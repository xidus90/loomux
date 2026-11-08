package corpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/xidus90/loomux/internal/code/store"
)

const (
	// corpusRecordVersion is the shape of the sidecar. A record of another
	// version is a record this code cannot read, which is the same as none.
	corpusRecordVersion = 3
	// binaryRuleVersion names the NUL rule the corpus was selected with.
	binaryRuleVersion = "nul-v1"
	// ignoreRuleVersion names the git semantics the corpus was selected with.
	ignoreRuleVersion = "git-v1"
	// corpusRecordName is the sidecar beside the graph and the index.
	corpusRecordName = "corpus.json"
)

// Stamp is a file's size and modification time, the pair a probe compares
// before it reads anything.
type Stamp struct {
	Size  int64 `json:"size"`
	MTime int64 `json:"mtime"`
}

// GitPaths are the paths Load resolved, so the probe needs no git call.
type GitPaths struct {
	Root    string `json:"root"`
	Prefix  string `json:"prefix"`
	Index   string `json:"index"`
	Config  string `json:"config"`
	Exclude string `json:"exclude"`
	Global  string `json:"global"`
}

// Print is one search file's fingerprint. Binary marks the files the corpus
// leaves out; the record still carries their hash so a probe can tell that
// nothing but the binary decision changed.
type Print struct {
	Size   int64  `json:"size"`
	MTime  int64  `json:"mtime"`
	Hash   string `json:"hash"`
	Binary bool   `json:"binary"`
}

// Record is the search corpus's sidecar: the settings it was selected with,
// the ignore sources it read, the stamps of .git/index and .git/config, and
// one print per file. It never contains a path's bytes.
type Record struct {
	Version     int              `json:"version"`
	Settings    []byte           `json:"settings"`
	IgnoreCase  bool             `json:"ignorecase"`
	Sources     []Source         `json:"sources"`
	Stamp       Stamp            `json:"index_stamp"`
	ConfigStamp Stamp            `json:"config_stamp"`
	Tracked     []string         `json:"tracked"`
	GitPaths    GitPaths         `json:"git_paths"`
	Files       map[string]Print `json:"files"`
	// Digest is hex(Digest(record)), stored so a probe reads it instead of
	// hashing the file list again.
	Digest string `json:"digest"`
	// The selection's counters, so a clean probe can report what the corpus
	// left out without selecting it again.
	Ignored       int   `json:"ignored"`
	SkippedSize   int   `json:"skipped_size"`
	SkippedBinary int   `json:"skipped_binary"`
	MaxFilesize   int64 `json:"max_filesize"`
}

// RecordPath is where the search corpus's record lives.
func RecordPath(root string) string { return store.CachePath(root, corpusRecordName) }

// CanonicalSettings is the text the digest binds to: every input the corpus
// selection depends on that is not the tree itself.
func CanonicalSettings(maxFilesize int64, ignoreCase bool) []byte {
	return []byte(fmt.Sprintf("max_filesize=%d;binary=%s;ignore=%s;ignorecase=%t",
		maxFilesize, binaryRuleVersion, ignoreRuleVersion, ignoreCase))
}

// NewRecord assembles the record from the state the build just used. A nil
// rules value is a build outside a git repository: the record then carries no
// sources and no stamps, and the probe walks without ignore evaluation.
func NewRecord(rules *Rules, prints map[string]Print, maxFilesize int64) Record {
	rec := Record{Version: corpusRecordVersion, Files: prints, MaxFilesize: maxFilesize}
	fillGitState(&rec, rules, maxFilesize)
	sum := Digest(rec)
	rec.Digest = hex.EncodeToString(sum[:])
	return rec
}

// fillGitState copies what the digest and the probe need from rules into rec.
func fillGitState(rec *Record, rules *Rules, maxFilesize int64) {
	if rules == nil {
		rec.Settings = CanonicalSettings(maxFilesize, false)
		return
	}
	indexPath, indexSize, indexMTime := rules.IndexStamp()
	configPath, configSize, configMTime := rules.ConfigStamp()
	rec.Settings = CanonicalSettings(maxFilesize, rules.IgnoreCase())
	rec.IgnoreCase = rules.IgnoreCase()
	rec.Sources = rules.Sources()
	rec.Stamp = Stamp{Size: indexSize, MTime: indexMTime}
	rec.ConfigStamp = Stamp{Size: configSize, MTime: configMTime}
	rec.Tracked = rules.TrackedPaths()
	rec.GitPaths = GitPaths{
		Root:    rules.RepoTop(),
		Prefix:  rules.Prefix(),
		Index:   indexPath,
		Config:  configPath,
		Exclude: rules.excludePath,
		Global:  rules.globalPath,
	}
}

