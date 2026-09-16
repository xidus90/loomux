package cli

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
)

// brainSubcommands are the verbs of `loomux brain`, in the order cli.py's
// _build_parser adds them: argparse names its choices in that order.
var brainSubcommands = []string{"search", "catalog", "read", "neighbors", "status"}

// brainResultCount is the default of `search -n` (cli.py:483).
const brainResultCount = 5

// brainOption is one option of a brain subcommand. Every option of the five
// takes exactly one value (cli.py:471-501, 451), so the reader knows no other
// arity.
type brainOption struct {
	flag       string
	metavar    string
	fallback   string
	required   bool
	choices    []string
	atLeastOne bool
}

// brainParser is the argument shape of one subcommand: at most one positional
// and the options in the order the reference declares them, which is the order
// argparse reports missing ones in.
type brainParser struct {
	name       string
	positional string
	options    []brainOption
}

// brainArgs is a successful parse: the positional, every option's value by flag
// (fallbacks included) and the count behind -n.
type brainArgs struct {
	positional string
	values     map[string]string
	n          int
}

// brainUsageError is argparse's ArgumentError. top marks the one error the
// reference reports from the top-level parser instead of the subcommand's:
// unrecognized arguments, which parse_args checks after the subparser returned.
type brainUsageError struct {
	top     bool
	message string
}

// brainToken is one classified argument, as _parse_known_args sees it: 'A' for
// a value, 'O' for an option and '-' for the first "--".
type brainToken struct {
	kind        byte
	option      *brainOption
	explicit    string
	hasExplicit bool
}

// brainScan is the state of one parse.
type brainScan struct {
	parser     brainParser
	args       []string
	tokens     []brainToken
	out        brainArgs
	positional bool
	extras     []string
}

// brainTopUsage is the usage line of `loomux brain` itself.
func brainTopUsage() string {
	return "usage: loomux brain {" + strings.Join(brainSubcommands, ",") + "} ..."
}

// brainChoices renders a choice list the way argparse's _check_value does:
// every choice through repr, joined by a comma and a space.
func brainChoices(choices []string) string {
	quoted := make([]string, len(choices))
	for i, choice := range choices {
		quoted[i] = pytext.Repr(choice)
	}
	return strings.Join(quoted, ", ")
}

// brainParserFor answers the argument shape of one subcommand as cli.py:471-501
// and :542 declare it, without --state-dir: loomux reads its state from
// config.StateDir and the legacy directory, never from a flag.
func brainParserFor(name string) (brainParser, bool) {
	scope := brainOption{flag: "--scope", metavar: "SCOPE", fallback: "all"}
	requiredScope := brainOption{flag: "--scope", metavar: "SCOPE", required: true}
	channel := brainOption{
		flag:     "--channel",
		fallback: string(privacy.ChannelLocal),
		choices:  []string{string(privacy.ChannelLocal), string(privacy.ChannelCloud)},
	}
	switch name {
	case "search":
		profile := brainOption{
			flag:     "--profile",
			fallback: string(search.ProfileFast),
			choices:  []string{string(search.ProfileFast), string(search.ProfileFull), string(search.ProfileKeyword)},
		}
		count := brainOption{flag: "-n", metavar: "N", fallback: strconv.Itoa(brainResultCount), atLeastOne: true}
		return brainParser{name: name, positional: "query", options: []brainOption{scope, profile, count, channel}}, true
	case "catalog":
		return brainParser{name: name, options: []brainOption{scope, channel}}, true
	case "read":
		section := brainOption{flag: "--section", metavar: "SECTION"}
		return brainParser{name: name, positional: "relative", options: []brainOption{requiredScope, section, channel}}, true
	case "neighbors":
		return brainParser{name: name, positional: "relative", options: []brainOption{requiredScope, channel}}, true
	case "status":
		return brainParser{name: name, options: []brainOption{channel}}, true
	}
	return brainParser{}, false
}

