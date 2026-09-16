package search_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestLauncher_DirectBinary(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "mytool.exe")
	if err := os.WriteFile(binPath, []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd, err := search.ResolveLauncher(binPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 1 || cmd[0] != binPath {
		t.Errorf("expected [%q], got %v", binPath, cmd)
	}
}

func TestLauncher_NotFound(t *testing.T) {
	_, err := search.Launcher("nonexistent-tool-xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent tool, got nil")
	}
	if !strings.Contains(err.Error(), "cannot find") {
		t.Errorf("expected error mentioning 'cannot find', got: %v", err)
	}
}

func TestLauncher_NpmBatchShimWithBesideNode(t *testing.T) {
	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "qmd.cmd")
	nodePath := filepath.Join(tmpDir, "node.exe")
	scriptRel := filepath.Join("node_modules", "qmd", "bin", "qmd.js")
	scriptPath := filepath.Join(tmpDir, scriptRel)

	if err := os.WriteFile(nodePath, []byte("node binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte("// qmd script"), 0o644); err != nil {
		t.Fatal(err)
	}

	shimContent := `@ECHO startup
"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*
`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := search.ResolveLauncher(shimPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 2 {
		t.Fatalf("expected 2 elements in cmd, got %d: %v", len(cmd), cmd)
	}
	if cmd[0] != nodePath {
		t.Errorf("expected node executable %q, got %q", nodePath, cmd[0])
	}
	if cmd[1] != scriptPath {
		t.Errorf("expected script path %q, got %q", scriptPath, cmd[1])
	}
}

func TestLauncher_UnrecognizedBatchShim(t *testing.T) {
	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "custom.cmd")
	if err := os.WriteFile(shimPath, []byte("@echo unsupported batch shim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error for unrecognized batch shim, got nil")
	}
	if !strings.Contains(err.Error(), "is a batch shim this version does not know how to bypass") {
		t.Errorf("expected error about batch shim bypass, got: %v", err)
	}
}

func TestLauncher_NodeIsBatchShim(t *testing.T) {
	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "qmd.cmd")
	nodeShimPath := filepath.Join(tmpDir, "node.cmd")

	if err := os.WriteFile(nodeShimPath, []byte("@echo node shim"), 0o644); err != nil {
		t.Fatal(err)
	}

	shimContent := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when node is a batch shim, got nil")
	}
	if !strings.Contains(err.Error(), "is a batch shim; running it would hand the query to cmd.exe") {
		t.Errorf("expected batch shim refusal error, got: %v", err)
	}
}

func TestLauncher_Success(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "dummytool.exe")
	if err := os.WriteFile(binPath, []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpDir)
	cmd, err := search.Launcher("dummytool.exe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 1 || cmd[0] != binPath {
		t.Errorf("expected [%q], got %v", binPath, cmd)
	}
}

func TestLauncher_UnreadableFile(t *testing.T) {
	_, err := search.ResolveLauncher("C:\\nonexistent\\path\\tool.cmd")
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
	if !strings.Contains(err.Error(), "cannot read") {
		t.Errorf("expected 'cannot read' error, got: %v", err)
	}
}

func TestLauncher_BesideNodeBat(t *testing.T) {
	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "qmd.cmd")
	nodeBatPath := filepath.Join(tmpDir, "node.bat")
	if err := os.WriteFile(nodeBatPath, []byte("@echo bat"), 0o755); err != nil {
		t.Fatal(err)
	}
	shimContent := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when node.bat is beside, got nil")
	}
	if !strings.Contains(err.Error(), "is a batch shim") {
		t.Errorf("expected batch shim refusal error, got: %v", err)
	}
}

func TestLauncher_NodeOnPath(t *testing.T) {
	tmpDir := t.TempDir()
	shimDir := filepath.Join(tmpDir, "shim")
	nodeDir := filepath.Join(tmpDir, "node")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nodeExe := filepath.Join(nodeDir, "node.exe")
	if err := os.WriteFile(nodeExe, []byte("echo node"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", nodeDir)

	shimPath := filepath.Join(shimDir, "qmd.cmd")
	scriptRel := filepath.Join("node_modules", "qmd", "bin", "qmd.js")
	scriptPath := filepath.Join(shimDir, scriptRel)
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte("// script"), 0o644); err != nil {
		t.Fatal(err)
	}

	shimContent := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := search.ResolveLauncher(shimPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 2 || cmd[0] != nodeExe || cmd[1] != scriptPath {
		t.Errorf("unexpected cmd: %v", cmd)
	}
}

func TestLauncher_NodeNotFoundOnPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	shimPath := filepath.Join(tmpDir, "qmd.cmd")
	shimContent := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(shimPath, []byte(shimContent), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := search.ResolveLauncher(shimPath)
	if err == nil {
		t.Fatal("expected error when node not on PATH, got nil")
	}
	if !strings.Contains(err.Error(), "cannot find 'node' on PATH") {
		t.Errorf("expected cannot find node error, got: %v", err)
	}
}
