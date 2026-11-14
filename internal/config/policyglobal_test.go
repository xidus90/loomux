package config

import (
	"os"
	"strings"
	"testing"
)

func writeGlobal(t *testing.T, body string) string {
	t.Helper()
	state := t.TempDir()
	if err := os.WriteFile(GlobalConfigPath(state), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestReadGlobalPolicy(t *testing.T) {
	if policy, err := ReadGlobalPolicy(t.TempDir()); err != nil || len(policy.Paths)+len(policy.Commands) != 0 {
		t.Fatalf("a missing file is empty: %+v, %v", policy, err)
	}
	if policy, err := ReadGlobalPolicy(""); err != nil || len(policy.Paths)+len(policy.Commands) != 0 {
		t.Fatalf("no state directory is empty: %+v, %v", policy, err)
	}
	state := writeGlobal(t, "[model]\nenabled = true\n\n[[policy.paths.rules]]\nmatch = \"secrets/**\"\nreason = \"keys\"\n\n[[policy.commands.rules]]\nregex = \"curl\"\nreason = \"no curl\"\n")
	policy, err := ReadGlobalPolicy(state)
	if err != nil || len(policy.Paths) != 1 || policy.Paths[0].Reason != "keys" || len(policy.Commands) != 1 || policy.Commands[0].Reason != "no curl" {
		t.Fatalf("%+v, %v", policy, err)
	}
	for name, body := range map[string]string{
		"a malformed glob":     "[[policy.paths.rules]]\nmatch = \"a/[b\"\nreason = \"x\"\n",
		"a missing reason":     "[[policy.paths.rules]]\nmatch = \"a\"\n",
		"a regex that fails":   "[[policy.commands.rules]]\nregex = \"(\"\nreason = \"x\"\n",
		"a guard table":        "[guard]\nmode = \"strict\"\n",
		"an empty guard table": "[guard]\n",
	} {
		state := writeGlobal(t, body)
		if _, err := ReadGlobalPolicy(state); err == nil || !strings.Contains(err.Error(), GlobalConfigPath(state)) {
			t.Errorf("%s: err %v, want one naming the file", name, err)
		}
	}
}

func TestReadGlobalPolicyRefusesAFileItCannotRead(t *testing.T) {
	state := t.TempDir()
	if err := os.Mkdir(GlobalConfigPath(state), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadGlobalPolicy(state); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("err %v", err)
	}
}

func TestEffectivePolicyKeepsGlobalRules(t *testing.T) {
	state := writeGlobal(t, "[[policy.paths.rules]]\nmatch = [\"secrets/**\"]\nreason = \"global\"\n")
	policy, err := ReadEffectivePolicy(t.TempDir(), state)
	if err != nil || len(policy.Paths) != 1 || policy.Paths[0].Reason != "global" {
		t.Fatalf("%+v, %v", policy, err)
	}
	root := writeConfig(t, "[guard]\nmode = \"strict\"\n\n[[policy.paths.rules]]\nmatch = \"own/*\"\nreason = \"project\"\n\n[[policy.commands.rules]]\nregex = \"x\"\nreason = \"p\"\n")
	policy, err = ReadEffectivePolicy(root, state)
	if err != nil || len(policy.Paths) != 2 || policy.Paths[0].Reason != "global" || policy.Paths[1].Reason != "project" || len(policy.Commands) != 1 || !policy.Strict {
		t.Fatalf("global first, [guard] from the project: %+v, %v", policy, err)
	}
	if _, err := ReadEffectivePolicy(t.TempDir(), writeGlobal(t, "[guard]\n")); err == nil {
		t.Fatal("a broken global file must fail")
	}
	if _, err := ReadEffectivePolicy(writeConfig(t, "[[policy.paths.rules]\n"), state); err == nil {
		t.Fatal("a broken project file must fail")
	}
}

func TestParsePolicyNamesTheFileItWasGiven(t *testing.T) {
	if _, err := ParsePolicy("some/where.toml", []byte("[[policy.paths.rules]\n")); err == nil || !strings.Contains(err.Error(), "some/where.toml") {
		t.Fatalf("err %v", err)
	}
}
