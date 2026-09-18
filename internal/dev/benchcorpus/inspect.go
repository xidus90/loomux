package benchcorpus

import (
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
)

// InspectProject inspects an fs.FS filesystem for Claude configuration, git hooks,
// languages, and native build/test/lint tools.
func InspectProject(root fs.FS) ProjectSignals {
	var signals ProjectSignals
	signals.HasClaude, signals.ClaudeHooks = inspectClaude(root)
	signals.HasGitHooks = inspectGitHooks(root)
	signals.NativeTools, signals.Languages = inspectToolsAndLanguages(root)
	return signals
}

// inspectClaude reads the command hooks of Claude Code's settings, which
// group them per event and per matcher; events come out in name order so a
// report stays stable across runs.
func inspectClaude(root fs.FS) (bool, []ClaudeHook) {
	data, err := fs.ReadFile(root, ".claude/settings.json")
	if err != nil {
		data, err = fs.ReadFile(root, ".claude.json")
	}
	if err != nil {
		return false, nil
	}

	var parsed struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return true, nil
	}
	events := make([]string, 0, len(parsed.Hooks))
	for event := range parsed.Hooks {
		events = append(events, event)
	}
	sort.Strings(events)

	var hooks []ClaudeHook
	for _, event := range events {
		for _, group := range parsed.Hooks[event] {
			for _, h := range group.Hooks {
				if h.Type == "command" && h.Command != "" {
					hooks = append(hooks, ClaudeHook{Event: event, Command: h.Command})
				}
			}
		}
	}
	return true, hooks
}

func inspectGitHooks(root fs.FS) bool {
	if _, err := fs.Stat(root, ".githooks"); err == nil {
		return true
	}
	if entries, err := fs.ReadDir(root, ".git/hooks"); err == nil && len(entries) > 0 {
		return true
	}
	return false
}

func inspectToolsAndLanguages(root fs.FS) ([]ToolSignal, []string) {
	var tools []ToolSignal
	langMap := make(map[string]bool)

	addTool := func(tool, category, configFile string) {
		tools = append(tools, ToolSignal{Tool: tool, Category: category, ConfigFile: configFile})
	}

	entries, err := fs.ReadDir(root, ".")
	if err != nil {
		return nil, nil
	}

	for _, entry := range entries {
		name := entry.Name()

		// Python
		if name == "pyproject.toml" {
			langMap["python"] = true
			data, _ := fs.ReadFile(root, name)
			content := string(data)
			if strings.Contains(content, "ruff") {
				addTool("ruff", "lint", name)
			}
			if strings.Contains(content, "mypy") {
				addTool("mypy", "typecheck", name)
			}
			if strings.Contains(content, "pytest") {
				addTool("pytest", "test", name)
			}
		}

		// JS/TS
		if name == "package.json" {
			langMap["javascript"] = true
			data, _ := fs.ReadFile(root, name)
			inspectPackageJSON(data, name, addTool, langMap)
		}
		if name == "tsconfig.json" {
			langMap["typescript"] = true
			addTool("tsc", "typecheck", name)
		}

		// Go
		if name == "go.mod" {
			langMap["go"] = true
			addTool("go-vet", "lint", name)
			addTool("go-test", "test", name)
		}
		if name == ".golangci.yml" || name == ".golangci.yaml" {
			addTool("golangci-lint", "lint", name)
		}

		// Rust
		if name == "Cargo.toml" {
			langMap["rust"] = true
			addTool("cargo-clippy", "lint", name)
			addTool("cargo-test", "test", name)
			addTool("cargo-fmt", "format", name)
		}

		// Java
		if name == "pom.xml" {
			langMap["java"] = true
			addTool("mvn-test", "test", name)
		}

		// C#
		if strings.HasSuffix(name, ".csproj") || strings.HasSuffix(name, ".sln") {
			langMap["csharp"] = true
			addTool("dotnet-test", "test", name)
		}

		// C++
		if name == "CMakeLists.txt" {
			langMap["cpp"] = true
			addTool("cmake", "build", name)
		}
		if name == ".clang-format" {
			addTool("clang-format", "format", name)
		}
		if name == ".clang-tidy" {
			addTool("clang-tidy", "lint", name)
		}

		// Ruby
		if name == "Gemfile" {
			langMap["ruby"] = true
			data, _ := fs.ReadFile(root, name)
			content := string(data)
			if strings.Contains(content, "rubocop") {
				addTool("rubocop", "lint", name)
			}
			if strings.Contains(content, "rspec") {
				addTool("rspec", "test", name)
			}
		}

		// PHP
		if name == "composer.json" {
			langMap["php"] = true
			data, _ := fs.ReadFile(root, name)
			content := string(data)
			if strings.Contains(content, "phpunit") {
				addTool("phpunit", "test", name)
			}
			if strings.Contains(content, "phpstan") {
				addTool("phpstan", "typecheck", name)
			}
		}

		// Shell
		if strings.HasSuffix(name, ".sh") || name == ".shellcheckrc" {
			langMap["shell"] = true
			addTool("shellcheck", "lint", name)
		}
	}

	var languages []string
	for l := range langMap {
		languages = append(languages, l)
	}
	sort.Strings(languages)

	return tools, languages
}

func inspectPackageJSON(data []byte, filename string, addTool func(tool, category, file string), langMap map[string]bool) {
	var pkg struct {
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	allText := ""
	for _, v := range pkg.Scripts {
		allText += " " + v
	}
	for k := range pkg.Dependencies {
		allText += " " + k
	}
	for k := range pkg.DevDependencies {
		allText += " " + k
	}

	if strings.Contains(allText, "typescript") || strings.Contains(allText, "tsc") {
		langMap["typescript"] = true
	}
	if strings.Contains(allText, "eslint") {
		addTool("eslint", "lint", filename)
	}
	if strings.Contains(allText, "prettier") {
		addTool("prettier", "format", filename)
	}
	if strings.Contains(allText, "tsc") {
		addTool("tsc", "typecheck", filename)
	}
	if strings.Contains(allText, "vitest") {
		addTool("vitest", "test", filename)
	}
	if strings.Contains(allText, "jest") {
		addTool("jest", "test", filename)
	}
}
