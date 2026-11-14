package web

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
	"sync"
)

//go:embed assets
var assets embed.FS

// pageNames are the pages, each a file under assets/ that defines "content"
// for the layout.
var pageNames = []string{"login", "dashboard", "dashboard_edit", "settings", "preview", "raw", "confirm",
	"wizard_path", "wizard_facts", "wizard_plan", "wizard_done"}

// pages parses the pages on first use, so that starting the process, or any
// other command of the binary, costs none of it. Each page is the layout and
// its own file in a set of its own: two pages cannot redefine each other's
// "content".
var pages = sync.OnceValue(func() map[string]*template.Template {
	out := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		out[name] = template.Must(template.ParseFS(assets, "assets/layout.html", "assets/"+name+".html"))
	}
	return out
})

// view is what every page is rendered with: the layout's own needs, and the
// page's data.
type view struct {
	// CSRF is the token of the request's session, empty without one. A form
	// carries it as the hidden field the middleware reads.
	CSRF    string
	Version string
	// Nav is the projects the header links to, empty without a session or
	// when the lists cannot be read; the machine's settings stay linked.
	Nav  []Project
	Data any
}

// page renders a page inside the layout, with the session's form token.
func (s *server) page(w http.ResponseWriter, r *http.Request, name string, data any) {
	s.render(w, r, http.StatusOK, name, data)
}

// render is page with a status. The page is rendered in full before anything
// is written, so a template that fails is a 500 and not half a page.
func (s *server) render(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	v := view{Version: s.version, Data: data}
	if sess := sessionOf(r); sess != nil {
		v.CSRF = sess.csrf
		v.Nav, _ = Projects(s.options.StateDir)
	}
	var out bytes.Buffer
	tpl, found := pages()[name]
	if !found {
		log.Printf("web: no page %q", name)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(&out, "layout.html", v); err != nil {
		log.Printf("web: render %q: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(out.Bytes())
}

// style serves the one style sheet, which the pages link instead of carrying
// inline: the content security policy allows no inline style.
func style(w http.ResponseWriter, _ *http.Request) {
	// The file is embedded, so reading it cannot fail.
	data, _ := assets.ReadFile("assets/style.css")
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(data)
}
