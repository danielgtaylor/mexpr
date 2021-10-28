package mexpr

import "fmt"

func isNumber(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	}
	return false
}

func toNumber(ast *Node, v interface{}) (float64, Error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint8:
		return float64(n), nil
	case uint16:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case float32:
		return float64(n), nil
	}
	return 0, NewError(ast.Offset, "unable to convert to number")
}

func isString(v interface{}) bool {
	switch v.(type) {
	case string, rune, byte, []byte:
		return true
	}
	return false
}

func toString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case rune:
		return string(s)
	case byte:
		return string(s)
	case []byte:
		return string(s)
	}
	return fmt.Sprintf("%v", v)
}

func isSlice(v interface{}) bool {
	if _, ok := v.([]interface{}); ok {
		return true
	}
	return false
}

func toBool(v interface{}) bool {
	switch n := v.(type) {
	case bool:
		return n
	case int:
		return n > 0
	case int8:
		return n > 0
	case int16:
		return n > 0
	case int32:
		return n > 0
	case int64:
		return n > 0
	case uint:
		return n > 0
	case uint8:
		return n > 0
	case uint16:
		return n > 0
	case uint32:
		return n > 0
	case uint64:
		return n > 0
	case float32:
		return n > 0
	case float64:
		return n > 0
	case string:
		return len(n) > 0
	case []byte:
		return len(n) > 0
	case []interface{}:
		return len(n) > 0
	case map[string]interface{}:
		return len(n) > 0
	}
	return false
}
