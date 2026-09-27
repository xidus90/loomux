package runs

import (
	"strings"
	"testing"
)

// A number another run took in the meantime is computed again, not shared.
func TestClaimComputesAgainWhenTheNumberIsTaken(t *testing.T) {
	root := t.TempDir()
	if err := WriteMarker(MarkerPath(root, "0001"), Marker{Flow: "other"}); err != nil {
		t.Fatal(err)
	}
	offered := []string{"0001", "0002"}
	next := func(string) string {
		id := offered[0]
		offered = offered[1:]
		return id
	}
	id, err := claim(root, Marker{Flow: "mine"}, next)
	if err != nil || id != "0002" {
		t.Fatalf("id = %q, err = %v", id, err)
	}
	taken, err := ReadMarker(MarkerPath(root, "0001"))
	if err != nil || taken.Flow != "other" {
		t.Fatalf("the taken marker was overwritten: %+v, %v", taken, err)
	}
}

func TestClaimGivesUpAfterTenTakenNumbers(t *testing.T) {
	root := t.TempDir()
	if err := WriteMarker(MarkerPath(root, "0001"), Marker{Flow: "other"}); err != nil {
		t.Fatal(err)
	}
	asked := 0
	_, err := claim(root, Marker{Flow: "mine"}, func(string) string {
		asked++
		return "0001"
	})
	if err == nil || !strings.Contains(err.Error(), "no free run number") || asked != 10 {
		t.Fatalf("asked %d times, err = %v", asked, err)
	}
}
