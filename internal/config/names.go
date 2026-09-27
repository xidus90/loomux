package config

// The two name rules, as a message states them.
const (
	IdentifierRule = "[A-Za-z_][A-Za-z0-9_]*"
	FlowNameRule   = "[a-z][a-z0-9-]*"
)

// IsIdentifier reports whether name follows the rule a role, a model, a node,
// a field and a parameter name share. It is the rule a flow's conditions can
// scan, and a key segment of .loomux/config.toml can hold without quotes.
func IsIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

// IsFlowName reports whether name follows the rule of a flow's name. The name
// is the flow's folder, so it is lower case -- Windows does not tell Review
// from review -- and it starts with a letter, because go:embed leaves out a
// name that starts with "_" or ".".
func IsFlowName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}
