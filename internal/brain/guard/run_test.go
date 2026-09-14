package guard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRefuseWritesBothChannelsAndBlocks(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Refuse(&out, &errb, "no"); code != 2 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(out.String(), `"permissionDecision": "deny"`) || errb.String() != "no\n" {
		t.Fatalf("out %q err %q", out.String(), errb.String())
	}
}

func TestRunStaysSilentWhenItAllows(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(
		filepath.Join(tmp, "vault", "demo", "x.md")))
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	errs := &bytes.Buffer{}
	code := Run(bytes.NewReader(body), out, errs, state)
	if code != 0 || out.Len() != 0 || errs.Len() != 0 {
		t.Fatalf("code %d, stdout %q, stderr %q", code, out, errs)
	}
}

// asciiEnvelope is the byte shape the barrier writes: `json.dumps` with
// its default separators, `ensure_ascii=True` and no trailing newline.
// `strconv.Quote` stands in for the string rendering only because every
// reason built here is plain ASCII; the escaping itself is measured
// against `json.dumps` by hand, in
// TestPythonJSONStringRendersWhatJSONDumpsRenders.
func asciiEnvelope(reason string) string {
	quoted := strconv.Quote(reason)
	return `{"decision": "deny", "reason": ` + quoted +
		`, "hookSpecificOutput": {"hookEventName": "PreToolUse", ` +
		`"permissionDecision": "deny", "permissionDecisionReason": ` +
		quoted + `}}`
}

func TestRunPrintsTheDenyEnvelope(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	errs := &bytes.Buffer{}
	code := Run(bytes.NewReader(body), out, errs, state)
	if code != blockingExit {
		t.Fatalf("a refusal must block, got %d", code)
	}
	// The reason goes to stderr as well as into the envelope. The host
	// takes it from `permissionDecisionReason` when the JSON parses and
	// from stderr when it does not, and only one of those two can be
	// checked from inside this process.
	if !strings.Contains(errs.String(), "lies outside every writable tree") {
		t.Fatalf("stderr carried no reason: %q", errs)
	}
	envelope := map[string]any{}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v (%q)", err, out)
	}
	reason, _ := envelope["reason"].(string)
	if !strings.Contains(reason, "lies outside every writable tree") {
		t.Fatalf("no reason in %q", out)
	}
	if out.String() != asciiEnvelope(reason) {
		t.Fatalf("envelope\n got %s\nwant %s", out, asciiEnvelope(reason))
	}
	if strings.HasSuffix(out.String(), "\n") {
		t.Fatal("the envelope must carry no trailing newline")
	}
}

func TestTheEnvelopeCarriesTheHostSpecificHalf(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	Run(bytes.NewReader(body), out, &bytes.Buffer{}, state)
	envelope := map[string]any{}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["decision"] != "deny" {
		t.Fatalf("decision %v", envelope["decision"])
	}
	specific, ok := envelope["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("no hookSpecificOutput in %q", out)
	}
	for key, want := range map[string]string{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": envelope["reason"].(string),
	} {
		if specific[key] != want {
			t.Fatalf("%s = %v, want %v", key, specific[key], want)
		}
	}
}

func TestRunRefusesBrokenJSON(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	out := &bytes.Buffer{}
	code := Run(strings.NewReader("{not json"), out, &bytes.Buffer{}, state)
	if code != blockingExit {
		t.Fatalf("got %d", code)
	}
	if !strings.Contains(out.String(),
		"loomux cannot read the hook payload, so it refuses") {
		t.Fatalf("stdout %q", out)
	}
}

func TestRunRefusesAPayloadThatIsNotAnObject(t *testing.T) {
	// The names are `type(payload).__name__` as Python spells them, which
	// is why an integer and a float are told apart although JSON has one
	// number type.
	for body, name := range map[string]string{
		"[]":     "list",
		"\"x\"":  "str",
		"3":      "int",
		"3.5":    "float",
		"1e3":    "float",
		"true":   "bool",
		"false":  "bool",
		"null":   "NoneType",
		"-17":    "int",
		"[1, 2]": "list",
	} {
		out := &bytes.Buffer{}
		code := Run(strings.NewReader(body), out, &bytes.Buffer{}, "")
		if code != blockingExit {
			t.Fatalf("%s: got %d", body, code)
		}
		want := "expected an object, found " + name
		if !strings.Contains(out.String(), want) {
			t.Fatalf("%s: stdout %q wants %q", body, out, want)
		}
	}
}

