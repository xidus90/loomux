// Package web is the local web interface: one process on a loopback port,
// reached through a login code that only the terminal that started it knows.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/xidus90/loomux/internal/setup"
)

// DefaultPort is the port the interface asks for first. It is a fixed number
// so that a bookmark survives a restart; when something else holds it, the
// operating system chooses.
const DefaultPort = 7411

// shutdownTimeout bounds the orderly stop; what is still in flight after it
// loses.
const shutdownTimeout = 5 * time.Second

// Options are what one run needs. Everything hangs off StateDir, so that a
// test isolates its interface from the real one.
type Options struct {
	StateDir string
	Version  string
	// Code is the one-time login code the starting process handed over.
	Code string
	// Listen opens the listener for a port, 0 for any. Nil listens on the
	// loopback address, and only there.
	Listen func(port int) (net.Listener, error)
	// SetupDeps builds what a project setup runs with. The command line fills
	// it, so this package knows nothing of how a setup acts.
	SetupDeps func(root string, f setup.Facts, scope string, stdout, stderr io.Writer) setup.Deps
	// Gather reads what a project already is. The command line fills it with
	// the host's own view of the machine (home directory, git, installed
	// binaries), which this package has no business knowing. SetupDeps and
	// Gather are preconditions of the Add project pages: the command line
	// always fills both, and a page reached without them fails the request.
	Gather func(root string) (setup.Facts, error)
	// SearchStatus is what `loomux brain status` prints for the vault of a
	// state directory. The command line fills it, because the search engine
	// behind it is not this package's to start; the dashboard shows an
	// interface without one as having no answer.
	SearchStatus func(stateDir string) (string, error)
	// Now is the clock of the login code's lifetime; nil is time.Now.
	Now func() time.Time
}

// server is one run's state.
type server struct {
	port     int
	version  string
	sessions *sessions
	options  Options
	now      func() time.Time
}

// newServer is the state of a run on a port. The login code goes to the
// sessions, which forget it once it is exchanged; the options the server
// keeps for its lifetime do not hold it.
func newServer(port int, o Options, now func() time.Time) *server {
	sessions := newSessions(o.Code, now)
	o.Code = ""
	return &server{port: port, version: o.Version, sessions: sessions, options: o, now: now}
}

// listenLoopback is the default Listen. Loopback, never a wildcard: the
// interface writes configuration and must not be on the network.
func listenLoopback(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

// Run listens, writes ui.json and serves until ctx ends. It returns nil on an
// orderly stop.
//
// The port is DefaultPort and, when that cannot be had for any reason, a free
// one: a second user of the machine, or a service of another program, must
// not keep the interface from starting.
func Run(ctx context.Context, o Options) error {
	if o.Code == "" {
		return errors.New("no login code: the interface is started by `loomux ui`")
	}
	listen := o.Listen
	if listen == nil {
		listen = listenLoopback
	}
	listener, err := listen(DefaultPort)
	if err != nil {
		if listener, err = listen(0); err != nil {
			return fmt.Errorf("listen: %w", err)
		}
	}
	// Listen is a seam a test may fill with anything; the production one
	// returns a TCP listener.
	port := listener.Addr().(*net.TCPAddr).Port
	now := o.Now
	if now == nil {
		now = time.Now
	}
	s := newServer(port, o, now)

	// The state is written after the listener exists and before serving
	// starts: a starter that finds ui.json finds a port that answers as soon
	// as the first request is read.
	if err := writeState(o.StateDir, State{PID: os.Getpid(), Port: port, Started: now()}); err != nil {
		listener.Close()
		return err
	}
	defer removeOwnState(o.StateDir)

	httpServer := &http.Server{Handler: s.chain(s.routes()), ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- httpServer.Serve(listener) }()
	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = httpServer.Shutdown(shutdown)
	<-served
	return nil
}

// removeOwnState removes ui.json when it still names this process: another
// interface started later is not the one that ends now.
func removeOwnState(stateDir string) {
	if state, err := ReadState(stateDir); err == nil && state.PID == os.Getpid() {
		_ = removeState(stateDir)
	}
}

// routes are the pages of the interface. A route that writes is a POST: the
// middleware holds every one of those behind the session's form token.
func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /assets/style.css", style)
	mux.HandleFunc("GET /{$}", s.dashboard)
	mux.HandleFunc("GET /dashboard/edit", s.dashboardEdit)
	mux.HandleFunc("POST /dashboard/edit", s.dashboardChange)
	mux.HandleFunc("POST /projects/{id}/forget", s.forget)
	mux.HandleFunc("GET /settings/{id}", s.settings)
	mux.HandleFunc("GET /settings/{id}/raw", s.raw)
	mux.HandleFunc("POST /settings/{id}/preview", s.preview)
	mux.HandleFunc("POST /settings/{id}/apply", s.apply)
	mux.HandleFunc("POST /settings/{id}/rules/preview", s.rulePreview)
	mux.HandleFunc("POST /settings/{id}/proposals/{pid}/apply", s.applyProposal)
	mux.HandleFunc("POST /settings/{id}/proposals/{pid}/reject", s.rejectProposal)
	mux.HandleFunc("GET /settings/{id}/llm", s.llmPage)
	mux.HandleFunc("POST /settings/{id}/llm", s.llmSet)
	mux.HandleFunc("GET /projects/new", s.wizardPath)
	mux.HandleFunc("POST /projects/new/facts", s.wizardFacts)
	mux.HandleFunc("POST /projects/new/plan", s.wizardPlan)
	mux.HandleFunc("POST /projects/new/apply", s.wizardApply)
	mux.HandleFunc("POST /logout", s.logout)
	return mux
}

// health tells the starter, and a later `loomux ui stop`, which process
// answers on this port.
func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"pid": os.Getpid()})
}

// loginView is the data of the login page: the code of the link it was
// opened with, or that a posted code was refused.
type loginView struct {
	Code    string
	Refused bool
}

// loginPage is where every request without a session ends up. Opened with
// the link's code, it shows a button that posts the code and spends nothing
// itself: a browser that preloads an address, or a link preview, fetches the
// link before the user does.
func (s *server) loginPage(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "login", loginView{Code: r.URL.Query().Get("code")})
}

// login exchanges the posted code for a session cookie.
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	id, _, ok := s.sessions.open(r.PostFormValue("code"))
	if !ok {
		s.render(w, r, http.StatusForbidden, "login", loginView{Refused: true})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(s.port),
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// logout ends the session. The code is spent, so signing in again takes a new
// `loomux ui`.
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	s.sessions.close(sessionOf(r).id)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(s.port),
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
