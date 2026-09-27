package expr

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/flow"
)

// Names tells the parser what a condition may refer to.
type Names struct {
	Fields     map[string]flow.Type
	Params     map[string]flow.Type
	Predicates map[string]flow.Predicate
}

// ParseCondition reads one `when` and checks every name and type in it.
//
// The language is small on purpose: comparisons `<field> <op> <value>` and
// predicate names, joined only by `|` or only by `&`, without brackets.
// Everything it can say is checked here, before a run spends a model call.
func ParseCondition(source string, names Names) (flow.Condition, error) {
	tokens, err := scan(source)
	if err != nil {
		return nil, err
	}
	terms, joiner, err := split(tokens)
	if err != nil {
		return nil, err
	}
	conditions := make([]flow.Condition, 0, len(terms))
	for _, term := range terms {
		condition, err := parseTerm(term, names)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}
	if joiner == pipe {
		return anyOf(conditions), nil
	}
	return allOf(conditions), nil
}

type kind int

const (
	ident kind = iota
	number
	text
	operator
	pipe
	amp
	emptyList
)

type token struct {
	kind  kind
	value string // for text, the content without quotes and escapes
}

func scan(source string) ([]token, error) {
	var tokens []token
	for i := 0; i < len(source); {
		c := source[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '|':
			tokens = append(tokens, token{pipe, "|"})
			i++
		case c == '&':
			tokens = append(tokens, token{amp, "&"})
			i++
		case strings.HasPrefix(source[i:], "[]"):
			tokens = append(tokens, token{emptyList, "[]"})
			i += 2
		case c == '"':
			value, width, err := quoted(source[i:])
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{text, value})
			i += width
		case strings.IndexByte("=!<>", c) >= 0:
			op := operatorAt(source[i:])
			if op == "" {
				return nil, fmt.Errorf("unknown operator at column %d; operators are ==, !=, <, <=, >, >=", i+1)
			}
			tokens = append(tokens, token{operator, op})
			i += len(op)
		case c == '-' || isDigit(c):
			end := i + 1
			for end < len(source) && isDigit(source[end]) {
				end++
			}
			if source[i:end] == "-" {
				return nil, fmt.Errorf("a lone - at column %d is not a number", i+1)
			}
			tokens = append(tokens, token{number, source[i:end]})
			i = end
		case isLetter(c):
			end := i + 1
			for end < len(source) && (isLetter(source[end]) || isDigit(source[end])) {
				end++
			}
			tokens = append(tokens, token{ident, source[i:end]})
			i = end
		default:
			return nil, fmt.Errorf("unexpected %q at column %d", c, i+1)
		}
	}
	return tokens, nil
}

// operatorAt names the operator at the start of rest. The two-character ones
// are tried first, so "<=" is never read as "<" followed by "=".
func operatorAt(rest string) string {
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if strings.HasPrefix(rest, op) {
			return op
		}
	}
	return ""
}

