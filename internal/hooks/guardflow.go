package hooks

import (
	"regexp"
	"slices"
	"strings"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/config"
)

// runFilesReason refuses an agent a run's journal and marker, by path and by
// shell: an answered entry it wrote itself would hand the next resume without
// an answer a gate no human answered.
const runFilesReason = "a flow's journal and marker are written by loomux, not by the party the gates ask"

// bundledFlows are the names of the catalog this binary carries; a test
// replaces it to see the guard with an empty one.
var bundledFlows = flows.Names

// answersAGate says whether a shell line runs `loomux flow resume` with an
// answer. A gate asks a human; an agent that answered its own gates would
// approve its own plan and its own push. The line is read the way
// writesConfiguration reads it, and has the same holes.
func answersAGate(line string) bool {
	for _, variant := range lineVariants(line) {
		for _, segment := range segments(variant) {
			for _, words := range readings(segment) {
				if readingAnswers(words) {
					return true
				}
			}
		}
	}
	return false
}

// readingAnswers judges one reading from its head and from every word after
// a lone { or }, for readingWrites' reason: a block opens a command.
func readingAnswers(words []string) bool {
	for i, w := range words {
		if (w == "{" || w == "}") && wordsAnswer(words[i+1:]) {
			return true
		}
	}
	return wordsAnswer(words)
}

// wordsAnswer says whether one reading, past what runs in front of the
// program, is loomux flow resume with an answer, or a Start-Process of
// loomux, whose arguments the words cannot see.
func wordsAnswer(words []string) bool {
	words = dropPrefixes(words)
	if len(words) == 0 {
		return false
	}
	if startsLoomux(words) {
		return true
	}
	args, ok := loomuxArgs(words)
	return ok && len(args) > 1 && args[0] == "flow" && args[1] == "resume" && namesAnswer(args[2:])
}

// namesAnswer is any spelling of --answer the flag package takes.
func namesAnswer(args []string) bool {
	for _, arg := range args {
		name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		if name != arg && (name == "answer" || strings.HasPrefix(name, "answer=")) {
			return true
		}
	}
	return false
}

// protectedFlows are the flows whose folder under .loomux/flows only a human
// writes: the catalog of this binary, and every name [flow] overrides lets a
// project flow hide or overlay. The config holds the second half because the
// guard's catalog need not be the one the run loads. They come sorted, each
// once: a catalog name in overrides is how a project lets its overlay run.
func protectedFlows(root string) ([]string, error) {
	settings, err := config.ReadFlowSettings(root)
	names := append(bundledFlows(), settings.Overrides...)
	slices.Sort(names)
	return slices.Compact(names), err
}

// bundledFlowReason refuses a write into the folder of the flow name.
func bundledFlowReason(name string) string {
	return "a bundled flow's gates and instructions are a human's to change; give your flow a name of its own, or ask the user to hide or overlay `" + name + "`"
}

// unreadableFlowsReason refuses every write under .loomux/flows: a [flow] that
// will not read may name any folder there.
func unreadableFlowsReason(err error) string {
	return "loomux cannot read [flow] of .loomux/config.toml, so it refuses writes under .loomux/flows: " + err.Error()
}

// flowFolderReasons judges a write to rel, a path relative to the project,
// against the protected flows. The folder name is compared in lower case:
// Windows and macOS keep Example and example as one folder. The config is
// read only for a path under .loomux/flows, as ignoredFlowFolders reads it.
func flowFolderReasons(root, rel string) []string {
	rest, under := strings.CutPrefix(strings.ToLower(rel), flowsDir+"/")
	if !under {
		return nil
	}
	name, _, _ := strings.Cut(rest, "/")
	protected, err := protectedFlows(root)
	if err != nil {
		return []string{unreadableFlowsReason(err)}
	}
	if at := slices.IndexFunc(protected, func(p string) bool { return strings.ToLower(p) == name }); at >= 0 {
		return []string{bundledFlowReason(protected[at])}
	}
	return nil
}

// flowFolderCommand is the rule for a shell line that writes or removes a
// protected flow's folder or a file in it, in writeSource's forms and so in
// any case. It is built per call, since the names depend on the project's
// config, and answers false when no flow is protected: an empty list of
// names would match every folder under .loomux/flows.
//
// A config that will not read gives a rule over every name, with that
// reason, as flowFolderReasons refuses every path.
func flowFolderCommand(root string) (config.CommandRule, bool) {
	protected, err := protectedFlows(root)
	names, reason := `[^\s;|&'"<>/\\]+`, ""
	switch {
	case err != nil:
		reason = unreadableFlowsReason(err)
	case len(protected) == 0:
		return config.CommandRule{}, false
	default:
		quoted := make([]string, len(protected))
		for i, name := range protected {
			quoted[i] = regexp.QuoteMeta(name)
		}
		names = strings.Join(quoted, "|")
		reason = bundledFlowReason(strings.Join(protected, "`, `"))
	}
	// A glob in a flow's place reaches the flows for every verb: the shell
	// expands it onto their folders, and an agent writing a flow of its own
	// spells its name. A removal also reaches them through .loomux/flows
	// itself, so that a copy into it stays open.
	source := writeSource(
		`['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+flows[/\\]+(?:`+names+`|`+globName+`)(?:[/\\][^\s;|&'"<>]*)?['"]?`,
		`['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+flows(?:[/\\]+(?:`+names+`|`+globName+`)(?:[/\\][^\s;|&'"<>]*)?)?[/\\]*['"]?`)
	return config.CommandRule{Regex: regexp.MustCompile(source), Source: source, Reason: reason}, true
}
