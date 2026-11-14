package web

import (
	"net/http"

	"github.com/xidus90/loomux/internal/config"
)

// notRegistered is what a project outside the registry gets in place of the
// llm_config switch.
const notRegistered = "only a registered area has this switch"

// llmOpens is what the switch opens, as docs/en/configuration.md words it.
const llmOpens = "With the switch on, an agent may edit exactly this project's `.loomux/config.toml` with file tools and run `loomux config set|unset <key> [<value>] --yes` (without `--root`, `--global` and `--propose`, as a single plain line, from a shell standing at the project root; after a `cd` elsewhere the line stays refused)"

// llmLoosens is what the human accepts with it, from the same section.
const llmLoosens = "With the switch on the LLM can also loosen checks (`guard.mode`, `modules.*`, `[policy]` and others); you accept that when you turn it on."

// confirmView is the data of the confirmation page of the switch.
type confirmView struct {
	ID, Opens, Loosens string
	On                 bool
}

// registered finds the project an address names, answering the request
// itself when there is none or when it is no registered area.
func (s *server) registered(w http.ResponseWriter, r *http.Request) (Project, bool) {
	_, project, ok := s.target(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return project, false
	}
	if !project.Registered {
		http.Error(w, notRegistered, http.StatusConflict)
		return project, false
	}
	return project, true
}

// llmPage is the page that names what the switch opens, where it is turned.
func (s *server) llmPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.registered(w, r)
	if !ok {
		return
	}
	s.page(w, r, "confirm", confirmView{ID: r.PathValue("id"), Opens: llmOpens, Loosens: llmLoosens, On: project.LLMConfig})
}

// llmSet turns the switch of the registered area the address names.
func (s *server) llmSet(w http.ResponseWriter, r *http.Request) {
	project, ok := s.registered(w, r)
	if !ok {
		return
	}
	on := r.PostFormValue("on")
	if on != "true" && on != "false" {
		http.Error(w, "on must be true or false", http.StatusBadRequest)
		return
	}
	if err := config.SetLLMConfig(s.options.StateDir, project.Scope, on == "true"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings/"+r.PathValue("id"), http.StatusSeeOther)
}
