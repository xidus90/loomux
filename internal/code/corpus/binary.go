package corpus

import "bytes"

// IsBinary reports whether data is a binary file by the rule the design
// fixes: it contains a NUL byte. The whole file is checked, not just the head
// ripgrep would read, because the build holds the bytes anyway. UTF-16 files
// contain NUL bytes and therefore count as binary.
func IsBinary(data []byte) bool {
	return bytes.IndexByte(data, 0) >= 0
}
