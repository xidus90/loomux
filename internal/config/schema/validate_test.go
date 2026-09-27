package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsTheRealConfigurationOfLoomux(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".loomux", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := Validate(string(data)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAcceptsEveryDefault(t *testing.T) {
	t.Chdir(t.TempDir())
	var b strings.Builder
	section := ""
	for _, k := range Keys() {
		if k.Name == "" || k.Default == "" {
			continue
		}
		if k.Section != section {
			fmt.Fprintf(&b, "[%s]\n", k.Section)
			section = k.Section
		}
		fmt.Fprintf(&b, "%s = %s\n", k.Name, k.Default)
	}
	if !strings.Contains(b.String(), "[area]") {
		b.WriteString("[area]\nscope = \"project/x\"\n")
	}
	if err := Validate(b.String()); err != nil {
		t.Fatalf("%v\n%s", err, b.String())
	}
}

// Each case names a piece of its reader's message, so a case cannot pass
// because some other reader refused the text first.
func TestValidateRefusesWhatAReaderRefuses(t *testing.T) {
	t.Chdir(t.TempDir())
	for name, c := range map[string]struct{ text, want string }{
		"commit language": {"[commit]\nlanguage = \"fr\"\n", "[commit].language"},
		"modules":         {"[modules]\nbrain = 1\n", "[modules] brain"},
		"policy regex":    {"[[policy.commands.rules]]\nregex = '(?!x)'\nreason = \"r\"\n", "does not compile"},
		"verify":          {"[verify]\nnope = 1\n", "nope"},
		"privacy":         {"[area]\nscope = \"project/x\"\n[privacy]\nmode = \"open\"\n", "[privacy] mode"},
		"worktree":        {"[worktree]\nmirror = \"x\"\n", "worktree"},
		"toml":            {"[commit\n", "not valid TOML"},
	} {
		t.Run(name, func(t *testing.T) {
			err := Validate(c.text)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want the reader's %q, got: %v", c.want, err)
			}
			if !strings.Contains(err.Error(), ".loomux/config.toml") || strings.Contains(err.Error(), os.TempDir()) {
				t.Fatalf("error must name the project file, not the scratch copy: %v", err)
			}
		})
	}
}

func TestValidateReportsAScratchDirectoryThatCannotBeMade(t *testing.T) {
	t.Chdir(t.TempDir())
	missing := filepath.Join(t.TempDir(), "missing")
	for _, name := range []string{"TMP", "TEMP", "TMPDIR"} {
		t.Setenv(name, missing)
	}
	// The error must be the scratch directory's: past a failed scratch the
	// readers would judge an empty path and fail for a reason of their own.
	if err := Validate("[commit]\n"); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("want the scratch directory's error, got %v", err)
	}
}

func TestValidateAsksTheAgentAndFlowReaders(t *testing.T) {
	t.Chdir(t.TempDir())
	for text, want := range map[string]string{
		"[agent]\ndefault = \"w\"\n":  "not under [agent.models]",
		"[flow]\ndefault = \"Dev\"\n": "[flow] default must be a flow name",
	} {
		err := Validate(text)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), ".loomux/config.toml") {
			t.Errorf("Validate(%q) = %v, want %q named against .loomux/config.toml", text, err, want)
		}
	}
	if err := Validate("[agent.models.w]\nprovider = \"claude\"\n[agent.roles]\nreviewer = \"w\"\n[flow]\noverrides = [\"example\"]\n"); err != nil {
		t.Fatalf("a sound file: %v", err)
	}
}
