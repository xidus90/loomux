package programs

import "testing"

func TestEveryProgramNamesItsInstaller(t *testing.T) {
	names := map[string]bool{}
	for _, p := range All() {
		if p.Name == "" || p.Install == "" || names[p.Name] {
			t.Fatalf("%+v", p)
		}
		names[p.Name] = true
	}
	for _, name := range []string{"git", "qmd", "pdftotext", "yt-dlp", "ollama"} {
		if !names[name] {
			t.Errorf("%s is missing", name)
		}
	}
}

func TestInstallFindsTheCommandByName(t *testing.T) {
	if got := Install("pdftotext"); got != "winget install --id oschwartz10612.Poppler -e" {
		t.Fatal(got)
	}
	if got := Install("nothing"); got != "" {
		t.Fatal(got)
	}
}
