package compat

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestByStackGroupsJobsUnderEveryStack(t *testing.T) {
	jobs := []Job{
		{Entry: "php/php-8.2/php-syntax-8.2", OS: "linux"},
		{Entry: "go/go-1.27/go-module", OS: "linux"},
		{Entry: "php/php-8.2/php-syntax-8.2", OS: "windows"},
	}
	by := ByStack(jobs)
	if len(by) != len(runtimeOf) {
		t.Fatalf("stacks %d, want %d", len(by), len(runtimeOf))
	}
	want := []Job{jobs[0], jobs[2]}
	if !reflect.DeepEqual(by["php"], want) {
		t.Errorf("php = %v, want %v", by["php"], want)
	}
	if !reflect.DeepEqual(by["go"], jobs[1:2]) {
		t.Errorf("go = %v", by["go"])
	}
	// An empty stack must encode as [], which the workflow tests for.
	if by["java"] == nil || len(by["java"]) != 0 {
		t.Errorf("java = %#v, want an empty non-nil list", by["java"])
	}
}

// TestWorkflowRunsEveryStack holds .github/workflows/compat.yml to the stacks:
// each runs as a job of its own through compat-run.yml, and the required
// compat job waits for all of them.
func TestWorkflowRunsEveryStack(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/compat.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Uses string            `yaml:"uses"`
			With map[string]string `yaml:"with"`
			// A string or a list, as Actions allows both.
			Needs yaml.Node `yaml:"needs"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	needs := wf.Jobs["compat"].Needs
	var gate []string
	if err := needs.Decode(&gate); err != nil {
		t.Fatal(err)
	}
	for stack := range runtimeOf {
		j, ok := wf.Jobs[stack]
		switch {
		case !ok:
			t.Errorf("no job %q", stack)
		case j.Uses != "./.github/workflows/compat-run.yml" || j.With["stack"] != stack:
			t.Errorf("job %q: uses %q with stack %q", stack, j.Uses, j.With["stack"])
		case !slices.Contains(gate, stack):
			t.Errorf("compat does not need %q", stack)
		}
	}
}