// Digest is the SHA-256 over the canonical settings, the ignore sources
// (path-sorted, by their content hash) and the search corpus's (path, hash)
// pairs in path order. It depends on content and settings, never on a time:
// the stamps stay out, the probe compares them itself.
func Digest(rec Record) [32]byte {
	h := sha256.New()
	writeDigestLine(h, "settings", string(rec.Settings))
	sources := append([]Source(nil), rec.Sources...)
	sort.Slice(sources, func(i, j int) bool { return sources[i].Rel < sources[j].Rel })
	for _, s := range sources {
		writeDigestLine(h, "source", s.Rel+"\x00"+hex.EncodeToString(s.Hash[:]))
	}
	rels := make([]string, 0, len(rec.Files))
	for rel := range rec.Files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		writeDigestLine(h, "file", rel+"\x00"+rec.Files[rel].Hash)
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// writeDigestLine frames one line so that no path can imitate a separator:
// sha256.New never returns an error.
func writeDigestLine(h interface{ Write([]byte) (int, error) }, kind, rest string) {
	_, _ = h.Write([]byte(kind))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(rest))
	_, _ = h.Write([]byte{'\n'})
}

// Write stores the record atomically and then its binary mirror. The mirror
// is a cache of the JSON file: when only it cannot be written the record stands,
// the next reader takes the JSON file, and the error wraps ErrMirror so that
// the caller can say why the fast path is gone.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode; Record holds only strings, ints, bools, byte slices and maps thereof
func Write(root string, rec Record) error {
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	final := RecordPath(root)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return err
	}
	if err := replaceFile(final, body); err != nil {
		return err
	}
	return WriteMirror(final, rec)
}

// stampKey opens the git index stamp in the layout Write produces. A path
// cannot imitate it: a file's key sits one level deeper, and a source's rel
// is a string, whose quotes JSON escapes.
var stampKey = []byte("\n  \"index_stamp\": ")

// Restamp stores stamp as the git index stamp of the record at root, which
// was read from the file with the stamp was. A record file that moved on
// since belongs to another writer and is left alone. Only the stamp's bytes
// change, so the file comes out as Write would write it without marshalling
// the file list again; the mirror is written anew to bind to it.
func Restamp(root string, was Stamp, rec Record, stamp Stamp) error {
	path := RecordPath(root)
	body, err := readFile(path)
	if err != nil {
		return err
	}
	// Stat after the read: a swap before it shows here.
	if FileStamp(path) != was {
		return nil
	}
	rec.Stamp = stamp
	head, tail, found := bytes.Cut(body, stampKey)
	_, rest, closed := bytes.Cut(tail, []byte("}"))
	if !found || !closed {
		return Write(root, rec)
	}
	// A Stamp holds two ints, which json always encodes.
	value, _ := json.MarshalIndent(stamp, "  ", "  ")
	if err := replaceFile(path, slices.Concat(head, stampKey, value, rest)); err != nil {
		return err
	}
	return WriteMirror(path, rec)
}

// FileStamp is the stamp of the file at path, the empty stamp when it cannot
// be read.
func FileStamp(path string) Stamp {
	info, err := os.Stat(path)
	if err != nil {
		return Stamp{}
	}
	return Stamp{Size: info.Size(), MTime: info.ModTime().UnixNano()}
}

// Read loads the record, or reports that there is none this code may use. A
// record is a cache: a broken one costs the fast path and nothing more.
func Read(root string) (Record, bool) {
	path := RecordPath(root)
	var mirrored Record
	if ReadMirror(path, &mirrored) && usable(mirrored) {
		return mirrored, true
	}
	b, err := readFile(path)
	if err != nil {
		return Record{}, false
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		return Record{}, false
	}
	if !usable(rec) {
		return Record{}, false
	}
	return rec, true
}

// usable reports whether this code may rely on rec.
func usable(rec Record) bool {
	return rec.Version == corpusRecordVersion && rec.Files != nil && len(rec.Settings) != 0 && rec.Digest != ""
}
