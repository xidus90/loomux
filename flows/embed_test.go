package flows

import (
	"io/fs"
	"reflect"
	"strings"
	"testing"
)

func TestEveryCatalogFolderIsEmbeddedWithoutItsTests(t *testing.T) {
	if got := Names(); !reflect.DeepEqual(got, []string{"example"}) {
		t.Fatalf("Names() = %v", got)
	}
	err := fs.WalkDir(FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "_test") {
			t.Errorf("%s is embedded", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(FS(), "example/questions/approve.md"); err != nil {
		t.Fatal(err)
	}
}
