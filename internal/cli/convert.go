package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hosts"
)

// convertTools is the seam to pdftotext and yt-dlp; the replay of the
// recorded cases answers from the world's fixture instead.
var convertTools = convert.SystemTools()

// convertCommand turns the inboxes, or one named file, into Markdown. It
// writes into an area, so it stands at the top level beside reindex; the
// exit code is 1 as soon as anything is left for a person.
func convertCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux convert", flag.ContinueOnError)
	flags.SetOutput(stderr)
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return 2
	}
	if len(free) > 1 {
		fmt.Fprintf(stderr, "loomux convert: unrecognized arguments: %s\n", strings.Join(free[1:], " "))
		return 2
	}
	if code, off := brainModuleOff("convert", stderr); off {
		return code
	}
	ctx := context.Background()
	var outcome convert.Outcome
	var stopped error
	if len(free) == 1 {
		target, message := convert.ConvertFile(ctx, convertTools, free[0])
		if target != "" {
			outcome.Written = []string{target}
		}
		if message != "" {
			outcome.Skipped = []string{message}
		}
	} else {
		lookup := config.NewArtifactLookup()
		areas, err := config.ReadRegistry(lookup.Primary)
		if err != nil {
			return reportReconcileError(stderr, err)
		}
		entries, err := convert.Areas(areas, lookup.Primary)
		if err != nil {
			return reportReconcileError(stderr, err)
		}
		// An inbox that cannot be listed stops the run with what the inboxes
		// before it did; that is printed first, so nothing written goes
		// unlisted.
		outcome, stopped = convert.ConvertAll(ctx, convertTools, entries, lookup.Primary)
	}
	for _, written := range outcome.Written {
		fmt.Fprintln(stdout, written)
	}
	// Beside the written paths, not among the leftovers: a suggestion is no
	// work waiting.
	for _, suggestion := range outcome.Suggested {
		fmt.Fprintf(stdout, "suggested: %s\n", suggestion)
	}
	for _, message := range outcome.Skipped {
		fmt.Fprintf(stderr, "skipped: %s\n", message)
	}
	if stopped != nil {
		return reportReconcileError(stderr, stopped)
	}
	if len(outcome.Skipped) > 0 {
		return 1
	}
	return 0
}

// fetchCommand puts a video's subtitles into an area's inbox. The area must
// be writable: a read-only area takes no file, a fetched one included.
func fetchCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux fetch", flag.ContinueOnError)
	flags.SetOutput(stderr)
	scope := flags.String("scope", "knowledge", "which area's inbox receives the file")
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return 2
	}
	switch {
	case len(free) == 0:
		fmt.Fprintln(stderr, "loomux fetch: the following arguments are required: url")
		return 2
	case len(free) > 1:
		fmt.Fprintf(stderr, "loomux fetch: unrecognized arguments: %s\n", strings.Join(free[1:], " "))
		return 2
	case strings.HasPrefix(free[0], "-"):
		// yt-dlp would read it as an option, whatever loomux's `--` ended.
		fmt.Fprintf(stderr, "loomux fetch: a URL does not begin with '-': %s\n", free[0])
		return 2
	case namesOnlyAPlaylist(free[0]):
		// `--no-playlist` does not narrow it: yt-dlp would walk every entry
		// into the same names.
		fmt.Fprintf(stderr, "loomux fetch: a URL that names only a playlist is not fetched, give the URL of one video: %s\n", free[0])
		return 2
	}
	if code, off := brainModuleOff("fetch", stderr); off {
		return code
	}
	lookup := config.NewArtifactLookup()
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	entries, err := convert.Areas(areas, lookup.Primary)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	i := slices.IndexFunc(entries, func(a convert.Area) bool { return a.Scope == *scope })
	switch {
	case i < 0:
		return reportReconcileError(stderr, fmt.Errorf("no area named %s in the registry", pytext.Repr(*scope)))
	case entries[i].ReadOnly:
		return reportReconcileError(stderr, fmt.Errorf("area %s is read-only; fetch writes into no read-only area", pytext.Repr(*scope)))
	case entries[i].Inbox == "":
		return reportReconcileError(stderr, fmt.Errorf("area %s declares no inbox; add `[layout] inbox = ...` to its manifest", pytext.Repr(*scope)))
	}
	subs, err := convert.Fetch(convertTools, free[0])
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	target, err := convert.ToInbox(subs, free[0], entries[i].Inbox)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	fmt.Fprintln(stdout, target)
	return 0
}

// playlistHosts are the hosts that serve YouTube's playlist page.
var playlistHosts = []string{"youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com"}

// namesOnlyAPlaylist reports whether raw names a YouTube playlist and no
// video, which `--no-playlist` does not narrow: the playlist page, and a
// watch page with `list=` but no `v=`, which yt-dlp sends to the playlist
// page before it asks for `--no-playlist`. An address without a scheme is
// read as one with https, as a browser would.
func namesOnlyAPlaylist(raw string) bool {
	if !hasScheme(raw) {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || !slices.Contains(playlistHosts, strings.ToLower(u.Hostname())) {
		return false
	}
	switch strings.TrimSuffix(u.Path, "/") {
	case "/playlist":
		return true
	case "/watch":
		query := u.Query()
		return filled(query["list"]) && !filled(query["v"])
	}
	return false
}

// hasScheme reports whether raw begins with `<scheme>://`: a `://` further
// on, in a query value, does not make an address one with a scheme.
func hasScheme(raw string) bool {
	before, _, found := strings.Cut(raw, "://")
	return found && !strings.ContainsAny(before, "/?#")
}

// filled reports whether a query parameter has a value that is not empty,
// as yt-dlp reads it: its `parse_qs` drops blank values before it takes the
// first, so `v=&v=<id>` names a video and `v=` alone does not.
func filled(values []string) bool {
	return slices.ContainsFunc(values, func(value string) bool { return value != "" })
}

// brainModuleOff refuses a command of the brain module where the project
// found above the working directory switched the module off. Outside a
// project nothing is switched off.
func brainModuleOff(name string, stderr io.Writer) (int, bool) {
	root, err := hosts.FindRoot(".")
	if err != nil {
		return 0, false
	}
	modules, err := config.ReadModules(root)
	if err != nil {
		return reportReconcileError(stderr, err), true
	}
	if modules.Brain {
		return 0, false
	}
	fmt.Fprintf(stderr, "loomux %s: the brain module is off in %s ([modules] brain = false)\n", name, config.ManifestPath(root))
	return 1, true
}
