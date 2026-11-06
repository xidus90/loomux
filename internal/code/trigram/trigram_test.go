package trigram

import (
	"reflect"
	"testing"
)

// pack is the test's view of one trigram: it fails the test instead of
// returning a bool every table case would have to check.
func pack(t *testing.T, s string) uint32 {
	t.Helper()
	v, ok := Pack(s)
	if !ok {
		t.Fatalf("Pack(%q) not packable", s)
	}
	return v
}

// TestPack pins the byte layout and the ASCII folding: three bytes, big
// endian, uppercase folded to lowercase, and nothing that is not ASCII.
func TestPack(t *testing.T) {
	if v := pack(t, "abc"); v != 0x616263 {
		t.Fatalf("Pack(abc) = %#x, want %#x", v, 0x616263)
	}
	if v := pack(t, "ABC"); v != 0x616263 {
		t.Fatalf("Pack(ABC) = %#x, want %#x (folded like abc)", v, 0x616263)
	}
	if v := pack(t, "aBc"); v != 0x616263 {
		t.Fatalf("Pack(aBc) = %#x, want %#x", v, 0x616263)
	}
	if v := pack(t, "ab1"); v != 0x616231 {
		t.Fatalf("Pack(ab1) = %#x, want %#x", v, 0x616231)
	}
	for _, s := range []string{"ab", "abcd", "", "äb", "abä", "ä"} {
		if v, ok := Pack(s); ok {
			t.Fatalf("Pack(%q) = %#x, true; want not packable", s, v)
		}
	}
}

// TestGrams pins the sliding window: every three adjacent bytes in order, and
// no trigram that contains a non-ASCII byte.
func TestGrams(t *testing.T) {
	cases := []struct {
		name string
		data string
		want []uint32
	}{
		{"empty", "", nil},
		{"shorter than a trigram", "ab", nil},
		{"exactly one", "abc", []uint32{0x616263}},
		{"two windows", "abcd", []uint32{0x616263, 0x626364}},
		{"uppercase folds", "ABCD", []uint32{0x616263, 0x626364}},
		{"newline counts as a byte", "ab\nc", []uint32{0x61620a, 0x620a63}},
		{"non-ascii windows are skipped", "ab\xc3\xa4c", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Grams([]byte(c.data)); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Grams(%q) = %#x, want %#x", c.data, got, c.want)
			}
		})
	}
}

// TestGramsDeduplicates pins the set: a repeated trigram appears once, and the
// output is sorted.
func TestGramsDeduplicates(t *testing.T) {
	want := []uint32{pack(t, "abc"), pack(t, "bca"), pack(t, "cab")}
	if got := Grams([]byte("abcabc")); !reflect.DeepEqual(got, want) {
		t.Fatalf("Grams(abcabc) = %#x, want %#x", got, want)
	}
}

// TestGramsDeterministic pins that the same bytes yield the same slice twice.
func TestGramsDeterministic(t *testing.T) {
	a := Grams([]byte("abcabc"))
	b := Grams([]byte("abcabc"))
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Grams is not deterministic: %#x and %#x", a, b)
	}
}
