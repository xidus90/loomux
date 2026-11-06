package trigram

import "sort"

// foldByte is the index's case folding: ASCII uppercase becomes lowercase,
// every other byte stays as it is. It is one-way on purpose; a query folds the
// same way, so "ABC" and "abc" meet in the same trigram.
func foldByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// Pack turns exactly three bytes into a trigram, ASCII case folded, big endian.
// ok == false means the bytes are not a trigram the index stores: a length
// other than three, or a non-ASCII byte. The folded index cannot represent a
// non-ASCII letter's case, so it does not pretend to.
func Pack(s string) (uint32, bool) {
	if len(s) != 3 {
		return 0, false
	}
	var v uint32
	for i := 0; i < 3; i++ {
		if s[i] >= 0x80 {
			return 0, false
		}
		v = v<<8 | uint32(foldByte(s[i]))
	}
	return v, true
}

// Grams returns the trigrams of data: every three adjacent bytes, ASCII case
// folded, sorted ascending and without duplicates.
//
// A window that contains a non-ASCII byte is skipped, matching Pack: the index
// stores no trigram it could not also be asked for. For UTF-8 text that only
// drops the windows that touch a multi-byte character; the ASCII windows
// around them stay.
func Grams(data []byte) []uint32 {
	if len(data) < 3 {
		return nil
	}
	grams := make([]uint32, 0, len(data)-2)
	for i := 0; i+2 < len(data); i++ {
		if data[i] >= 0x80 || data[i+1] >= 0x80 || data[i+2] >= 0x80 {
			continue
		}
		grams = append(grams, uint32(foldByte(data[i]))<<16|
			uint32(foldByte(data[i+1]))<<8|
			uint32(foldByte(data[i+2])))
	}
	if len(grams) == 0 {
		return nil
	}
	sort.Slice(grams, func(i, j int) bool { return grams[i] < grams[j] })
	out := grams[:1]
	for _, g := range grams[1:] {
		if g != out[len(out)-1] {
			out = append(out, g)
		}
	}
	return out
}
