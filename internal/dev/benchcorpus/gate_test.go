package benchcorpus

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestGateLanes(t *testing.T) {
	t.Run("reads pre-commit gate commands trimming whitespace and ignoring comments", func(t *testing.T) {
		fsys := fstest.MapFS{
			".githooks/pre-commit": &fstest.MapFile{
				Data: []byte("#!/bin/sh\n# gate\nset -e\n\n  go test ./... -count=1\ngofmt -l .\n"),
			},
		}

		got := gateLanes(fsys)
		want := []string{"set -e", "go test ./... -count=1", "gofmt -l ."}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("gateLanes() = %v, want %v", got, want)
		}
	})

	t.Run("returns nil if pre-commit does not exist", func(t *testing.T) {
		fsys := fstest.MapFS{}
		got := gateLanes(fsys)
		if got != nil {
			t.Errorf("gateLanes() = %v, want nil", got)
		}
	})

	t.Run("returns nil if pre-commit contains only comments and empty lines", func(t *testing.T) {
		fsys := fstest.MapFS{
			".githooks/pre-commit": &fstest.MapFile{
				Data: []byte("#!/bin/sh\n# comment only\n   \n"),
			},
		}
		got := gateLanes(fsys)
		if got != nil {
			t.Errorf("gateLanes() = %v, want nil", got)
		}
	})

	t.Run("follows scripts the hook runs one level deep", func(t *testing.T) {
		fsys := fstest.MapFS{
			".githooks/pre-commit": &fstest.MapFile{
				Data: []byte("#!/bin/sh\nsh ci/gate.sh\n. ./ci/env.sh\nsh ci/missing.sh\n"),
			},
			"ci/gate.sh": &fstest.MapFile{
				Data: []byte("#!/bin/sh\n# gate\ngo vet ./...\nsh ci/nested.sh\n"),
			},
			"ci/env.sh":    &fstest.MapFile{Data: []byte("export GOFLAGS=-mod=mod\n")},
			"ci/nested.sh": &fstest.MapFile{Data: []byte("go test ./...\n")},
		}

		got := gateLanes(fsys)
		want := []string{
			"sh ci/gate.sh", "go vet ./...", "sh ci/nested.sh",
			". ./ci/env.sh", "export GOFLAGS=-mod=mod",
			"sh ci/missing.sh",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("gateLanes() = %v, want %v", got, want)
		}
	})
}
