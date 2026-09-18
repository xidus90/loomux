package serve

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// NewToken is one listener's secret: 32 bytes of randomness, base64url.
//
// Drawing it cannot fail, so there is nothing to return but the token. Since
// Go 1.24 crypto/rand.Read never returns an error: it fills the buffer
// entirely or crashes the program irrecoverably. go.mod is at go 1.26.0, so
// that guarantee holds here, and an error result would only be a dead branch
// at every call site.
func NewToken() string {
	buffer := make([]byte, 32)
	_, _ = rand.Read(buffer)
	return base64.RawURLEncoding.EncodeToString(buffer)
}

// RequireToken refuses every request that does not carry the listener's token,
// before anything MCP-shaped happens.
//
// The whole header value is compared, scheme included, rather than the token
// alone. That makes "bearer secret" a refusal as much as "Basic secret" is,
// which is stricter than RFC 6750 asks for and is what the only client here --
// the MCP SDK, which always writes "Bearer " -- sends anyway. One code path is
// worth more than a tolerance nobody exercises.
//
// The comparison is constant-time. The secret never leaves this machine, but a
// timing side channel is cheap to avoid and expensive to argue about. What the
// early exit on unequal lengths leaks is the length of "Bearer " plus a token,
// and that is a constant of the design, not a secret.
func RequireToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got := []byte(strings.TrimSpace(req.Header.Get("Authorization")))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	})
}
