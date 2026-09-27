package load

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
)

// Dir is where a project keeps its flows, one folder each, below the root.
const Dir = ".loomux/flows"

// Where a found flow came from, as list, show and the marker spell it.
const (
	OriginProject = "project"
	OriginBundled = "bundled"
	OriginOverlay = "bundled+overlay"
	OriginHides   = "project (hides bundled)"
)

// Found is a flow located by name and ready to load.
type Found struct {
	Name     string
	Origin   string
	File     string   // how messages name its flow.toml
	Files    fs.FS    // the flow's folder as a run reads it; see flow.Graph.Texts
	Overlays []string // the project's files laid over a bundled flow, sorted
	Warnings []string // what of the project was ignored for a bundled flow, and why
}

// Find locates a flow: a project folder under Dir, a bundled one, or a
// bundled one with the project's files laid over it. A project may hide or
// overlay a bundled flow only when [flow] overrides names it; otherwise the
// bundled flow runs and the project folder is named in a warning. That is the
// guard's second line: a folder an agent wrote cannot take a bundled flow's
// gates away, whichever binary judged the write.
func Find(root string, bundled fs.FS, settings config.FlowSettings, name string) (Found, error) {
	if !config.IsFlowName(name) {
		return Found{}, fmt.Errorf("%q is not a flow name; a flow name is %s", name, config.FlowNameRule)
	}
	isBundled := exists(bundled, name+"/flow.toml")
	shown := Dir + "/" + name
	// What the project holds is read from Dir's own listing, so a folder
	// counts only when it is spelled exactly like the flow: a case-blind file
	// system would otherwise find Example for example on one OS and not on
	// the next.
	folders, files, err := underDir(root)
	isFolder, isFile := slices.Contains(folders, name), slices.Contains(files, name)
	switch {
	case isBundled && !settings.Allows(name):
		// Decided before the folder is judged: a folder the project may not
		// use cannot stop the bundled flow either, whatever it holds.
		found := bundledFound(bundled, name)
		switch {
		case err != nil:
			found.Warnings = []string{err.Error()}
		case isFolder || isFile:
			found.Warnings = []string{shown + " is ignored: [flow] overrides does not name it"}
		}
		return found, nil
	case err != nil:
		return Found{}, err
	case isFile:
		return Found{}, errors.New(notAFolder(root, name))
	case !isFolder && isBundled:
		return bundledFound(bundled, name), nil
	case !isFolder:
		return Found{}, fmt.Errorf("no flow named %q; known flows: %s", name, offer(Names(root, bundled)))
	}
	folder := filepath.Join(root, filepath.FromSlash(Dir), name)
	full, overlays, err := contents(os.DirFS(folder), shown)
	switch {
	case err != nil:
		return Found{}, err
	case full && !isBundled:
		return Found{Name: name, Origin: OriginProject, File: shown + "/flow.toml", Files: os.DirFS(folder)}, nil
	case !isBundled:
		return Found{}, fmt.Errorf("%s overlays nothing: there is no bundled flow %q", shown, name)
	case full:
		return Found{Name: name, Origin: OriginHides, File: shown + "/flow.toml", Files: os.DirFS(folder)}, nil
	}
	for _, file := range overlays {
		if !exists(bundled, name+"/"+file) {
			return Found{}, fmt.Errorf("%s overlays nothing: bundled flow %q has no %s", shown, name, file)
		}
	}
	base := bundledFound(bundled, name)
	base.Origin, base.Overlays = OriginOverlay, overlays
	base.Files = overlayFS{base: base.Files, top: os.DirFS(folder), files: overlays}
	return base, nil
}

// bundledFound is the catalog's flow of that name, as it ships.
func bundledFound(bundled fs.FS, name string) Found {
	// Sub fails only on a name that is no valid path, and IsFlowName let none
	// of those through.
	files, _ := fs.Sub(bundled, name)
	return Found{Name: name, Origin: OriginBundled, File: "bundled:" + name + "/flow.toml", Files: files}
}

func exists(fsys fs.FS, path string) bool {
	_, err := fs.Stat(fsys, path)
	return err == nil
}

// notAFolder is how a message names what stands directly under Dir and is no
// folder: a file, or a link -- a symbolic link, or a junction, which Windows
// lists as an irregular entry. A link is refused like a file, and named as a
// link so that the message does not call a folder somewhere else a file.
func notAFolder(root, name string) string {
	shown := Dir + "/" + name
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(Dir), name))
	if err == nil && info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
		return shown + " is a link; a flow is a folder with flow.toml"
	}
	return shown + " is a file; a flow is a folder with flow.toml"
}

// hidden names are nobody's flow: .gitkeep, .DS_Store, an editor's .draft/.
// The loader passes over a name starting with "." wherever it looks -- under
// Dir and inside a project folder -- as go:embed does for the catalog.
func hidden(name string) bool { return strings.HasPrefix(name, ".") }

