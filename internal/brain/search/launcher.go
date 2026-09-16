package search

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	shimSuffixes   = map[string]bool{".cmd": true, ".bat": true}
	npmShimPattern = regexp.MustCompile(`"%_prog%"\s+"%dp0%[\\/]+([^"]+)"\s+%\*`)
)

// Launcher resolves the command to execute name, bypassing npm batch shims on Windows
// to prevent passing user text to cmd.exe.
func Launcher(name string) ([]string, error) {
	executable, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("cannot find %q on PATH", name)
	}
	return ResolveLauncher(executable)
}

// ResolveLauncher takes an already resolved executable path and unwraps batch shims if necessary.
func ResolveLauncher(executable string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(executable))
	if !shimSuffixes[ext] {
		return []string{executable}, nil
	}

	content, err := os.ReadFile(executable)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", executable, err)
	}

	match := npmShimPattern.FindSubmatch(content)
	if match == nil {
		return nil, fmt.Errorf("%s is a batch shim this version does not know how to bypass; running it would hand the query to cmd.exe", executable)
	}
	scriptRel := string(match[1])

	dir := filepath.Dir(executable)
	besideExe := filepath.Join(dir, "node.exe")
	besideCmd := filepath.Join(dir, "node.cmd")
	besideBat := filepath.Join(dir, "node.bat")
	var node string
	if info, err := os.Stat(besideExe); err == nil && !info.IsDir() {
		node = besideExe
	} else if info, err := os.Stat(besideCmd); err == nil && !info.IsDir() {
		node = besideCmd
	} else if info, err := os.Stat(besideBat); err == nil && !info.IsDir() {
		node = besideBat
	} else {
		nodeLooked, err := exec.LookPath("node")
		if err != nil {
			return nil, fmt.Errorf("cannot find 'node' on PATH to run %s", executable)
		}
		node = nodeLooked
	}

	nodeExt := strings.ToLower(filepath.Ext(node))
	if shimSuffixes[nodeExt] {
		return nil, fmt.Errorf("%s is a batch shim; running it would hand the query to cmd.exe", node)
	}

	scriptPath := filepath.Join(dir, filepath.FromSlash(scriptRel))
	return []string{node, scriptPath}, nil
}
