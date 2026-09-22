package apply

// pyyaml.go stands in for the two PyYAML calls `_advance` makes:
// `yaml.safe_load` (through `parse_frontmatter`) and `yaml.safe_dump(...,
// sort_keys=False, allow_unicode=True, default_flow_style=False)`. yaml.v3
// parses; what PyYAML's SafeLoader makes of a node, and every byte its
// Emitter writes, is ported from PyYAML 6.0.3 (resolver.py, constructor.py,
// representer.py, emitter.py) as far as a safe_load can produce the value.
//
// Refused where PyYAML would go on: an alias to a collection, a date or a
// datetime (PyYAML writes an `&id001` anchor for a shared object), any
// explicit tag, and a merge key `<<`; and, because yaml.v3's scanner refuses
// them first, the escape `\/` and a tab first on a block scalar's first
// line. Each is a row of the parity list. Where PyYAML's scanner is the
// stricter one, on tabs, pyyaml_tabs.go refuses what it refuses.

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type pyKind int

const (
	pyNone pyKind = iota
	pyBool
	pyInt
	pyFloat
	pyStr
	pyDate
	pyDateTime
	pyList
	pyDict
)

// pyValue is what safe_load returns, in the shape the dump needs. A scalar
// carries text, its representation as the representer writes it (for a str
// the string itself), str, Python's `str()` of it, and key, the class of
// values Python's `==` puts it in, so a mapping keeps one key per class.
type pyValue struct {
	kind   pyKind
	text   string
	str    string
	key    string
	items  []*pyValue
	keys   []*pyValue
	values []*pyValue
}

func newStr(s string) *pyValue { return &pyValue{kind: pyStr, text: s, str: s, key: "s:" + s} }

func newInt(n int) *pyValue { return newBigInt(big.NewInt(int64(n))) }

func newBigInt(n *big.Int) *pyValue {
	s := n.String()
	return &pyValue{kind: pyInt, text: s, str: s, key: "n:" + s}
}

func newDict() *pyValue { return &pyValue{kind: pyDict} }

// get is `dict.get(key)` for a str key; nil when it is absent.
func (d *pyValue) get(key string) *pyValue {
	for i, k := range d.keys {
		if k.key == "s:"+key {
			return d.values[i]
		}
	}
	return nil
}

// set is `dict[key] = value` for a str key: in place when present, last
// when not.
func (d *pyValue) set(key string, value *pyValue) {
	d.put(newStr(key), value)
}

// put is `dict[key] = value`: an equal key keeps its place and its first
// spelling, as a Python dict does.
func (d *pyValue) put(key, value *pyValue) {
	for i, k := range d.keys {
		if k.key == key.key {
			d.values[i] = value
			return
		}
	}
	d.keys = append(d.keys, key)
	d.values = append(d.values, value)
}

// pythonStr is `str(value)` for the scalars a `doc_id` can be, with nil as
// Python's None. A list or a mapping answers false: its `str()` is a repr
// that no doc_id of a case can equal.
func pythonStr(value *pyValue) (string, bool) {
	if value == nil {
		return "None", true
	}
	if value.kind == pyList || value.kind == pyDict {
		return "", false
	}
	return value.str, true
}

// nonPrintable is `Reader.NON_PRINTABLE` (reader.py): PyYAML refuses the
// whole stream when it holds one of these characters raw.
var nonPrintable = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`[^\t\n\r\x20-\x7e\x{85}\x{a0}-\x{d7ff}\x{e000}-\x{fffd}\x{10000}-\x{10ffff}]`)
})

