package setup

import (
	"os/exec"

	"github.com/xidus90/loomux/internal/programs"
)

// tool is a program loomux calls and the command that installs it. init
// never runs the command; it names it.
type tool struct{ name, install string }

// tools are checked on every run; the list lives in internal/programs,
// which convert reads as well.
func tools() []tool {
	var out []tool
	for _, p := range programs.All() {
		out = append(out, tool{p.Name, p.Install})
	}
	return out
}

// lookPath finds a program on the PATH; tests replace it so no result
// depends on what the machine has installed.
var lookPath = exec.LookPath

// missingTools is one note per tool that is not on the PATH.
func missingTools() []string {
	var notes []string
	for _, t := range tools() {
		if _, err := lookPath(t.name); err != nil {
			notes = append(notes, t.name+" is not on PATH; install it with: "+t.install)
		}
	}
	return notes
}
