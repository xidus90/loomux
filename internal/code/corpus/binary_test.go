package corpus

import "testing"

func TestIsBinary(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "empty file is text", data: nil, want: false},
		{name: "plain UTF-8 is text", data: []byte("hello\n"), want: false},
		{name: "NUL at the start", data: []byte{0, 'a'}, want: true},
		{name: "NUL in the middle", data: []byte("ab\x00cd"), want: true},
		{name: "NUL at the last byte", data: []byte("ab\x00"), want: true},
		{name: "UTF-16 with BOM counts as binary", data: []byte{0xff, 0xfe, 'h', 0x00}, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsBinary(c.data); got != c.want {
				t.Fatalf("IsBinary(%q) = %v, want %v", c.data, got, c.want)
			}
		})
	}
}
