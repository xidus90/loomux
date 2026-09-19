package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

var knownLegacyHooks = []struct {
	scriptName string
	reason     string
}{
	{"ulguard", "superseded by 'loomux hook pre-tool-use' and 'loomux hook post-tool-use'"},
	{"brain guard", "merged into 'loomux hook pre-tool-use'"},
	{"guard_paths.py", "superseded by 'loomux hook pre-tool-use'"},
	{"format_on_edit.py", "superseded by 'loomux hook post-tool-use'"},
	{"post_edit.py", "superseded by 'loomux hook post-tool-use'"},
	{"wiki_gate.py", "superseded by 'loomux wiki-gate' in Stop hook"},
	{"generate_index.py", "superseded by ultra-brain catalog/reindex and wiki-gate"},
	{"lint.py", "superseded by 'loomux lint' and 'loomux hook post-tool-use'"},
}

type LegacyFinding struct {
	Event   string
	Command string
	Reason  string
}

func auditSettings(root string) ([]LegacyFinding, bool, bool) {
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, false, false
	}

	var parsed struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, false, false
	}

	hasPreGuard := false
	hasPostEdit := false
	var findings []LegacyFinding

	for event, entries := range parsed.Hooks {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				cmd := h.Command
				if strings.Contains(cmd, "loomux") && strings.Contains(cmd, "pre-tool-use") {
					hasPreGuard = true
				}
				if strings.Contains(cmd, "loomux") && strings.Contains(cmd, "post-tool-use") {
					hasPostEdit = true
				}
				for _, leg := range knownLegacyHooks {
					if strings.Contains(cmd, leg.scriptName) {
						findings = append(findings, LegacyFinding{
							Event:   event,
							Command: cmd,
							Reason:  leg.reason,
						})
					}
				}
			}
		}
	}

	return findings, hasPreGuard, hasPostEdit
}

//coverage:exempt filepath.Abs fails only when os.Getwd does, which no test on the platforms this runs on can provoke
func Status(stdout io.Writer, stderr io.Writer, root string) int {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}

	facts := detect.Detect(os.DirFS(root))
	facts.Stacks = stacksWithWiki(facts.Stacks, root)
	// The same precedence the hook uses: the manifest's [layout] wiki first,
	// then detection, then wiki/. Reading detection first named a directory the
	// manifest does not mean.
	wikiDir := wikiDirFor(root)

	hasStack := func(s string) bool {
		for _, stack := range facts.Stacks {
			if stack == s {
				return true
			}
		}
		return false
	}

	fmt.Fprintln(stdout, "================================================================================")
	fmt.Fprintln(stdout, " loomux Hook Inspection")
	fmt.Fprintln(stdout, "================================================================================")
	fmt.Fprintf(stdout, "Project Root:    %s\n", absRoot)
	fmt.Fprintf(stdout, "Detected Stacks: %v\n", facts.Stacks)

	// From the stacks list, which is what decides the lane below: reading
	// facts.WikiMode instead let this report list the stack `wiki`, print the
	// wiki lane, and call the wiki disabled in between -- detection does not
	// read the one key a loomux project declares its bundle with. The mode is
	// gone with it; it was "brain" wherever it was set at all and empty for
	// every wiki the manifest declares, so it distinguished nothing.
	if hasStack("wiki") {
		fmt.Fprintf(stdout, "UltraBrain Wiki: Active (Bundle Directory: '%s')\n", wikiDir)
	} else {
		fmt.Fprintln(stdout, "UltraBrain Wiki: Inactive / Disabled (default)")
	}

	fmt.Fprintln(stdout, "\n--------------------------------------------------------------------------------")
	fmt.Fprintln(stdout, " Configured Lifecycle Hooks & Concurrent Tools")
	fmt.Fprintln(stdout, "--------------------------------------------------------------------------------")

	fmt.Fprintln(stdout, "[PreToolUse] (Matcher: Write|Edit|NotebookEdit|Bash|PowerShell)")
	fmt.Fprintln(stdout, "  -> loomux hook pre-tool-use (policy and write barrier)")

	fmt.Fprintln(stdout, "\n[PostToolUse] (Matcher: Write|Edit|NotebookEdit)")
	// The lanes as post-edit plans them, from [verify] and the presets: a
	// second list here drifted from what ran.
	eff, lanesErr := editLoad(root, facts)
	writeEditLanes(stdout, eff, lanesErr)
	if hasStack("wiki") && eff.Config.Stacks["wiki"]["lint"].Lane.Off {
		fmt.Fprintf(stdout, "     * *.md (in %s): off [config]\n", wikiDir)
	} else if hasStack("wiki") {
		fmt.Fprintf(stdout, "     * *.md (in %s): loomux lint <target-file>\n", wikiDir)
		fmt.Fprintln(stdout, "     * *.md (outside): [SKIPPED] Instant 0ms exit")
	} else {
		fmt.Fprintln(stdout, "     * *.md:           [SKIPPED] Instant 0ms exit (Wiki disabled)")
	}
	writeIgnored(stdout, eff.Ignored)

	fmt.Fprintln(stdout, "\n[Stop] (Session End Gate)")
	if hasStack("wiki") {
		fmt.Fprintln(stdout, "  -> loomux wiki-gate --root \"${CLAUDE_PROJECT_DIR}\" (Git-Drift & OKF Bundle Validation)")
	} else {
		fmt.Fprintln(stdout, "  -> No Stop-Hook configured (Wiki disabled)")
	}

	fmt.Fprintln(stdout, "\n--------------------------------------------------------------------------------")
	fmt.Fprintln(stdout, " Hook Audit & Redundancy Check (.claude/settings.json)")
	fmt.Fprintln(stdout, "--------------------------------------------------------------------------------")

	findings, hasPre, hasPost := auditSettings(root)
	if hasPre {
		fmt.Fprintln(stdout, " [OK] PreToolUse:  'loomux hook pre-tool-use' installed")
	} else {
		fmt.Fprintln(stdout, " [INFO] PreToolUse: 'loomux hook pre-tool-use' not found in .claude/settings.json")
	}

	if hasPost {
		fmt.Fprintln(stdout, " [OK] PostToolUse: 'loomux hook post-tool-use' installed")
	} else {
		fmt.Fprintln(stdout, " [INFO] PostToolUse: 'loomux hook post-tool-use' not found in .claude/settings.json")
	}

	if len(findings) == 0 {
		fmt.Fprintln(stdout, " [OK] No obsolete or redundant legacy hooks found.")
	} else {
		fmt.Fprintln(stdout, " [WARNING] Redundant or obsolete legacy hooks detected in .claude/settings.json:")
		for _, f := range findings {
			fmt.Fprintf(stdout, "   • [%s] %s\n     Reason: %s\n", f.Event, f.Command, f.Reason)
		}
	}
	renderLaneTools(stdout, unavailableLanes(eff, exec.LookPath))

	fmt.Fprintln(stdout, "================================================================================")

	return ExitOK
}

