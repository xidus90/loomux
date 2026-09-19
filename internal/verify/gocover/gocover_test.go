package gocover

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const sample = "github.com/xidus90/loomux/internal/a/a.go:10:\tFull\t\t100.0%\n" +
	"github.com/xidus90/loomux/internal/a/a.go:20:\tPartial\t\t85.7%\n" +
	"github.com/xidus90/loomux/cmd/loomux/main.go:9:\tmain\t\t0.0%\n" +
	"total:\t\t\t\t(statements)\t98.0%\n"

func TestParseReadsEveryFunctionLineAndSkipsTotal(t *testing.T) {
	lines, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || lines[1] != (Line{File: "github.com/xidus90/loomux/internal/a/a.go", Line: 20, Func: "Partial", Percent: 85.7}) {
		t.Fatalf("%+v", lines)
	}
}

func TestParseRefusesAMalformedLine(t *testing.T) {
	if _, err := Parse(strings.NewReader("garbage\n")); err == nil {
		t.Fatal("want error")
	}
	if _, err := Parse(strings.NewReader("x.go:abc:\tF\t1.0%\n")); err == nil {
		t.Fatal("want error for a line number that is not a number")
	}
	if _, err := Parse(strings.NewReader("x.go:1:\tF\tabc%\n")); err == nil {
		t.Fatal("want error for a percentage that is not a number")
	}
}

func TestParseRefusesALocationWithoutALineNumber(t *testing.T) {
	if _, err := Parse(strings.NewReader("x.go\tF\t1.0%\n")); err == nil {
		t.Fatal("want error for a location without a colon")
	}
}

func files(m map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if s, ok := m[p]; ok {
			return []byte(s), nil
		}
		return nil, errors.New("no such file")
	}
}

func TestGatePassesWhenEveryShortfallIsExempt(t *testing.T) {
	lines, _ := Parse(strings.NewReader(sample))
	read := files(map[string]string{
		"internal/a/a.go":    strings.Repeat("\n", 18) + "//coverage:exempt the error arm needs a full disk\nfunc Partial() {}\n",
		"cmd/loomux/main.go": strings.Repeat("\n", 7) + "//coverage:exempt process entry\nfunc main() {}\n",
	})
	var out bytes.Buffer
	if code := Gate(lines, "github.com/xidus90/loomux", read, &out); code != 0 {
		t.Fatalf("code %d: %s", code, out.String())
	}
}

func TestGateFailsAndNamesEveryUnexemptShortfall(t *testing.T) {
	lines, _ := Parse(strings.NewReader(sample))
	read := files(map[string]string{
		"internal/a/a.go":    strings.Repeat("\n", 18) + "//coverage:exempt \nfunc Partial() {}\n",
		"cmd/loomux/main.go": "func main() {}\n",
	})
	var out bytes.Buffer
	code := Gate(lines, "github.com/xidus90/loomux", read, &out)
	if code != 1 || !strings.Contains(out.String(), "internal/a/a.go:20 Partial 85.7%") ||
		!strings.Contains(out.String(), "cmd/loomux/main.go:9 main 0.0%") {
		t.Fatalf("code %d: %s", code, out.String())
	}
}

func TestGateFailsWhenTheSourceCannotBeRead(t *testing.T) {
	lines, _ := Parse(strings.NewReader(sample))
	var out bytes.Buffer
	if code := Gate(lines, "github.com/xidus90/loomux", files(nil), &out); code != 1 {
		t.Fatalf("code %d", code)
	}
}

func TestModulePath(t *testing.T) {
	got, err := ModulePath([]byte("// x\nmodule github.com/a/b\n\ngo 1.25.0\n"))
	if err != nil || got != "github.com/a/b" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := ModulePath([]byte("go 1.25.0\n")); err == nil || err.Error() != "go.mod has no module line" {
		t.Fatalf("want error, got %v", err)
	}
}

func TestTotal(t *testing.T) {
	out := []byte("github.com/a/b/x.go:3:\tF\t100.0%\ntotal:\t(statements)\t97.5%\n")
	if got, err := Total(out); err != nil || got != 97.5 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := Total([]byte("x.go:1:\tF\t1%\n")); err == nil || err.Error() != "no total line" {
		t.Fatalf("want error, got %v", err)
	}
}

func TestTotalRefusesAnUnreadableNumber(t *testing.T) {
	if _, err := Total([]byte("total:\t(statements)\tabc%\n")); err == nil {
		t.Fatal("want error")
	}
}