// usage is the subcommand's usage line: options first, the positional last,
// as argparse formats it, on one line and without -h and --state-dir.
func (p brainParser) usage() string {
	var b strings.Builder
	b.WriteString("usage: loomux brain " + p.name)
	for _, opt := range p.options {
		metavar := opt.metavar
		if opt.choices != nil {
			metavar = "{" + strings.Join(opt.choices, ",") + "}"
		}
		if opt.required {
			fmt.Fprintf(&b, " %s %s", opt.flag, metavar)
		} else {
			fmt.Fprintf(&b, " [%s %s]", opt.flag, metavar)
		}
	}
	if p.positional != "" {
		b.WriteString(" " + p.positional)
	}
	return b.String()
}

func (p brainParser) option(flag string) *brainOption {
	for i := range p.options {
		if p.options[i].flag == flag {
			return &p.options[i]
		}
	}
	return nil
}

// classify is argparse's _parse_optional without abbreviations: a word that is
// not an option of this subcommand but looks like one becomes an unknown
// option, which ends among the unrecognized arguments.
func (p brainParser) classify(arg string) brainToken {
	if arg == "" || arg[0] != '-' {
		return brainToken{kind: 'A'}
	}
	if opt := p.option(arg); opt != nil {
		return brainToken{kind: 'O', option: opt}
	}
	if len(arg) == 1 {
		return brainToken{kind: 'A'}
	}
	if flag, value, found := strings.Cut(arg, "="); found {
		if opt := p.option(flag); opt != nil {
			return brainToken{kind: 'O', option: opt, explicit: value, hasExplicit: true}
		}
	}
	// A single-dash option carries its value glued on: -n3.
	if arg[1] != '-' {
		if opt := p.option(arg[:2]); opt != nil {
			return brainToken{kind: 'O', option: opt, explicit: arg[2:], hasExplicit: true}
		}
	}
	if brainLooksNegative(arg) || strings.Contains(arg, " ") {
		return brainToken{kind: 'A'}
	}
	return brainToken{kind: 'O'}
}

// brainLooksNegative is argparse's _negative_number_matcher `-\.?\d` matched at
// the start of the word. No option of the five looks like a number, so such a
// word is always a value.
func brainLooksNegative(arg string) bool {
	rest := strings.TrimPrefix(arg[1:], ".")
	r, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsDigit(r)
}

// parse reads the arguments behind the subcommand the way argparse's
// _parse_known_args does for this shape: positional and options interleaved,
// errors in the order argparse raises them -- a bad option value while
// scanning, then missing arguments, then unrecognized ones.
func (p brainParser) parse(args []string) (brainArgs, *brainUsageError) {
	s := &brainScan{
		parser: p,
		args:   args,
		tokens: make([]brainToken, len(args)),
		out:    brainArgs{values: map[string]string{}, n: brainResultCount},
	}
	last := -1
	ended := false
	for i, arg := range args {
		switch {
		case ended:
			s.tokens[i] = brainToken{kind: 'A'}
		case arg == "--":
			s.tokens[i] = brainToken{kind: '-'}
			ended = true
		default:
			s.tokens[i] = p.classify(arg)
		}
		if s.tokens[i].kind == 'O' {
			last = i
		}
	}

	start := 0
	for start <= last {
		next := start
		for s.tokens[next].kind != 'O' {
			next++
		}
		if start != next {
			if end := s.positionalAt(start); end > start {
				start = end
				continue
			}
			s.extras = append(s.extras, args[start:next]...)
			start = next
		}
		var err *brainUsageError
		if start, err = s.optionalAt(start); err != nil {
			return brainArgs{}, err
		}
	}
	end := s.positionalAt(start)
	s.extras = append(s.extras, args[end:]...)

	var missing []string
	if p.positional != "" && !s.positional {
		missing = append(missing, p.positional)
	}
	for _, opt := range p.options {
		if _, given := s.out.values[opt.flag]; given {
			continue
		}
		if opt.required {
			missing = append(missing, opt.flag)
			continue
		}
		s.out.values[opt.flag] = opt.fallback
	}
	if len(missing) > 0 {
		return brainArgs{}, &brainUsageError{message: "the following arguments are required: " + strings.Join(missing, ", ")}
	}
	if len(s.extras) > 0 {
		return brainArgs{}, &brainUsageError{top: true, message: "unrecognized arguments: " + strings.Join(s.extras, " ")}
	}
	return s.out, nil
}

