package corpus

import (
	"fmt"
	"path"
	"strings"
)

// DefaultMaxFilesize is the largest file the search corpus reads, 64 MiB. It
// is a suggestion the design adopted; tgrep's own default is unmeasured until
// the comparison in Teil C.
const DefaultMaxFilesize int64 = 67_108_864

// ResolveMaxFilesize applies the order the design fixes: the flag wins over
// the configuration, the configuration over the default. A nil pointer means
// "not given". 0 means unlimited; a negative value is an error, whichever
// source carried it.
func ResolveMaxFilesize(flag, configured *int64) (int64, error) {
	switch {
	case flag != nil:
		return checkedSize(*flag, "flag")
	case configured != nil:
		return checkedSize(*configured, "configuration")
	}
	return DefaultMaxFilesize, nil
}

// checkedSize refuses a negative limit and names where it came from.
func checkedSize(value int64, origin string) (int64, error) {
	if value < 0 {
		return 0, fmt.Errorf("max_filesize %d from the %s is negative", value, origin)
	}
	return value, nil
}

// WithinSizeLimit reports whether a file of size bytes belongs to the search
// corpus. The limit is inclusive, and 0 means unlimited.
func WithinSizeLimit(size, maxFilesize int64) bool {
	return maxFilesize == 0 || size <= maxFilesize
}

// SearchFile is one file of the search corpus.
type SearchFile struct {
	Rel   string
	Abs   string
	Size  int64
	MTime int64
}

// SearchCorpus selects the files the search reads: everything the walk found
// that git does not ignore and whose name does not start with a dot, within the
// size limit. A dot directory never reaches this filter, because the walk does
// not enter one; a dot file is the ripgrep default the design keeps for v1.
//
// ignored and skippedSize count what the selection left out; a binary check
// needs the bytes and happens when a caller reads them. Symlinks stay out: the
// design does not follow them in the search corpus.
func SearchCorpus(listing Listing, maxFilesize int64) (files []SearchFile, ignored, skippedSize int) {
	files = make([]SearchFile, 0, len(listing.Entries))
	for _, e := range listing.Entries {
		if e.Ignored {
			ignored++
			continue
		}
		if e.Link {
			continue
		}
		if strings.HasPrefix(path.Base(e.Rel), ".") {
			continue
		}
		if !WithinSizeLimit(e.Size, maxFilesize) {
			skippedSize++
			continue
		}
		files = append(files, SearchFile{Rel: e.Rel, Abs: e.Abs, Size: e.Size, MTime: e.MTime})
	}
	return files, ignored, skippedSize
}
