package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardAllowsSafeFileWrite(t *testing.T) {
	root := t.TempDir()
	payload := `{"tool_name": "Write", "tool_input": {"file_path": "src/main.py", "content": "print(1)"}}`
	var stderr bytes.Buffer

	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitOK {
		t.Fatalf("code = %d, want ExitOK (0). stderr: %s", code, stderr.String())
	}

	// Test absolute path
	absPath := filepath.Join(root, "src", "main.py")
	payloadAbs := `{"tool_name": "Write", "tool_input": {"file_path": "` + filepath.ToSlash(absPath) + `", "content": "print(1)"}}`
	stderr.Reset()
	if code := runGuard(strings.NewReader(payloadAbs), &stderr, root); code != ExitOK {
		t.Fatalf("abs path code = %d, want ExitOK", code)
	}

	// Test outside path
	payloadOutside := `{"tool_name": "Write", "tool_input": {"file_path": "../outside.py", "content": "print(1)"}}`
	stderr.Reset()
	if code := runGuard(strings.NewReader(payloadOutside), &stderr, root); code != ExitOK {
		t.Fatalf("outside path code = %d, want ExitOK", code)
	}
}

func TestGuardBlocksProtectedBuiltinPaths(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name string
		path string
	}{
		{"env file", ".env"},
		{"env local", ".env.local"},
		{"ssh key", "id_rsa"},
		{"pem file", "server.pem"},
		{"key file", "cert.key"},
		{"p12 file", "cert.p12"},
		{"aws secret", ".aws/credentials"},
		{"lock file", "uv.lock"},
		{"no-verify", ".claude/.no-verify"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"tool_name": "Edit", "tool_input": {"file_path": "` + tc.path + `", "new_string": "x"}}`
			var stderr bytes.Buffer
			code := runGuard(strings.NewReader(payload), &stderr, root)
			if code != ExitDenied {
				t.Fatalf("[%s] code = %d, want ExitDenied (2)", tc.name, code)
			}
			if !strings.Contains(stderr.String(), "ultraloom policy refused this Edit") {
				t.Fatalf("stderr missing refusal:\n%s", stderr.String())
			}
		})
	}
}

func TestGuardBlocksConfiguredProtectedPaths(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ultraloom"), 0o755)
	policyTOML := `
[[policy.paths.rules]]
match  = "migrations/[0-9][0-9][0-9][0-9]_*.py"
reason = "Django migrations are protected"
`
	os.WriteFile(filepath.Join(root, ".ultraloom", "policy.toml"), []byte(policyTOML), 0o644)

	payload := `{"tool_name": "Write", "tool_input": {"file_path": "migrations/0001_initial.py"}}`
	var stderr bytes.Buffer
	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitDenied {
		t.Fatalf("code = %d, want ExitDenied (2)", code)
	}
	if !strings.Contains(stderr.String(), "Django migrations are protected") {
		t.Fatalf("stderr missing configured reason:\n%s", stderr.String())
	}
}

func TestGuardBlocksNotebookEdit(t *testing.T) {
	root := t.TempDir()
	payload := `{"tool_name": "NotebookEdit", "tool_input": {"notebook_path": ".env"}}`
	var stderr bytes.Buffer
	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitDenied {
		t.Fatalf("code = %d, want ExitDenied", code)
	}
}

func TestGuardBlocksGitPushOnBashAndPowerShell(t *testing.T) {
	root := t.TempDir()
	for _, tool := range []string{"Bash", "PowerShell"} {
		payload := `{"tool_name": "` + tool + `", "tool_input": {"command": "git push origin main"}}`
		var stderr bytes.Buffer
		code := runGuard(strings.NewReader(payload), &stderr, root)
		if code != ExitDenied {
			t.Fatalf("[%s] code = %d, want ExitDenied", tool, code)
		}
		if !strings.Contains(stderr.String(), "Whether commits reach the remote is a human's decision.") {
			t.Fatalf("[%s] stderr missing reason:\n%s", tool, stderr.String())
		}
	}
}

func TestGuardAllowsSafeCommands(t *testing.T) {
	root := t.TempDir()
	payload := `{"tool_name": "Bash", "tool_input": {"command": "git status"}}`
	var stderr bytes.Buffer
	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitOK {
		t.Fatalf("code = %d, want ExitOK. stderr: %s", code, stderr.String())
	}
}

func TestGuardBlocksConfiguredForbiddenCommand(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ultraloom"), 0o755)
	policyTOML := `
[[policy.commands.rules]]
regex  = "(^|\\s)pip\\s+install"
reason = "use uv add instead"
`
	os.WriteFile(filepath.Join(root, ".ultraloom", "policy.toml"), []byte(policyTOML), 0o644)

	payload := `{"tool_name": "Bash", "tool_input": {"command": "pip install requests"}}`
	var stderr bytes.Buffer
	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitDenied {
		t.Fatalf("code = %d, want ExitDenied", code)
	}
	if !strings.Contains(stderr.String(), "use uv add instead") {
		t.Fatalf("stderr missing reason:\n%s", stderr.String())
	}
}

func TestGuardHandlesMalformedPayloads(t *testing.T) {
	root := t.TempDir()

	// Empty input
	var stderr bytes.Buffer
	if code := runGuard(strings.NewReader(""), &stderr, root); code != ExitInternal {
		t.Fatalf("code = %d, want ExitInternal", code)
	}

	// Invalid JSON
	stderr.Reset()
	if code := runGuard(strings.NewReader("not json"), &stderr, root); code != ExitInternal {
		t.Fatalf("code = %d, want ExitInternal", code)
	}

	// Missing tool_name
	stderr.Reset()
	if code := runGuard(strings.NewReader(`{"tool_input": {}}`), &stderr, root); code != ExitInternal {
		t.Fatalf("code = %d, want ExitInternal", code)
	}
}

func TestGuardHandlesBrokenPolicyTOML(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ultraloom"), 0o755)
	os.WriteFile(filepath.Join(root, ".ultraloom", "policy.toml"), []byte("broken toml === "), 0o644)

	payload := `{"tool_name": "Write", "tool_input": {"file_path": "test.txt"}}`
	var stderr bytes.Buffer
	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitDenied {
		t.Fatalf("code = %d, want ExitDenied (2) for broken policy file", code)
	}
}

func TestRelativePathHelper(t *testing.T) {
	root := t.TempDir()
	rel := relativePath(filepath.Join(root, "a", "b.txt"), root)
	if rel != "a/b.txt" {
		t.Fatalf("rel = %q, want a/b.txt", rel)
	}
	outside := relativePath("../outside.txt", root)
	if !strings.Contains(outside, "outside.txt") {
		t.Fatalf("outside = %q", outside)
	}
	otherDir := t.TempDir()
	absOutside := relativePath(filepath.Join(otherDir, "other.txt"), root)
	if !strings.Contains(absOutside, "other.txt") {
		t.Fatalf("absOutside = %q", absOutside)
	}
}

func TestGuardBlocksMultiEdit(t *testing.T) {
	root := t.TempDir()
	payload := `{"tool_name": "MultiEdit", "tool_input": {"file_path": ".env"}}`
	var stderr bytes.Buffer
	code := runGuard(strings.NewReader(payload), &stderr, root)
	if code != ExitDenied {
		t.Fatalf("code = %d, want ExitDenied for MultiEdit on .env", code)
	}
}

func TestLoadPolicyUnreadable(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, ".ultraloom", "policy.toml")
	_ = os.MkdirAll(policyDir, 0755) // directory instead of file causes read error
	_, err := loadPolicy(root)
	if err == nil {
		t.Fatal("expected error for directory policy.toml, got nil")
	}
}