// quoted reads the double-quoted text at the start of rest and reports how many
// bytes it took. A backslash takes the next byte literally, so \" and \\ are
// the way to write those two.
func quoted(rest string) (string, int, error) {
	var value strings.Builder
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case '"':
			return value.String(), i + 1, nil
		case '\\':
			i++
			if i < len(rest) {
				value.WriteByte(rest[i])
			}
		default:
			value.WriteByte(rest[i])
		}
	}
	return "", 0, fmt.Errorf("text %s has no closing quote", rest)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isLetter(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// split cuts the tokens at | or & and refuses a condition that uses both:
// without brackets, "a | b & c" has two readings and neither is obviously meant.
func split(tokens []token) ([][]token, kind, error) {
	if len(tokens) == 0 {
		return nil, 0, fmt.Errorf("the condition is empty")
	}
	joiner := kind(-1)
	var terms [][]token
	var current []token
	for _, tok := range tokens {
		if tok.kind != pipe && tok.kind != amp {
			current = append(current, tok)
			continue
		}
		if joiner != -1 && joiner != tok.kind {
			return nil, 0, fmt.Errorf("the condition mixes | and &; use only one of them")
		}
		joiner = tok.kind
		if len(current) == 0 {
			return nil, 0, fmt.Errorf("%s has no term before it", tok.value)
		}
		terms = append(terms, current)
		current = nil
	}
	if len(current) == 0 {
		return nil, 0, fmt.Errorf("%s has no term after it", tokens[len(tokens)-1].value)
	}
	return append(terms, current), joiner, nil
}

func parseTerm(term []token, names Names) (flow.Condition, error) {
	if len(term) == 1 && term[0].kind == ident {
		return predicateTerm(term[0].value, names)
	}
	if len(term) == 3 && term[0].kind == ident && term[1].kind == operator {
		return comparisonTerm(term[0].value, term[1].value, term[2], names)
	}
	return nil, fmt.Errorf("cannot read %q; expected <field> <op> <value> or a predicate name", render(term))
}

func render(term []token) string {
	parts := make([]string, len(term))
	for i, tok := range term {
		parts[i] = tok.value
		if tok.kind == text {
			parts[i] = strconv.Quote(tok.value)
		}
	}
	return strings.Join(parts, " ")
}

func predicateTerm(name string, names Names) (flow.Condition, error) {
	if predicate, ok := names.Predicates[name]; ok {
		return predicateCondition{predicate}, nil
	}
	if _, ok := names.Fields[name]; ok {
		return nil, fmt.Errorf("field %q needs a comparison, e.g. %s == ...", name, name)
	}
	return nil, fmt.Errorf("names %q, which is neither a field nor a predicate; known predicates: %s",
		name, known(names.Predicates))
}

func comparisonTerm(field, op string, value token, names Names) (flow.Condition, error) {
	fieldType, ok := names.Fields[field]
	if !ok {
		return nil, fmt.Errorf("reads %q; known fields: %s", field, known(names.Fields))
	}
	c := comparison{field: field, op: op, kind: fieldType}
	var valueType flow.Type
	switch value.kind {
	case text:
		valueType, c.literal = flow.String, value.value
	case number:
		n, err := strconv.Atoi(value.value)
		if err != nil {
			return nil, fmt.Errorf("%s is too large for an int", value.value)
		}
		valueType, c.literal = flow.Int, n
	case emptyList:
		valueType = flow.StringList
	case ident:
		if value.value == "true" || value.value == "false" {
			valueType, c.literal = flow.Bool, value.value == "true"
			break
		}
		paramType, ok := names.Params[value.value]
		if !ok {
			return nil, fmt.Errorf("compares %q with %q, which is no parameter; known parameters: %s",
				field, value.value, known(names.Params))
		}
		valueType, c.param = paramType, value.value
	default:
		return nil, fmt.Errorf("cannot compare %q with %s", field, value.value)
	}
	if valueType != fieldType {
		return nil, fmt.Errorf("compares %s field %q with a %s", fieldType, field, valueType)
	}
	if fieldType == flow.StringList && c.param != "" {
		return nil, fmt.Errorf("list field %q compares only with []", field)
	}
	if op != "==" && op != "!=" && fieldType != flow.Int {
		return nil, fmt.Errorf("%s needs an int, but %q is %s", op, field, fieldType)
	}
	return c, nil
}

type comparison struct {
	field   string
	op      string
	kind    flow.Type
	literal flow.Value
	param   string // set: compare with this parameter instead of literal
}

func (c comparison) Holds(state flow.State, params flow.Params) bool {
	left := state.Fields[c.field]
	right := c.literal
	if c.param != "" {
		right = params[c.param]
	}
	switch c.kind {
	case flow.Int:
		return compareInts(left.(int), c.op, right.(int))
	case flow.StringList:
		empty := len(left.([]string)) == 0
		return empty == (c.op == "==")
	default:
		return (left == right) == (c.op == "==")
	}
}

func compareInts(left int, op string, right int) bool {
	switch op {
	case "==":
		return left == right
	case "!=":
		return left != right
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	default:
		return left >= right
	}
}

type predicateCondition struct{ holds flow.Predicate }

func (p predicateCondition) Holds(state flow.State, params flow.Params) bool {
	return p.holds(state, params)
}

type anyOf []flow.Condition

func (terms anyOf) Holds(state flow.State, params flow.Params) bool {
	for _, term := range terms {
		if term.Holds(state, params) {
			return true
		}
	}
	return false
}

type allOf []flow.Condition

func (terms allOf) Holds(state flow.State, params flow.Params) bool {
	for _, term := range terms {
		if !term.Holds(state, params) {
			return false
		}
	}
	return true
}

// known lists the names a message offers, sorted, or says there are none.
func known[T any](names map[string]T) string {
	if len(names) == 0 {
		return "none"
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	slices.Sort(list)
	return strings.Join(list, ", ")
}
