package hooks

import (
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
// writesConfiguration reads it, and has the same holes; with anyProgram, in
// strict mode, every program knownProgram does not name counts as loomux.
func answersAGate(line string, anyProgram bool) bool {
	for _, variant := range lineVariants(line) {
		for _, segment := range segments(variant) {
			for _, words := range readings(segment) {
				if readingAnswers(words, anyProgram) {
					return true
				}
			}
		}
	}
	return false
}

// readingAnswers judges one reading from its head and from every word after
// a lone { or }, for readingWrites' reason: a block opens a command.
func readingAnswers(words []string, anyProgram bool) bool {
	for i, w := range words {
		if (w == "{" || w == "}") && wordsAnswer(words[i+1:], anyProgram) {
			return true
		}
	}
	return wordsAnswer(words, anyProgram)
}

// wordsAnswer says whether one reading, past what runs in front of the
// program, is loomux flow resume with an answer, or a Start-Process of
// loomux, whose arguments the words cannot see.
func wordsAnswer(words []string, anyProgram bool) bool {
	words = dropPrefixes(words)
	if len(words) == 0 {
		return false
	}
	if startsLoomux(words) {
		return true
	}
	args, ok := programArgs(words, anyProgram)
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

// flowFolderReasons judges a write to rel against the protected flows: rel
// under a .loomux/flows folder, the project's or one under any directory, the
// way the built-in rules keep .loomux. Every such folder in rel counts, the
// outer and the inner alike, and each protected flow among them is named once.
// The folder name is compared in lower case: Windows and macOS keep Example
// and example as one folder. The flows are asked for only for such a path.
func flowFolderReasons(rel string, flows func() ([]string, error)) []string {
	names := flowFolderName(rel)
	if len(names) == 0 {
		return nil
	}
	protected, err := flows()
	if err != nil {
		return []string{unreadableFlowsReason(err)}
	}
	var reasons []string
	for _, name := range names {
		at := slices.IndexFunc(protected, func(p string) bool { return strings.ToLower(p) == name })
		if at >= 0 && !slices.Contains(reasons, bundledFlowReason(protected[at])) {
			reasons = append(reasons, bundledFlowReason(protected[at]))
		}
	}
	return reasons
}

// flowFolderName is, in path order and lower case, the element right below
// every .loomux/flows folder in rel; none when rel lies below no such folder.
func flowFolderName(rel string) []string {
	var names []string
	rest := "/" + strings.ToLower(rel)
	for {
		at := strings.Index(rest, "/"+flowsDir+"/")
		if at < 0 {
			return names
		}
		// The search goes on from the slash that closes this flows folder,
		// so a flows folder right inside the flow is found as well.
		rest = rest[at+len(flowsDir)+1:]
		name, _, _ := strings.Cut(rest[1:], "/")
		names = append(names, name)
	}
}
