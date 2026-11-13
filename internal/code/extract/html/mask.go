package html

// mask is src with every byte outside spans blanked to a space, line
// breaks kept: the copy a part's extractor reads, at the page's offsets
// and lines. A sep other than 0 is written on the first byte after each
// span, the < of </script>, so that a statement left open in one script
// ends there and does not run on into the next.
func mask(src string, spans [][2]int, sep byte) string {
	out := []byte(src)
	for i, c := range out {
		if c != '\n' && c != '\r' {
			out[i] = ' '
		}
	}
	for _, s := range spans {
		copy(out[s[0]:s[1]], src[s[0]:s[1]])
		if sep != 0 && s[1] < len(out) {
			out[s[1]] = sep
		}
	}
	return string(out)
}
