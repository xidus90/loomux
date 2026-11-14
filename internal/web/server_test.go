package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testCode = "one-time-code"

// noRedirect keeps the answer a handler gave instead of the page behind it.
var noRedirect = &http.Client{
	Timeout:       5 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// loopbackAny listens on a free port whatever port was asked for, so a test
// never competes for DefaultPort.
func loopbackAny(int) (net.Listener, error) { return net.Listen("tcp", "127.0.0.1:0") }

// ui is a running interface and the clock its login code reads.
type ui struct {
	t        *testing.T
	stateDir string
	port     int
	base     string
	advance  *atomic.Int64
	cancel   context.CancelFunc
	once     sync.Once
	cookie   string
	err      error
	done     chan error
}

// startUI runs the interface on a free loopback port in a state directory of
// its own and waits until it has written ui.json.
func startUI(t *testing.T) *ui { return startUIWith(t, nil) }

// startUIWith is startUI with the options adjusted before the run starts.
func startUIWith(t *testing.T, adjust func(*Options)) *ui {
	t.Helper()
	// The test world never reaches the real state.
	stateDir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", stateDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var advance atomic.Int64
	origin := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	options := Options{
		StateDir: stateDir,
		Version:  "9.9.9",
		Code:     testCode,
		Listen:   loopbackAny,
		Now:      func() time.Time { return origin.Add(time.Duration(advance.Load())) },
	}
	if adjust != nil {
		adjust(&options)
	}
	go func() { done <- Run(ctx, options) }()
	u := &ui{t: t, stateDir: stateDir, advance: &advance, cancel: cancel, done: done}
	t.Cleanup(func() {
		u.stop()
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if state, err := ReadState(stateDir); err == nil {
			u.port = state.Port
			u.base = fmt.Sprintf("http://127.0.0.1:%d", state.Port)
			return u
		}
		if time.Now().After(deadline) {
			t.Fatal("the interface did not write ui.json")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// req is one request: whatever is left empty is not sent.
type req struct {
	method, path, host, origin, cookie string
	form                               url.Values
}

func (u *ui) do(r req) *http.Response {
	u.t.Helper()
	method := r.method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if r.form != nil {
		body = strings.NewReader(r.form.Encode())
	}
	request, err := http.NewRequest(method, u.base+r.path, body)
	if err != nil {
		u.t.Fatal(err)
	}
	if r.form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if r.host != "" {
		request.Host = r.host
	}
	if r.origin != "" {
		request.Header.Set("Origin", r.origin)
	}
	if r.cookie != "" {
		request.Header.Set("Cookie", r.cookie)
	}
	resp, err := noRedirect.Do(request)
	if err != nil {
		u.t.Fatal(err)
	}
	u.t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func bodyOf(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// login posts a code as the sign-in button does.
func (u *ui) login(code string) *http.Response {
	u.t.Helper()
	return u.do(req{method: "POST", path: "/login", form: url.Values{"code": {code}}})
}

// signIn spends the code and returns the cookie header and the form token of
// the session it bought.
func (u *ui) signIn() (cookie, token string) {
	u.t.Helper()
	resp := u.login(testCode)
	if resp.StatusCode != http.StatusSeeOther {
		u.t.Fatalf("login = %d, want 303", resp.StatusCode)
	}
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		u.t.Fatalf("login set %d cookies, want one", len(cookies))
	}
	cookie = cookies[0].Name + "=" + cookies[0].Value
	home := bodyOf(u.t, u.do(req{path: "/", cookie: cookie}))
	_, after, found := strings.Cut(home, `name="csrf" value="`)
	if !found {
		u.t.Fatalf("the home page carries no form token: %s", home)
	}
	token, _, _ = strings.Cut(after, `"`)
	return cookie, token
}

func TestLoginTradesTheCodeForACookieOnce(t *testing.T) {
	u := startUI(t)
	resp := u.login(testCode)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("login = %d to %q, want 303 to /", resp.StatusCode, resp.Header.Get("Location"))
	}
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == cookieName(u.port) {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("no session cookie named after the port")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" {
		t.Errorf("cookie %+v, want HttpOnly, SameSite=Strict and Path=/", cookie)
	}
	if cookie.MaxAge != 0 || !cookie.Expires.IsZero() {
		t.Errorf("cookie %+v outlives the browser session", cookie)
	}
	if second := u.login(testCode); second.StatusCode != http.StatusForbidden {
		t.Errorf("the second use of the code = %d, want 403", second.StatusCode)
	}
}

// A browser that preloads the address, or a link preview, fetches it before
// the user does; only the button's post may spend the code.
func TestOpeningTheLinkSpendsNothing(t *testing.T) {
	u := startUI(t)
	for range 2 {
		resp := u.do(req{path: "/login?code=" + testCode})
		text := bodyOf(t, resp)
		if resp.StatusCode != http.StatusOK || len(resp.Cookies()) != 0 {
			t.Fatalf("the link = %d with %d cookies, want 200 and none", resp.StatusCode, len(resp.Cookies()))
		}
		if !strings.Contains(text, `<form method="post" action="/login">`) ||
			!strings.Contains(text, `name="code" value="`+testCode+`"`) {
			t.Fatalf("the link shows no sign-in button carrying the code: %s", text)
		}
	}
	if resp := u.login(testCode); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("the button after two opened links = %d, want 303", resp.StatusCode)
	}
}

func TestAWrongCodeCostsNothing(t *testing.T) {
	u := startUI(t)
	for _, guess := range []string{"nope", testCode + "x", testCode[:3]} {
		resp := u.login(guess)
		if resp.StatusCode != http.StatusForbidden || len(resp.Cookies()) != 0 {
			t.Fatalf("code %q = %d with %d cookies, want 403 and none", guess, resp.StatusCode, len(resp.Cookies()))
		}
		if !strings.Contains(bodyOf(t, resp), "wrong, already used or expired") {
			t.Errorf("code %q: the refusal does not say why", guess)
		}
	}
	if resp := u.login(testCode); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("the right code after wrong ones = %d, want 303", resp.StatusCode)
	}
}

func TestTheCodeExpiresAfterTwoMinutes(t *testing.T) {
	u := startUI(t)
	u.advance.Store(int64(2*time.Minute + time.Second))
	if resp := u.login(testCode); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("an expired code = %d, want 403", resp.StatusCode)
	}
}

func TestTheCodeHoldsUntilTheLastSecondOfItsLifetime(t *testing.T) {
	u := startUI(t)
	u.advance.Store(int64(2 * time.Minute))
	if resp := u.login(testCode); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("a code exactly two minutes old = %d, want 303", resp.StatusCode)
	}
}

func TestWithoutASessionEverythingLeadsToTheHint(t *testing.T) {
	u := startUI(t)
	for _, path := range []string{"/", "/settings", "/assets/other.css", "/login/"} {
		resp := u.do(req{path: path})
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
			t.Errorf("%s = %d to %q, want 303 to /login", path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
	page := u.do(req{path: "/login"})
	if text := bodyOf(t, page); page.StatusCode != http.StatusOK || !strings.Contains(text, "<code>loomux ui</code>") {
		t.Errorf("the login page = %d %q, want the hint to run loomux ui", page.StatusCode, text)
	}
}

func TestAnUnknownCookieIsNoSession(t *testing.T) {
	u := startUI(t)
	resp := u.do(req{path: "/", cookie: cookieName(u.port) + "=forged"})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("a forged cookie = %d, want 303", resp.StatusCode)
	}
}

// A browser sends a cookie to every port of the host, so the handle of one
// interface reaches whatever else listens on 127.0.0.1; it must not be a
// session of an interface on another port.
func TestTheSessionCookieOfAnotherPortIsNoSession(t *testing.T) {
	u := startUI(t)
	cookie, _ := u.signIn()
	_, handle, _ := strings.Cut(cookie, "=")
	resp := u.do(req{path: "/", cookie: cookieName(u.port+1) + "=" + handle})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("the handle under another port's name = %d, want 303", resp.StatusCode)
	}
}

// The code is exchanged once and forgotten: after it bought a session the
// process holds none of it, neither in the sessions nor in the options the
// server keeps.
func TestTheLoginCodeIsForgottenOnceExchanged(t *testing.T) {
	srv := newServer(7411, Options{Code: testCode}, time.Now)
	if srv.options.Code != "" {
		t.Errorf("the options keep the code: %q", srv.options.Code)
	}
	s := srv.sessions
	if _, _, ok := s.open(testCode); !ok {
		t.Fatal("the code was not accepted")
	}
	if s.code != "" {
		t.Errorf("the code stays in memory: %q", s.code)
	}
	if _, _, ok := s.open(""); ok {
		t.Error("the empty code buys a session once the real one is forgotten")
	}
}

func TestACookieOfTheRightNameAmongOthersStillCounts(t *testing.T) {
	u := startUI(t)
	cookie, _ := u.signIn()
	resp := u.do(req{path: "/", cookie: cookieName(u.port) + "=tossed-by-another-service; " + cookie})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home with two cookies = %d, want 200", resp.StatusCode)
	}
}

