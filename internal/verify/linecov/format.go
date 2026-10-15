package linecov

import "bytes"

// Record is one file's lines as a report spells them, before the path is
// resolved against the root.
type Record struct {
	Path   string
	Bases  []string // Cobertura <source> entries, tried in order
	Suffix bool     // JaCoCo: Path is package/file and is found by its ending
	Lines  map[int]uint64
}

// ReadContext carries what a format needs from outside its report.
type ReadContext struct {
	Module string // module path from go.mod under the root, "" without one
}

// Format reads one kind of coverage report. A new format is one type and one
// entry in formats.
type Format interface {
	Name() string
	Detect(head []byte) bool
	Read(content []byte, ctx ReadContext) ([]Record, []string)
}

func formats() []Format {
	return []Format{goProfileFormat{}, jacocoFormat{}, coberturaFormat{}, lcovFormat{}}
}

// Detect names the format of content by its first bytes, or nil.
func Detect(content []byte) Format {
	h := head(content)
	for _, f := range formats() {
		if f.Detect(h) {
			return f
		}
	}
	return nil
}

func head(content []byte) []byte {
	return bytes.TrimLeft(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")), " \t\r\n")
}