// contents says what a project folder holds: a flow of its own (full, it has
// a flow.toml) or the files it lays over a bundled flow. A flow of its own
// may also carry the README.md and _test/ a catalog flow has, so a copied
// template loads unchanged; the loader reads neither. Anything else is
// refused by name, so a stray README beside an overlay is a message and not a
// silent overlay. Which files are stray is known only after the walk: a
// README.md sorts before the flow.toml that allows it.
func contents(folder fs.FS, shown string) (bool, []string, error) {
	var files, others []string
	full := false
	err := fs.WalkDir(folder, ".", func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case path != "." && hidden(entry.Name()) && entry.IsDir():
			return fs.SkipDir
		case entry.IsDir() || hidden(entry.Name()):
			return nil
		}
		switch {
		case path == "flow.toml":
			full = true
		case strings.HasPrefix(path, "instructions/") || strings.HasPrefix(path, "questions/"):
			files = append(files, path)
		default:
			others = append(others, path)
		}
		return nil
	})
	switch {
	case err != nil:
		// The file system names only a path inside the folder (`open .`).
		return false, nil, fmt.Errorf("%s cannot be read: %w", shown, err)
	case full:
		for _, path := range others {
			if path != "README.md" && !strings.HasPrefix(path, "_test/") {
				return false, nil, fmt.Errorf("%s/%s is neither flow.toml, README.md nor a file under instructions/, questions/ or _test/", shown, path)
			}
		}
		return true, nil, nil
	case len(others) > 0:
		return false, nil, fmt.Errorf("%s/%s is neither flow.toml nor a file under instructions/ or questions/", shown, others[0])
	case len(files) == 0:
		return false, nil, fmt.Errorf("%s holds no flow.toml and nothing to overlay", shown)
	}
	slices.Sort(files)
	return false, files, nil
}

// Names are the flows a project can name: every folder under Dir and every
// flow of the catalog, sorted and each once. A folder is named whatever it
// holds, so that Find can say what is wrong with it; a missing Dir is a
// project without flows of its own.
func Names(root string, bundled fs.FS) []string {
	names, _, _ := underDir(root) // Find names a Dir it cannot read
	shipped, _ := fs.ReadDir(bundled, ".")
	for _, entry := range shipped {
		if entry.IsDir() && exists(bundled, entry.Name()+"/flow.toml") {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// underDir splits what a project keeps directly under Dir into folders and
// files, as the listing spells them, hidden names left out; a link counts
// among the files. A missing Dir is a project without flows of its own; one
// that cannot be listed -- a file, or a folder the process may not read -- is
// named. Stat tells a file from an absent Dir: Windows lists a file as a path
// it cannot find, POSIX as no directory.
func underDir(root string) (folders, files []string, err error) {
	dir := filepath.Join(root, filepath.FromSlash(Dir))
	held, err := os.ReadDir(dir)
	info, statErr := os.Stat(dir)
	switch {
	case errors.Is(statErr, fs.ErrNotExist):
		return nil, nil, nil
	case statErr == nil && !info.IsDir():
		err = errors.New("it is a file")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s cannot be read as a folder of flows: %w", Dir, err)
	}
	for _, entry := range held {
		switch {
		case hidden(entry.Name()):
		case entry.IsDir():
			folders = append(folders, entry.Name())
		default:
			files = append(files, entry.Name())
		}
	}
	return folders, files, nil
}

// Entry is one flow a project can name, as list shows it.
type Entry struct {
	Name     string
	Origin   string   // empty when Find refused the folder, or for a file that is no flow
	Problem  string   // empty when the flow loads; why it does not otherwise
	Default  bool     // [flow] default names it
	Warnings []string // what of the project was ignored for a bundled flow, and why
}

// List names every flow of the project and the catalog once, sorted, each
// found and loaded the way a run would. A flow that will not load is listed
// with the reason rather than left out: a flow that disappears from the list
// reads as a flow nobody wrote. For the same reason a file directly under Dir
// -- a single-file flow of an older format, say -- is listed by its file name
// with the reason it is no flow.
func List(root string, bundled fs.FS, settings config.FlowSettings, catalog *flow.Catalog) []Entry {
	names := Names(root, bundled)
	_, files, _ := underDir(root) // Find names a Dir it cannot read
	entries := make([]Entry, 0, len(names)+len(files))
	for _, file := range files {
		// A file named like a bundled flow is that flow's entry: Find says
		// what is wrong with it.
		if !slices.Contains(names, file) {
			entries = append(entries, Entry{Name: file, Problem: notAFolder(root, file)})
		}
	}
	for _, name := range names {
		entry := Entry{Name: name, Default: name == settings.Default}
		found, err := Find(root, bundled, settings, name)
		if err == nil {
			entry.Origin, entry.Warnings = found.Origin, found.Warnings
			_, err = Load(found, catalog)
		}
		if err != nil {
			entry.Problem = err.Error()
		}
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	return entries
}

// SettingsWarnings name the overrides of [flow] that point at a flow the
// catalog does not ship. The config check cannot see them, for it reads no
// flows. A default that names no flow is no warning: MissingDefault refuses
// it.
func SettingsWarnings(bundled fs.FS, settings config.FlowSettings) []string {
	var warnings []string
	for _, name := range settings.Overrides {
		if !exists(bundled, name+"/flow.toml") {
			warnings = append(warnings, fmt.Sprintf("[flow] overrides names %q, which no bundled flow has", name))
		}
	}
	return warnings
}

// MissingDefault refuses a [flow] default that is no flow of the project or
// the catalog, and is nil for an unset default. It is a refusal and not a
// warning because a run without a name would start nothing, and every command
// that reads the default says so in these words. The config check cannot see
// it either.
func MissingDefault(root string, bundled fs.FS, settings config.FlowSettings) error {
	if settings.Default == "" || slices.Contains(Names(root, bundled), settings.Default) {
		return nil
	}
	return fmt.Errorf("[flow] default names %q, which is no flow here", settings.Default)
}

func offer(found []string) string {
	if len(found) == 0 {
		return "none"
	}
	return strings.Join(found, ", ")
}
