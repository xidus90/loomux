package blast_test

import (
	"errors"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestQuoteLine(t *testing.T) {
	fileContent := `package main

import "fmt"

func Process() {
	fmt.Println("start")
	RenderAll()
	Render()
	fmt.Println("end")
}
`
	reader := func(p string) ([]byte, error) {
		if p == "main.go" {
			return []byte(fileContent), nil
		}
		return nil, errors.New("file not found")
	}

	span := model.Span("L5-L10")

	// 1. Matches Render on line 8, ignores RenderAll on line 7 due to word boundary
	lineNum, line, ok := blast.QuoteLine(reader, "main.go", span, "Render")
	if !ok || lineNum != 8 || line != "Render()" {
		t.Fatalf("want line 8 Render(), got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}

	// 2. Target not found
	lineNum, line, ok = blast.QuoteLine(reader, "main.go", span, "NonExistent")
	if ok || lineNum != 0 || line != "" {
		t.Fatalf("want not found, got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}

	// 3. Reader error
	lineNum, line, ok = blast.QuoteLine(reader, "missing.go", span, "Render")
	if ok || lineNum != 0 || line != "" {
		t.Fatalf("want error handled, got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}

	// 4. Invalid span
	lineNum, line, ok = blast.QuoteLine(reader, "main.go", "invalid", "Render")
	if ok || lineNum != 0 || line != "" {
		t.Fatalf("want invalid span handled, got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}

	// Inverted span (to < from)
	lineNum, line, ok = blast.QuoteLine(reader, "main.go", "L10-L5", "Render")
	if ok || lineNum != 0 || line != "" {
		t.Fatalf("want inverted span handled, got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}

	// 5. Nil reader or empty target
	lineNum, line, ok = blast.QuoteLine(nil, "main.go", span, "Render")
	if ok || lineNum != 0 || line != "" {
		t.Fatalf("want nil reader handled, got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}
	lineNum, line, ok = blast.QuoteLine(reader, "main.go", span, "")
	if ok || lineNum != 0 || line != "" {
		t.Fatalf("want empty target handled, got lineNum=%d line=%q ok=%v", lineNum, line, ok)
	}
}
