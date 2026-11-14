package schema

import (
	"slices"
	"strings"
	"testing"
)

func TestUnknownListsWhatNoKeyDescribes(t *testing.T) {
	text := `
[commit]
language = "de"
colour = "red"

[[policy.paths.rules]]
match = ["a"]
reason = "r"

[policy.commands]
extra = 1

[agent.models.fast]
provider = "claude"
temperature = 3

[agent.roles]
reviewer = "fast"

[verify.profiles]
edit = ["lint"]

[verify.go]
test = "go test"

[dotnet]
sdk = "8"

[verify.python.tools]
ruff = "0.9"
`
	got, err := Unknown(Keys(), text)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"agent.models.fast.temperature", "commit.colour", "dotnet", "policy.commands.extra", "verify.go"}
	if !slices.Equal(got, want) {
		t.Errorf("Unknown = %v, want %v", got, want)
	}
}

func TestUnknownOfAFileThatKnowsEverythingIsEmpty(t *testing.T) {
	got, err := Unknown(Keys(), "[commit]\nlanguage = \"de\"\n")
	if err != nil || len(got) != 0 {
		t.Errorf("Unknown = %v, %v", got, err)
	}
}

func TestUnknownReadsTheKeysItIsGiven(t *testing.T) {
	// The machine-wide file has no [commit].
	got, err := Unknown(GlobalKeys(), "[commit]\nlanguage = \"de\"\n[model]\nenabled = true\n")
	if err != nil || !slices.Equal(got, []string{"commit"}) {
		t.Errorf("Unknown = %v, %v", got, err)
	}
}

func TestUnknownRefusesATextThatIsNoTOML(t *testing.T) {
	if _, err := Unknown(Keys(), "[commit\n"); err == nil || !strings.Contains(err.Error(), "not valid TOML") {
		t.Errorf("err = %v", err)
	}
}
