package web

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// maxFormBytes bounds what a form may send; the interface's forms are a few
// fields.
const maxFormBytes = 1 << 20

// formField is the hidden field every form carries its session's token in.
const formField = "csrf"

// publicPaths are what answers without a session: the health probe of the
// starter, the page that says how to sign in, and the style sheet that page
// needs. Whole paths, never prefixes.
var publicPaths = []string{"/health", "/login", "/assets/style.css"}

type sessionKey struct{}

// sessionOf is the session the middleware found, nil on a public request
// without one.
func sessionOf(r *http.Request) *session {
	s, _ := r.Context().Value(sessionKey{}).(*session)
	return s
}

// chain wraps the routes in the order a request is judged: headers for every
// answer, then the Host header, then cross-origin protection, then the session
// and, on anything that is not a read, the form token.
func (s *server) chain(routes http.Handler) http.Handler {
	h := s.requireSession(routes)
	h = http.NewCrossOriginProtection().Handler(h)
	h = s.checkHost(h)
	return secureHeaders(h)
}

// secureHeaders are on every answer. The login code travels in a URL once, so
// no referrer leaves the page it lands on, and nothing is cached or framed.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// checkHost refuses a request whose Host header is not this listener's own
// address. A page of another origin can aim a name of its own at 127.0.0.1
// (DNS rebinding) and then read the answers as same-origin; its requests
// carry its name in Host.
func (s *server) checkHost(next http.Handler) http.Handler {
	allowed := []string{fmt.Sprintf("127.0.0.1:%d", s.port), fmt.Sprintf("localhost:%d", s.port)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !slices.Contains(allowed, strings.ToLower(r.Host)) {
			http.Error(w, "misdirected request", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireSession sends a request without a session to the page that says how
// to get one, and refuses a write without its session's form token.
func (s *server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := s.sessions.find(r, cookieName(s.port))
		if sess == nil && !slices.Contains(publicPaths, r.URL.Path) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			// Only the body: a token in the query would be one more place a
			// token ends up in a log.
			r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
			// The sign-in form has no session yet; the login code it posts
			// is its token.
			signIn := r.Method == http.MethodPost && r.URL.Path == "/login"
			if !signIn && (sess == nil || !sess.formToken(r.PostFormValue(formField))) {
				http.Error(w, "missing or wrong form token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess)))
	})
}
