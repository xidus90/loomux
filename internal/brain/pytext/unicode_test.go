package pytext

import "testing"

func TestNFCLikePython(t *testing.T) {
	// unicodedata.normalize("NFC", s) on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"e\U00000301", "\U000000e9"},
		{"\U0000212b", "\U000000c5"},
		{"a\U00000301\U00000328", "\U00000105\U00000301"},
		{"\U00001e9b\U00000323", "\U00001e9b\U00000323"},
		{"\U0000ac00\U000011a8", "\U0000ac01"},
		{"\U00000958", "\U00000915\U0000093c"},
	}
	for _, c := range cases {
		if got := NFC(c.in); got != c.want {
			t.Errorf("NFC(%+q) = %+q, want %+q", c.in, got, c.want)
		}
	}
}

func TestNFCMatchesPythonOverTheWholeRange(t *testing.T) {
	// sha256 over unicodedata.normalize("NFC", chr(c)) + "\0" for every code
	// point but the surrogates; x/text differs from Python on none of them.
	const python = "6e0aaa17a82be5b6f410942f1f99ae77832fb630e98780c6eaf6779ab278235d"
	if got := rangeDigest(NFC, false); got != python {
		t.Errorf("digest %s, want %s", got, python)
	}
}

func TestCaseFoldLikePython(t *testing.T) {
	// s.casefold() on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"\U000000df", "ss"},
		{"\U00001e9e", "ss"},
		{"\U0000fb01", "fi"},
		{"\U000003a3", "\U000003c3"},
		{"\U000003c2", "\U000003c3"},
		{"\U00000130", "i\U00000307"},
		{"\U00000149", "\U000002bcn"},
		{"\U000001f1", "\U000001f3"},
		{"ABC", "abc"},
		{"\U000000df.md", "ss.md"},
		{"\U000003a3\U00000391\U000003a3", "\U000003c3\U000003b1\U000003c3"},
		{"\U00000130x", "i\U00000307x"},
		{"\U0000fb01le.MD", "file.md"},
		{"Stra\U000000dfe", "strasse"},
		{"\U000001c5", "\U000001c6"},
		{"\U00001f88", "\U00001f00\U000003b9"},
		{"A\U0000030a", "a\U0000030a"},
		{"\U000013a0\U0000ab70\U000013f0\U000013f8", "\U000013a0\U000013a0\U000013f0\U000013f0"},
	}
	for _, c := range cases {
		if got := CaseFold(c.in); got != c.want {
			t.Errorf("CaseFold(%+q) = %+q, want %+q", c.in, got, c.want)
		}
	}
}

func TestCaseFoldMatchesPythonOverTheWholeRange(t *testing.T) {
	// sha256 over chr(c).casefold() + "\0" for every code point but the
	// surrogates and the Unicode 17 additions. Without the Cherokee repair in
	// CaseFold this digest differs.
	const python = "10e93868ae4b30b2d8d735f20f016cae6203cd34bad840c9a7037911cef978d8"
	if got := rangeDigest(CaseFold, true); got != python {
		t.Errorf("digest %s, want %s", got, python)
	}
}
