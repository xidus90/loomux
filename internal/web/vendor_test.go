package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
)

// forceGraphSHA256 pins the embedded force-graph build; AGENTS.md names its
// version and origin.
const forceGraphSHA256 = "f59db59ba92ff5a76ed93d7a4d77a703b176260e5d9cd29f4f8eb4f6c369a629"

func TestForceGraphIsThePinnedFile(t *testing.T) {
	data, err := assets.ReadFile("assets/vendor/force-graph.min.js")
	if err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != forceGraphSHA256 {
		t.Errorf("force-graph.min.js is %x, not the pinned build", sum)
	}
}

func TestForceGraphIsServedAsScript(t *testing.T) {
	u := startUI(t)
	cookie, _ := u.signIn()
	resp := u.do(req{path: "/assets/vendor/force-graph.min.js", cookie: cookie})
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/javascript; charset=utf-8" ||
		!strings.Contains(bodyOf(t, resp), "ForceGraph") {
		t.Errorf("force-graph.min.js = %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}
