package convert

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

const json3 = `{"events": [
 {"tStartMs": 0, "segs": [{"utf8": "Hallo zusammen, das hier"}]},
 {"tStartMs": 3000, "segs": [{"utf8": "\n"}]},
 {"tStartMs": 4200, "segs": [{"utf8": " ist mein Second Brain."}]}
]}`

// ytdlp stands for yt-dlp: it leaves files in its working directory and
// ends as told, and records the call and the directory it found.
type ytdlp struct {
	files    map[string]string
	exit     int
	stderr   string
	err      error // yt-dlp did not start
	timedOut bool
	calls    []child.Spec
	fresh    []bool // the directory was empty and right under os.TempDir when yt-dlp started
}

func (y *ytdlp) tools() Tools {
	return Tools{
		Look: func(name string) (string, error) { return `C:\bin\` + name + ".exe", nil },
		Run: func(spec child.Spec) child.Result {
			y.calls = append(y.calls, spec)
			entries, err := os.ReadDir(spec.Dir)
			y.fresh = append(y.fresh, err == nil && len(entries) == 0 && sameDir(filepath.Dir(spec.Dir), os.TempDir()))
			for name, text := range y.files {
				os.WriteFile(filepath.Join(spec.Dir, name), []byte(text), 0o644)
			}
			return child.Result{Code: y.exit, Stderr: y.stderr, Err: y.err, TimedOut: y.timedOut}
		},
	}
}

// sameDir compares by spelling first and asks the file system where the
// spelling differs: a short 8.3 name and its long form are one directory.
func sameDir(a, b string) bool {
	if a == b {
		return true
	}
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ai, bi)
}

func info(title string) string {
	return `{"id": "mHSOsy_usAg", "title": "` + title + `", "subtitles": {}, "automatic_captions": {"de": [{"ext": "json3"}]}}`
}

func TestTheFragmentsComeBackWithTheirMarks(t *testing.T) {
	y := &ytdlp{files: map[string]string{"v.info.json": info("Der echte Weg"), "v.de.json3": json3}}
	subs, err := Fetch(y.tools(), "https://www.youtube.com/watch?v=mHSOsy_usAg")
	if err != nil || subs.Title != "Der echte Weg" || !slices.Equal(subs.Fragments, []Fragment{{"00:00:00", "Hallo zusammen, das hier"}, {"00:00:04", "ist mein Second Brain."}}) {
		t.Fatalf("%+v %v", subs, err)
	}
}

func TestYtdlpIsCalledOnceInAFreshDirectory(t *testing.T) {
	y := &ytdlp{files: map[string]string{"v.info.json": info("x"), "v.de.json3": json3}}
	Fetch(y.tools(), "https://www.youtube.com/watch?v=mHSOsy_usAg&list=PL1")
	if len(y.calls) != 1 {
		t.Fatalf("%d calls", len(y.calls))
	}
	if !y.fresh[0] {
		t.Fatal("yt-dlp did not start in an empty directory right under os.TempDir")
	}
	call := y.calls[0]
	want := []string{`C:\bin\yt-dlp.exe`, "--ignore-config", "--no-playlist", "--no-progress", "--skip-download",
		"--write-subs", "--write-auto-subs", "--sub-langs", "de,en", "--sub-format", "json3", "--write-info-json",
		"--ignore-errors", "-o", "v", "https://www.youtube.com/watch?v=mHSOsy_usAg&list=PL1"}
	if !slices.Equal(call.Argv, want) || call.Timeout != 10*time.Minute {
		t.Fatalf("%q", call.Argv)
	}
	if _, err := os.Stat(call.Dir); err == nil {
		t.Fatal("the working directory outlived the fetch")
	}
}

func TestAMarkPastAHundredMinutesKeepsTheHour(t *testing.T) {
	late := `{"events": [{"tStartMs": 6300000, "segs": [{"utf8": "Spät im Video."}]}]}`
	y := &ytdlp{files: map[string]string{"v.info.json": info("x"), "v.de.json3": late}}
	subs, _ := Fetch(y.tools(), "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if subs.Fragments[0].Mark != "01:45:00" {
		t.Fatal(subs.Fragments)
	}
}

// Review Focus 4: the English track failed, the German one landed. Under
// --ignore-errors a failed track is a warning and yt-dlp ends with 0
// (measured); an error it still reports after writing can end it with 1,
// and what it wrote decides.
func TestAFailedSecondTrackDoesNotLoseTheFirst(t *testing.T) {
	y := &ytdlp{exit: 1, files: map[string]string{"v.info.json": info("x"), "v.de.json3": json3}}
	if subs, err := Fetch(y.tools(), "https://youtu.be/mHSOsy_usAg"); err != nil || len(subs.Fragments) != 2 {
		t.Fatalf("%+v %v", subs, err)
	}
}

func TestTheTrackIsChosenAsTheReferenceChoosesIt(t *testing.T) {
	json3s := func(langs ...string) map[string][]track {
		out := map[string][]track{}
		for _, lang := range langs {
			out[lang] = []track{{Ext: "json3"}}
		}
		return out
	}
	for _, c := range []struct {
		name   string
		meta   infoJSON
		lang   string
		chosen bool
	}{
		{"manual de", infoJSON{Subtitles: json3s("de", "en"), Automatic: json3s("de")}, "de", true},
		{"manual en before automatic de", infoJSON{Subtitles: json3s("en"), Automatic: json3s("de")}, "en", true},
		{"automatic de", infoJSON{Automatic: json3s("en", "de")}, "de", true},
		{"automatic en", infoJSON{Automatic: json3s("en")}, "en", true},
		// A language that has tracks and none in json3 ends its kind's search.
		{"manual de without json3", infoJSON{Subtitles: map[string][]track{"de": {{Ext: "vtt"}}, "en": {{Ext: "json3"}}}, Automatic: json3s("en")}, "en", true},
		{"nothing", infoJSON{}, "", false},
	} {
		if lang, ok := chooseTrack(c.meta); lang != c.lang || ok != c.chosen {
			t.Errorf("%s: %q %v", c.name, lang, ok)
		}
	}
}

// "No subtitle track" is kept for a video that lists none in json3, and for
// a json3 track of whitespace alone -- the two cases the reference answers
// so.
func TestWithoutASubtitleTrackThereIsNothingToFetch(t *testing.T) {
	for _, files := range []map[string]string{
		{"v.info.json": `{"title": "x", "subtitles": {}, "automatic_captions": {}}`},
		{"v.info.json": `{"title": "x", "subtitles": {}, "automatic_captions": {"de": [{"ext": "vtt"}]}}`, "v.de.vtt": "WEBVTT"},
		{"v.info.json": info("x"), "v.de.json3": "  \n"},
	} {
		_, err := Fetch((&ytdlp{files: files}).tools(), "https://www.youtube.com/watch?v=aaaaaaaaaaa")
		if err == nil || !strings.Contains(err.Error(), "no subtitle track to fetch, and this system does no ASR") {
			t.Errorf("%v: %v", files, err)
		}
	}
}

// Under --ignore-errors a chosen track whose download failed is a warning:
// no file, exit 0. That is no missing track but yt-dlp's failure, and the
// line about the chosen language says why -- yt-dlp asks for de before en,
// so the last line can be the other language's.
func TestAChosenTrackYtdlpDidNotWriteNamesWhy(t *testing.T) {
	de := "WARNING: Unable to download video subtitles for 'de': HTTP Error 429: Too Many Requests"
	en := "WARNING: Unable to download video subtitles for 'en': HTTP Error 429: Too Many Requests"
	for _, c := range []struct {
		stderr string
		exit   int
		want   string
	}{
		{"WARNING: No supported JavaScript runtime\r\n" + de + "\r\n" + en + "\r\n", 0, "https://x: yt-dlp wrote no v.de.json3 (exit 0): " + de},
		{"ERROR: something\nWARNING: the last line\n", 1, "https://x: yt-dlp wrote no v.de.json3 (exit 1): WARNING: the last line"},
		{"", 0, "https://x: yt-dlp wrote no v.de.json3 (exit 0)"},
		// The line about the language can be the first one yt-dlp wrote.
		{de + "\nWARNING: the last line\n", 0, "https://x: yt-dlp wrote no v.de.json3 (exit 0): " + de},
	} {
		y := &ytdlp{exit: c.exit, stderr: c.stderr, files: map[string]string{"v.info.json": info("x")}}
		if _, err := Fetch(y.tools(), "https://x"); err == nil || err.Error() != c.want {
			t.Errorf("%q: %v", c.stderr, err)
		}
	}
}

// A yt-dlp that did not start, or ran out of time, says so where it left no
// file to read; the exit code of such a run is -1 and its stderr says
// nothing.
func TestAYtdlpThatDidNotRunToItsEndSaysSo(t *testing.T) {
	for _, c := range []struct {
		y    *ytdlp
		want string
	}{
		{&ytdlp{exit: -1, err: errors.New("first\nsecond")}, "https://x: yt-dlp did not start: first: second"},
		{&ytdlp{exit: -1, timedOut: true}, "https://x: yt-dlp took longer than 10m0s"},
		{&ytdlp{exit: -1, timedOut: true, files: map[string]string{"v.info.json": info("x")}}, "https://x: yt-dlp took longer than 10m0s"},
	} {
		if _, err := Fetch(c.y.tools(), "https://x"); err == nil || err.Error() != c.want {
			t.Errorf("%+v: %v", c.y, err)
		}
	}
}

// What yt-dlp wrote decides, not how it ended: a run that ran out of time
// after the chosen track landed still gives the track.
func TestATimedOutRunWhoseTrackLandedStillSucceeds(t *testing.T) {
	y := &ytdlp{exit: -1, timedOut: true, files: map[string]string{"v.info.json": info("x"), "v.de.json3": json3}}
	if subs, err := Fetch(y.tools(), "https://x"); err != nil || len(subs.Fragments) != 2 {
		t.Fatalf("%+v %v", subs, err)
	}
}

func TestWhatYtdlpLeavesUnreadableIsAnError(t *testing.T) {
	for _, files := range []map[string]string{
		{},
		{"v.info.json": "not json"},
		{"v.info.json": info("x"), "v.de.json3": "[1, 2]"},
	} {
		if _, err := Fetch((&ytdlp{exit: 1, files: files}).tools(), "https://x"); err == nil {
			t.Errorf("%v passed", files)
		}
	}
}

// Without an info JSON the message carries the last line yt-dlp wrote to
// stderr, and no colon where it wrote none. The recorded stderr comes from
// the run that succeeded; it stands in here only for several lines, of
// which the last one is taken.
func TestAFetchWithoutVideoNamesYtdlpsLastLine(t *testing.T) {
	recorded := read(t, filepath.Join("..", "..", "..", "testdata", "convert", "ytdlp", "mHSOsy_usAg", "stderr"))
	for stderr, want := range map[string]string{
		recorded: "https://x: yt-dlp found no video (exit 1): WARNING: Unable to download video subtitles for 'en': HTTP Error 429: Too Many Requests",
		"":       "https://x: yt-dlp found no video (exit 1)",
	} {
		if _, err := Fetch((&ytdlp{exit: 1, stderr: stderr}).tools(), "https://x"); err == nil || err.Error() != want {
			t.Errorf("%q: %v", stderr, err)
		}
	}
}

// A chosen track that is there and does not read is an error of its own,
// not "yt-dlp wrote no v.<lang>.json3 ...": that answer is kept for a
// chosen track yt-dlp never wrote.
func TestAChosenTrackThatDoesNotReadIsAnError(t *testing.T) {
	tools := Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run: func(spec child.Spec) child.Result {
			os.WriteFile(filepath.Join(spec.Dir, "v.info.json"), []byte(info("x")), 0o644)
			os.Mkdir(filepath.Join(spec.Dir, "v.de.json3"), 0o755)
			return child.Result{}
		},
	}
	_, err := Fetch(tools, "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err == nil || !strings.Contains(err.Error(), "v.de.json3 cannot be read") {
		t.Fatal(err)
	}
}

func TestAMissingYtdlpNamesItsInstaller(t *testing.T) {
	tools := Tools{Look: func(string) (string, error) { return "", exec.ErrNotFound }}
	_, err := Fetch(tools, "https://x")
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is not on PATH; install it with: winget install --id yt-dlp.yt-dlp -e") {
		t.Fatal(err)
	}
}

// A temporary directory that cannot be made stops the fetch before yt-dlp
// is started.
func TestAFetchWithoutATemporaryDirectoryStartsNothing(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// os.TempDir reads TMP under Windows and TMPDIR elsewhere.
	t.Setenv("TMP", file)
	t.Setenv("TMPDIR", file)
	tools := Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run:  func(child.Spec) child.Result { t.Fatal("yt-dlp was started"); return child.Result{} },
	}
	if _, err := Fetch(tools, "https://x"); err == nil {
		t.Fatal("the fetch went on without a directory")
	}
}

func TestTheFileLandsInTheInboxInBracketForm(t *testing.T) {
	inbox := filepath.Join(t.TempDir(), "00 Eingang")
	subs := Subtitles{Title: "Der echte Weg", Fragments: []Fragment{{"00:00:00", "Hallo."}, {"00:00:04", "Weiter."}}}
	path, err := ToInbox(subs, "https://www.youtube.com/watch?v=mHSOsy_usAg", inbox)
	if err != nil || filepath.Base(path) != "Der echte Weg (mHSOsy_usAg).txt" || read(t, path) != "[00:00:00] Hallo.\n\n[00:00:04] Weiter.\n" {
		t.Fatalf("%q %v", path, err)
	}
	again, _ := ToInbox(subs, "https://www.youtube.com/watch?v=mHSOsy_usAg", inbox)
	if entries, _ := os.ReadDir(inbox); again != path || len(entries) != 1 {
		t.Fatal("the second fetch did not reuse the name")
	}
}

func TestTheTitleBecomesAUsableName(t *testing.T) {
	inbox := t.TempDir()
	one := []Fragment{{"00:00", "Text"}}
	for _, c := range []struct{ title, url, want string }{
		{"Was: RAG / Wiki?", "https://www.youtube.com/watch?v=mHSOsy_usAg", "Was RAG  Wiki (mHSOsy_usAg).txt"},
		{"???:::", "https://www.youtube.com/watch?v=mHSOsy_usAg", "video (mHSOsy_usAg).txt"},
		{"x", "https://www.youtube.com/shorts/mHSOsy_usAg", "x (mHSOsy_usAg).txt"},
		{"x", "https://www.youtube.com/live/mHSOsy_usAg", "x (mHSOsy_usAg).txt"},
		{"x", "https://vimeo.com/123", "x.txt"},
		// Only what Windows refuses goes: "=" stays, "<" and ">" do not.
		{"a=b <c>", "https://vimeo.com/123", "a=b c.txt"},
		{strings.Repeat("\U000000e4", 300), "https://vimeo.com/1", strings.Repeat("\U000000e4", 150) + ".txt"},
	} {
		path, err := ToInbox(Subtitles{Title: c.title, Fragments: one}, c.url, inbox)
		if err != nil || filepath.Base(path) != c.want {
			t.Errorf("%q: %q %v", c.title, filepath.Base(path), err)
		}
	}
}

// An inbox that cannot be made, and a target that cannot be written, are
// errors, and no path is named.
func TestAnInboxThatTakesNoFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	subs := Subtitles{Title: "x", Fragments: []Fragment{{"00:00:00", "Text"}}}
	if path, err := ToInbox(subs, "https://vimeo.com/1", filepath.Join(file, "x")); err == nil || path != "" {
		t.Errorf("an inbox under a file: %q %v", path, err)
	}
	if err := os.Mkdir(filepath.Join(dir, "x.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if path, err := ToInbox(subs, "https://vimeo.com/1", dir); err == nil || path != "" {
		t.Errorf("a directory where the file goes: %q %v", path, err)
	}
}

// The recording of Task 1, fed through Fetch and ToInbox, writes what the
// reference's fetch and to_inbox wrote for it. The recording was made with
// the command line Fetch runs, --ignore-errors included.
func TestTheRecordingLandsAsTheReferenceWroteIt(t *testing.T) {
	recording := filepath.Join("..", "..", "..", "testdata", "convert", "ytdlp", "mHSOsy_usAg")
	argv := strings.TrimSpace(read(t, filepath.Join(recording, "argv")))
	if want := "yt-dlp " + strings.Join(ytdlpArgs("https://www.youtube.com/watch?v=mHSOsy_usAg"), " "); argv != want {
		t.Fatalf("the recording ran %q, Fetch runs %q", argv, want)
	}
	files := map[string]string{}
	entries, err := os.ReadDir(filepath.Join(recording, "files"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		files[e.Name()] = read(t, filepath.Join(recording, "files", e.Name()))
	}
	subs, err := Fetch((&ytdlp{files: files}).tools(), "https://www.youtube.com/watch?v=mHSOsy_usAg")
	if err != nil {
		t.Fatal(err)
	}
	path, err := ToInbox(subs, "https://www.youtube.com/watch?v=mHSOsy_usAg", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSpace(read(t, filepath.Join(recording, "expected", "name")))
	if filepath.Base(path) != name || read(t, path) != read(t, filepath.Join(recording, "expected", name)) {
		t.Fatalf("%q differs from the reference's %q", filepath.Base(path), name)
	}
}
