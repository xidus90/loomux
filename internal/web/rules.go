package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// ruleFields are the lists a rule form may append to or remove from, each
// with the fields of one entry. A list of globs takes them one per line.
var ruleFields = map[string][2]string{
	"policy.paths.rules":    {"match", "reason"},
	"policy.commands.rules": {"regex", "reason"},
	"commit.allow":          {"regex", "reason"},
}

// ruleList is the globs a form sent, one per line of each value, as the TOML
// literal of an array.
func ruleList(values []string) string {
	var items []string
	for _, v := range values {
		for line := range strings.SplitSeq(v, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				items = append(items, line)
			}
		}
	}
	return tomlList(items)
}

// rulePreview shows what appending a rule to a list, or removing one from
// the file's own list, would change. It writes nothing; the preview's apply
// form does.
func (s *server) rulePreview(w http.ResponseWriter, r *http.Request) {
	tgt, id, text, ok := s.current(w, r)
	if !ok {
		return
	}
	// The middleware read the body for the form token; this only exposes
	// the values.
	_ = r.ParseForm()
	section := r.PostForm.Get("section")
	fields, known := ruleFields[section]
	_, listed := tgt.Lookup(section)
	var next string
	var err error
	switch _, remove := r.PostForm["remove"]; {
	case !known || !listed:
		err = fmt.Errorf("%q is not a list of rules of this file", section)
	case remove:
		var index int
		if index, err = strconv.Atoi(r.PostForm.Get("remove")); err != nil {
			err = fmt.Errorf("%q is not the number of a rule", r.PostForm.Get("remove"))
			break
		}
		next, err = tgt.RemoveRule(text, section, index)
	default:
		first := config.QuoteTOML(strings.TrimSpace(r.PostForm.Get(fields[0])))
		if fields[0] == "match" {
			first = ruleList(r.PostForm["match"])
		}
		next, err = tgt.AppendRule(text, section, [][2]string{
			{fields[0], first},
			{"reason", config.QuoteTOML(strings.TrimSpace(r.PostForm.Get("reason")))},
		})
	}
	s.showPreview(w, r, id, text, settle(next, text), err, false)
}
