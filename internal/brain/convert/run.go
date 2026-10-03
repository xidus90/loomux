package convert

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/model"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// openness orders the privacy modes from closed to open (run.py:34-36).
var openness = map[string]int{"local_only": 0, "manual_cloud": 1, "automatic_cloud": 2}

// Area is a registered area as convert sees it: its declaration, its inbox
// and the privacy mode it declares.
type Area struct {
	config.Area
	Manifest *config.Manifest // nil where the area declares nothing
	Inbox    string           // "" where the declaration names none
	Mode     string
}

// Areas reads every area's declaration before a file is touched: one broken
// declaration stops the run, as reading the registry stops the reference's.
// An area without a declaration has no inbox and counts as manual_cloud.
func Areas(areas []config.Area, stateDir string) ([]Area, error) {
	out := make([]Area, 0, len(areas))
	for _, a := range areas {
		entry := Area{Area: a, Mode: "manual_cloud"}
		manifest, err := config.ReadAreaDeclaration(config.ManifestDir(a, stateDir))
		switch {
		case config.IsUndeclared(err):
		case err != nil:
			return nil, err
		default:
			rel, err := manifest.InboxLayout()
			if err != nil {
				return nil, err
			}
			entry.Manifest, entry.Mode = manifest, manifest.PrivacyMode
			if rel != "" {
				entry.Inbox = filepath.Join(a.Path, rel)
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// Outcome is what a run wrote, what it left for a person, and where it
// would file what it wrote; a suggestion is a line, not a move.
type Outcome struct {
	Written, Skipped, Suggested []string
}

type describer interface {
	Describe(context.Context, string) (string, bool)
}

type placer interface {
	Place(context.Context, string, []string) (string, bool)
}

// statSource is the seam a test replaces to see a failing stat.
var statSource = os.Stat

// readDir is the seam a test replaces to see an inbox that cannot be listed;
// a deny on the directory does not stop an account that lists anyway, such as
// an administrator on a CI runner.
var readDir = os.ReadDir

type run struct {
	ctx context.Context
	pdf *pdftotext
}

// ConvertFile converts one named path without asking the model: it belongs
// to no area, and the next run over the inboxes fills the sentence in.
func ConvertFile(ctx context.Context, tools Tools, path string) (string, string) {
	r := &run{ctx: ctx, pdf: newPDFToText(tools)}
	target, message, _ := r.one(path, nil, nil, nil)
	return target, message
}

// ConvertAll goes through every writable inbox in registry order. The model
// settings are read and the proposers built before the first file, so a
// misconfiguration stops the run with nothing converted and nothing left
// unlisted. No error in a single file stops it. An inbox that cannot be
// listed does: the Outcome returned with that error holds what earlier
// inboxes wrote, and the caller prints it before the error.
func ConvertAll(ctx context.Context, tools Tools, areas []Area, stateDir string) (Outcome, error) {
	settings, err := config.ReadModelSettings(stateDir)
	if err != nil {
		return Outcome{}, err
	}
	var usable []Area
	for _, a := range areas {
		if a.Inbox != "" && !a.ReadOnly && isDir(a.Inbox) {
			usable = append(usable, a)
		}
	}
	// A nil *model.Proposer in an interface would not be nil, so only a
	// proposer that exists goes into the maps.
	describers, placers := map[string]describer{}, map[string]placer{}
	if settings.Enabled {
		for _, a := range usable {
			d, err := model.ProposerFor(settings, a.Manifest, "describe")
			if err != nil {
				return Outcome{}, err
			}
			if d != nil {
				describers[a.Scope] = d
			}
			p, err := model.ProposerFor(settings, a.Manifest, "place")
			if err != nil {
				return Outcome{}, err
			}
			if p != nil {
				placers[a.Scope] = p
			}
		}
	}
	// The register, less what nobody may file into, in registry order.
	var register []Area
	for _, a := range areas {
		if !a.ReadOnly {
			register = append(register, a)
		}
	}
	r := &run{ctx: ctx, pdf: newPDFToText(tools)}
	var out Outcome
	for _, a := range usable {
		pl := placers[a.Scope]
		var scopes []string
		if pl != nil {
			for _, target := range register {
				if openness[target.Mode] <= openness[a.Mode] {
					scopes = append(scopes, target.Scope)
				}
			}
		}
		entries, err := readDir(a.Inbox)
		if err != nil {
			return out, err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		for _, name := range sortedOn(runtime.GOOS, names) {
			path := filepath.Join(a.Inbox, name)
			if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() || strings.EqualFold(pySuffix(name), ".md") {
				continue
			}
			target, message, suggestion := r.one(path, describers[a.Scope], pl, scopes)
			if target != "" {
				out.Written = append(out.Written, target)
			}
			if message != "" {
				out.Skipped = append(out.Skipped, message)
			}
			if suggestion != "" {
				out.Suggested = append(out.Suggested, suggestion)
			}
		}
	}
	return out, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// sortedOn is sorted(inbox.iterdir()): Python compares paths in lower case
// under Windows, by code point elsewhere.
func sortedOn(goos string, names []string) []string {
	key := func(s string) string { return s }
	if goos == "windows" {
		key = strings.ToLower
	}
	slices.SortStableFunc(names, func(a, b string) int { return strings.Compare(key(a), key(b)) })
	return names
}

// one converts path and answers the target it wrote ("" where nothing was
// written), what it left behind, and where the written file belongs.
func (r *run) one(path string, d describer, p placer, scopes []string) (string, string, string) {
	name := filepath.Base(path)
	format := Detect(path)
	if format == Unsupported {
		return "", name + ": no converter knows this format", ""
	}
	// The source's extension rides along: doku.pdf and doku.txt must not
	// overwrite each other.
	targetName := name + ".md"
	target := filepath.Join(filepath.Dir(path), targetName)
	kept := ""
	if _, err := os.Stat(target); err == nil {
		existing, err := pytext.ReadText(target)
		if err != nil {
			return "", targetName + ": cannot be read as UTF-8 text", ""
		}
		if _, ours := ConvertedBy(existing); !ours {
			return "", targetName + ": not written by us, left untouched", ""
		}
		// A standing sentence is never traded for another; a head without
		// one is asked again on every run.
		kept, _ = DescriptionOf(existing)
	}
	var body, converter, skip string
	asr := false
	if format == PDF {
		e, err := r.pdf.extract(path)
		var tool *toolError
		switch {
		case errors.As(err, &tool):
			return "", name + ": " + tool.Error(), ""
		case err != nil:
			return "", fmt.Sprintf("%s: cannot be read as a PDF (%v)", name, err), ""
		case e.text == "" && e.pages == 0:
			return "", name + ": no pages to extract", ""
		case e.text == "":
			return "", name + ": no extractable text, looks like a scan", ""
		}
		if e.skipped > 0 {
			skip = fmt.Sprintf("%s: %d page(s) skipped as scanned", name, e.skipped)
		}
		body, converter = e.text, PDFConverter
	} else {
		source, err := pytext.ReadText(path)
		if err != nil {
			return "", name + ": cannot be read as UTF-8 text", ""
		}
		body, converter, asr = ToParagraphs(source, format), TranscriptConverter, true
	}
	info, err := statSource(path)
	if err != nil {
		return "", fmt.Sprintf("%s: cannot be read (%v)", name, err), ""
	}
	description := kept
	if description == "" && d != nil {
		description, _ = d.Describe(r.ctx, body)
	}
	head := Head{SourceURL: SourceURLFrom(name), Retrieved: info.ModTime().UTC(), Converter: converter, ASR: asr, Description: description}
	written, err := writeIfChanged(target, head.String()+body+"\n")
	if err != nil {
		return "", fmt.Sprintf("%s: cannot be written (%v)", targetName, err), ""
	}
	if !written {
		return "", skip, ""
	}
	// Only a file this run wrote is placed: every file stays in its inbox,
	// and asking on every run would repeat the line for every file waiting.
	if p == nil {
		return target, skip, ""
	}
	scope, ok := p.Place(r.ctx, body, scopes)
	if !ok {
		return target, skip, ""
	}
	return target, skip, targetName + ": belongs in " + scope + ", left in the inbox"
}

// writeIfChanged writes only on change, so an untouched file keeps its
// time; the comparison is byte for byte, as `newline=""` reads.
func writeIfChanged(path, text string) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && string(existing) == text {
		return false, nil
	}
	return true, lock.ReplaceText(path, text)
}
