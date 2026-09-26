package benchreport

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestStampIsUTCToTheMinute(t *testing.T) {
	at := time.Date(2026, 9, 26, 23, 59, 30, 0, time.FixedZone("CEST", 2*3600))
	if got := Stamp(at); got != "2026-09-26-2159" {
		t.Fatalf("Stamp = %q", got)
	}
}

func TestReportJSONIsIndentedWithANewlineAndOmitsAnEmptyPayload(t *testing.T) {
	r := Report{Schema: Schema, Command: "hooks", Stamp: "s",
		Environment: Environment{OS: "windows", Arch: "amd64", CPU: "c", Go: "go1.27.0", Loomux: "dev"},
		Timings:     []Timing{Summarize("a", 1, []float64{2})}}
	got, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(got, []byte("}\n")) || bytes.Contains(got, []byte(`"payload"`)) {
		t.Fatalf("unexpected JSON:\n%s", got)
	}
	if !bytes.Contains(got, []byte("\n  \"schema\": 1,")) {
		t.Fatalf("not indented by two:\n%s", got)
	}
}

func TestReportJSONRefusesAPayloadThatIsNoJSON(t *testing.T) {
	r := Report{Schema: Schema, Payload: json.RawMessage("{")}
	if got, err := r.JSON(); err == nil {
		t.Fatalf("no error, got:\n%s", got)
	}
}

func TestReportJSONNamesTheBackboneOnlyWhenKnown(t *testing.T) {
	for backbone, want := range map[string]bool{"vulkan": true, "": false} {
		r := Report{Schema: Schema, Environment: Environment{Backbone: backbone}}
		got, err := r.JSON()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(got, []byte(`"backbone": "vulkan"`)) != want || bytes.Contains(got, []byte(`"backbone"`)) != want {
			t.Fatalf("backbone %q:\n%s", backbone, got)
		}
	}
}