func TestRunRefusesTrailingData(t *testing.T) {
	// `json.load` reads the whole stream and calls trailing bytes an
	// error; a decoder that stopped at the first value would judge half a
	// payload and say nothing about the rest.
	out := &bytes.Buffer{}
	code := Run(strings.NewReader("{} {}"), out, &bytes.Buffer{}, "")
	if code != blockingExit {
		t.Fatalf("got %d", code)
	}
	if !strings.Contains(out.String(), "cannot read the hook payload") {
		t.Fatalf("stdout %q", out)
	}
}

func TestRunAcceptsWhitespaceAroundThePayload(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(
		filepath.Join(tmp, "vault", "demo", "x.md")))
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	code := Run(strings.NewReader("\n "+string(body)+" \n"), out,
		&bytes.Buffer{}, state)
	if code != 0 || out.Len() != 0 {
		t.Fatalf("code %d, stdout %q", code, out)
	}
}

// failing is a channel that is already gone -- the case the second
// recover in `Run` exists for.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, errors.New("gone") }

// short is a channel that accepts less than it was given. Python's
// `stdout.write` raises on a short write; an unchecked Go `Write` would
// report success and deliver half an object, and half a deny is no deny.
type short struct{}

func (short) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestRunFallsBackToTheBlockingExitCode(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}
	errs := &bytes.Buffer{}
	code := Run(bytes.NewReader(body), failing{}, errs, state)
	if code != 2 {
		t.Fatalf("a dead pipe must block, got %d", code)
	}
	if !strings.Contains(errs.String(), "lies outside every writable tree") {
		t.Fatalf("stderr %q", errs)
	}
}

func TestAShortWriteBlocksToo(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}
	if code := Run(bytes.NewReader(body), short{}, &bytes.Buffer{},
		state); code != 2 {
		t.Fatalf("got %d", code)
	}
}

func TestAStderrThatIsGoneTooStillBlocks(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	body, err := json.Marshal(writeCall(filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}
	if code := Run(bytes.NewReader(body), failing{}, failing{},
		state); code != 2 {
		t.Fatalf("got %d", code)
	}
}

// brokenReader is a stdin that fails mid-read, which `json.load` raises
// through and `main`'s outer catch turns into a refusal.
type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) {
	return 0, errors.New("the pipe is gone")
}

func TestRunRefusesAStdinThatCannotBeRead(t *testing.T) {
	out := &bytes.Buffer{}
	code := Run(brokenReader{}, out, &bytes.Buffer{}, "")
	if code != blockingExit {
		t.Fatalf("got %d", code)
	}
	if !strings.Contains(out.String(), "loomux broke down") {
		t.Fatalf("stdout %q", out)
	}
}

