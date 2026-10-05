package pytext

import "testing"

// The expected values are what Python 3.14's urlsplit and unquote answered
// for each input on 2026-09-23, run through `uv run python`.

func TestSplitURLAnswersWhatUrlsplitAnswers(t *testing.T) {
	for _, c := range []struct{ in, scheme, netloc, path string }{
		{"a.md", "", "", "a.md"},
		{"topics/b.md?x=1#frag", "", "", "topics/b.md"},
		{"/abs/p.md", "", "", "/abs/p.md"},
		{"../up.md", "", "", "../up.md"},
		{"https://ex.invalid/x", "https", "ex.invalid", "/x"},
		{"mailto:a@b", "mailto", "", "a@b"},
		{"c:/x/y.md", "c", "", "/x/y.md"},
		{`C:\x`, "c", "", `\x`},
		{"brain://project/space/topics/y", "brain", "project", "/space/topics/y"},
		{"BRAIN://Project/x", "brain", "Project", "/x"},
		{"//host/path?q", "", "host", "/path"},
		{"//host", "", "host", ""},
		{"  \t sp.md", "", "", "sp.md"},
		{"a\tb\nc.md", "", "", "abc.md"},
		{"1abc:rest", "", "", "1abc:rest"},
		{"a+b-c.d:rest", "a+b-c.d", "", "rest"},
		{"ab_c:rest", "", "", "ab_c:rest"},
		{":nope", "", "", ":nope"},
		{"#only", "", "", ""},
		{"?q#f", "", "", ""},
		{"x%20y.md", "", "", "x%20y.md"},
		{"", "", "", ""},
		{" ", "", "", ""},
		{"\x00\x1f x.md", "", "", "x.md"},
	} {
		scheme, netloc, path := SplitURL(c.in)
		if scheme != c.scheme || netloc != c.netloc || path != c.path {
			t.Errorf("SplitURL(%q) = %q %q %q, want %q %q %q", c.in, scheme, netloc, path, c.scheme, c.netloc, c.path)
		}
	}
}

func TestUnquoteAnswersWhatUnquoteAnswers(t *testing.T) {
	for in, want := range map[string]string{
		"a.md":          "a.md",
		"x%20y.md":      "x y.md",
		"%zz%2":         "%zz%2",
		"%":             "%",
		"a%2Fb":         "a/b",
		"%41%4a%4A":     "AJJ",
		"%C3%A4.md":     "ä.md",
		"%e2%82%ac":     "€",
		"%e2%82":        "\ufffd",
		"%e0%a0":        "\ufffd",
		"%e0%80":        "\ufffd\ufffd",
		"%ed%a0":        "\ufffd\ufffd",
		"%ed%9f":        "\ufffd",
		"%f0%90%80":     "\ufffd",
		"%f0%80":        "\ufffd\ufffd",
		"%f4%90":        "\ufffd\ufffd",
		"%f4%8f":        "\ufffd",
		"%f1%80":        "\ufffd",
		"%c2":           "\ufffd",
		"%c0%af":        "\ufffd\ufffd",
		"%80x":          "\ufffdx",
		"%e1%80%41":     "\ufffdA",
		"%ff%fe":        "\ufffd\ufffd",
		"ä%20literally": "ä literally",
	} {
		if got := Unquote(in); got != want {
			t.Errorf("Unquote(%q) = %q, want %q", in, got, want)
		}
	}
}

// The boundaries the mutation round of 2026-09-23 asked about, measured with
// stufe-3c-orakel/boundaries.py against Python 3.14.

func TestSplitURLAtTheEdgesOfASchemeAndANetworkLocation(t *testing.T) {
	for _, c := range []struct{ in, scheme, netloc, path string }{
		{"///x", "", "", "/x"},
		{"z:r", "z", "", "r"},
		{"Z:r", "z", "", "r"},
		{"a0:r", "a0", "", "r"},
		{"a9:r", "a9", "", "r"},
		{"a/:r", "", "", "a/:r"},
	} {
		scheme, netloc, path := SplitURL(c.in)
		if scheme != c.scheme || netloc != c.netloc || path != c.path {
			t.Errorf("SplitURL(%q) = %q %q %q, want %q %q %q", c.in, scheme, netloc, path, c.scheme, c.netloc, c.path)
		}
	}
}

func TestUnquoteAtTheFirstAndLastLeadByteOfEveryRange(t *testing.T) {
	for in, want := range map[string]string{
		"%C2%80":       "\u0080",
		"%DF%BF":       "\u07ff",
		"%E0%A0%80":    "\u0800",
		"%E1%80%80":    "\u1000",
		"%EF%BF%BF":    "\uffff",
		"%ED%9F%BF":    "\ud7ff",
		"%F0%90%80%80": "\U00010000",
		"%F1%80%80%80": "\U00040000",
		"%F3%BF%BF%BF": "\U000fffff",
		"%F4%8F%BF%BF": "\U0010ffff",
		"%C1%BF":       "\ufffd\ufffd",
		"%F5%80":       "\ufffd\ufffd",
		"%EF%BF":       "\ufffd",
		"%F3%BF%BF":    "\ufffd",
		"%C2":          "\ufffd",
		"%E0%9F%80":    "\ufffd\ufffd\ufffd",
	} {
		if got := Unquote(in); got != want {
			t.Errorf("Unquote(%q) = %q, want %q", in, got, want)
		}
	}
}
