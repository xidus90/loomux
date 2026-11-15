package rust

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// attr is one attribute in front of an item, read as text: the grammar leaves
// the arguments of #[class(rename = X)] an unparsed token tree, and the few
// attributes read here need no more than their words.
type attr struct {
	name string // the last segment of the path: "test" for tokio::test
	args string // the token tree with its parentheses, "" without one
	eq   string // the string after "=", without its quotes: y.rs for path = "y.rs"
}

// attrsOf is the attribute of an attribute_item. An item the parser rebuilt
// without one yields none.
func (x *extractor) attrsOf(item *gts.Node) []attr {
	var out []attr
	for _, c := range item.Children() {
		if x.doc.Type(c) != "attribute" {
			continue
		}
		out = append(out, attr{
			name: lastSegment(x.doc.Text(c.Child(0))),
			args: x.doc.Text(x.doc.Field(c, "arguments")),
			eq:   strings.Trim(x.doc.Text(x.doc.Field(c, "value")), `"`),
		})
	}
	return out
}

// lastSegment is the last name of a path: "test" for "tokio::test".
func lastSegment(path string) string {
	if i := strings.LastIndex(path, "::"); i >= 0 {
		return path[i+2:]
	}
	return path
}

// find is the first attribute of the given name.
func find(attrs []attr, name string) (attr, bool) {
	for _, a := range attrs {
		if a.name == name {
			return a, true
		}
	}
	return attr{}, false
}

// words are the comma-separated parts of an attribute's arguments, outside
// the parentheses.
func (a attr) words() []string {
	if a.args == "" {
		return nil
	}
	parts := strings.Split(strings.Trim(a.args, "()"), ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// value is the word after "key =" among the arguments, "" without one.
func (a attr) value(key string) string {
	for _, w := range a.words() {
		if k, v, ok := strings.Cut(w, "="); ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// isTest is whether the attributes make a function a test: #[test] or any
// attribute whose path ends in test, such as #[tokio::test].
func isTest(attrs []attr) bool {
	_, ok := find(attrs, "test")
	return ok
}

// cfgTest is whether the attributes compile their item for tests alone,
// #[cfg(test)]; a condition that merely mentions test (all(test, ...)) does
// not count.
func cfgTest(attrs []attr) bool {
	for _, a := range attrs {
		if a.name == "cfg" && strings.Join(strings.Fields(a.args), "") == "(test)" {
			return true
		}
	}
	return false
}

// godotClass is the Godot name of the class a struct declares, "" for a
// struct that is none: derive(GodotClass), qualified or not, makes it one,
// and #[class(rename = X)] names it X instead of the struct's own name.
func godotClass(attrs []attr, structName string) string {
	for _, a := range attrs {
		for _, w := range a.words() {
			if a.name == "derive" && lastSegment(w) == "GodotClass" {
				if class, ok := find(attrs, "class"); ok && class.value("rename") != "" {
					return class.value("rename")
				}
				return structName
			}
		}
	}
	return ""
}
