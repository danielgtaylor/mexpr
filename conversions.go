package mexpr

import (
	"fmt"
	"reflect"
	"time"
	"unicode/utf8"
)

func isNumber(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case func() int, func() int8, func() int16, func() int32, func() int64:
		return true
	case func() uint, func() uint8, func() uint16, func() uint32, func() uint64:
		return true
	case func() float32, func() float64:
		return true
	}
	return false
}

func toNumber(ast *Node, v any) (float64, Error) {
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
	case func() int:
		return float64(n()), nil
	case func() int8:
		return float64(n()), nil
	case func() int16:
		return float64(n()), nil
	case func() int32:
		return float64(n()), nil
	case func() int64:
		return float64(n()), nil
	case func() uint:
		return float64(n()), nil
	case func() uint8:
		return float64(n()), nil
	case func() uint16:
		return float64(n()), nil
	case func() uint32:
		return float64(n()), nil
	case func() uint64:
		return float64(n()), nil
	case func() float32:
		return float64(n()), nil
	case func() float64:
		return n(), nil
	}
	return 0, NewError(ast.Offset, ast.Length, "unable to convert to number: %v", v)
}

func isString(v any) bool {
	switch v.(type) {
	case string, rune, byte, []byte:
		return true
	case func() string:
		return true
	}
	return false
}

func toString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case rune:
		return string(s)
	case byte:
		return string(s)
	case []byte:
		return string(s)
	case func() string:
		return s()
	}
	return fmt.Sprintf("%v", v)
}

func stringLength(v string) int {
	return utf8.RuneCountInString(v)
}

func runeRangeToByteOffsets(v string, startIdx, endIdx int) (int, int) {
	if startIdx <= 0 {
		startIdx = 0
	}
	if endIdx < startIdx {
		endIdx = startIdx
	}

	offset := 0
	for i := 0; i < startIdx && offset < len(v); i++ {
		_, size := utf8.DecodeRuneInString(v[offset:])
		offset += size
	}

	startOffset := offset
	for i := startIdx; i < endIdx && offset < len(v); i++ {
		_, size := utf8.DecodeRuneInString(v[offset:])
		offset += size
	}

	return startOffset, offset
}

func stringIndex(v string, idx int) string {
	start, end := runeRangeToByteOffsets(v, idx, idx+1)
	return v[start:end]
}

func stringSlice(v string, start, end int) string {
	from, to := runeRangeToByteOffsets(v, start, end+1)
	return v[from:to]
}

func isByteArrayOrSlice(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if t.Kind() != reflect.Array && t.Kind() != reflect.Slice {
		return false
	}
	return t.Elem().Kind() == reflect.Uint8
}

func sliceLen(v any) (int, bool) {
	switch s := v.(type) {
	case []any:
		return len(s), true
	case []int:
		return len(s), true
	case []float64:
		return len(s), true
	case []string:
		return len(s), true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || isByteArrayOrSlice(rv.Type()) {
		return 0, false
	}
	if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
		return 0, false
	}
	return rv.Len(), true
}

func isSlice(v any) bool {
	_, ok := sliceLen(v)
	return ok
}

func sliceItem(v any, idx int) (any, bool) {
	switch s := v.(type) {
	case []any:
		return s[idx], true
	case []int:
		return s[idx], true
	case []float64:
		return s[idx], true
	case []string:
		return s[idx], true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || isByteArrayOrSlice(rv.Type()) {
		return nil, false
	}
	if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
		return nil, false
	}
	return rv.Index(idx).Interface(), true
}

func sliceRange(v any, start, end int) (any, bool) {
	switch s := v.(type) {
	case []any:
		return s[start : end+1], true
	case []int:
		return s[start : end+1], true
	case []float64:
		return s[start : end+1], true
	case []string:
		return s[start : end+1], true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || isByteArrayOrSlice(rv.Type()) {
		return nil, false
	}
	if rv.Kind() == reflect.Array {
		copyValue := reflect.New(rv.Type()).Elem()
		copyValue.Set(rv)
		return copyValue.Slice(start, end+1).Interface(), true
	}
	if rv.Kind() == reflect.Slice {
		return rv.Slice(start, end+1).Interface(), true
	}
	return nil, false
}

func appendSliceItems(dst []any, v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return append(dst, s...), true
	case []int:
		for _, item := range s {
			dst = append(dst, item)
		}
		return dst, true
	case []float64:
		for _, item := range s {
			dst = append(dst, item)
		}
		return dst, true
	case []string:
		for _, item := range s {
			dst = append(dst, item)
		}
		return dst, true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || isByteArrayOrSlice(rv.Type()) {
		return nil, false
	}
	if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
		return nil, false
	}
	for idx := 0; idx < rv.Len(); idx++ {
		dst = append(dst, rv.Index(idx).Interface())
	}
	return dst, true
}

