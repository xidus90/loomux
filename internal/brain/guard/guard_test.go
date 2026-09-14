package guard

import (
	"strings"
	"testing"
)

func TestWriteTargetsCollectsEveryKnownKey(t *testing.T) {
	got := WriteTargets(map[string]any{"file_path": "a", "notebook_path": "b", "TargetFile": "c", "target_file": "d", "other": "e", "file_path_empty": ""})
	if strings.Join(got, ",") != "a,b,c,d" {
		t.Fatalf("%v", got)
	}
}

func TestIsWritingTool(t *testing.T) {
	if !IsWritingTool("Edit") || !IsWritingTool("write_to_file") || IsWritingTool("Bash") {
		t.Fatal("wrong classification")
	}
}

func TestCallReadsBothPayloadShapes(t *testing.T) {
	name, args := Call(map[string]any{"tool_name": "Write", "tool_input": map[string]any{"file_path": "a"}})
	if name != "Write" || args["file_path"] != "a" {
		t.Fatalf("host shape: %q %v", name, args)
	}
	name, args = Call(map[string]any{"toolCall": map[string]any{"name": "write_to_file", "args": map[string]any{"TargetFile": "b"}}})
	if name != "write_to_file" || args["TargetFile"] != "b" {
		t.Fatalf("antigravity shape: %q %v", name, args)
	}
}
