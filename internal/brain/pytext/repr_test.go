package pytext

import "testing"

// bs is one backslash. The expected texts spell Python's four-digit escape
// as bs + "u...", so that nothing between the plan and the file can turn
// the escape into the character it names.
const bs = "\\"

func TestReprQuotesAndEscapesLikePython(t *testing.T) {
	// Every right-hand side was printed by Python 3.14.7 of the reference:
	// print(ascii(s), "=>", ascii(repr(s))).
	cases := []struct{ in, want string }{
		{"x", `'x'`},
		{"it's", `"it's"`},
		{`say "hi"`, `'say "hi"'`},
		{`both ' and "`, `'both \' and "'`},
		{"a\tb", `'a\tb'`},
		{"\x00", `'\x00'`},
		{"\x7f", `'\x7f'`},
		{"\U000000e9", "'\U000000e9'"},
		{"\U0000200b", "'" + bs + "u200b'"},
		{"\U0001f600", "'\U0001f600'"},
		{`a\b`, `'a\\b'`},
		{`'\`, `"'\\"`},
		{`"\'`, `'"\\\''`},
		{"a\rb\nc", `'a\rb\nc'`},
		{"", `''`},
		{"\U00000080", `'\x80'`},
		{"\U000000a0", `'\xa0'`},
		{"\U000000ad", `'\xad'`},
		{"\U000000ff", "'\U000000ff'"},
		{"\U00000100", "'\U00000100'"},
		{"\U00002028", "'" + bs + "u2028'"},
		{"\U0000feff", "'" + bs + "ufeff'"},
		{"\U0000ffff", "'" + bs + "uffff'"},
		{"\U000e0001", `'\U000e0001'`},
		{"\U0010ffff", `'\U0010ffff'`},
		{" ", `' '`},
		{"\x1b", `'\x1b'`},
		{"\U00000378", "'" + bs + "u0378'"},
		{"\U0001fae9", "'\U0001fae9'"},
	}
	for _, c := range cases {
		if got := Repr(c.in); got != c.want {
			t.Errorf("Repr(%+q) = %+q, want %+q", c.in, got, c.want)
		}
	}
}

func TestReprWritesAByteThatIsNotUTF8AsHex(t *testing.T) {
	if got, want := Repr("a\xffb"), `'a\xffb'`; got != want {
		t.Errorf("Repr = %s, want %s", got, want)
	}
}

func TestReprMatchesPythonOverTheWholeRange(t *testing.T) {
	// sha256 over repr(chr(c)) + "\0" for every code point but the
	// surrogates and the 4803 that Unicode 17 assigns.
	const python = "61bb4be799689f2c003eb8932471a2f8d2b63c3f74c9c313a461037097a31cc9"
	if got := rangeDigest(Repr, true); got != python {
		t.Errorf("digest %s, want %s", got, python)
	}
}
