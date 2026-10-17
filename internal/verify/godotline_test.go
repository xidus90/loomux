package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProject puts body in a project.godot of its own directory and answers
// the file's path. godotWorld answers the directory instead, for the tests
// that resolve a lane; these want the file.
func writeProject(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "project.godot")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestALineReadsItsFeaturesVersion(t *testing.T) {
	want, _, err := projectGodot(writeProject(t, "config/features=PackedStringArray(\"4.7\")\n"))
	if want != (Line{Major: "4", Minor: "7"}) || err != nil {
		t.Fatalf("%#v %v", want, err)
	}
	if !want.Known() || want.String() != "4.7" {
		t.Fatalf("Known() = %v, String() = %q", want.Known(), want.String())
	}
}

func TestAThreeProjectFallsBackToConfigVersion(t *testing.T) {
	// Godot 3 writes no config/features at all.
	want, _, err := projectGodot(writeProject(t, "config_version=4\n[application]\n"))
	if want != (Line{Major: "3"}) || err != nil {
		t.Fatalf("%#v %v", want, err)
	}
	if want.Known() || want.String() != "3" {
		t.Fatalf("Known() = %v, String() = %q", want.Known(), want.String())
	}
}

func TestFeaturesBeatConfigVersion(t *testing.T) {
	// Godot writes config/features; config_version only counts the file's form.
	want, _, _ := projectGodot(writeProject(t, "config_version=4\nconfig/features=PackedStringArray(\"4.7\")\n"))
	if want != (Line{Major: "4", Minor: "7"}) {
		t.Fatalf("%#v", want)
	}
}

func TestAnUnknownConfigVersionNamesTheFile(t *testing.T) {
	// 3 was written by Godot 3.0 (FORMAT_VERSION 3 there, CONFIG_VERSION 4
	// from 3.1 on), and a 6 has not been written yet: neither is guessed at,
	// and 3.0 is a line nobody maintains.
	path := writeProject(t, "config_version=3\n")
	if _, _, err := projectGodot(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("%v", err)
	}
}

func TestAFourProjectWithoutFeaturesStaysAnError(t *testing.T) {
	// config_version=5 spans 4.0 to 4.8, so it is no measure for a comparison.
	path := writeProject(t, "config_version=5\n")
	if _, _, err := projectGodot(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("%v", err)
	}
}

func TestAKnownLineMatchesMajorAndMinor(t *testing.T) {
	l := Line{Major: "4", Minor: "7"}
	if !l.Matches("4.7") || l.Matches("4.8") || l.Matches("3.6") {
		t.Fatal("a named minor must be compared")
	}
}

func TestAThreeLineMatchesTheMajorAlone(t *testing.T) {
	// config_version names no minor, so there is none to compare.
	l := Line{Major: "3"}
	if !l.Matches("3.6") || !l.Matches("3.5") || l.Matches("4.7") {
		t.Fatal("the major alone decides where the project named no minor")
	}
}
