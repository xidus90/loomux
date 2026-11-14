package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/target"
)

// bom is the byte order mark a file may start with; it is the file's, not
// the text's.
const bom = "\xef\xbb\xbf"

// hashOf names a text in the form that carries it from the preview to the
// write.
func hashOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// settle gives next the line endings and the byte order mark of the file it
// replaces: a browser sends a text field with CRLF whatever the file holds,
// and one changed line must not turn every line into a change. A file that
// is not there yet is LF. The preview diffs the settled text and the write
// puts the same one in place.
func settle(next, current string) string {
	ending := "\n"
	if strings.Contains(current, "\r\n") {
		ending = "\r\n"
	}
	next = strings.ReplaceAll(strings.ReplaceAll(next, "\r\n", "\n"), "\n", ending)
	next = strings.TrimPrefix(next, bom)
	if strings.HasPrefix(current, bom) {
		next = bom + next
	}
	return next
}

// previewView is the data of the preview page. Before and Next are empty
// when there is nothing to apply.
type previewView struct {
	ID, Diff, Error string
	Before, Next    string
	// Changed says the file is no longer the one the form was made from.
	Changed bool
}

// current finds the target an address names and reads its text. It answers
// the request itself when it cannot.
func (s *server) current(w http.ResponseWriter, r *http.Request) (tgt target.Target, id, text string, ok bool) {
	id = r.PathValue("id")
	tgt, _, found := s.target(id)
	if !found {
		http.NotFound(w, r)
		return tgt, id, "", false
	}
	text, err := tgt.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return tgt, id, "", false
	}
	return tgt, id, text, true
}

// showPreview renders what writing next over current would change. A next
// that did not pass its check is no next: err is shown and nothing offers to
// write.
func (s *server) showPreview(w http.ResponseWriter, r *http.Request, id, current, next string, err error, changed bool) {
	v := previewView{ID: id, Changed: changed}
	if err != nil {
		v.Error = err.Error()
	} else {
		v.Diff = edit.Diff(current, next)
		v.Before, v.Next = hashOf(current), next
	}
	s.page(w, r, "preview", v)
}

// preview computes the text a form asks for and shows its diff. It writes
// nothing.
func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	tgt, id, text, ok := s.current(w, r)
	if !ok {
		return
	}
	// The body is parsed already (the form token), and this is a no-op then.
	_ = r.ParseForm()
	var next string
	var err error
	switch _, isRaw := r.PostForm["raw"]; {
	// The field's presence is the choice: an empty text is a real request.
	case isRaw:
		next, err = tgt.Validated(settle(r.PostForm.Get("raw"), text))
	case r.PostForm.Get("reset") == "1":
		next, err = tgt.Unset(text, r.PostForm.Get("key"))
	default:
		next, err = tgt.Change(text, r.PostForm.Get("key"), r.PostForm.Get("value"))
	}
	s.showPreview(w, r, id, text, settle(next, text), err, false)
}

// apply writes the text of a preview when the file is still the one the
// preview was made from. Otherwise it writes nothing and shows the diff
// against the file as it is now, so that what is written is always what was
// shown last. The text is checked again either way.
func (s *server) apply(w http.ResponseWriter, r *http.Request) {
	tgt, id, text, ok := s.current(w, r)
	if !ok {
		return
	}
	next, err := tgt.Validated(settle(r.PostFormValue("next"), text))
	stale := hashOf(text) != r.PostFormValue("before")
	if err == nil && !stale {
		err = tgt.Write(text, next)
	}
	if err != nil || stale {
		s.showPreview(w, r, id, text, next, err, stale)
		return
	}
	http.Redirect(w, r, "/settings/"+id, http.StatusSeeOther)
}

// rawView is the data of the raw text page.
type rawView struct{ ID, Text string }

// raw shows the whole file in a text field. It reads no entries, so a file
// that does not parse is shown too: this is where it is repaired.
func (s *server) raw(w http.ResponseWriter, r *http.Request) {
	_, id, text, ok := s.current(w, r)
	if !ok {
		return
	}
	s.page(w, r, "raw", rawView{ID: id, Text: strings.TrimPrefix(text, bom)})
}
