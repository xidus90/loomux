package web

import (
	"html/template"
	"net/http"
	"slices"

	"github.com/xidus90/loomux/internal/pathkey"
)

// Tile is one panel of the dashboard. The list of tiles is fixed in code; what
// the user chooses is which of them show and in which order.
type Tile struct {
	ID, Title string
	// Render builds the body of the tile. csrf is the form token of the
	// session the page is drawn for, for a tile that carries a form. A tile
	// whose source fails returns the error, which the page shows in the
	// tile's own frame and nowhere else.
	Render func(s *server, csrf string) (template.HTML, error)
}

// tileView is a tile as the page draws it.
type tileView struct {
	ID, Title string
	Body      template.HTML
	Error     string
}

// dashboardView is the data of the dashboard page.
type dashboardView struct {
	// Notice is why the arrangement is not the user's.
	Notice string
	Tiles  []tileView
}

// editRow is one tile on the arrangement page.
type editRow struct {
	ID, Title   string
	Shown       bool
	First, Last bool
}

// editView is the data of the arrangement page.
type editView struct {
	Notice string
	Rows   []editRow
}

// writePrefs is how the dashboard's handlers save ui.toml; a test replaces it
// to see what a failing disk makes of a request.
var writePrefs = WritePrefs

// prefsNotice is what the pages say when ui.toml does not parse: the file
// counts as missing, so the defaults show.
func prefsNotice(err error) string { return "ui.toml could not be read: " + err.Error() }

// shownOrder is the ids of the tiles that show, in order. Without a tile list
// in ui.toml every tile shows in the fixed order; a name the list holds that
// is no tile, or that it holds twice, is passed over.
func shownOrder(all []Tile, p Prefs) []string {
	ids := make([]string, 0, len(all))
	if !p.TilesSet {
		for _, t := range all {
			ids = append(ids, t.ID)
		}
		return ids
	}
	for _, id := range p.Tiles {
		known := slices.ContainsFunc(all, func(t Tile) bool { return t.ID == id })
		if known && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}

// tileByID is the tile of a known id.
func tileByID(all []Tile, id string) Tile {
	return all[slices.IndexFunc(all, func(t Tile) bool { return t.ID == id })]
}

// dashboard is the start page.
func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	s.showDashboard(w, r, tiles())
}

// showDashboard draws the tiles of all that the user's arrangement shows. A
// ui.toml that does not read is said at the top and the defaults show.
func (s *server) showDashboard(w http.ResponseWriter, r *http.Request, all []Tile) {
	prefs, err := ReadPrefs(s.options.StateDir)
	var v dashboardView
	if err != nil {
		prefs, v.Notice = Prefs{}, prefsNotice(err)
	}
	csrf := sessionOf(r).csrf
	for _, id := range shownOrder(all, prefs) {
		t := tileByID(all, id)
		tv := tileView{ID: t.ID, Title: t.Title}
		body, err := t.Render(s, csrf)
		if err != nil {
			tv.Error = err.Error()
		} else {
			tv.Body = body
		}
		v.Tiles = append(v.Tiles, tv)
	}
	s.page(w, r, "dashboard", v)
}

// editPage is the arrangement page: the shown tiles in order, then the hidden
// ones. notice and status are for a change that was refused.
func (s *server) editPage(w http.ResponseWriter, r *http.Request, status int, notice string) {
	prefs, err := ReadPrefs(s.options.StateDir)
	if err != nil {
		prefs, notice = Prefs{}, joinNotice(notice, prefsNotice(err))
	}
	all := tiles()
	shown := shownOrder(all, prefs)
	v := editView{Notice: notice}
	for i, id := range shown {
		v.Rows = append(v.Rows, editRow{ID: id, Title: tileByID(all, id).Title, Shown: true, First: i == 0, Last: i == len(shown)-1})
	}
	for _, t := range all {
		if !slices.Contains(shown, t.ID) {
			v.Rows = append(v.Rows, editRow{ID: t.ID, Title: t.Title})
		}
	}
	s.render(w, r, status, "dashboard_edit", v)
}

// joinNotice joins two reasons, the one that is there first.
func joinNotice(first, second string) string {
	if first == "" {
		return second
	}
	return first + "; " + second
}

func (s *server) dashboardEdit(w http.ResponseWriter, r *http.Request) {
	s.editPage(w, r, http.StatusOK, "")
}

// arrange applies one action to the order of the shown tiles.
func arrange(order []string, id, action string) []string {
	order = slices.Clone(order)
	at := slices.Index(order, id)
	switch action {
	case "hide":
		if at >= 0 {
			order = slices.Delete(order, at, at+1)
		}
	case "show":
		if at < 0 {
			order = append(order, id)
		}
	case "up":
		if at > 0 {
			order[at-1], order[at] = order[at], order[at-1]
		}
	case "down":
		if at >= 0 && at < len(order)-1 {
			order[at+1], order[at] = order[at], order[at+1]
		}
	}
	return order
}

// dashboardChange shows, hides or moves one tile and writes the new
// arrangement to ui.toml. An arrangement that comes out as it was is not
// written, so that a first "up" at the top does not turn the defaults into a
// list of the user's.
func (s *server) dashboardChange(w http.ResponseWriter, r *http.Request) {
	all := tiles()
	id, action := r.PostFormValue("tile"), r.PostFormValue("action")
	if !slices.ContainsFunc(all, func(t Tile) bool { return t.ID == id }) ||
		!slices.Contains([]string{"show", "hide", "up", "down"}, action) {
		http.Error(w, "unknown tile or action", http.StatusBadRequest)
		return
	}
	prefs, err := ReadPrefs(s.options.StateDir)
	if err != nil {
		// Writing from the defaults would drop the projects the file holds.
		s.editPage(w, r, http.StatusConflict, "nothing written")
		return
	}
	before := shownOrder(all, prefs)
	after := arrange(before, id, action)
	if !slices.Equal(before, after) {
		prefs.Tiles, prefs.TilesSet = after, true
		if err := writePrefs(s.options.StateDir, prefs); err != nil {
			s.editPage(w, r, http.StatusInternalServerError, "nothing written: "+err.Error())
			return
		}
	}
	http.Redirect(w, r, "/dashboard/edit", http.StatusSeeOther)
}

// forget drops a project from the list of ui.toml. The project's own files
// are not touched, and a project that is registered stays listed through the
// registry.
func (s *server) forget(w http.ResponseWriter, r *http.Request) {
	// The list is read before it is changed: writing from a ui.toml that did
	// not read would drop what the file holds.
	prefs, err := ReadPrefs(s.options.StateDir)
	if err != nil {
		http.Error(w, "nothing written: "+err.Error(), http.StatusConflict)
		return
	}
	projects, err := Projects(s.options.StateDir)
	if err != nil {
		http.Error(w, "nothing written: "+err.Error(), http.StatusConflict)
		return
	}
	at := slices.IndexFunc(projects, func(p Project) bool { return p.ID == r.PathValue("id") })
	if at < 0 {
		http.NotFound(w, r)
		return
	}
	key := pathkey.Key(projects[at].Root)
	prefs.Projects = slices.DeleteFunc(slices.Clone(prefs.Projects), func(path string) bool { return pathkey.Key(path) == key })
	if err := writePrefs(s.options.StateDir, prefs); err != nil {
		http.Error(w, "nothing written: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
