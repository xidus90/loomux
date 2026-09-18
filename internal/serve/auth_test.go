package serve_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xidus90/loomux/internal/serve"
)

func TestNewTokenIsThirtyTwoRandomBytes(t *testing.T) {
	first := serve.NewToken()
	raw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("decode %q: %v", first, err)
	}
	if len(raw) != 32 {
		t.Errorf("token carries %d bytes, want 32", len(raw))
	}
	if second := serve.NewToken(); first == second {
		t.Error("two tokens are the same")
	}
}

func TestRequireTokenLetsTheRightTokenThrough(t *testing.T) {
	handler := serve.RequireToken("secret", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Errorf("status %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestRequireTokenRefusesAWrongToken(t *testing.T) {
	var reached bool
	handler := serve.RequireToken("secret", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	for _, header := range []string{
		// Shorter or longer than "Bearer secret": refused at the
		// length short-circuit of the constant-time compare.
		"", "Bearer", "Bearer wrong", "secret", "Basic secret", "Bearer secre",
		// Exactly as long as "Bearer secret": these reach the
		// byte-by-byte comparison itself. The second one pins that the
		// scheme is compared case-sensitively.
		"Bearer secres", "bearer secret",
	} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q gave status %d, want 401", header, rec.Code)
		}
	}
	if reached {
		t.Error("a refused request reached the handler")
	}
}