// positionalAt is consume_positionals for at most one positional: the pattern
// `-*A-*` from start, the "--" inside it dropped. It answers where the scan
// goes on, which is start when nothing was taken.
func (s *brainScan) positionalAt(start int) int {
	if s.parser.positional == "" || s.positional {
		return start
	}
	i := start
	for i < len(s.tokens) && s.tokens[i].kind == '-' {
		i++
	}
	if i == len(s.tokens) || s.tokens[i].kind != 'A' {
		return start
	}
	s.out.positional = s.args[i]
	s.positional = true
	i++
	for i < len(s.tokens) && s.tokens[i].kind == '-' {
		i++
	}
	return i
}

// optionalAt is consume_optional for an option of one value: glued on, or the
// next word when that word is a value.
func (s *brainScan) optionalAt(start int) (int, *brainUsageError) {
	tok := s.tokens[start]
	if tok.option == nil {
		s.extras = append(s.extras, s.args[start])
		return start + 1, nil
	}
	value, stop := tok.explicit, start+1
	if !tok.hasExplicit {
		if stop == len(s.tokens) || s.tokens[stop].kind != 'A' {
			return 0, &brainUsageError{message: "argument " + tok.option.flag + ": expected one argument"}
		}
		value, stop = s.args[stop], stop+1
	}
	return stop, s.take(tok.option, value)
}

// take is _get_values for one option: the type first, then the choices.
func (s *brainScan) take(opt *brainOption, value string) *brainUsageError {
	if opt.atLeastOne {
		n, err := brainAtLeastOne(value)
		if err != nil {
			return &brainUsageError{message: "argument " + opt.flag + ": " + err.Error()}
		}
		s.out.n = n
		value = strconv.Itoa(n)
	}
	if opt.choices != nil && !slices.Contains(opt.choices, value) {
		return &brainUsageError{message: fmt.Sprintf("argument %s: invalid choice: %s (choose from %s)",
			opt.flag, pytext.Repr(value), brainChoices(opt.choices))}
	}
	s.out.values[opt.flag] = value
	return nil
}

// brainAtLeastOne is cli.py's _at_least_one: int() of the text, refused below
// one. int() is read over ASCII digits only; a count past the machine's int is
// held at the largest one, which no engine returns fewer hits for.
func brainAtLeastOne(text string) (int, error) {
	digits, ok := brainPythonInt(pytext.Strip(text))
	if !ok {
		return 0, fmt.Errorf("invalid _at_least_one value: %s", pytext.Repr(text))
	}
	value, _ := new(big.Int).SetString(digits, 10)
	if value.Sign() < 1 {
		return 0, errors.New("must be at least 1, got " + value.String())
	}
	if !value.IsInt64() || value.Int64() > math.MaxInt {
		return math.MaxInt, nil
	}
	return int(value.Int64()), nil
}

// brainPythonInt checks the grammar int() accepts in base 10 -- a sign, then
// digits with single underscores between them -- and answers the number
// without the underscores.
func brainPythonInt(s string) (string, bool) {
	sign := ""
	if s != "" && (s[0] == '+' || s[0] == '-') {
		sign, s = s[:1], s[1:]
	}
	var digits strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			digits.WriteByte(c)
		case c == '_' && i > 0 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9':
		default:
			return "", false
		}
	}
	if digits.Len() == 0 {
		return "", false
	}
	return sign + digits.String(), true
}