// TestNoArmEverLeavesWithOne is the one contract the whole module rests
// on: the host reads exit 1 as a *non-blocking* error and lets the write
// through, so a guard that fails with 1 is no guard at all. Kept beside
// the sharper test above, which asks which of 0 and 2 each arm gives; this
// one holds the floor even for arms nobody has thought to name yet.
func TestNoArmEverLeavesWithOne(t *testing.T) {
	tmp := t.TempDir()
	good := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	none := filepath.Join(tmp, "nostate")
	broken := filepath.Join(tmp, "brokenstate")
	write(t, filepath.Join(broken, "registry.toml"), "[[area]\n")
	allowed, err := json.Marshal(writeCall(
		filepath.Join(tmp, "vault", "demo", "x.md")))
	if err != nil {
		t.Fatal(err)
	}
	refused, err := json.Marshal(writeCall(
		filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(writeCall(
		filepath.Join(tmp, "a", ".loomux", "config.toml")))
	if err != nil {
		t.Fatal(err)
	}
	arms := []struct {
		name   string
		stdin  io.Reader
		state  string
		stdout io.Writer
		stderr io.Writer
	}{
		{"allowed", bytes.NewReader(allowed), good, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"refused", bytes.NewReader(refused), good, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"manifest", bytes.NewReader(manifest), good, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"no registry", bytes.NewReader(refused), none, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"broken registry", bytes.NewReader(refused), broken,
			&bytes.Buffer{}, &bytes.Buffer{}},
		{"broken json", strings.NewReader("{oops"), good,
			&bytes.Buffer{}, &bytes.Buffer{}},
		{"not an object", strings.NewReader("[]"), good, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"trailing data", strings.NewReader("{} 1"), good,
			&bytes.Buffer{}, &bytes.Buffer{}},
		{"empty stdin", strings.NewReader(""), good, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"unreadable stdin", brokenReader{}, good, &bytes.Buffer{},
			&bytes.Buffer{}},
		{"dead stdout, refusal", bytes.NewReader(refused), good,
			failing{}, &bytes.Buffer{}},
		{"dead stdout, broken payload", strings.NewReader("{oops"), good,
			failing{}, &bytes.Buffer{}},
		{"both channels dead", bytes.NewReader(refused), good, failing{},
			failing{}},
		{"short stdout", bytes.NewReader(refused), good, short{},
			&bytes.Buffer{}},
	}
	for _, arm := range arms {
		code := Run(arm.stdin, arm.stdout, arm.stderr, arm.state)
		if code != 0 && code != 2 {
			t.Fatalf("%s left with %d", arm.name, code)
		}
	}
}

func TestABrokenPayloadIsNamedAsBrokenAndNotAsTheWrongShape(t *testing.T) {
	// Two mutants lived here, and both left the refusal standing while
	// swapping its reason: one dropped the decode error and let a nil
	// payload be reported as "expected an object", the other dropped it
	// inside `decodeOnly` and reported "Extra data". Both are true of
	// nothing -- `{oops` is neither a value of the wrong shape nor a
	// value with something after it -- and a reason that names the wrong
	// defect sends the reader looking in the wrong place.
	//
	// The words asserted are this decoder's own, the way the missing-file
	// message elsewhere in this binary is. Python says "Expecting property
	// name enclosed in double quotes"; the corpus that held the two
	// barriers against each other carried that difference as a named
	// excuse, and went with them in 6bceba4. Nothing compares the wordings
	// now, so this test is the only thing keeping this one honest.
	out := &bytes.Buffer{}
	Run(strings.NewReader("{oops"), out, &bytes.Buffer{}, "")
	said := out.String()
	if !strings.Contains(said, "invalid character") {
		t.Fatalf("the decoder's own complaint is missing: %q", said)
	}
	for _, wrong := range []string{"expected an object", "Extra data"} {
		if strings.Contains(said, wrong) {
			t.Fatalf("a broken payload was called %q: %s", wrong, said)
		}
	}
}

// TestEveryRefusalBlocksAndEveryAllowancePasses is the contract this
// module was rebuilt around on 2026-09-06. A refusal used to leave with 0
// and rely on the host reading the envelope off stdout. Measured on this
// host, it does not: a probe file landed under `10 Rohquellen` while the
// refusal sat unread. The documented behaviour is that exit 2 blocks
// regardless of what stdout says, so the exit code carries the decision
// and the envelope explains it.
//
// The arms are split by outcome rather than merged into one "never 1"
// sweep, because "not 1" was already true of the version that let writes
// through. Only "a refusal is 2" catches that.
func TestEveryRefusalBlocksAndEveryAllowancePasses(t *testing.T) {
	tmp := t.TempDir()
	good := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	none := filepath.Join(tmp, "nostate")
	broken := filepath.Join(tmp, "brokenstate")
	write(t, filepath.Join(broken, "registry.toml"), "[[area]\n")
	allowed, err := json.Marshal(writeCall(
		filepath.Join(tmp, "vault", "demo", "x.md")))
	if err != nil {
		t.Fatal(err)
	}
	refused, err := json.Marshal(writeCall(
		filepath.Join(tmp, "repo", "a.py")))
	if err != nil {
		t.Fatal(err)
	}

	blocking := []struct {
		name  string
		stdin io.Reader
		state string
	}{
		{"outside every tree", bytes.NewReader(refused), good},
		{"no registry", bytes.NewReader(refused), none},
		{"broken registry", bytes.NewReader(refused), broken},
		{"broken json", strings.NewReader("{oops"), good},
		{"not an object", strings.NewReader("[]"), good},
		{"trailing data", strings.NewReader("{} 1"), good},
		{"empty stdin", strings.NewReader(""), good},
		{"unreadable stdin", brokenReader{}, good},
	}
	for _, arm := range blocking {
		out := &bytes.Buffer{}
		errs := &bytes.Buffer{}
		if code := Run(arm.stdin, out, errs, arm.state); code != blockingExit {
			t.Errorf("%s: left with %d, want %d", arm.name, code, blockingExit)
		}
		// Whichever channel the host reads, it must find a reason there.
		if out.Len() == 0 && errs.Len() == 0 {
			t.Errorf("%s: blocked without saying why", arm.name)
		}
	}

	// And the other direction, which is the one a too-eager guard breaks:
	// an allowed write leaves with 0 and says nothing at all.
	out := &bytes.Buffer{}
	errs := &bytes.Buffer{}
	if code := Run(bytes.NewReader(allowed), out, errs, good); code != 0 {
		t.Errorf("an allowed write left with %d, want 0", code)
	}
	if out.Len() != 0 || errs.Len() != 0 {
		t.Errorf("an allowed write spoke: stdout %q, stderr %q", out, errs)
	}
}
