// Package programs names the external programs loomux calls and the command
// that installs each. loomux never runs the command; init and convert name
// it.
package programs

// Program is one external program and its installer.
type Program struct{ Name, Install string }

// All are the programs in the order init checks them. The winget ids were
// each confirmed with `winget search` on 2026-09-24; pdftotext comes with
// Poppler.
func All() []Program {
	return []Program{
		{"git", "winget install --id Git.Git -e"},
		// winget has no qmd; the command is the one of the project's README,
		// https://github.com/tobi/qmd (read 2026-09-24).
		{"qmd", "npm install -g @tobilu/qmd"},
		{"pdftotext", "winget install --id oschwartz10612.Poppler -e"},
		{"yt-dlp", "winget install --id yt-dlp.yt-dlp -e"},
		{"ollama", "winget install --id Ollama.Ollama -e"},
	}
}

// Install is the command that installs name, or "" for a program loomux
// does not name.
func Install(name string) string {
	for _, p := range All() {
		if p.Name == name {
			return p.Install
		}
	}
	return ""
}
