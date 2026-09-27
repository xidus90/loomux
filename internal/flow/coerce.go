package flow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Coerce turns a value from outside into the Go type its declared Type stands
// for. It is the only table of its kind: TOML hands out int64, a journal read
// with UseNumber hands out json.Number, and JSON hands out []any for a list.
// Written twice, the two copies would drift, and a flow would mean one thing
// when loaded and another when resumed.
func Coerce(t Type, raw any) (Value, error) {
	switch t {
	case String:
		text, ok := raw.(string)
		if !ok {
			return nil, notA(raw, t)
		}
		return text, nil
	case Int:
		return toInt(raw)
	case Bool:
		flag, ok := raw.(bool)
		if !ok {
			return nil, notA(raw, t)
		}
		return flag, nil
	case StringList:
		return toList(raw)
	default:
		return nil, fmt.Errorf("unknown type %q", string(t))
	}
}

func toInt(raw any) (Value, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		return fromFloat(raw, value)
	case json.Number:
		number, err := strconv.ParseInt(value.String(), 10, 64)
		if err == nil {
			return int(number), nil
		}
		// Out of an int64's range is not "not a whole number": 10^20 is one,
		// and calling it a fraction would send the reader looking for a dot.
		if errors.Is(err, strconv.ErrRange) {
			return nil, notA(raw, Int)
		}
		// ParseInt cannot spell 1.0 or 1e3, and both are whole numbers. They
		// take the same test a float64 takes, so one literal cannot read as an
		// int from a journal decoded without UseNumber and as a fraction from
		// the same journal decoded with it.
		asFloat, floatErr := value.Float64()
		if floatErr != nil {
			return nil, notA(raw, Int)
		}
		return fromFloat(raw, asFloat)
	default:
		return nil, notA(raw, Int)
	}
}

// fromFloat is the one whole-number test both shapes of a number go through.
//
// The range is tested first: converting a float outside an int64 is undefined
// in the spec and wraps on amd64, so the whole-number test would fire and call
// 10^19 a fraction -- the very mislabel the ErrRange branch above avoids.
func fromFloat(raw any, value float64) (Value, error) {
	// 2^63 exactly as a float64: every int64 is below it and at or above its
	// negation.
	const beyondInt64 = 1 << 63
	if value < -beyondInt64 || value >= beyondInt64 {
		return nil, notA(raw, Int)
	}
	if value != float64(int64(value)) {
		return nil, notWhole(raw)
	}
	return int(value), nil
}

func toList(raw any) (Value, error) {
	switch value := raw.(type) {
	case []string:
		// A copy: a State is never changed in place, and handing back the
		// caller's own slice would let a write through it reach a stored value.
		list := make([]string, len(value))
		copy(list, value)
		return list, nil
	case []any:
		list := make([]string, 0, len(value))
		for _, item := range value {
			text, err := Coerce(String, item)
			if err != nil {
				return nil, err
			}
			list = append(list, text.(string))
		}
		return list, nil
	default:
		return nil, notA(raw, StringList)
	}
}

func notA(raw any, t Type) error {
	return fmt.Errorf("cannot read %v as %s", raw, string(t))
}

func notWhole(raw any) error {
	return fmt.Errorf("cannot read %v as int; it is not a whole number", raw)
}
