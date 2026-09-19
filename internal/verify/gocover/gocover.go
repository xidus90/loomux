// Package gocover turns `go tool cover -func` output into a gate: every
// function at 100%, or an exemption with a reason written above it.
package gocover

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const marker = "//coverage:exempt "

// Line is one function row of `go tool cover -func`.
type Line struct {
	File    string
	Line    int
	Func    string
	Percent float64
}

// Parse reads the rows and skips the closing total.
func Parse(r io.Reader) ([]Line, error) {
	var lines []Line
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "total:") || strings.TrimSpace(text) == "" {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected cover line %q", text)
		}
		location := strings.Split(strings.TrimSuffix(fields[0], ":"), ":")
		if len(location) < 2 {
			return nil, fmt.Errorf("unexpected location in %q", text)
		}
		number, err := strconv.Atoi(location[len(location)-1])
		if err != nil {
			return nil, fmt.Errorf("line number in %q: %w", text, err)
		}
		percent, err := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		if err != nil {
			return nil, fmt.Errorf("percentage in %q: %w", text, err)
		}
		lines = append(lines, Line{
			File:    strings.Join(location[:len(location)-1], ":"),
			Line:    number,
			Func:    fields[1],
			Percent: percent,
		})
	}
	return lines, scanner.Err()
}

// Gate reports every function below 100% whose declaration is not preceded
// by an exemption with a reason, and returns 1 if there is any.
func Gate(lines []Line, module string, read func(string) ([]byte, error), w io.Writer) int {
	code := 0
	for _, l := range lines {
		if l.Percent >= 100 {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(l.File, module), "/")
		if exempt(read, rel, l.Line) {
			continue
		}
		fmt.Fprintf(w, "not covered: %s:%d %s %.1f%%\n", rel, l.Line, l.Func, l.Percent)
		code = 1
	}
	return code
}

func exempt(read func(string) ([]byte, error), path string, line int) bool {
	data, err := read(path)
	if err != nil {
		return false
	}
	rows := strings.Split(string(data), "\n")
	if line < 2 || line-2 >= len(rows) {
		return false
	}
	above := strings.TrimSpace(rows[line-2])
	return strings.HasPrefix(above+" ", marker) && strings.TrimSpace(strings.TrimPrefix(above, strings.TrimSpace(marker))) != ""
}

// ModulePath is the argument of the `module` line of a go.mod: the prefix
// the cover tool puts in front of every file it names.
func ModulePath(gomod []byte) (string, error) {
	for _, row := range strings.Split(string(gomod), "\n") {
		fields := strings.Fields(row)
		if len(fields) >= 2 && fields[0] == "module" {
			return strings.Trim(fields[1], `"`), nil
		}
	}
	return "", errors.New("go.mod has no module line")
}

// Total is the statement coverage of the closing `total:` line.
func Total(out []byte) (float64, error) {
	for _, row := range strings.Split(string(out), "\n") {
		fields := strings.Fields(row)
		if len(fields) == 0 || fields[0] != "total:" {
			continue
		}
		percent, err := strconv.ParseFloat(strings.TrimSuffix(fields[len(fields)-1], "%"), 64)
		if err != nil {
			return 0, fmt.Errorf("total in %q: %w", row, err)
		}
		return percent, nil
	}
	return 0, errors.New("no total line")
}
