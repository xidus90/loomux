package resolve

import "testing"

func TestImportDirSkipsACargoModule(t *testing.T) {
	// A Cargo module has an empty Path, which an empty specifier would match.
	if dir, ok := importDir("", []Module{{Cargo: &CargoManifest{}}}); ok {
		t.Fatalf("got %q, want no module", dir)
	}
}
