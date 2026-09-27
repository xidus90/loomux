package expr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/flow"
)

// ParseCap reads max_visits as TOML decoded it: absent, an integer, or a text
// naming an int parameter, optionally plus a non-negative integer. It is the
// only arithmetic a flow file has.
func ParseCap(raw any, params map[string]flow.Type) (flow.Cap, error) {
	switch value := raw.(type) {
	case nil:
		return flow.Cap{Add: 1}, nil
	case int64:
		if value < 1 {
			return flow.Cap{}, fmt.Errorf("max_visits must be at least 1, got %d", value)
		}
		return flow.Cap{Add: int(value)}, nil
	case string:
		return capText(value, params)
	default:
		return flow.Cap{}, fmt.Errorf("max_visits must be an integer or \"<parameter> + <integer>\", got %v", raw)
	}
}

func capText(source string, params map[string]flow.Type) (flow.Cap, error) {
	name, add, hasAdd := strings.Cut(source, "+")
	name = strings.TrimSpace(name)
	limit := flow.Cap{Param: name}
	if hasAdd {
		n, err := strconv.Atoi(strings.TrimSpace(add))
		if err != nil || n < 0 {
			return flow.Cap{}, fmt.Errorf("max_visits %q is not \"<parameter> + <integer>\"", source)
		}
		limit.Add = n
	}
	paramType, ok := params[name]
	if !ok {
		return flow.Cap{}, fmt.Errorf("max_visits %q names no parameter; known parameters: %s", source, known(params))
	}
	if paramType != flow.Int {
		return flow.Cap{}, fmt.Errorf("max_visits parameter %q is %s, not int", name, paramType)
	}
	return limit, nil
}
