package providers

import (
	"encoding/json"
	"strconv"
)

// asInt converts a JSON scalar into a non-negative int.
func asInt(v any) int {
	switch x := v.(type) {
	case int:
		if x < 0 {
			return 0
		}
		return x
	case int64:
		if x < 0 {
			return 0
		}
		return int(x)
	case float64:
		if x < 0 {
			return 0
		}
		return int(x)
	case json.Number:
		n, err := x.Int64()
		if err != nil || n < 0 {
			return 0
		}
		return int(n)
	case string:
		n, err := strconv.Atoi(x)
		if err != nil || n < 0 {
			return 0
		}
		return n
	default:
		return 0
	}
}

// stringify converts common scalar types to a string.
func stringify(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return strconv.Itoa(int(s))
	case int:
		return strconv.Itoa(s)
	default:
		b, _ := json.Marshal(s)
		return string(b)
	}
}

// trim returns at most n bytes from b.
func trim(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}
