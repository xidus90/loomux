package stylesheet

import (
	"bytes"
	"regexp"
	"strings"
	"sync"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// The raw edges of SCSS: what @include, @extend and a call in a value name.

// member is the m of an @include ns.m, after the ns the grammar takes for
// the name.
var member = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^\.([^\s;({]*)`) })

// extended is the selector an @extend names.
var extended = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`@extend\s+([%.][\w-]+)`) })

// rejectedImport is what an ERROR holds of the imports the grammar
// rejects: an @use with a namespace, an @import of a list of paths.
var rejectedImport = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`@use\s+["']([^"']+)["']\s+as\s+([\w-]+|\*)|@import\s+(?:"[^"]*"|'[^']*')(?:\s*,\s*(?:"[^"]*"|'[^']*'))*`)
})

// include records the mixin n names as a calls edge from s. The grammar
// takes ns of ns.m alone as the name, so m is read from the source after
// it. include reports whether the grammar also ended the statement there,
// so that the caller can take the selector it made of .m for what it is.
func (x *extractor) include(n *gts.Node, s scope) bool {
	id := n.Child(1)
	name, receiver := x.doc.Text(id), ""
	if m := member().FindSubmatch(x.doc.Src[id.EndByte():]); m != nil {
		name, receiver = string(m[1]), name
	}
	x.calls = append(x.calls, x.edge(s, model.RelationCalls, name, receiver, "mixin", n.StartByte()))
	return bytes.HasPrefix(x.doc.Src[n.EndByte():], []byte("."))
}

// extend records every @extend in text, which starts at off, as an extends
// edge from s.
func (x *extractor) extend(text string, off uint32, s scope) {
	for _, m := range extended().FindAllStringSubmatchIndex(text, -1) {
		x.extends = append(x.extends, x.edge(s, model.RelationExtends, text[m[2]:m[3]], "", "", off+uint32(m[0])))
	}
}

// call records a call in a value as a calls edge from s, ns.f split at the
// dot. Plain CSS calls nothing of the repository.
func (x *extractor) call(n *gts.Node, s scope) {
	if !x.scss {
		return
	}
	name, receiver := x.doc.Text(n.Child(0)), ""
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name, receiver = name[i+1:], name[:i]
	}
	x.calls = append(x.calls, x.edge(s, model.RelationCalls, name, receiver, "function", n.StartByte()))
}

// rejected reads an ERROR as text: an @use with its namespace, an @import
// of a list and an @extend, the forms the grammar rejects, and nothing
// else.
func (x *extractor) rejected(n *gts.Node, s scope) {
	text := x.doc.Text(n)
	for _, m := range rejectedImport().FindAllStringIndex(text, -1) {
		x.importStatement(text[m[0]:m[1]], n.StartByte()+uint32(m[0]))
	}
	x.extend(text, n.StartByte(), s)
}

// extendPlaceholder is the % of @extend %p, which the grammar rejects and
// which closes the block around it early when a nested rule follows.
var extendPlaceholder = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`@extend\s+%`) })

// innerInterpolation is a #{$v} inside a name, var(--a-#{$v}-b) say: in a
// value it can take the rest of the file into one ERROR. One that calls a
// function is left alone, so that its call still counts, and so is one
// over several lines, which a mask would split into two names.
var innerInterpolation = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[\w-]#\{[^{}()\n]*\}|#\{[^{}()\n]*\}[\w-]`)
})

// mapKey is the colon after the first key of a map literal, ($k: v) or
// ("k": v), that stands where a value starts -- after a colon, an opening
// parenthesis or a comma: as the argument of a call it can take the rest
// of the file into one ERROR. A parenthesis after a name, with or without
// a space, opens parameters or arguments, m ($a: 1) or f($a: 1), and does
// not match.
var mapKey = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[:(,]\s*\(\s*(?:\$[\w-]+|"[^"\n]*"|'[^'\n]*')\s*:`)
})

// masked is a copy of src for the parser, the same length and with the
// same line breaks, with what the grammar breaks on made harmless: the %
// of @extend %p becomes a dot, the colon after a map key a comma, and
// #{$v} inside a name a run of underscores. Every text is still read from
// src.
func masked(src []byte) []byte {
	out := bytes.Clone(src)
	for _, m := range extendPlaceholder().FindAllIndex(out, -1) {
		fill(out[m[1]-1:m[1]], '.')
	}
	for _, m := range mapKey().FindAllIndex(out, -1) {
		fill(out[m[1]-1:m[1]], ',')
	}
	for _, m := range innerInterpolation().FindAllIndex(out, -1) {
		from := m[0] + bytes.IndexByte(out[m[0]:m[1]], '#')
		to := m[0] + bytes.IndexByte(out[m[0]:m[1]], '}')
		fill(out[from:to+1], '_')
	}
	return out
}

// fill overwrites b with c, keeping its line breaks, so that every line
// of the copy starts where the source's does.
func fill(b []byte, c byte) {
	for i := range b {
		if b[i] != '\n' && b[i] != '\r' {
			b[i] = c
		}
	}
}

// edge is a raw edge from s at byte offset off.
func (x *extractor) edge(s scope, rel model.Relation, name, receiver string, member model.Kind, off uint32) extract.RawEdge {
	return extract.RawEdge{Source: s.id, Relation: rel, Name: name, Receiver: receiver, Member: member, File: x.doc.Rel, Line: x.line(off)}
}
