package web

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/xidus90/loomux/internal/config/target"
)

// proposalRow is one open proposal of the page's target: the diff it would
// make to the file as it is now, or why it no longer holds.
type proposalRow struct {
	ID, Describe, Created, Diff, Error string
	// DiffHash names the diff the page shows; applying sends it back, so
	// that what is written is what was shown.
	DiffHash string
}

func proposalRows(views []target.ProposalView) []proposalRow {
	rows := make([]proposalRow, len(views))
	for i, v := range views {
		rows[i] = proposalRow{ID: v.ID, Created: v.Created, Diff: v.Diff, Error: v.Error, DiffHash: hashOf(v.Diff)}
		// A proposal file that did not read has no operation to describe.
		if v.Op != "" {
			rows[i].Describe = v.Describe()
		}
	}
	return rows
}

// onProposal runs act on the open proposal an address names, then returns to
// the settings page: to the page itself on success, and to it with the
// reason when the proposal is not there or act fails. Looking it up in the
// listing keeps an id from naming any other file.
func (s *server) onProposal(w http.ResponseWriter, r *http.Request, act func(tgt target.Target, id string) error) {
	tgt, _, ok := s.target(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	ids, err := tgt.IDs()
	if err == nil {
		err = fmt.Errorf("no open proposal %q", r.PathValue("pid"))
		if slices.Contains(ids, r.PathValue("pid")) {
			err = act(tgt, r.PathValue("pid"))
		}
	}
	if err != nil {
		s.showSettings(w, r, err)
		return
	}
	http.Redirect(w, r, "/settings/"+r.PathValue("id"), http.StatusSeeOther)
}

// applyProposal writes one proposal. It computes the proposal against the
// file once and writes that very result, if its diff is the one the form was
// made from: a second read for the write could meet another proposal file
// or another file text than the one compared.
func (s *server) applyProposal(w http.ResponseWriter, r *http.Request) {
	s.onProposal(w, r, func(tgt target.Target, id string) error {
		pending, err := tgt.Prepare(id)
		if err != nil {
			return err
		}
		if hashOf(pending.Diff) != r.PostFormValue("diff") {
			return errors.New("the proposal changed since the page was drawn; nothing written")
		}
		return tgt.ApplyPending(pending)
	})
}

// rejectProposal drops one proposal and leaves the file alone.
func (s *server) rejectProposal(w http.ResponseWriter, r *http.Request) {
	s.onProposal(w, r, func(tgt target.Target, id string) error {
		return tgt.Reject(id)
	})
}
