package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The warm-up counts the whole 2xx range as a warm model, as the question
// does: a 299 to the warm-up lets the question through.
func TestAWarmUpAnsweredWith299IsWarm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if isWarmUp(got) {
			w.WriteHeader(299)
			return
		}
		_, _ = w.Write([]byte(`{"response":"answer"}`))
	}))
	defer server.Close()
	client, _ := NewClient(settingsFor(server.URL))
	if text, ok := client.Ask(context.Background(), "p"); !ok || text != "answer" {
		t.Fatalf("%q %v", text, ok)
	}
}