func TestTheHomePageShowsWithASession(t *testing.T) {
	u := startUI(t)
	cookie, _ := u.signIn()
	resp := u.do(req{path: "/", cookie: cookie})
	text := bodyOf(t, resp)
	if resp.StatusCode != http.StatusOK || !strings.Contains(text, "9.9.9") || !strings.Contains(text, "Sign out") {
		t.Fatalf("home = %d %q, want the page with the version and a sign-out button", resp.StatusCode, text)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("content type %q", ct)
	}
	if notFound := u.do(req{path: "/nowhere", cookie: cookie}); notFound.StatusCode != http.StatusNotFound {
		t.Errorf("an unknown path = %d, want 404", notFound.StatusCode)
	}
}

func TestEveryAnswerCarriesTheSecurityHeaders(t *testing.T) {
	u := startUI(t)
	cookie, _ := u.signIn()
	for _, r := range []req{
		{path: "/health"}, {path: "/login"}, {path: "/assets/style.css"}, {path: "/", cookie: cookie},
		{path: "/", host: "evil.example"}, {path: "/nowhere", cookie: cookie},
	} {
		h := u.do(r).Header
		for name, want := range map[string]string{
			"Cache-Control":           "no-store",
			"Referrer-Policy":         "no-referrer",
			"X-Content-Type-Options":  "nosniff",
			"X-Frame-Options":         "DENY",
			"Content-Security-Policy": "default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'",
		} {
			if h.Get(name) != want {
				t.Errorf("%+v: %s is %q, want %q", r, name, h.Get(name), want)
			}
		}
	}
}

