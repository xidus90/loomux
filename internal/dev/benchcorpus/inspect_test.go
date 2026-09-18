package benchcorpus

import (
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"
)

func TestInspectProject(t *testing.T) {
	t.Run("Python with ruff mypy and pytest", func(t *testing.T) {
		fs := fstest.MapFS{
			"pyproject.toml": &fstest.MapFile{
				Data: []byte(`
[tool.ruff]
line-length = 88

[tool.mypy]
strict = true

[tool.pytest.ini_options]
minversion = "6.0"
`),
			},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PreToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"pytest"}]}],"PostToolUse":[{"hooks":[{"type":"command","command":"ruff check"},{"type":"prompt","prompt":"x"}]}]}}`),
			},
			".githooks/pre-commit": &fstest.MapFile{
				Data: []byte(`#!/bin/sh`),
			},
		}

		signals := InspectProject(fs)
		if !signals.HasClaude {
			t.Errorf("expected HasClaude to be true")
		}
		wantHooks := []ClaudeHook{{Event: "PostToolUse", Command: "ruff check"}, {Event: "PreToolUse", Command: "pytest"}}
		if !slices.Equal(signals.ClaudeHooks, wantHooks) {
			t.Errorf("expected hooks %v, got %v", wantHooks, signals.ClaudeHooks)
		}
		if !signals.HasGitHooks {
			t.Errorf("expected HasGitHooks to be true")
		}

		hasLang := false
		for _, l := range signals.Languages {
			if l == "python" {
				hasLang = true
			}
		}
		if !hasLang {
			t.Errorf("expected python language detected")
		}

		hasRuff, hasMypy, hasPytest := false, false, false
		for _, tool := range signals.NativeTools {
			if tool.Tool == "ruff" && tool.Category == "lint" {
				hasRuff = true
			}
			if tool.Tool == "mypy" && tool.Category == "typecheck" {
				hasMypy = true
			}
			if tool.Tool == "pytest" && tool.Category == "test" {
				hasPytest = true
			}
		}
		if !hasRuff || !hasMypy || !hasPytest {
			t.Errorf("expected ruff, mypy, pytest detected, got %+v", signals.NativeTools)
		}
	})

	t.Run("JavaScript TypeScript with package.json", func(t *testing.T) {
		fs := fstest.MapFS{
			"package.json": &fstest.MapFile{
				Data: []byte(`{
  "name": "my-app",
  "scripts": {
    "test": "vitest run",
    "lint": "eslint .",
    "format": "prettier --write .",
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "vitest": "^1.0.0",
    "eslint": "^8.0.0"
  }
}`),
			},
			"tsconfig.json": &fstest.MapFile{
				Data: []byte(`{}`),
			},
		}

		signals := InspectProject(fs)
		hasVitest, hasEslint, hasPrettier, hasTsc := false, false, false, false
		for _, tool := range signals.NativeTools {
			if tool.Tool == "vitest" {
				hasVitest = true
			}
			if tool.Tool == "eslint" {
				hasEslint = true
			}
			if tool.Tool == "prettier" {
				hasPrettier = true
			}
			if tool.Tool == "tsc" {
				hasTsc = true
			}
		}
		if !hasVitest || !hasEslint || !hasPrettier || !hasTsc {
			t.Errorf("expected vitest, eslint, prettier, tsc detected, got %+v", signals.NativeTools)
		}
	})

	t.Run("Go project with golangci-lint", func(t *testing.T) {
		fs := fstest.MapFS{
			"go.mod": &fstest.MapFile{
				Data: []byte("module example.com/foo\n\ngo 1.25\n"),
			},
			".golangci.yml": &fstest.MapFile{
				Data: []byte("linters:\n  enable:\n    - errcheck\n"),
			},
		}

		signals := InspectProject(fs)
		hasGoVet, hasGolangci := false, false
		for _, tool := range signals.NativeTools {
			if tool.Tool == "go-vet" {
				hasGoVet = true
			}
			if tool.Tool == "golangci-lint" {
				hasGolangci = true
			}
		}
		if !hasGoVet || !hasGolangci {
			t.Errorf("expected go-vet and golangci-lint, got %+v", signals.NativeTools)
		}
	})

	t.Run("Rust project", func(t *testing.T) {
		fs := fstest.MapFS{
			"Cargo.toml": &fstest.MapFile{
				Data: []byte("[package]\nname = \"foo\"\nversion = \"0.1.0\"\n"),
			},
		}

		signals := InspectProject(fs)
		hasClippy, hasTest := false, false
		for _, tool := range signals.NativeTools {
			if tool.Tool == "cargo-clippy" {
				hasClippy = true
			}
			if tool.Tool == "cargo-test" {
				hasTest = true
			}
		}
		if !hasClippy || !hasTest {
			t.Errorf("expected cargo-clippy and cargo-test, got %+v", signals.NativeTools)
		}
	})

	t.Run("Java, C#, C++, Ruby, PHP, Shell projects", func(t *testing.T) {
		fs := fstest.MapFS{
			"pom.xml":          &fstest.MapFile{Data: []byte("<project></project>")},
			"app.csproj":       &fstest.MapFile{Data: []byte("<Project></Project>")},
			"CMakeLists.txt":   &fstest.MapFile{Data: []byte("cmake_minimum_required(VERSION 3.10)")},
			".clang-format":    &fstest.MapFile{Data: []byte("BasedOnStyle: LLVM")},
			".clang-tidy":      &fstest.MapFile{Data: []byte("Checks: '*'")},
			"Gemfile":          &fstest.MapFile{Data: []byte("gem 'rspec'\ngem 'rubocop'")},
			"composer.json":    &fstest.MapFile{Data: []byte(`{"require-dev": {"phpunit/phpunit": "^9"}}`)},
			"script.sh":        &fstest.MapFile{Data: []byte("#!/bin/bash")},
			".git/hooks/apply": &fstest.MapFile{Data: []byte("#!/bin/sh")},
		}

		signals := InspectProject(fs)
		tools := make(map[string]bool)
		for _, tool := range signals.NativeTools {
			tools[tool.Tool] = true
		}
		expected := []string{"mvn-test", "dotnet-test", "cmake", "clang-format", "clang-tidy", "rubocop", "rspec", "phpunit", "shellcheck"}
		for _, exp := range expected {
			if !tools[exp] {
				t.Errorf("missing expected tool: %s", exp)
			}
		}
		if !signals.HasGitHooks {
			t.Errorf("expected HasGitHooks to be true from .git/hooks")
		}
	})

	t.Run("Empty filesystem", func(t *testing.T) {
		fs := fstest.MapFS{}
		signals := InspectProject(fs)
		if signals.HasClaude || signals.HasGitHooks || len(signals.NativeTools) != 0 || len(signals.Languages) != 0 {
			t.Errorf("expected empty signals, got %+v", signals)
		}
	})

	t.Run("Claude edge cases", func(t *testing.T) {
		// Fallback to .claude.json and hooks field
		fs1 := fstest.MapFS{
			".claude.json": &fstest.MapFile{Data: []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"pytest"}]}]}}`)},
		}
		sig1 := InspectProject(fs1)
		if !sig1.HasClaude || len(sig1.ClaudeHooks) != 1 || sig1.ClaudeHooks[0] != (ClaudeHook{Event: "Stop", Command: "pytest"}) {
			t.Errorf("expected claude hooks from .claude.json, got %+v", sig1)
		}

		// Invalid json in .claude/settings.json
		fs2 := fstest.MapFS{
			".claude/settings.json": &fstest.MapFile{Data: []byte(`invalid json`)},
		}
		sig2 := InspectProject(fs2)
		if !sig2.HasClaude || len(sig2.ClaudeHooks) != 0 {
			t.Errorf("expected HasClaude true with empty hooks on invalid json")
		}
	})

	t.Run("Package JSON edge cases", func(t *testing.T) {
		fs := fstest.MapFS{
			"package.json": &fstest.MapFile{
				Data: []byte(`{
  "scripts": {
    "test": "jest"
  },
  "dependencies": {
    "typescript": "^5.0.0"
  }
}`),
			},
		}
		signals := InspectProject(fs)
		hasJest := false
		for _, tool := range signals.NativeTools {
			if tool.Tool == "jest" {
				hasJest = true
			}
		}
		if !hasJest {
			t.Errorf("expected jest detected")
		}

		// Invalid json
		fsInvalid := fstest.MapFS{
			"package.json": &fstest.MapFile{Data: []byte(`invalid`)},
		}
		sigInvalid := InspectProject(fsInvalid)
		if len(sigInvalid.NativeTools) != 0 {
			t.Errorf("expected 0 tools on invalid package.json")
		}
	})

	t.Run("PHP with phpstan", func(t *testing.T) {
		fs := fstest.MapFS{
			"composer.json": &fstest.MapFile{
				Data: []byte(`{"require-dev": {"phpstan/phpstan": "^1.0"}}`),
			},
		}
		signals := InspectProject(fs)
		hasPhpstan := false
		for _, tool := range signals.NativeTools {
			if tool.Tool == "phpstan" {
				hasPhpstan = true
			}
		}
		if !hasPhpstan {
			t.Errorf("expected phpstan detected")
		}
	})

	t.Run("ReadDir error", func(t *testing.T) {
		badFS := errFS{}
		signals := InspectProject(badFS)
		if len(signals.NativeTools) != 0 || len(signals.Languages) != 0 {
			t.Errorf("expected empty on read error, got %+v", signals)
		}
	})
}

type errFS struct{}

func (errFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrPermission
}
