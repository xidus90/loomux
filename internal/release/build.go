package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/notices"
)

// Target is one platform a release ships a binary for.
type Target struct{ OS, Arch string }

// Targets are the platforms of every release, in the order of its assets.
var Targets = []Target{
	{"windows", "amd64"}, {"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
}

// GoBuild runs `go` with extra environment; tests replace it.
type GoBuild func(env []string, args ...string) error

// ExecGoBuild runs the real go tool and keeps its output in the error.
func ExecGoBuild(env []string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Build cross-compiles every target into out and writes NOTICE.md and
// SHA256SUMS next to them. It returns the file names in asset order.
func Build(version, channel, out string, run GoBuild) ([]string, error) {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	ldflags := fmt.Sprintf("-X github.com/xidus90/loomux/internal/cli.Version=%s -X github.com/xidus90/loomux/internal/cli.Channel=%s", version, channel)
	var names []string
	var sums strings.Builder
	for _, t := range Targets {
		name := fmt.Sprintf("loomux_%s_%s_%s", version, t.OS, t.Arch)
		if t.OS == "windows" {
			name += ".exe"
		}
		path := filepath.Join(out, name)
		env := []string{"CGO_ENABLED=0", "GOOS=" + t.OS, "GOARCH=" + t.Arch}
		if err := run(env, "build", "-trimpath", "-ldflags", ldflags, "-o", path, "./cmd/loomux"); err != nil {
			return nil, fmt.Errorf("build %s/%s: %w", t.OS, t.Arch, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		names = append(names, name)
	}
	// Every third-party license the binary carries goes out beside it; a
	// notice in the source tree never reaches whoever downloads a release.
	notice := []byte(notices.Text())
	if err := os.WriteFile(filepath.Join(out, "NOTICE.md"), notice, 0o644); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(notice)
	fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), "NOTICE.md")
	names = append(names, "NOTICE.md")
	if err := os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(sums.String()), 0o644); err != nil {
		return nil, err
	}
	return append(names, "SHA256SUMS"), nil
}
