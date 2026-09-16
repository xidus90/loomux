package search_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
)

type mockSession struct {
	callFunc  func(name string, args map[string]any) (map[string]any, error)
	closeFunc func() error
}

func (m *mockSession) Call(name string, args map[string]any) (map[string]any, error) {
	if m.callFunc != nil {
		return m.callFunc(name, args)
	}
	return nil, nil
}

func (m *mockSession) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestQmdMcpPort_ProfileMapping(t *testing.T) {
	tests := []struct {
		name        string
		profile     search.Profile
		checkParams func(t *testing.T, args map[string]any)
	}{
		{
			name:    "fast profile",
			profile: search.ProfileFast,
			checkParams: func(t *testing.T, args map[string]any) {
				if args["rerank"] != false {
					t.Errorf("expected rerank=false, got %v", args["rerank"])
				}
				searches, ok := args["searches"].([]map[string]any)
				if !ok || len(searches) != 1 {
					t.Fatalf("expected 1 search spec in searches, got %v", args["searches"])
				}
				if searches[0]["type"] != "vec" || searches[0]["query"] != "q" {
					t.Errorf("expected vec query 'q', got %v", searches[0])
				}
			},
		},
		{
			name:    "keyword profile",
			profile: search.ProfileKeyword,
			checkParams: func(t *testing.T, args map[string]any) {
				if args["rerank"] != false {
					t.Errorf("expected rerank=false, got %v", args["rerank"])
				}
				searches, ok := args["searches"].([]map[string]any)
				if !ok || len(searches) != 1 {
					t.Fatalf("expected 1 search spec in searches, got %v", args["searches"])
				}
				if searches[0]["type"] != "lex" || searches[0]["query"] != "q" {
					t.Errorf("expected lex query 'q', got %v", searches[0])
				}
			},
		},
		{
			name:    "full profile",
			profile: search.ProfileFull,
			checkParams: func(t *testing.T, args map[string]any) {
				if args["rerank"] != true {
					t.Errorf("expected rerank=true, got %v", args["rerank"])
				}
				if args["query"] != "q" {
					t.Errorf("expected query='q', got %v", args["query"])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedName string
			var capturedArgs map[string]any

			session := &mockSession{
				callFunc: func(name string, args map[string]any) (map[string]any, error) {
					capturedName = name
					capturedArgs = args
					return map[string]any{
						"structuredContent": map[string]any{
							"results": []any{},
						},
					}, nil
				},
			}

			port := search.NewQmdMcpPort(
				search.WithConnect(func(env map[string]string) (search.Session, error) {
					return session, nil
				}),
			)

			hits, err := port.Search("q", []string{"col1"}, tc.profile, 5)
			if err != nil {
				t.Fatalf("unexpected Search error: %v", err)
			}
			if len(hits) != 0 {
				t.Errorf("expected 0 hits, got %d", len(hits))
			}
			if capturedName != "query" {
				t.Errorf("expected call to 'query', got %q", capturedName)
			}
			if capturedArgs["limit"] != 5 {
				t.Errorf("expected limit=5, got %v", capturedArgs["limit"])
			}
			tc.checkParams(t, capturedArgs)
		})
	}
}

func TestQmdMcpPort_ResponseTranslation(t *testing.T) {
	session := &mockSession{
		callFunc: func(name string, args map[string]any) (map[string]any, error) {
			return map[string]any{
				"structuredContent": map[string]any{
					"results": []any{
						map[string]any{
							"file":    "col1/docs/page.md",
							"line":    42,
							"title":   "My Doc",
							"snippet": "some text",
							"score":   0.95,
							"docid":   "doc-xyz",
						},
						map[string]any{
							"file": "loose.md",
						},
					},
				},
			}, nil
		},
	}

	port := search.NewQmdMcpPort(
		search.WithConnect(func(env map[string]string) (search.Session, error) {
			return session, nil
		}),
	)

	hits, err := port.Search("test", []string{"col1"}, search.ProfileFast, 10)
	if err != nil {
		t.Fatalf("unexpected Search error: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}

	h0 := hits[0]
	if h0.Collection != "col1" || h0.Relative != "docs/page.md" || h0.Line != 42 || h0.Title != "My Doc" || h0.Snippet != "some text" || h0.Score != 0.95 || h0.ContentKey != "doc-xyz" {
		t.Errorf("unexpected hit 0: %+v", h0)
	}

	h1 := hits[1]
	if h1.Collection != "col1" || h1.Relative != "loose.md" || h1.Line != 1 || h1.Score != 0.0 {
		t.Errorf("unexpected hit 1 defaults: %+v", h1)
	}
}

func TestQmdMcpPort_Serialization(t *testing.T) {
	var inFlight int64
	var maxConcurrent int64

	session := &mockSession{
		callFunc: func(name string, args map[string]any) (map[string]any, error) {
			current := atomic.AddInt64(&inFlight, 1)
			defer atomic.AddInt64(&inFlight, -1)

			for {
				old := atomic.LoadInt64(&maxConcurrent)
				if current <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			return map[string]any{
				"structuredContent": map[string]any{
					"results": []any{},
				},
			}, nil
		},
	}

	port := search.NewQmdMcpPort(
		search.WithConnect(func(env map[string]string) (search.Session, error) {
			return session, nil
		}),
	)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = port.Search("concurrent", []string{"col"}, search.ProfileFast, 1)
		}()
	}
	wg.Wait()

	if maxConcurrent != 1 {
		t.Errorf("expected max concurrency 1 due to mutex serialization, got %d", maxConcurrent)
	}
}

func TestQmdMcpPort_ColdRetry(t *testing.T) {
	t.Run("succeeds on third call", func(t *testing.T) {
		var connects, calls int
		port := search.NewQmdMcpPort(
			search.WithColdAttempts(3),
			search.WithConnect(func(env map[string]string) (search.Session, error) {
				connects++
				return &mockSession{
					callFunc: func(name string, args map[string]any) (map[string]any, error) {
						calls++
						if calls < 3 {
							return nil, errors.New("the daemon hung up")
						}
						return map[string]any{
							"structuredContent": map[string]any{
								"results": []any{},
							},
						}, nil
					},
				}, nil
			}),
		)

		hits, err := port.Search("query", []string{"c"}, search.ProfileFast, 1)
		if err != nil {
			t.Fatalf("unexpected Search error: %v", err)
		}
		if len(hits) != 0 {
			t.Errorf("expected 0 hits, got %d", len(hits))
		}
		if calls != 3 {
			t.Errorf("expected 3 calls, got %d", calls)
		}
		if connects != 3 {
			t.Errorf("expected a fresh connection per attempt, got %d", connects)
		}
	})

	t.Run("a connect that fails is not tried again", func(t *testing.T) {
		var attempts int
		port := search.NewQmdMcpPort(
			search.WithColdAttempts(3),
			search.WithConnect(func(env map[string]string) (search.Session, error) {
				attempts++
				return nil, errors.New("daemon connection refused")
			}),
		)

		_, err := port.Search("query", []string{"c"}, search.ProfileFast, 1)
		if err == nil {
			t.Fatal("expected Search to fail on the first connect, got nil")
		}
		if err.Error() != "the search engine did not answer in 1 attempts: daemon connection refused" {
			t.Errorf("unexpected message: %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 connect attempt, got %d", attempts)
		}
	})

	t.Run("a connect that fails after a failed call ends the attempts", func(t *testing.T) {
		var connects int
		port := search.NewQmdMcpPort(
			search.WithColdAttempts(3),
			search.WithConnect(func(env map[string]string) (search.Session, error) {
				connects++
				if connects > 1 {
					return nil, errors.New("daemon connection refused")
				}
				return &mockSession{
					callFunc: func(name string, args map[string]any) (map[string]any, error) {
						return nil, errors.New("the daemon hung up")
					},
				}, nil
			}),
		)

		_, err := port.Search("query", []string{"c"}, search.ProfileFast, 1)
		if err == nil {
			t.Fatal("expected Search to fail, got nil")
		}
		if err.Error() != "the search engine did not answer in 2 attempts: the daemon hung up; daemon connection refused" {
			t.Errorf("unexpected message: %v", err)
		}
		if connects != 2 {
			t.Errorf("expected 2 connects, got %d", connects)
		}
	})
}

func TestQmdMcpPort_MaintenanceDelegation(t *testing.T) {
	fake := search.NewFakePort()
	fake.Listings["col1"] = search.ScriptedIndexed{Paths: []string{"a.md", "b.md"}}
	fake.Pending = []search.ScriptedPending{{Count: 5}}

	port := search.NewQmdMcpPort(
		search.WithCLI(fake),
		search.WithBackbone(search.BackboneCPU),
		search.WithPort(9000),
	)

	files, err := port.Indexed("col1")
	if err != nil || !reflect.DeepEqual(files, []string{"a.md", "b.md"}) {
		t.Errorf("unexpected Indexed result: %v, %v", files, err)
	}

	err = port.Refresh([]string{"col1"})
	if err != nil {
		t.Errorf("unexpected Refresh error: %v", err)
	}
	if len(fake.Refreshed) != 1 || !reflect.DeepEqual(fake.Refreshed[0], []string{"col1"}) {
		t.Errorf("unexpected fake.Refreshed: %v", fake.Refreshed)
	}

	count, err := port.NotYetSearchable()
	if err != nil || count != 5 {
		t.Errorf("unexpected NotYetSearchable: %d, %v", count, err)
	}

	err = port.Embed([]string{"col1"})
	if err != nil {
		t.Errorf("unexpected Embed error: %v", err)
	}
	if len(fake.Embedded) != 1 || !reflect.DeepEqual(fake.Embedded[0], []string{"col1"}) {
		t.Errorf("unexpected fake.Embedded: %v", fake.Embedded)
	}

	if err := port.Close(); err != nil {
		t.Errorf("unexpected Close error: %v", err)
	}
}

func TestQmdMcpPort_CallFailure(t *testing.T) {
	var calls int
	port := search.NewQmdMcpPort(
		search.WithColdAttempts(2),
		search.WithConnect(func(env map[string]string) (search.Session, error) {
			return &mockSession{
				callFunc: func(name string, args map[string]any) (map[string]any, error) {
					calls++
					return nil, errors.New("call failure")
				},
			}, nil
		}),
	)

	_, err := port.Search("q", []string{"c"}, search.ProfileFast, 1)
	if err == nil {
		t.Fatal("expected error on call failure, got nil")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestQmdMcpPort_TranslateReply_EdgeCases(t *testing.T) {
	// Malformed structuredContent and line handling
	session := &mockSession{
		callFunc: func(name string, args map[string]any) (map[string]any, error) {
			return map[string]any{
				"structuredContent": map[string]any{
					"results": []any{
						"not-a-map",
						map[string]any{
							"file":  "col/doc.md",
							"line":  float64(0), // <= 0 -> 1
							"score": 0.5,
						},
						map[string]any{
							"file": "col/doc2.md",
							"line": 15,
						},
					},
				},
			}, nil
		},
	}

	port := search.NewQmdMcpPort(
		search.WithConnect(func(env map[string]string) (search.Session, error) {
			return session, nil
		}),
	)

	hits, err := port.Search("q", []string{"col"}, search.ProfileFast, 2)
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if hits[0].Line != 1 {
		t.Errorf("expected hit 0 line 1, got %d", hits[0].Line)
	}
	if hits[1].Line != 15 {
		t.Errorf("expected hit 1 line 15, got %d", hits[1].Line)
	}

	// Empty / invalid structuredContent returns nil
	nilSession := &mockSession{
		callFunc: func(name string, args map[string]any) (map[string]any, error) {
			return map[string]any{"structuredContent": "not-a-map"}, nil
		},
	}
	nilPort := search.NewQmdMcpPort(
		search.WithConnect(func(env map[string]string) (search.Session, error) {
			return nilSession, nil
		}),
	)
	hits, err = nilPort.Search("q", nil, search.ProfileFast, 1)
	if err != nil || len(hits) != 0 {
		t.Errorf("expected empty hits for invalid structuredContent, got %v, %v", hits, err)
	}

	// results not a slice
	nilResSession := &mockSession{
		callFunc: func(name string, args map[string]any) (map[string]any, error) {
			return map[string]any{"structuredContent": map[string]any{"results": "not-slice"}}, nil
		},
	}
	nilResPort := search.NewQmdMcpPort(
		search.WithConnect(func(env map[string]string) (search.Session, error) {
			return nilResSession, nil
		}),
	)
	hits, err = nilResPort.Search("q", nil, search.ProfileFast, 1)
	if err != nil || len(hits) != 0 {
		t.Errorf("expected empty hits for invalid results, got %v, %v", hits, err)
	}
}

func TestFakePort(t *testing.T) {
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{
		{Hits: []search.SearchHit{{Title: "T1"}}},
		{Err: errors.New("search error")},
	}
	fake.Refreshes = []error{errors.New("refresh error")}
	fake.Listings["c1"] = search.ScriptedIndexed{Paths: []string{"x.md"}}
	fake.Listings["c_err"] = search.ScriptedIndexed{Err: errors.New("indexed error")}
	fake.Pending = []search.ScriptedPending{
		{Count: 10},
		{Err: errors.New("pending error")},
	}

	// First search succeeds
	hits, err := fake.Search("q", []string{"c1"}, search.ProfileFast, 1)
	if err != nil || len(hits) != 1 || hits[0].Title != "T1" {
		t.Errorf("unexpected search 1: %v, %v", hits, err)
	}

	// Second search returns scripted error
	_, err = fake.Search("q", []string{"c1"}, search.ProfileFast, 1)
	if err == nil || err.Error() != "search error" {
		t.Errorf("expected 'search error', got: %v", err)
	}

	// Third search runs out of scripted results
	_, err = fake.Search("q", []string{"c1"}, search.ProfileFast, 1)
	if err == nil || !strings.Contains(err.Error(), "ran out of scripted results") {
		t.Errorf("expected ran out of results error, got: %v", err)
	}

	// Refresh returns scripted error
	err = fake.Refresh([]string{"c1"})
	if err == nil || err.Error() != "refresh error" {
		t.Errorf("expected 'refresh error', got: %v", err)
	}

	// Unscripted refresh succeeds
	err = fake.Refresh([]string{"c2"})
	if err != nil {
		t.Errorf("unexpected unscripted refresh error: %v", err)
	}

	// Indexed succeeds
	files, err := fake.Indexed("c1")
	if err != nil || len(files) != 1 || files[0] != "x.md" {
		t.Errorf("unexpected indexed: %v, %v", files, err)
	}

	// Indexed returns scripted error
	_, err = fake.Indexed("c_err")
	if err == nil || err.Error() != "indexed error" {
		t.Errorf("expected 'indexed error', got: %v", err)
	}

	// Indexed unscripted collection returns error
	_, err = fake.Indexed("unscripted_col")
	if err == nil || !strings.Contains(err.Error(), "has no listing scripted") {
		t.Errorf("expected no listing scripted error, got: %v", err)
	}

	// NotYetSearchable succeeds
	pending, err := fake.NotYetSearchable()
	if err != nil || pending != 10 {
		t.Errorf("unexpected pending: %d, %v", pending, err)
	}

	// NotYetSearchable returns scripted error
	_, err = fake.NotYetSearchable()
	if err == nil || err.Error() != "pending error" {
		t.Errorf("expected 'pending error', got: %v", err)
	}

	// NotYetSearchable unscripted returns 0
	pending, err = fake.NotYetSearchable()
	if err != nil || pending != 0 {
		t.Errorf("unexpected unscripted pending: %d, %v", pending, err)
	}

	// Embed records call
	err = fake.Embed([]string{"c1"})
	if err != nil || len(fake.Embedded) != 1 {
		t.Errorf("unexpected embed: %v", err)
	}
}

func TestDefaultConnectWith_AlreadyReachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]any{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	_, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	var heard []string
	connFn := search.DefaultConnectWith(port, nil, nil, time.Second, func(m string) { heard = append(heard, m) })
	sess, err := connFn(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if len(heard) != 0 {
		t.Errorf("a daemon that answers must not be announced as starting: %v", heard)
	}
}

func TestDefaultConnectWith_DaemonStartFails(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return nil, errors.New("cannot launch qmd")
	}
	var heard []string
	connFn := search.DefaultConnectWith(64999, mockLauncher, nil, time.Millisecond, func(m string) { heard = append(heard, m) })
	_, err := connFn(nil)
	if err == nil || !strings.Contains(err.Error(), "cannot launch qmd") {
		t.Errorf("expected launch error, got: %v", err)
	}
	if len(heard) != 0 {
		t.Errorf("a start that failed must not be announced: %v", heard)
	}
}

func TestDefaultConnectWith_WaitTimeout(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return []string{"qmd"}, nil
	}
	mockSpawner := func(argv []string, env []string) error {
		return nil
	}
	connFn := search.DefaultConnectWith(64998, mockLauncher, mockSpawner, 5*time.Millisecond, nil)
	_, err := connFn(nil)
	if err == nil || !strings.Contains(err.Error(), "no qmd daemon answered") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestDefaultConnectWith_AnnouncesAStartOncePerPort(t *testing.T) {
	var heard []string
	spawned := 0
	connect := search.DefaultConnectWith(64997,
		func(string) ([]string, error) { return []string{"qmd"}, nil },
		func([]string, []string) error { spawned++; return nil },
		5*time.Millisecond,
		func(message string) { heard = append(heard, message) })
	// The notice belongs to the returned ConnectFunc, so it takes two calls of that one
	// function to see the guard hold; a failed connect no longer repeats inside a search.
	for i := 0; i < 2; i++ {
		if _, err := connect(nil); err == nil ||
			err.Error() != "no qmd daemon answered on http://localhost:64997/mcp within 5ms" {
			t.Fatalf("got %v", err)
		}
	}
	if spawned != 2 {
		t.Errorf("expected two starts, got %d", spawned)
	}
	if len(heard) != 1 || heard[0] != search.WarmingNotice {
		t.Errorf("expected the warming notice exactly once, got %q", heard)
	}
}

func TestWarmingNoticeIsTheDaemonsWording(t *testing.T) {
	const want = "starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm."
	if search.WarmingNotice != want {
		t.Fatalf("got %q", search.WarmingNotice)
	}
}

func TestNewQmdMcpPort_DefaultConnectRefusesAMissingQmdWithoutANotice(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var heard []string
	port := search.NewQmdMcpPort(search.WithPort(64996), search.WithColdAttempts(1),
		search.WithNotice(func(m string) { heard = append(heard, m) }))
	_, err := port.Search("q", []string{"c"}, search.ProfileFast, 1)
	if err == nil || err.Error() != "the search engine did not answer in 1 attempts: cannot find 'qmd' on PATH" {
		t.Fatalf("got %v", err)
	}
	if len(heard) != 0 {
		t.Errorf("no daemon was started, yet the notice came: %q", heard)
	}
}

func TestNewQmdMcpPort_Defaults(t *testing.T) {
	port := search.NewQmdMcpPort()
	if port == nil {
		t.Fatal("expected non-nil port")
	}
	_ = search.DefaultConnect(8765, nil)
}

// qmd's daemon numbers every snippet line of a query answer (dist/mcp/server.js:301,
// addLineNumbers(snippet, line)); `qmd query --json`, which the Python reference reads, does
// not. The port takes the numbers off again, and only numbers that are exactly the daemon's.
func TestQmdMcpPort_TakesTheDaemonsLineNumbersOffTheSnippet(t *testing.T) {
	for _, tc := range []struct {
		name, fields, want string
	}{
		{"numbered from the hit's line", `"line":12,"snippet":"12: @@ -11,4 @@ (10 before, 5 after)\n13: a\n14: b\r\n15: c"`, "@@ -11,4 @@ (10 before, 5 after)\na\nb\r\nc"},
		{"an empty snippet", `"line":7,"snippet":"7: "`, ""},
		{"numbers that are not the hit's line", `"line":12,"snippet":"5: a\n6: b"`, "5: a\n6: b"},
		{"no line in the reply", `"snippet":"1: a"`, "1: a"},
		{"one part without its number", `"line":12,"snippet":"12: a\nb"`, "12: a\nb"},
		{"a line below one counts from itself", `"line":0,"snippet":"0: a\n1: b"`, "a\nb"},
	} {
		for _, profile := range []search.Profile{search.ProfileKeyword, search.ProfileFast, search.ProfileFull} {
			t.Run(tc.name+"/"+string(profile), func(t *testing.T) {
				reply := `{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"results":[{"docid":"#abc123","file":"c/a.md","title":"A","score":0.5,` + tc.fields + `}]}}}`
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(reply))
				}))
				defer ts.Close()
				port := search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
					return &search.HTTPSession{URL: ts.URL}, nil
				}))
				hits, err := port.Search("q", []string{"c"}, profile, 1)
				if err != nil {
					t.Fatal(err)
				}
				if len(hits) != 1 {
					t.Fatalf("got %d hits", len(hits))
				}
				if hits[0].Snippet != tc.want {
					t.Fatalf("snippet %q, want %q", hits[0].Snippet, tc.want)
				}
			})
		}
	}
}
