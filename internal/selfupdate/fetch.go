package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sumsName is the checksum asset every release carries (internal/release).
const sumsName = "SHA256SUMS"

// AssetName is the release asset for one platform, named the way
// internal/release names it.
func AssetName(ver, goos, goarch string) string {
	name := fmt.Sprintf("loomux_%s_%s_%s", ver, goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// fetch downloads tag's binary for this platform into a directory of its own
// under dir, holds it against SHA256SUMS and against its own --version, and
// stages it as dir/loomux.new.exe for swap. Nothing is staged unless every
// check passed.
func fetch(ctx context.Context, o Options, tag, dir string) error {
	ver := strings.TrimPrefix(tag, "v")
	asset := AssetName(ver, o.GOOS, o.GOARCH)
	tmp, err := os.MkdirTemp(dir, "update-*")
	if err != nil {
		return fmt.Errorf("create download directory: %w", err)
	}
	defer os.RemoveAll(tmp)
	if _, err := call(ctx, o.Run, "gh", "release", "download", tag, "--repo", Repo,
		"--pattern", asset, "--pattern", sumsName, "--dir", tmp); err != nil {
		return err
	}
	if err := verifySum(tmp, asset, tag); err != nil {
		return err
	}
	path := filepath.Join(tmp, asset)
	out, err := call(ctx, o.Run, path, "--version")
	if err != nil {
		return fmt.Errorf("run downloaded binary: %w", err)
	}
	if !reportsVersion(string(out), ver) {
		return fmt.Errorf("downloaded binary reports %q, not loomux %s", firstLine(string(out)), ver)
	}
	return stage(path, filepath.Join(dir, "loomux.new.exe"), o.Now())
}

// verifySum holds the asset against its line in SHA256SUMS. The line may
// carry sha256sum's binary-mode marker before the name.
func verifySum(dir, asset, tag string) error {
	sums, err := os.ReadFile(filepath.Join(dir, sumsName))
	if err != nil {
		return fmt.Errorf("release %s has no %s", tag, sumsName)
	}
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			want = strings.ToLower(fields[0])
		}
	}
	if want == "" {
		return fmt.Errorf("%s of %s has no line for %s", sumsName, tag, asset)
	}
	data, err := os.ReadFile(filepath.Join(dir, asset))
	if err != nil {
		return fmt.Errorf("release %s has no asset %s", tag, asset)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}
	return nil
}

// reportsVersion accepts "loomux <ver>" alone or followed by the channel.
func reportsVersion(out, ver string) bool {
	line := firstLine(out)
	return line == "loomux "+ver || strings.HasPrefix(line, "loomux "+ver+" ")
}

// stage stamps the binary with now and moves it to target. The stamp is the
// activation: a bridge replaces serve only for a newer modification time
// (serve.State.OlderThan), and a download must not depend on what time gh
// happens to leave on the file.
func stage(path, target string, now time.Time) error {
	if err := os.Chtimes(path, now, now); err != nil {
		return fmt.Errorf("stamp %s: %w", path, err)
	}
	if err := os.Rename(path, target); err != nil {
		return fmt.Errorf("stage %s: %w", target, err)
	}
	return nil
}
