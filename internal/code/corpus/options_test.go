package corpus

import (
	"strings"
	"testing"
)

func int64Ptr(v int64) *int64 { return &v }

func TestResolveMaxFilesize(t *testing.T) {
	cases := []struct {
		name       string
		flag       *int64
		configured *int64
		want       int64
		wantErr    string
	}{
		{name: "default without input", want: DefaultMaxFilesize},
		{name: "configuration wins over the default", configured: int64Ptr(1024), want: 1024},
		{name: "flag wins over the configuration", flag: int64Ptr(2048), configured: int64Ptr(1024), want: 2048},
		{name: "a zero flag means unlimited", flag: int64Ptr(0), configured: int64Ptr(1024), want: 0},
		{name: "a zero configuration means unlimited", configured: int64Ptr(0), want: 0},
		{name: "negative flag", flag: int64Ptr(-1), wantErr: "flag"},
		{name: "negative configuration", configured: int64Ptr(-2), wantErr: "configuration"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveMaxFilesize(c.flag, c.configured)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("ResolveMaxFilesize = %d, %v; want an error naming the %s", got, err, c.wantErr)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("ResolveMaxFilesize = %d, %v; want %d", got, err, c.want)
			}
		})
	}
}

func TestSizeLimitBoundary(t *testing.T) {
	const limit = 16
	listing := Listing{Entries: []Entry{
		{Rel: "under.txt", Size: limit - 1},
		{Rel: "exact.txt", Size: limit},
		{Rel: "over.txt", Size: limit + 1},
	}}
	files, ignored, skipped := SearchCorpus(listing, limit)
	if ignored != 0 || skipped != 1 {
		t.Fatalf("ignored=%d skipped=%d, want 0 and 1", ignored, skipped)
	}
	if len(files) != 2 || files[0].Rel != "under.txt" || files[1].Rel != "exact.txt" {
		t.Fatalf("files = %+v, want under.txt and exact.txt", files)
	}

	all, _, skipped := SearchCorpus(listing, 0)
	if skipped != 0 || len(all) != 3 {
		t.Fatalf("unlimited: skipped=%d files=%d, want 0 and 3", skipped, len(all))
	}
}

func TestSearchCorpusFilters(t *testing.T) {
	listing := Listing{Entries: []Entry{
		{Rel: "keep.go", Size: 1},
		{Rel: "ignored.go", Size: 1, Ignored: true},
		{Rel: ".hidden.go", Size: 1},
		{Rel: "link.go", Size: 1, Link: true},
		{Rel: "dir/keep.md", Size: 1},
	}}
	files, ignored, skipped := SearchCorpus(listing, DefaultMaxFilesize)
	if ignored != 1 || skipped != 0 {
		t.Fatalf("ignored=%d skipped=%d, want 1 and 0", ignored, skipped)
	}
	var got []string
	for _, f := range files {
		got = append(got, f.Rel)
	}
	want := []string{"keep.go", "dir/keep.md"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", got, want)
	}
}
