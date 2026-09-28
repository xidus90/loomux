package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// benchStrictLine is a long shell line with wrappers, braces and globs whose
// unknown program names a path the project keeps: strict mode refuses it
// after resolving every target, the default mode lets it pass.
const benchStrictLine = "sudo -u root env X=1 timeout -s KILL 60 frob --out bin/app.exe src/notes/{a,b,c}/*.md ; " +
	"tee -a build/log.txt < in.txt | xargs -n 1 echo && rm -rf build/tmp/* 2>&1 ; git status"

// benchGuardProject is a project root with a manifest and the folders the
// line names, so that the file system has something to resolve.
func benchGuardProject(b *testing.B) string {
	b.Helper()
	b.Setenv(config.StateDirEnv, b.TempDir())
	root := b.TempDir()
	for _, dir := range []string{".loomux", "src/notes/a", "bin", "build/tmp"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			b.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	return root
}

// BenchmarkCheckTool is the guard's judgement of benchStrictLine in both
// modes, the file system resolution of strict mode included.
func BenchmarkCheckTool(b *testing.B) {
	root := benchGuardProject(b)
	paths := []config.PathRule{{Match: []string{"bin/*"}, Reason: "build output"}}
	for _, mode := range []struct {
		name    string
		policy  config.Policy
		refuses bool
	}{
		{"default", config.Policy{Paths: paths}, false},
		{"strict", config.Policy{Paths: paths, Strict: true}, true},
	} {
		b.Run(mode.name, func(b *testing.B) {
			if got := checkTool(root, "Bash", command(benchStrictLine), mode.policy); (len(got) > 0) != mode.refuses {
				b.Fatalf("reasons %q, want a refusal: %v", got, mode.refuses)
			}
			for b.Loop() {
				checkTool(root, "Bash", command(benchStrictLine), mode.policy)
			}
		})
	}
}