// unavailableLanes names every tool a configured lane would start that this
// machine does not have.
//
// Built from the lanes [verify] and the presets lay out -- what the hook and
// `loomux check` run -- and not from the lane list printed above it. A second
// list drifts, and this one has to be about the lanes that actually run.
//
// This report is the standing answer to the same question the hook answers
// per edit: a lane whose tool is missing is dropped there and named as it
// goes; here the whole set is listed at once, before an edit rather than
// after one. Without lanes -- a config they cannot be read from -- it lists
// nothing; the report names that error above.
func unavailableLanes(eff verify.Effective, look func(string) (string, error)) []string {
	seen := map[string]bool{}
	var missing []string
	for _, stack := range eff.Active {
		for _, kind := range verify.Kinds() {
			lane := eff.Stacks[stack][kind].Lane
			for _, command := range slices.Concat(lane.Commands, lane.OnFile, []string{lane.Measuring, lane.Measure}) {
				fields := strings.Fields(command)
				// {loomux} is this binary, which is running; the wiki lane
				// runs in this process and has no command at all.
				if len(fields) == 0 || fields[0] == "{loomux}" || seen[fields[0]] {
					continue
				}
				seen[fields[0]] = true
				if _, err := look(fields[0]); err != nil {
					missing = append(missing, fields[0])
				}
			}
		}
	}
	sort.Strings(missing)
	return missing
}

// writeEditLanes lists the lanes of the `edit` profile for every active
// stack: the on-file form where a lane has one, since that is what an edit
// runs, and the layer that shaped it. A lane that is not defined runs nothing
// and is left out; one switched off says so.
func writeEditLanes(w io.Writer, eff verify.Effective, err error) {
	if err != nil {
		fmt.Fprintf(w, "  -> loomux hook post-tool-use:\n     [ERROR] the lanes cannot be read: %v\n", err)
		return
	}
	// The defaults hold an edit profile and a config can only replace it.
	kinds, _ := verify.ExpandProfile(eff.Config, "edit")
	fmt.Fprintf(w, "  -> loomux hook post-tool-use (profile `edit`: %s):\n", strings.Join(kinds, ", "))
	for _, stack := range eff.Active {
		globs := []string{}
		for _, ext := range slices.Sorted(maps.Keys(eff.Extensions)) {
			if eff.Extensions[ext] == stack {
				globs = append(globs, "*"+ext)
			}
		}
		for _, kind := range kinds {
			r := eff.Stacks[stack][kind]
			head := fmt.Sprintf("     * %s (%s) %s:", stack, strings.Join(globs, ", "), kind)
			if r.Lane.Off {
				fmt.Fprintf(w, "%s off [%s]\n", head, r.Origin)
				continue
			}
			cmds := r.Lane.OnFile
			if len(cmds) == 0 && stack != "project" {
				cmds = r.Lane.Commands
			}
			if !r.Defined || len(cmds) == 0 {
				continue
			}
			tags := r.Origin
			if r.Lane.Threaded {
				tags += ", parallel"
			}
			fmt.Fprintf(w, "%s %s [%s]\n", head, strings.Join(cmds, " ; "), tags)
		}
	}
}

// writeIgnored names the extensions post-edit leaves alone.
func writeIgnored(w io.Writer, ignored []string) {
	if len(ignored) == 0 {
		return
	}
	fmt.Fprintf(w, "     * ignored (%s): [SKIPPED] no lane runs\n", strings.Join(ignored, ", "))
}

func renderLaneTools(stdout io.Writer, missing []string) {
	fmt.Fprintln(stdout, "\n--------------------------------------------------------------------------------")
	fmt.Fprintln(stdout, " Lane Tools On This Machine")
	fmt.Fprintln(stdout, "--------------------------------------------------------------------------------")
	if len(missing) == 0 {
		fmt.Fprintln(stdout, " [OK] Every configured lane's tool is on PATH.")
		return
	}
	for _, tool := range missing {
		fmt.Fprintf(stdout, " [WARN] %s is not on PATH: its lane is configured and skipped on every edit.\n", tool)
	}
}
