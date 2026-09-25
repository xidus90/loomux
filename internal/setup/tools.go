package setup

import "os/exec"

// tool is a program loomux calls and the command that installs it. init
// never runs the command; it names it.
type tool struct{ name, install string }

// tools are checked on every run. The winget ids were each confirmed with
// `winget search` on 2026-09-24; pdftotext comes with Poppler.
func tools() []tool {
	return []tool{
		{"git", "winget install --id Git.Git -e"},
		// winget has no qmd; the command is the one of the project's README,
		// https://github.com/tobi/qmd (read 2026-09-24).
		{"qmd", "npm install -g @tobilu/qmd"},
		{"pdftotext", "winget install --id oschwartz10612.Poppler -e"},
		{"yt-dlp", "winget install --id yt-dlp.yt-dlp -e"},
		{"ollama", "winget install --id Ollama.Ollama -e"},
	}
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
