package trigram

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/lock"
)

// TestWriteWaitsForAMappedReader pins that a rebuild does not fail while
// another reader has the index mapped. Older Windows refuses the rename until
// the mapping is gone, and Write retries until the reader closes; newer builds
// replace the file at once and the mapped reader keeps its bytes.
func TestWriteWaitsForAMappedReader(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, testDigest(), testExtractor, testSettings, testCounters, testFiles()); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	ix, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	refused := make(chan struct{}, 1)
	old := lock.OnRetry
	lock.OnRetry = func() {
		select {
		case refused <- struct{}{}:
		default:
		}
	}
	t.Cleanup(func() { lock.OnRetry = old })

	next := []File{{Rel: "new.txt", Data: []byte("fresh")}}
	done := make(chan error, 1)
	go func() { done <- Write(root, testDigest(), testExtractor, testSettings, testCounters, next) }()
	select {
	case <-refused:
		// This Windows refuses to replace a mapped file: Close is what lets
		// Write finish.
		if err := ix.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if err := <-done; err != nil {
			t.Fatalf("Write after Close: %v", err)
		}
	case err := <-done:
		// This Windows replaces the file at once (newer builds rename with
		// POSIX semantics): the reader keeps answering from the old bytes.
		if err != nil {
			t.Fatalf("Write while mapped: %v", err)
		}
		got, used := ix.Candidates([][]string{{"main"}})
		if !used || !reflect.DeepEqual(got, []string{"a.go", "b.py"}) {
			t.Fatalf("old reader: Candidates(main) = %v, %v; want a.go and b.py", got, used)
		}
		if err := ix.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}

	ix, err = Read(root)
	if err != nil {
		t.Fatalf("Read of the new index: %v", err)
	}
	defer ix.Close()
	if got, want := ix.Files(), []string{"new.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
}