// loadFrontmatter is `parse_frontmatter(text, strict=True)` after its match:
// safe_load, a mapping or an error, and every top-level key made a str.
func loadFrontmatter(text string) (*pyValue, error) {
	if loc := nonPrintable().FindStringIndex(text); loc != nil {
		r, _ := utf8.DecodeRuneInString(text[loc[0]:])
		return nil, fmt.Errorf("unacceptable character #x%04x: special characters are not allowed", r)
	}
	decoder := yaml.NewDecoder(strings.NewReader(text))
	var document yaml.Node
	err := decoder.Decode(&document)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	var second yaml.Node
	if next := decoder.Decode(&second); !errors.Is(next, io.EOF) {
		return nil, errors.New("expected a single document in the stream")
	}
	if err := checkTabs(text, &document); err != nil {
		return nil, err
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("frontmatter is not a mapping")
	}
	loaded, err := construct(document.Content[0])
	if err != nil {
		return nil, err
	}
	// `{str(key): value for key, value in loaded.items()}`: the keys become
	// strings, and two that now read alike fall into one.
	meta := newDict()
	for i, key := range loaded.keys {
		meta.put(newStr(key.str), loaded.values[i])
	}
	return meta, nil
}

// construct is SafeConstructor over one yaml.v3 node.
func construct(node *yaml.Node) (*pyValue, error) {
	if node.Style&yaml.TaggedStyle != 0 {
		return nil, fmt.Errorf("explicit tag %s is not supported", node.Tag)
	}
	switch node.Kind {
	case yaml.AliasNode:
		// The anchor came first and was constructed without an error, so
		// the same call cannot fail here.
		value, _ := construct(node.Alias)
		// SafeRepresenter.ignore_aliases: only None, str, bool, int and
		// float are written again in full; anything else shared gets an
		// anchor this dump does not write.
		if value.kind > pyStr {
			return nil, errors.New("an alias to a collection or a timestamp is not supported")
		}
		return value, nil
	case yaml.SequenceNode:
		list := &pyValue{kind: pyList, items: []*pyValue{}}
		for _, child := range node.Content {
			item, err := construct(child)
			if err != nil {
				return nil, err
			}
			list.items = append(list.items, item)
		}
		return list, nil
	case yaml.MappingNode:
		dict := newDict()
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if keyNode.Kind == yaml.ScalarNode && keyNode.Style == 0 && keyNode.Value == "<<" {
				return nil, errors.New("merge keys are not supported")
			}
			key, err := construct(keyNode)
			if err != nil {
				return nil, err
			}
			if key.kind == pyList || key.kind == pyDict {
				return nil, errors.New("while constructing a mapping: found unhashable key")
			}
			value, err := construct(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			dict.put(key, value)
		}
		return dict, nil
	default:
		return constructScalar(node)
	}
}

// constructScalar resolves a scalar the way PyYAML 1.1 does: only a plain
// scalar is resolved, a quoted or block one is always a str.
func constructScalar(node *yaml.Node) (*pyValue, error) {
	if node.Style != 0 {
		return newStr(node.Value), nil
	}
	value := node.Value
	switch implicitTag(value) {
	case "null":
		return &pyValue{kind: pyNone, text: "null", str: "None", key: "none"}, nil
	case "bool":
		if b := strings.ToLower(value); b == "yes" || b == "true" || b == "on" {
			return &pyValue{kind: pyBool, text: "true", str: "True", key: "n:1"}, nil
		}
		return &pyValue{kind: pyBool, text: "false", str: "False", key: "n:0"}, nil
	case "int":
		return constructInt(value)
	case "float":
		return constructFloat(value), nil
	case "timestamp":
		return constructTimestamp(value)
	case "str":
		return newStr(value), nil
	default:
		return nil, fmt.Errorf("could not determine a constructor for the tag 'tag:yaml.org,2002:%s'", implicitTag(value))
	}
}

