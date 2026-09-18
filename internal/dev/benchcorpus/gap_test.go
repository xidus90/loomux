package benchcorpus

import (
	"errors"
	"strings"
	"testing"
)

func TestAuditGaps(t *testing.T) {
	mockLookPathSuccess := func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	mockLookPathFailure := func(file string) (string, error) {
		return "", errors.New("not found")
	}

	t.Run("Full coverage scenario", func(t *testing.T) {
		signals := ProjectSignals{
			Languages: []string{"python"},
			NativeTools: []ToolSignal{
				{Tool: "ruff", Category: "lint", ConfigFile: "pyproject.toml: [tool.ruff]"},
				{Tool: "mypy", Category: "typecheck", ConfigFile: "pyproject.toml: [tool.mypy]"},
			},
		}
		detectedStacks := []string{"python"}
		executedLanes := []string{"ruff check --output-format=concise .", "mypy --no-error-summary --no-pretty"}

		audits, gaps, rate := AuditGaps(signals, detectedStacks, executedLanes, mockLookPathSuccess)
		if len(audits) != 2 {
			t.Fatalf("expected 2 audits, got %d", len(audits))
		}
		if audits[0].Lane == "" || audits[1].Lane == "" {
			t.Errorf("expected both tools to have matched lanes, got %+v", audits)
		}
		if !audits[0].OnPath || !audits[1].OnPath {
			t.Errorf("expected tools to be on path")
		}
		if len(gaps) != 0 {
			t.Errorf("expected 0 gaps, got %v", gaps)
		}
		if rate != 100.0 {
			t.Errorf("expected 100.0 coverage rate, got %v", rate)
		}
	})

	t.Run("Missing lane scenario", func(t *testing.T) {
		signals := ProjectSignals{
			Languages: []string{"python"},
			NativeTools: []ToolSignal{
				{Tool: "pytest", Category: "test", ConfigFile: "pytest.ini"},
			},
		}
		detectedStacks := []string{"python"}
		executedLanes := []string{"ruff check ."}

		audits, gaps, rate := AuditGaps(signals, detectedStacks, executedLanes, mockLookPathSuccess)
		if len(audits) != 1 {
			t.Fatalf("expected 1 audit, got %d", len(audits))
		}
		if audits[0].Lane != "" {
			t.Errorf("expected no lane for pytest, got %s", audits[0].Lane)
		}
		if len(gaps) != 1 || !strings.Contains(gaps[0], "pytest") {
			t.Errorf("expected pytest gap, got %v", gaps)
		}
		if rate != 0.0 {
			t.Errorf("expected 0.0 coverage rate, got %v", rate)
		}
	})

	t.Run("Missing stack scenario", func(t *testing.T) {
		signals := ProjectSignals{
			Languages: []string{"csharp"},
			NativeTools: []ToolSignal{
				{Tool: "dotnet-test", Category: "test", ConfigFile: "app.csproj"},
			},
		}
		detectedStacks := []string{} // C# not recognized by Loomux
		executedLanes := []string{}

		audits, gaps, rate := AuditGaps(signals, detectedStacks, executedLanes, mockLookPathSuccess)
		if len(audits) != 1 {
			t.Fatalf("expected 1 audit, got %d", len(audits))
		}
		if audits[0].Lane != "" {
			t.Errorf("expected no lane, got %s", audits[0].Lane)
		}
		if len(gaps) == 0 {
			t.Fatalf("expected gaps reported")
		}
		if rate != 0.0 {
			t.Errorf("expected 0.0 coverage rate, got %v", rate)
		}
	})

	t.Run("Tool missing from PATH", func(t *testing.T) {
		signals := ProjectSignals{
			Languages: []string{"python"},
			NativeTools: []ToolSignal{
				{Tool: "ruff", Category: "lint", ConfigFile: "pyproject.toml"},
			},
		}
		detectedStacks := []string{"python"}
		executedLanes := []string{"ruff check ."}

		audits, gaps, rate := AuditGaps(signals, detectedStacks, executedLanes, mockLookPathFailure)
		if len(audits) != 1 {
			t.Fatalf("expected 1 audit, got %d", len(audits))
		}
		if audits[0].OnPath {
			t.Errorf("expected on_path false")
		}
		if len(gaps) != 1 || !strings.Contains(gaps[0], "PATH") {
			t.Errorf("expected missing from PATH gap, got %v", gaps)
		}
		if rate != 100.0 { // Lane exists in Loomux, but tool is not in PATH
			t.Errorf("expected 100.0 lane coverage rate, got %v", rate)
		}
	})

	t.Run("Empty native tools", func(t *testing.T) {
		signals := ProjectSignals{
			Languages:   []string{"go"},
			NativeTools: []ToolSignal{},
		}
		audits, gaps, rate := AuditGaps(signals, []string{"go"}, []string{"go vet ./..."}, mockLookPathSuccess)
		if len(audits) != 0 || len(gaps) != 0 || rate != 100.0 {
			t.Errorf("expected empty audits and 100.0 rate, got audits=%v, gaps=%v, rate=%v", audits, gaps, rate)
		}
	})

	t.Run("All tool executables and matching lanes", func(t *testing.T) {
		signals := ProjectSignals{
			Languages: []string{"various"},
			NativeTools: []ToolSignal{
				{Tool: "cargo-test", Category: "test", ConfigFile: "Cargo.toml"},
				{Tool: "dotnet-test", Category: "test", ConfigFile: "App.csproj"},
				{Tool: "mvn-test", Category: "test", ConfigFile: "pom.xml"},
				{Tool: "go-test", Category: "test", ConfigFile: "go.mod"},
				{Tool: "go-vet", Category: "lint", ConfigFile: "go.mod"},
				{Tool: "gofmt", Category: "format", ConfigFile: "go.mod"},
				{Tool: "custom-tool", Category: "custom", ConfigFile: "tool.json"},
			},
		}
		executedLanes := []string{
			"cargo test --workspace",
			"dotnet test --no-build",
			"mvn test -B",
			"go test ./...",
			"go vet ./...",
			"gofmt -l .",
		}
		audits, gaps, rate := AuditGaps(signals, []string{"rust", "dotnet", "java", "go"}, executedLanes, mockLookPathSuccess)
		if len(audits) != 7 {
			t.Fatalf("expected 7 audits, got %d", len(audits))
		}
		if len(gaps) != 1 || !strings.Contains(gaps[0], "custom-tool") {
			t.Errorf("expected 1 gap for custom-tool, got %v", gaps)
		}
		expectedRate := float64(6) / float64(7) * 100.0
		if rate != expectedRate {
			t.Errorf("expected rate %f, got %f", expectedRate, rate)
		}
	})
}