func TestHealthNamesThisProcessWithoutASession(t *testing.T) {
	u := startUI(t)
	resp := u.do(req{path: "/health"})
	if want := fmt.Sprintf(`{"pid":%d}`, os.Getpid()); strings.TrimSpace(bodyOf(t, resp)) != want {
		t.Errorf("health does not say %s", want)
	}
	if pid, err := Health(u.port); err != nil || pid != os.Getpid() {
		t.Errorf("Health = %d, %v, want this process", pid, err)
	}
}

func TestTheStyleSheetNeedsNoSession(t *testing.T) {
	u := startUI(t)
	resp := u.do(req{path: "/assets/style.css"})
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/css; charset=utf-8" ||
		!strings.Contains(bodyOf(t, resp), "--bg") {
		t.Errorf("style.css = %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}

func TestAForeignHostIsRefusedEverywhere(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	for _, host := range []string{
		"evil.example", "evil.example:" + fmt.Sprint(u.port), "127.0.0.1", "127.0.0.1:1", "[::1]:" + fmt.Sprint(u.port),
		"localhost", "127.0.0.1.evil.example:" + fmt.Sprint(u.port), "",
	} {
		if host == "" {
			continue
		}
		for _, r := range []req{
			{path: "/health", host: host}, {method: "POST", path: "/login", host: host, form: url.Values{"code": {testCode}}},
			{path: "/", host: host, cookie: cookie}, {path: "/assets/style.css", host: host},
			{method: "POST", path: "/logout", host: host, cookie: cookie, form: url.Values{"csrf": {token}}},
		} {
			if resp := u.do(r); resp.StatusCode != http.StatusMisdirectedRequest {
				t.Errorf("Host %q on %s %s = %d, want 421", host, r.method, r.path, resp.StatusCode)
			}
		}
	}
}

func TestBothNamesOfTheListenerAreAccepted(t *testing.T) {
	u := startUI(t)
	for _, host := range []string{
		fmt.Sprintf("127.0.0.1:%d", u.port), fmt.Sprintf("localhost:%d", u.port), fmt.Sprintf("LocalHost:%d", u.port),
	} {
		if resp := u.do(req{path: "/health", host: host}); resp.StatusCode != http.StatusOK {
			t.Errorf("Host %q = %d, want 200", host, resp.StatusCode)
		}
	}
}

func TestAPostNeedsTheSessionsFormToken(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	for name, r := range map[string]req{
		"no field":        {method: "POST", path: "/logout", cookie: cookie, form: url.Values{}},
		"no body":         {method: "POST", path: "/logout", cookie: cookie},
		"wrong token":     {method: "POST", path: "/logout", cookie: cookie, form: url.Values{"csrf": {token + "x"}}},
		"empty token":     {method: "POST", path: "/logout", cookie: cookie, form: url.Values{"csrf": {""}}},
		"token in query":  {method: "POST", path: "/logout?csrf=" + token, cookie: cookie, form: url.Values{}},
		"put without it":  {method: "PUT", path: "/logout", cookie: cookie},
		"delete":          {method: "DELETE", path: "/", cookie: cookie},
		"post to a read":  {method: "POST", path: "/", cookie: cookie, form: url.Values{}},
		"login, no token": {method: "POST", path: "/login", form: url.Values{}},
	} {
		if resp := u.do(r); resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, resp.StatusCode)
		}
	}
	// Still signed in after every refusal.
	if resp := u.do(req{path: "/", cookie: cookie}); resp.StatusCode != http.StatusOK {
		t.Fatalf("the session did not survive the refusals: %d", resp.StatusCode)
	}
}

func TestAPostWithATokenButNoSessionIsSentToTheHint(t *testing.T) {
	u := startUI(t)
	_, token := u.signIn()
	resp := u.do(req{method: "POST", path: "/logout", form: url.Values{"csrf": {token}}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Errorf("a token without its cookie = %d to %q, want 303 to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestAPostFromAnotherOriginIsRefused(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	form := url.Values{"csrf": {token}}
	for _, origin := range []string{"http://evil.example", "null", "http://127.0.0.1:1", "http://localhost:" + fmt.Sprint(u.port)} {
		resp := u.do(req{method: "POST", path: "/logout", cookie: cookie, origin: origin, form: form})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Origin %q = %d, want 403", origin, resp.StatusCode)
		}
	}
	if resp := u.do(req{path: "/", cookie: cookie}); resp.StatusCode != http.StatusOK {
		t.Fatalf("the session did not survive: %d", resp.StatusCode)
	}
	// The page's own origin is what a real form sends.
	own := u.do(req{method: "POST", path: "/logout", cookie: cookie, origin: u.base, form: form})
	if own.StatusCode != http.StatusSeeOther {
		t.Errorf("a post from its own origin = %d, want 303", own.StatusCode)
	}
}

func TestLogoutEndsTheSessionForGood(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	resp := u.do(req{method: "POST", path: "/logout", cookie: cookie, form: url.Values{"csrf": {token}}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("logout = %d to %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	var cleared bool
	for _, c := range resp.Cookies() {
		cleared = cleared || (c.Name == cookieName(u.port) && c.MaxAge < 0)
	}
	if !cleared {
		t.Error("logout did not clear the cookie")
	}
	if again := u.do(req{path: "/", cookie: cookie}); again.StatusCode != http.StatusSeeOther {
		t.Errorf("the old cookie after logout = %d, want 303", again.StatusCode)
	}
	if code := u.login(testCode); code.StatusCode != http.StatusForbidden {
		t.Errorf("the spent code after logout = %d, want 403", code.StatusCode)
	}
}

func TestPageIsA500WhenThePageIsUnknownOrBroken(t *testing.T) {
	s := &server{version: "1"}
	for name, data := range map[string]any{"nowhere": nil, "login": 42} {
		w := httptest.NewRecorder()
		s.page(w, httptest.NewRequest("GET", "/", nil), name, data)
		if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "<html") {
			t.Errorf("page %q = %d %q, want a bare 500", name, w.Code, w.Body.String())
		}
	}
}

func TestRunWritesItsStateAndRemovesItOnStop(t *testing.T) {
	u := startUI(t)
	state, err := ReadState(u.stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if state.PID != os.Getpid() || state.Port != u.port || time.Since(state.Started) > time.Minute {
		t.Errorf("state %+v, want this process on port %d", state, u.port)
	}
	// Nothing secret is on disk.
	raw, _ := os.ReadFile(StatePath(u.stateDir))
	if strings.Contains(string(raw), testCode) {
		t.Error("ui.json holds the login code")
	}
	if err := u.stop(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(StatePath(u.stateDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ui.json is still there after the stop: %v", err)
	}
}

func TestRunLeavesAnotherInterfacesStateAlone(t *testing.T) {
	u := startUI(t)
	foreign := State{PID: os.Getpid() + 1, Port: 1, Started: time.Now()}
	if err := writeState(u.stateDir, foreign); err != nil {
		t.Fatal(err)
	}
	u.stop()
	if state, err := ReadState(u.stateDir); err != nil || state.PID != foreign.PID {
		t.Errorf("the state of another interface is gone: %+v, %v", state, err)
	}
}

func TestRunFallsBackToAFreePortWhenTheDefaultIsTaken(t *testing.T) {
	var asked []int
	listen := func(port int) (net.Listener, error) {
		asked = append(asked, port)
		if port == DefaultPort {
			return nil, errors.New("address already in use")
		}
		return loopbackAny(port)
	}
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, Options{StateDir: dir, Code: testCode, Listen: listen}) }()
	waitForState(t, dir)
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(asked) != 2 || asked[0] != DefaultPort || asked[1] != 0 {
		t.Errorf("ports asked %v, want the default and then any", asked)
	}
}

func TestRunUsesTheDefaultPortWhenItIsFree(t *testing.T) {
	var asked []int
	listen := func(port int) (net.Listener, error) {
		asked = append(asked, port)
		return loopbackAny(port)
	}
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, Options{StateDir: dir, Code: testCode, Listen: listen}) }()
	waitForState(t, dir)
	cancel()
	<-done
	if len(asked) != 1 || asked[0] != DefaultPort {
		t.Errorf("ports asked %v, want only the default", asked)
	}
}

func TestRunListensOnLoopbackByDefault(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, Options{StateDir: dir, Code: testCode}) }()
	state := waitForState(t, dir)
	// Whether DefaultPort was free or not, the interface is on loopback only.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", state.Port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	listener, err := listenLoopback(0)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if ip := listener.Addr().(*net.TCPAddr).IP; !ip.IsLoopback() {
		t.Errorf("the default listener is on %v, want loopback", ip)
	}
	cancel()
	<-done
}

func TestRunFailsWhenNoPortCanBeHad(t *testing.T) {
	err := Run(context.Background(), Options{StateDir: t.TempDir(), Code: testCode,
		Listen: func(int) (net.Listener, error) { return nil, errors.New("no ports") }})
	if err == nil || !strings.Contains(err.Error(), "no ports") {
		t.Fatalf("Run: %v, want the failure to listen", err)
	}
}

func TestRunRefusesToStartWithoutACode(t *testing.T) {
	err := Run(context.Background(), Options{StateDir: t.TempDir(),
		Listen: func(int) (net.Listener, error) { t.Fatal("it listened without a code"); return nil, nil }})
	if err == nil || !strings.Contains(err.Error(), "login code") {
		t.Fatalf("Run: %v, want the missing code named", err)
	}
}

func TestRunFailsWhenItCannotWriteItsState(t *testing.T) {
	dir := t.TempDir()
	// ui.json as a folder cannot be replaced by a file.
	if err := os.MkdirAll(filepath.Join(StatePath(dir), "keep"), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := loopbackAny(0)
	if err != nil {
		t.Fatal(err)
	}
	err = Run(context.Background(), Options{StateDir: dir, Code: testCode,
		Listen: func(int) (net.Listener, error) { return listener, nil }})
	if err == nil || !strings.Contains(err.Error(), "ui.json") {
		t.Fatalf("Run: %v, want the failed state file named", err)
	}
	if _, err := listener.Accept(); err == nil {
		t.Error("the listener stayed open after the failed start")
	}
}

func TestRunReturnsWhatEndsServingEarly(t *testing.T) {
	listener, err := loopbackAny(0)
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	err = Run(context.Background(), Options{StateDir: t.TempDir(), Code: testCode,
		Listen: func(int) (net.Listener, error) { return listener, nil }})
	if err == nil {
		t.Fatal("Run returned nil for a listener that was already closed")
	}
}

func waitForState(t *testing.T, dir string) *State {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if state, err := ReadState(dir); err == nil {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatal("no ui.json")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEveryPageParses(t *testing.T) {
	for _, name := range pageNames {
		if pages()[name] == nil {
			t.Errorf("page %q did not parse", name)
		}
	}
}

// stop ends the interface and returns what Run returned; the cleanup calls it
// too, so a test may have stopped it already.
func (u *ui) stop() error {
	u.once.Do(func() {
		u.cancel()
		u.err = <-u.done
	})
	return u.err
}

func TestARefusedHostDoesNotSpendTheCode(t *testing.T) {
	u := startUI(t)
	if resp := u.do(req{method: "POST", path: "/login", host: "evil.example", form: url.Values{"code": {testCode}}}); resp.StatusCode != http.StatusMisdirectedRequest {
		t.Fatalf("login on a foreign host = %d, want 421", resp.StatusCode)
	}
	if resp := u.login(testCode); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("the code after a refused host = %d, want 303", resp.StatusCode)
	}
}
