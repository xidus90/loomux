package convert

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/programs"
)

// fetchTimeout bounds one yt-dlp call: extraction and two subtitle
// downloads on a slow line.
const fetchTimeout = 10 * time.Minute

// maxStem keeps a title from an arbitrary portal below Windows' 255
// characters, with room for the id and the extension (fetch.py:74-78). It
// counts code points, as the reference does, where Windows counts UTF-16
// units and Linux 255 bytes: 150 characters outside the BMP, or 150 x `ä`
// on Linux, would still be too long. Not measured; like the reference, the
// bound is made for Windows.
const maxStem = 150

var watchID = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?:v=|youtu\.be/|shorts/|live/)([0-9A-Za-z_-]{11})`)
})

// unsafeName is what Windows will not have in a file name, and every
// control character.
var unsafeName = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`) })

// Subtitles is what a fetch brings back: the title names the file, the
// fragments fill it.
type Subtitles struct {
	Title     string
	Fragments []Fragment
}

// Fragment is one piece of speech and the mark it began at, hh:mm:ss.
type Fragment struct{ Mark, Text string }

type track struct {
	Ext string `json:"ext"`
}

type infoJSON struct {
	Title     string             `json:"title"`
	Subtitles map[string][]track `json:"subtitles"`
	Automatic map[string][]track `json:"automatic_captions"`
}

// Fetch has yt-dlp write a video's subtitles into a fresh directory and
// reads the language the reference would have chosen. Within a language
// yt-dlp picks the track, the last json3 one listed, where the reference
// took the first -- a released deviation. loomux itself never speaks to the
// network: what the portal asks of a client, yt-dlp keeps up with.
func Fetch(tools Tools, url string) (Subtitles, error) {
	exe, err := tools.Look("yt-dlp")
	if err != nil {
		return Subtitles{}, fmt.Errorf("yt-dlp is not on PATH; install it with: %s", programs.Install("yt-dlp"))
	}
	dir, err := os.MkdirTemp("", "loomux-fetch-")
	if err != nil {
		return Subtitles{}, err
	}
	defer os.RemoveAll(dir)
	res := tools.Run(child.Spec{Argv: append([]string{exe}, ytdlpArgs(url)...), Dir: dir, Timeout: fetchTimeout})
	// The exit code decides nothing, and neither does a timeout: what yt-dlp
	// wrote does. Under --ignore-errors a failed track (429 on an
	// auto-translated one, measured) is a warning and the info JSON is
	// still written; an error yt-dlp reports after writing can end it with
	// 1, and what it wrote may be the one to take.
	data, err := os.ReadFile(filepath.Join(dir, "v.info.json"))
	if err != nil {
		if err := unfinished(url, res); err != nil {
			return Subtitles{}, err
		}
		return Subtitles{}, fmt.Errorf("%s: yt-dlp found no video %s", url, ended(res.Code, lastLine(res.Stderr)))
	}
	var meta infoJSON
	if err := json.Unmarshal(data, &meta); err != nil {
		return Subtitles{}, fmt.Errorf("%s: yt-dlp wrote an unreadable v.info.json: %v", url, err)
	}
	lang, ok := chooseTrack(meta)
	if !ok {
		return Subtitles{}, noTrack(url)
	}
	// The info JSON lists the chosen language in json3, so a missing file is
	// no missing track but a download that failed, a warning under
	// --ignore-errors; one that is there and does not read is an error of
	// its own.
	name := "v." + lang + ".json3"
	raw, err := os.ReadFile(filepath.Join(dir, name))
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := unfinished(url, res); err != nil {
			return Subtitles{}, err
		}
		return Subtitles{}, fmt.Errorf("%s: yt-dlp wrote no %s %s", url, name, ended(res.Code, lineAbout(res.Stderr, lang)))
	case err != nil:
		return Subtitles{}, fmt.Errorf("%s: yt-dlp's %s cannot be read: %v", url, name, err)
	}
	if pytext.Strip(string(raw)) == "" {
		return Subtitles{}, noTrack(url)
	}
	fragments, err := parseJSON3(raw)
	if err != nil {
		return Subtitles{}, fmt.Errorf("%s: the subtitle track is no json3: %v", url, err)
	}
	return Subtitles{Title: meta.Title, Fragments: fragments}, nil
}

// noTrack is the reference's answer for a video that offers no track to
// read (fetch.py:64-65).
func noTrack(url string) error {
	return fmt.Errorf("%s: no subtitle track to fetch, and this system does no ASR", url)
}