func concatSlices(left, right any) (any, bool) {
	switch l := left.(type) {
	case []any:
		if r, ok := right.([]any); ok {
			out := make([]any, 0, len(l)+len(r))
			out = append(out, l...)
			out = append(out, r...)
			return out, true
		}
	case []int:
		if r, ok := right.([]int); ok {
			out := make([]int, 0, len(l)+len(r))
			out = append(out, l...)
			out = append(out, r...)
			return out, true
		}
	case []float64:
		if r, ok := right.([]float64); ok {
			out := make([]float64, 0, len(l)+len(r))
			out = append(out, l...)
			out = append(out, r...)
			return out, true
		}
	case []string:
		if r, ok := right.([]string); ok {
			out := make([]string, 0, len(l)+len(r))
			out = append(out, l...)
			out = append(out, r...)
			return out, true
		}
	}

	leftLen, ok := sliceLen(left)
	if !ok {
		return nil, false
	}
	rightLen, ok := sliceLen(right)
	if !ok {
		return nil, false
	}
	out := make([]any, 0, leftLen+rightLen)
	out, ok = appendSliceItems(out, left)
	if !ok {
		return nil, false
	}
	return appendSliceItems(out, right)
}

func iterateSlice(v any, yield func(any) bool) bool {
	switch s := v.(type) {
	case []any:
		for _, item := range s {
			if !yield(item) {
				return true
			}
		}
		return true
	case []int:
		for _, item := range s {
			if !yield(item) {
				return true
			}
		}
		return true
	case []float64:
		for _, item := range s {
			if !yield(item) {
				return true
			}
		}
		return true
	case []string:
		for _, item := range s {
			if !yield(item) {
				return true
			}
		}
		return true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || isByteArrayOrSlice(rv.Type()) {
		return false
	}
	if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
		return false
	}
	for idx := 0; idx < rv.Len(); idx++ {
		if !yield(rv.Index(idx).Interface()) {
			return true
		}
	}
	return true
}

func recursiveDeepEqual(left, right any) bool {
	l := normalize(left)
	r := normalize(right)

	switch lv := l.(type) {
	case float64:
		rv, ok := r.(float64)
		return ok && lv == rv
	case string:
		rv, ok := r.(string)
		return ok && lv == rv
	case bool:
		rv, ok := r.(bool)
		return ok && lv == rv
	}

	if lLen, ok := sliceLen(l); ok {
		rLen, ok := sliceLen(r)
		if !ok || lLen != rLen {
			return false
		}
		for idx := 0; idx < lLen; idx++ {
			leftItem, _ := sliceItem(l, idx)
			rightItem, _ := sliceItem(r, idx)
			if !recursiveDeepEqual(leftItem, rightItem) {
				return false
			}
		}
		return true
	}

	switch lv := l.(type) {
	case map[string]any:
		rv, ok := r.(map[string]any)
		if !ok || len(lv) != len(rv) {
			return false
		}
		for key, leftValue := range lv {
			rightValue, ok := rv[key]
			if !ok || !recursiveDeepEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	case map[any]any:
		rv, ok := r.(map[any]any)
		if !ok || len(lv) != len(rv) {
			return false
		}
		for key, leftValue := range lv {
			rightValue, ok := rv[key]
			if !ok || !recursiveDeepEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	}

	return reflect.DeepEqual(l, r)
}

// toTime converts a string value into a time.Time if possible, otherwise
// returns a zero time.
func toTime(v any) time.Time {
	vStr := toString(v)
	if t, err := time.Parse(time.RFC3339, vStr); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05", vStr); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", vStr); err == nil {
		return t
	}
	return time.Time{}
}

func toBool(v any) bool {
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
	case map[string]any:
		return len(n) > 0
	case map[any]any:
		return len(n) > 0
	}
	if l, ok := sliceLen(v); ok {
		return l > 0
	}
	return false
}

// normalize an input for equality checks. All numbers -> float64, []byte to
// string, etc. Since `rune` is an alias for int32, we can't differentiate it
// for comparison with strings.
func normalize(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case []byte:
		return string(n)
	case func() int:
		return float64(n())
	case func() int8:
		return float64(n())
	case func() int16:
		return float64(n())
	case func() int32:
		return float64(n())
	case func() int64:
		return float64(n())
	case func() uint:
		return float64(n())
	case func() uint8:
		return float64(n())
	case func() uint16:
		return float64(n())
	case func() uint32:
		return float64(n())
	case func() uint64:
		return float64(n())
	case func() float32:
		return float64(n())
	case func() float64:
		return n()
	case func() string:
		return n()
	case func() bool:
		return n()
	}

	return v
}

// deepEqual returns whether two values are deeply equal.
func deepEqual(left, right any) bool {
	return recursiveDeepEqual(left, right)
}
