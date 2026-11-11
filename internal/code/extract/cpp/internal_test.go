package cpp

import (
	"reflect"
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The C++ grammar builds a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "m.cpp", "class K {};\n")
	if err == nil || !strings.Contains(err.Error(), "parse m.cpp") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}

func TestChunksStartAtNamespaceLevelLines(t *testing.T) {
	src := []byte("void a() { x\n  y();\n}\n#if Z\n[[n]] int b;\n_c d;\n  e;\n}\nF g;\n")
	if got, want := chunks(src, 5), []int{28, 41, 54}; !reflect.DeepEqual(got, want) {
		t.Errorf("chunks = %v, want %v: the [[, _ and F lines", got, want)
	}

	// Too many starts are thinned to every k-th, k = ceil(600/256) = 3.
	many := []byte("x\n" + strings.Repeat("f;\n", 600))
	got := chunks(many, 0)
	if len(got) != 200 || got[0] != 2 || got[1] != 11 {
		t.Errorf("chunks = %d starting %v, want 200 starting [2 11 ...]", len(got), got[:2])
	}
}

func TestMaskKeepsEveryOffset(t *testing.T) {
	src := "struct K {\r\n  GDCLASS(K, Base)\r\n  Q_OBJECT\r\n  int x;\r\n};\nEND"
	want := "struct K {\r\n                  \r\n          \r\n  int x;\r\n};\n   "
	if got := mask([]byte(src)); string(got) != want {
		t.Errorf("mask =\n%q\nwant\n%q", got, want)
	}
	alias := "template <class T> using X = typename A<B<T>>::type;\nusing Y =\n  A<B<T> >::type;\nusing Z = A<T>;\nusing W = A::B;\nusing namespace q;\nusing V = A<\n B<T>>::type;\n"
	aliasWant := "template <class T> using X = int" + strings.Repeat(" ", 19) + ";\nusing Y =\n  int" + strings.Repeat(" ", 11) + ";\nusing Z = A<T>;\nusing W = A::B;\nusing namespace q;\nusing V = A<\n B<T>>::type;\n"
	if got := mask([]byte(alias)); string(got) != aliasWant {
		t.Errorf("mask =\n%q\nwant\n%q", got, aliasWant)
	}
	// Only code is masked: what merely reads like an alias in a comment or a
	// literal stays, and the code after it is not blanked up to the next ';'.
	for _, keep := range []string{
		"// using X = the A<T>::type\nint f() { return a<b>::c; }\n",
		"// using X = the A<T>::type\nfoo(b);\n",
		"int a; // using X = A<T>::type;\n",
		"int a; // using X = A<T>::type;",
		"const char* s = \"using X = A<T>::type;",
		"/** using Vec = std::vector<int> */\nint x;\n",
		"/* using X = A<T>::type */\nint f() { return a<b>::c; }\n",
		"/* using X = A<T>::type;\n int x;",
		"const char* s = \"using X = A<T>::type;\";\n",
		"const char* s = \"a\\\"using X = A<T>::type;\";\n",
		"auto r = R\"x(using X = A<T>::type;)x\";\n",
		"auto r = R\"x(using X = A<T>::type;\n",
		"char c = 'a'; char d = '\\''; const char* s = \"'\" \"using X = A<T>::type;\";\n",
		"using X = A<T>::t /* ; */;\n",
		// The right side stops at a brace, even in plain code.
		"using X = decltype([]{ return a<b>::c; }());\n",
	} {
		if got := mask([]byte(keep)); string(got) != keep {
			t.Errorf("mask(%q) = %q, want it unchanged", keep, got)
		}
	}
	// Code after a literal that merely looks open is still masked.
	for _, open := range []string{
		"char c = '\"'; using X = A<T>::type;\n",
		"int n = 1'000; using X = A<T>::type;\n",
		"auto r = R\"abcdefghijklmnopqrstuvwxyz\"; using X = A<T>::type;\n",
		"const char* s = \"abc\nusing X = A<T>::type;\n",
		"/* c */ using X = A<T>::type; // d\n",
	} {
		if got := mask([]byte(open)); !strings.Contains(string(got), "= int") {
			t.Errorf("mask(%q) = %q, want the alias masked", open, got)
		}
	}
	if marks := gdclasses([]byte(src)); len(marks) != 1 || *marks[0] != (mark{at: strings.Index(src, "\n  GDCLASS") + 1, name: "K"}) {
		t.Errorf("gdclasses = %+v, want K at the GDCLASS line", marks)
	}
}