// The implicit resolvers of resolver.py, each tried only for the first
// characters it is registered for, in the order they are registered.
var (
	boolPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^(?:yes|Yes|YES|no|No|NO|true|True|TRUE|false|False|FALSE|on|On|ON|off|Off|OFF)$`)
	})
	floatPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^(?:[-+]?(?:[0-9][0-9_]*)\.[0-9_]*(?:[eE][-+][0-9]+)?|\.[0-9][0-9_]*(?:[eE][-+][0-9]+)?|[-+]?[0-9][0-9_]*(?::[0-5]?[0-9])+\.[0-9_]*|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)
	})
	intPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^(?:[-+]?0b[0-1_]+|[-+]?0[0-7_]+|[-+]?(?:0|[1-9][0-9_]*)|[-+]?0x[0-9a-fA-F_]+|[-+]?[1-9][0-9_]*(?::[0-5]?[0-9])+)$`)
	})
	nullPattern      = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^(?:~|null|Null|NULL)$`) })
	timestampPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^(?:[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]|[0-9][0-9][0-9][0-9]-[0-9][0-9]?-[0-9][0-9]?(?:[Tt]|[ \t]+)[0-9][0-9]?:[0-9][0-9]:[0-9][0-9](?:\.[0-9]*)?(?:[ \t]*(?:Z|[-+][0-9][0-9]?(?::[0-9][0-9])?))?)$`)
	})
	// timestampParts is `SafeConstructor.timestamp_regexp`.
	timestampParts = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^([0-9][0-9][0-9][0-9])-([0-9][0-9]?)-([0-9][0-9]?)(?:(?:[Tt]|[ \t]+)([0-9][0-9]?):([0-9][0-9]):([0-9][0-9])(?:\.([0-9]*))?(?:[ \t]*(Z|([-+])([0-9][0-9]?)(?::([0-9][0-9]))?))?)?$`)
	})
)

// implicitTag is `Resolver.resolve(ScalarNode, value, (True, False))`,
// without the `tag:yaml.org,2002:` prefix.
func implicitTag(value string) string {
	if value == "" {
		return "null"
	}
	first := value[0]
	switch {
	case strings.IndexByte("yYnNtTfFoO", first) >= 0 && boolPattern().MatchString(value):
		return "bool"
	case strings.IndexByte("-+0123456789.", first) >= 0 && floatPattern().MatchString(value):
		return "float"
	case strings.IndexByte("-+0123456789", first) >= 0 && intPattern().MatchString(value):
		return "int"
	case value == "<<":
		return "merge"
	case strings.IndexByte("~nN", first) >= 0 && nullPattern().MatchString(value):
		return "null"
	case first >= '0' && first <= '9' && timestampPattern().MatchString(value):
		return "timestamp"
	case value == "=":
		return "value"
	}
	return "str"
}

// constructInt is `construct_yaml_int`; an empty number after `0b` or `0x`
// is the ValueError Python's `int(”)` raises.
func constructInt(value string) (*pyValue, error) {
	value = strings.ReplaceAll(value, "_", "")
	sign := 1
	if value[0] == '-' {
		sign = -1
	}
	if value[0] == '+' || value[0] == '-' {
		value = value[1:]
	}
	n := new(big.Int)
	switch {
	case value == "0":
	case strings.HasPrefix(value, "0b"), strings.HasPrefix(value, "0x"):
		base := 2
		if value[1] == 'x' {
			base = 16
		}
		if _, ok := n.SetString(value[2:], base); !ok {
			return nil, fmt.Errorf("invalid literal for int() with base %d: %s", base, pyQuote(value[2:]))
		}
	case value[0] == '0':
		n.SetString(value, 8)
	case strings.Contains(value, ":"):
		parts := strings.Split(value, ":")
		base := big.NewInt(1)
		for i := len(parts) - 1; i >= 0; i-- {
			digit, _ := new(big.Int).SetString(parts[i], 10)
			n.Add(n, digit.Mul(digit, base))
			base.Mul(base, big.NewInt(60))
		}
	default:
		n.SetString(value, 10)
	}
	if sign < 0 {
		n.Neg(n)
	}
	return newBigInt(n), nil
}

// pyQuote is Python's repr of a string of ASCII digits.
func pyQuote(s string) string { return "'" + s + "'" }

// constructFloat is `construct_yaml_float`. What reaches it matched the
// float resolver, so ParseFloat only ever reports a range, and then its
// infinity is Python's.
func constructFloat(value string) *pyValue {
	value = strings.ToLower(strings.ReplaceAll(value, "_", ""))
	sign := 1.0
	if value[0] == '-' {
		sign = -1
	}
	if value[0] == '+' || value[0] == '-' {
		value = value[1:]
	}
	var f float64
	switch {
	case value == ".inf":
		f = sign * math.Inf(1)
	case value == ".nan":
		f = math.NaN()
	case strings.Contains(value, ":"):
		parts := strings.Split(value, ":")
		base := 1.0
		for i := len(parts) - 1; i >= 0; i-- {
			digit, _ := strconv.ParseFloat(parts[i], 64)
			f += digit * base
			base *= 60
		}
		f *= sign
	default:
		parsed, _ := strconv.ParseFloat(value, 64)
		f = sign * parsed
	}
	return newFloat(f)
}

// newFloat builds the value with `represent_float` as text and Python's
// `repr` as str. Every NaN is one key: the constructor hands out one shared
// `nan_value`, and a dict finds a key by identity before equality.
func newFloat(f float64) *pyValue {
	repr := floatRepr(f)
	v := &pyValue{kind: pyFloat, str: repr, key: "f:" + repr}
	switch {
	case math.IsNaN(f):
		v.text, v.key = ".nan", "nan"
	case math.IsInf(f, 1):
		v.text = ".inf"
	case math.IsInf(f, -1):
		v.text = "-.inf"
	default:
		v.text = repr
		if !strings.Contains(repr, ".") && strings.Contains(repr, "e") {
			v.text = strings.Replace(repr, "e", ".0e", 1)
		}
		if f == math.Trunc(f) {
			whole, _ := new(big.Float).SetFloat64(f).Int(nil)
			v.key = "n:" + whole.String()
		}
	}
	return v
}

// floatRepr is Python's `repr(float)`: the shortest digits that read back,
// in exponent form when the decimal exponent is below -4 or at least 16.
func floatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	exponent := strconv.FormatFloat(f, 'e', -1, 64)
	x, _ := strconv.Atoi(exponent[strings.IndexByte(exponent, 'e')+1:])
	if x < -4 || x >= 16 {
		return exponent
	}
	fixed := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(fixed, ".") {
		fixed += ".0"
	}
	return fixed
}

// constructTimestamp is `construct_yaml_timestamp`: a date, or a datetime
// that is naive without a zone and aware with one. A field out of range is
// the ValueError `datetime` raises.
func constructTimestamp(value string) (*pyValue, error) {
	m := timestampParts().FindStringSubmatch(value)
	year, month, day := atoi(m[1]), atoi(m[2]), atoi(m[3])
	if year < 1 || month < 1 || month > 12 || day < 1 || day > time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day() {
		return nil, fmt.Errorf("invalid date %q", value)
	}
	date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	if m[4] == "" {
		return &pyValue{kind: pyDate, text: date, str: date, key: "d:" + date}, nil
	}
	hour, minute, second := atoi(m[4]), atoi(m[5]), atoi(m[6])
	if hour > 23 || minute > 59 || second > 59 {
		return nil, fmt.Errorf("invalid time %q", value)
	}
	micro := 0
	if m[7] != "" {
		micro = atoi((m[7] + "00000")[:6])
	}
	clock := fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)
	if micro != 0 {
		clock += fmt.Sprintf(".%06d", micro)
	}
	text := date + " " + clock
	if m[8] == "" {
		return &pyValue{kind: pyDateTime, text: text, str: text, key: "t:" + text}, nil
	}
	offset := 0
	if m[9] != "" {
		offset = atoi(m[10])*60 + atoi(m[11])
		if offset >= 24*60 {
			return nil, fmt.Errorf("invalid offset %q", value)
		}
		if m[9] == "-" {
			offset = -offset
		}
	}
	sign, magnitude := '+', offset
	if offset < 0 {
		sign, magnitude = '-', -offset
	}
	text += fmt.Sprintf("%c%02d:%02d", sign, magnitude/60, magnitude%60)
	instant := time.Date(year, time.Month(month), day, hour, minute, second, micro*1000, time.FixedZone("", offset*60)).UTC()
	return &pyValue{kind: pyDateTime, text: text, str: text, key: "z:" + instant.Format(time.RFC3339Nano)}, nil
}

// atoi reads a run of ASCII digits a pattern has already matched; an empty
// run is 0, as `int(values['tz_minute'] or 0)`.
func atoi(digits string) int {
	n, _ := strconv.Atoi(digits)
	return n
}