// unfinished says why a run that left a file out never got to write it:
// yt-dlp did not start, or it ran out of time. For a run that ended by
// itself it is nil.
func unfinished(url string, res child.Result) error {
	switch {
	case res.Err != nil:
		return fmt.Errorf("%s: yt-dlp did not start: %s", url, oneLine(res.Err))
	case res.TimedOut:
		return fmt.Errorf("%s: yt-dlp took longer than %s", url, fetchTimeout)
	}
	return nil
}

// ended is how yt-dlp ended, for a message: its exit code and the line it
// said, with no colon where it said none.
func ended(code int, line string) string {
	if line == "" {
		return fmt.Sprintf("(exit %d)", code)
	}
	return fmt.Sprintf("(exit %d): %s", code, line)
}

// lastLine is the last line yt-dlp wrote to stderr, without a CR: the seam
// does not promise LF.
func lastLine(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// lineAbout is the last line of stderr about the language, which yt-dlp
// names as in "subtitles for 'de'", and the last line where none is:
// yt-dlp asks for de before en, so the last line can be the other
// language's.
func lineAbout(stderr, lang string) string {
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "for '"+lang+"'") {
			return strings.TrimSpace(lines[i])
		}
	}
	return lastLine(stderr)
}

// ytdlpArgs is the command line the recording under testdata/convert/ytdlp
// ran, flag for flag and in its order.
func ytdlpArgs(url string) []string {
	return []string{"--ignore-config", "--no-playlist", "--no-progress", "--skip-download",
		"--write-subs", "--write-auto-subs", "--sub-langs", "de,en", "--sub-format", "json3", "--write-info-json",
		"--ignore-errors", "-o", "v", url}
}

// chooseTrack is the reference's order (fetch.py:118-124): manual tracks
// before automatic ones, de before en within each -- and a language that
// has tracks but none in json3 ends its kind's search. It picks a language;
// which of its json3 tracks the file holds, yt-dlp decided. With yt-dlp
// 2026.08.19 that end skips no json3 track on YouTube: its YouTube
// extractor lists all seven formats for every language
// (`process_language`). Another extractor may list a language without
// json3; then the search ends there as the reference's did.
func chooseTrack(meta infoJSON) (string, bool) {
	for _, byLang := range []map[string][]track{meta.Subtitles, meta.Automatic} {
		lang, tracks := "de", byLang["de"]
		if len(tracks) == 0 {
			lang, tracks = "en", byLang["en"]
		}
		if slices.ContainsFunc(tracks, func(t track) bool { return t.Ext == "json3" }) {
			return lang, true
		}
	}
	return "", false
}

// parseJSON3 reads json3, not vtt: the vtt track scrolls and repeats every
// line. An event of whitespace alone is no fragment.
func parseJSON3(raw []byte) ([]Fragment, error) {
	var doc struct {
		Events []struct {
			TStartMs float64 `json:"tStartMs"`
			Segs     []struct {
				UTF8 string `json:"utf8"`
			} `json:"segs"`
		} `json:"events"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var out []Fragment
	for _, event := range doc.Events {
		var text strings.Builder
		for _, seg := range event.Segs {
			text.WriteString(seg.UTF8)
		}
		if t := pytext.Strip(text.String()); t != "" {
			out = append(out, Fragment{Mark: mark(int64(event.TStartMs)), Text: t})
		}
	}
	return out, nil
}

// mark is hh:mm:ss, not mm:ss: past minute 100 the bracket mark would stop
// matching, and the rest of the video would melt into one paragraph.
func mark(milliseconds int64) string {
	total := milliseconds / 1000
	return fmt.Sprintf("%02d:%02d:%02d", total/3600, total%3600/60, total%60)
}

// ToInbox files the fragments in bracket form, the form a person writes by
// hand, under the video's title.
func ToInbox(subs Subtitles, url, inbox string) (string, error) {
	stem := pytext.Strip(pytext.FirstRunes(pytext.Strip(unsafeName().ReplaceAllString(subs.Title, "")), maxStem))
	if stem == "" {
		stem = "video"
	}
	name := stem + ".txt"
	if m := watchID().FindStringSubmatch(url); m != nil {
		name = fmt.Sprintf("%s (%s).txt", stem, m[1])
	}
	lines := make([]string, len(subs.Fragments))
	for i, f := range subs.Fragments {
		lines[i] = "[" + f.Mark + "] " + f.Text
	}
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(inbox, name)
	if _, err := writeIfChanged(target, strings.Join(lines, "\n\n")+"\n"); err != nil {
		return "", err
	}
	return target, nil
}
