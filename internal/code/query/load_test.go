package query

import (
	"errors"
	"os"
	"testing"

	"github.com/xidus90/loomux/internal/code/store"
)

func TestLoadGraphErrors(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := loadGraph(root, true)
	if err == nil || errors.Is(err, ErrNoGraph) {
		t.Fatalf("want decode error, got %v", err)
	}
}
