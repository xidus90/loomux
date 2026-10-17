package verify

import (
	"os"
	"path"
	"slices"
	"strings"
	"testing"
)

// listing is a readDir that answers the same names for every directory.
func listing(names []string) func(string) ([]string, error) {
	return func(string) ([]string, error) { return names, nil }
}

func TestADirOffersItsEnginesInAStableOrder(t *testing.T) {
	entries := []string{"Godot_v4.7.2_win64.exe", "readme.txt", "Godot_v3.6.3_win64.exe", "project.godot"}
	got, _ := dirCandidates("C:/engines", listing(entries), func(p string) string { return p })
	want := []string{"C:/engines/Godot_v3.6.3_win64.exe", "C:/engines/Godot_v4.7.2_win64.exe"}
	if !slices.Equal(got, want) {
		t.Fatalf("%#v", got)
	}
}

func TestADirThatIsNotThereOffersNothing(t *testing.T) {
	// A directory is a source, not a contract: nothing found is not an error.
	fail := func(string) ([]string, error) { return nil, os.ErrNotExist }
	if got, _ := dirCandidates("C:/gone", fail, nil); got != nil {
		t.Fatalf("%#v", got)
	}
	if got, _ := dirCandidates("C:/empty", listing(nil), nil); got != nil {
		t.Fatalf("%#v", got)
	}
	if got, _ := dirCandidates("C:/other", listing([]string{"readme.txt"}), nil); got != nil {
		t.Fatalf("%#v", got)
	}
	if got, _ := dirCandidates("", listing([]string{"godot"}), nil); got != nil {
		t.Fatalf("an unset directory is not the working directory: %#v", got)
	}
}

func TestADirCandidateGetsTheConsoleBuild(t *testing.T) {
	// Godot's Windows download ships both builds side by side, and the window
	// one writes nothing to stdout: it would be discarded for saying no
	// version rather than for being the wrong one. Both names map to the same
	// console build, which is one candidate, not two.
	entries := []string{"Godot_v4.7.2_win64.exe", "Godot_v4.7.2_win64_console.exe"}
	got, _ := dirCandidates("C:/e", listing(entries), func(p string) string {
		return strings.Replace(p, "win64.exe", "win64_console.exe", 1)
	})
	want := []string{"C:/e/Godot_v4.7.2_win64_console.exe"}
	if !slices.Equal(got, want) {
		t.Fatalf("%#v", got)
	}
}

func TestADirNamesEnginesWhateverTheirCase(t *testing.T) {
	entries := []string{"GODOT_v4.7.exe", "godot", "Godot.app"}
	got, _ := dirCandidates("/e", listing(entries), func(p string) string { return p })
	if len(got) != 3 {
		t.Fatalf("%#v", got)
	}
	for _, p := range got {
		if path.Dir(p) != "/e" {
			t.Fatalf("%q is not in the directory", p)
		}
	}
}
