package web

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

const (
	// cookieBase starts the name of the session cookie, which carries a random
	// handle and no state: everything a session knows lives in this process.
	// Browsers scope cookies by host and not by port; the port in the name
	// keeps this interface from reading, or overwriting, the cookie of
	// another one on the same host.
	cookieBase = "loomux_ui"
	// codeLifetime is how long the login code may still be exchanged. It is
	// counted from the start of the process, which is what the browser opens
	// a moment later.
	codeLifetime = 2 * time.Minute
)

// sessions holds the one login code and the sessions it bought. Nothing of
// it is written down: a session ends with the process.
type sessions struct {
	mu     sync.Mutex
	code   string
	issued time.Time
	used   bool
	byID   map[string]string // session handle -> its form token
	now    func() time.Time
}

// cookieName is the session cookie of the interface on a port.
func cookieName(port int) string { return cookieBase + "_" + strconv.Itoa(port) }

func newSessions(code string, now func() time.Time) *sessions {
	return &sessions{code: code, issued: now(), byID: map[string]string{}, now: now}
}

// open exchanges the login code for a session: the handle for the cookie and
// the token its forms carry. A code buys one session, once, within
// codeLifetime; a wrong code costs nothing, so a page that merely guesses
// cannot use up the real one.
func (s *sessions) open(code string) (id, csrf string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// The comparison runs on every attempt and takes the same time for any
	// wrong code of the right length.
	right := subtle.ConstantTimeCompare([]byte(code), []byte(s.code)) == 1
	if !right || s.used || s.now().Sub(s.issued) > codeLifetime {
		return "", "", false
	}
	// The code has done its work and is not kept; used stops the empty one
	// from matching.
	s.used, s.code = true, ""
	id, csrf = serve.NewToken(), serve.NewToken()
	s.byID[id] = csrf
	return id, csrf, true
}

// find returns the session a request's cookies name, if any. A request may
// carry several cookies of the name (another service on the same host can set
// one); any known handle counts, an unknown one never does.
func (s *sessions) find(r *http.Request, name string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range r.CookiesNamed(name) {
		if csrf, ok := s.byID[c.Value]; ok {
			return &session{id: c.Value, csrf: csrf}
		}
	}
	return nil
}

// close forgets a session.
func (s *sessions) close(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, id)
}

// session is what a request with a valid cookie is.
type session struct {
	id   string
	csrf string
}

// formToken says whether got is this session's form token, in constant time.
func (s *session) formToken(got string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.csrf)) == 1
}
