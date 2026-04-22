package mexpr

import (
	"math"
	"reflect"
	"strings"
)

// InterpreterOption passes configuration settings when creating a new
// interpreter instance.
type InterpreterOption int

const (
	// StrictMode does extra checks like making sure identifiers exist.
	StrictMode InterpreterOption = iota

	// UnqoutedStrings enables the use of unquoted string values rather than
	// returning nil or a missing identifier error. Identifiers get priority
	// over unquoted strings.
	UnquotedStrings
)

// mapValues returns the values of the map m.
// The values will be in an indeterminate order.
func mapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// checkBounds returns an error if the index is out of bounds.
func checkBounds(ast *Node, input any, idx int) Error {
	if v, ok := input.([]any); ok {
		if idx < 0 || idx >= len(v) {
			return NewError(ast.Offset, ast.Length, "invalid index %d for slice of length %d", int(idx), len(v))
		}
	}
	if v, ok := input.(string); ok {
		return checkStringBounds(ast, stringLength(v), idx)
	}
	return nil
}

func checkStringBounds(ast *Node, length, idx int) Error {
	if idx < 0 || idx >= length {
		return NewError(ast.Offset, ast.Length, "invalid index %d for string of length %d", idx, length)
	}
	return nil
}

// Interpreter executes expression AST programs.
type Interpreter interface {
	Run(value any) (any, Error)
}

// NewInterpreter returns an interpreter for the given AST.
func NewInterpreter(ast *Node, options ...InterpreterOption) Interpreter {
	strict := false
	unquoted := false

	for _, opt := range options {
		switch opt {
		case StrictMode:
			strict = true
		case UnquotedStrings:
			unquoted = true
		}
	}

	return &interpreter{
		ast:      ast,
		strict:   strict,
		unquoted: unquoted,
	}
}

type interpreter struct {
	ast             *Node
	prevFieldSelect bool
	strict          bool
	unquoted        bool
}

func (i *interpreter) Run(value any) (any, Error) {
	return i.run(i.ast, value)
}

func (i *interpreter) run(ast *Node, value any) (any, Error) {
	if ast == nil {
		return nil, nil
	}

	fromSelect := i.prevFieldSelect
	i.prevFieldSelect = false

	switch ast.Type {
	case NodeIdentifier:
		if resolved, ok := resolveLazyValue(value); ok {
			value = resolved
		}
		switch ast.Value.(string) {
		case "@":
			return value, nil
		case "length":
			// Special pseudo-property to get the value's length.
			if s, ok := value.(func() string); ok {
				return stringLength(s()), nil
			}
			if s, ok := value.(string); ok {
				return stringLength(s), nil
			}
			if a, ok := value.([]any); ok {
				return len(a), nil
			}
		case "lower":
			if s, ok := value.(func() string); ok {
				return strings.ToLower(s()), nil
			}
			if s, ok := value.(string); ok {
				return strings.ToLower(s), nil
			}
		case "upper":
			if s, ok := value.(func() string); ok {
				return strings.ToUpper(s()), nil
			}
			if s, ok := value.(string); ok {
				return strings.ToUpper(s), nil
			}
		}
		if m, ok := value.(map[string]any); ok {
			if v, ok := m[ast.Value.(string)]; ok {
				if resolved, ok := resolveLazyValue(v); ok {
					return resolved, nil
				}
				return v, nil
			}
		}
		if m, ok := value.(map[any]any); ok {
			if v, ok := m[ast.Value]; ok {
				if resolved, ok := resolveLazyValue(v); ok {
					return resolved, nil
				}
				return v, nil
			}
		}
		if i.unquoted && !fromSelect {
			// Identifiers not found in the map are treated as strings, but only if
			// the previous item was not a `.` like `obj.field`.
			return ast.Value.(string), nil
		}
		if !i.strict {
			return nil, nil
		}
		return nil, NewError(ast.Offset, ast.Length, "cannot get %v from %v", ast.Value, value)
	case NodeFieldSelect:
		i.prevFieldSelect = true
		leftValue, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		i.prevFieldSelect = true
		return i.run(ast.Right, leftValue)
	case NodeArrayIndex:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		if !isSlice(resultLeft) && !isString(resultLeft) {
			return nil, NewError(ast.Offset, ast.Length, "can only index strings or arrays but got %v", resultLeft)
		}
		if ast.Right != nil && ast.Right.Type == NodeSlice {
			startValue, err := i.run(ast.Right.Left, value)
			if err != nil {
				return nil, err
			}
			endValue, err := i.run(ast.Right.Right, value)
			if err != nil {
				return nil, err
			}
			start, err := toNumber(ast.Right.Left, startValue)
			if err != nil {
				return nil, err
			}
			end, err := toNumber(ast.Right.Right, endValue)
			if err != nil {
				return nil, err
			}
			if left, ok := resultLeft.([]any); ok {
				if start < 0 {
					start += float64(len(left))
				}
				if end < 0 {
					end += float64(len(left))
				}
				if err := checkBounds(ast, left, int(start)); err != nil {
					return nil, err
				}
				if err := checkBounds(ast, left, int(end)); err != nil {
					return nil, err
				}
				if int(start) > int(end) {
					return nil, NewError(ast.Offset, ast.Length, "slice start cannot be greater than end")
				}
				return left[int(start) : int(end)+1], nil
			}
			left := toString(resultLeft)
			leftLen := stringLength(left)
			if start < 0 {
				start += float64(leftLen)
			}
			if end < 0 {
				end += float64(leftLen)
			}
			if err := checkStringBounds(ast, leftLen, int(start)); err != nil {
				return nil, err
			}
			if int(start) > int(end) {
				return nil, NewError(ast.Offset, ast.Length, "string slice start cannot be greater than end")
			}
			if err := checkStringBounds(ast, leftLen, int(end)); err != nil {
				return nil, err
			}
			return stringSlice(left, int(start), int(end)), nil
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		if isSlice(resultRight) && len(resultRight.([]any)) == 2 {
			start, err := toNumber(ast, resultRight.([]any)[0])
			if err != nil {
				return nil, err
			}
			end, err := toNumber(ast, resultRight.([]any)[1])
			if err != nil {
				return nil, err
			}
			if left, ok := resultLeft.([]any); ok {
				if start < 0 {
					start += float64(len(left))
				}
				if end < 0 {
					end += float64(len(left))
				}
				if err := checkBounds(ast, left, int(start)); err != nil {
					return nil, err
				}
				if err := checkBounds(ast, left, int(end)); err != nil {
					return nil, err
				}
				if int(start) > int(end) {
					return nil, NewError(ast.Offset, ast.Length, "slice start cannot be greater than end")
				}
				return left[int(start) : int(end)+1], nil
			}
			left := toString(resultLeft)
			leftLen := stringLength(left)
			if start < 0 {
				start += float64(leftLen)
			}
			if end < 0 {
				end += float64(leftLen)
			}
			if err := checkStringBounds(ast, leftLen, int(start)); err != nil {
				return nil, err
			}
			if int(start) > int(end) {
				return nil, NewError(ast.Offset, ast.Length, "string slice start cannot be greater than end")
			}
			if err := checkStringBounds(ast, leftLen, int(end)); err != nil {
				return nil, err
			}
			return stringSlice(left, int(start), int(end)), nil
		}
		if isNumber(resultRight) {
			idx, err := toNumber(ast, resultRight)
			if err != nil {
				return nil, err
			}
			if left, ok := resultLeft.([]any); ok {
				if idx < 0 {
					idx += float64(len(left))
				}
				if err := checkBounds(ast, left, int(idx)); err != nil {
					return nil, err
				}
				return left[int(idx)], nil
			}
			left := toString(resultLeft)
			leftLen := stringLength(left)
			if idx < 0 {
				idx += float64(leftLen)
			}
			if err := checkStringBounds(ast, leftLen, int(idx)); err != nil {
				return nil, err
			}
			return stringIndex(left, int(idx)), nil
		}
		return nil, NewError(ast.Offset, ast.Length, "array index must be number or slice %v", resultRight)
	case NodeSlice:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		return []any{resultLeft, resultRight}, nil
	case NodeLiteral:
		return ast.Value, nil
	case NodeSign:
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		right, err := toNumber(ast, resultRight)
		if err != nil {
			return nil, err
		}
		if ast.Value.(string) == "-" {
			right = -right
		}
		return right, nil
	case NodeAdd, NodeSubtract, NodeMultiply, NodeDivide, NodeModulus, NodePower:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		if ast.Type == NodeAdd {
			if isString(resultLeft) || isString(resultRight) {
				return toString(resultLeft) + toString(resultRight), nil
			}
			if isSlice(resultLeft) && isSlice(resultRight) {
				tmp := append([]any{}, resultLeft.([]any)...)
				return append(tmp, resultRight.([]any)...), nil
			}
		}
		if isNumber(resultLeft) && isNumber(resultRight) {
			left, err := toNumber(ast.Left, resultLeft)
			if err != nil {
				return nil, err
			}
			right, err := toNumber(ast.Right, resultRight)
			if err != nil {
				return nil, err
			}
			switch ast.Type {
			case NodeAdd:
				return left + right, nil
			case NodeSubtract:
				return left - right, nil
			case NodeMultiply:
				return left * right, nil
			case NodeDivide:
				if right == 0.0 {
					return nil, NewError(ast.Offset, ast.Length, "cannot divide by zero")
				}
				return left / right, nil
			case NodeModulus:
				if int(right) == 0 {
					return nil, NewError(ast.Offset, ast.Length, "cannot divide by zero")
				}
				return int(left) % int(right), nil
			case NodePower:
				return math.Pow(left, right), nil
			}
		}
		return nil, NewError(ast.Offset, ast.Length, "cannot operate on incompatible types %v and %v", resultLeft, resultRight)
	case NodeEqual, NodeNotEqual, NodeLessThan, NodeLessThanEqual, NodeGreaterThan, NodeGreaterThanEqual:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		if ast.Type == NodeEqual {
			return deepEqual(resultLeft, resultRight), nil
		}
		if ast.Type == NodeNotEqual {
			return !deepEqual(resultLeft, resultRight), nil
		}

		left, err := toNumber(ast.Left, resultLeft)
		if err != nil {
			return nil, err
		}
		right, err := toNumber(ast.Right, resultRight)
		if err != nil {
			return nil, err
		}

		switch ast.Type {
		case NodeGreaterThan:
			return left > right, nil
		case NodeGreaterThanEqual:
			return left >= right, nil
		case NodeLessThan:
			return left < right, nil
		case NodeLessThanEqual:
			return left <= right, nil
		}
	case NodeAnd, NodeOr:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		left := toBool(resultLeft)
		switch ast.Type {
		case NodeAnd:
			if !left {
				return false, nil
			}
			resultRight, err := i.run(ast.Right, value)
			if err != nil {
				return nil, err
			}
			return toBool(resultRight), nil
		case NodeOr:
			if left {
				return true, nil
			}
			resultRight, err := i.run(ast.Right, value)
			if err != nil {
				return nil, err
			}
			return toBool(resultRight), nil
		}
	case NodeBefore, NodeAfter:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		leftTime := toTime(resultLeft)
		if leftTime.IsZero() {
			return nil, NewError(ast.Offset, ast.Length, "unable to convert %v to date or time", resultLeft)
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		rightTime := toTime(resultRight)
		if rightTime.IsZero() {
			return nil, NewError(ast.Offset, ast.Length, "unable to convert %v to date or time", resultRight)
		}
		if ast.Type == NodeBefore {
			return leftTime.Before(rightTime), nil
		} else {
			return leftTime.After(rightTime), nil
		}
	case NodeIn, NodeContains, NodeStartsWith, NodeEndsWith:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		switch ast.Type {
		case NodeIn:
			if a, ok := resultRight.([]any); ok {
				for _, item := range a {
					if deepEqual(item, resultLeft) {
						return true, nil
					}
				}
				return false, nil
			}
			if m, ok := resultRight.(map[string]any); ok {
				_, ok := m[toString(resultLeft)]
				return ok, nil
			}
			if m, ok := resultRight.(map[any]any); ok {
				_, ok := m[resultLeft]
				return ok, nil
			}
			return strings.Contains(toString(resultRight), toString(resultLeft)), nil
		case NodeContains:
			if a, ok := resultLeft.([]any); ok {
				for _, item := range a {
					if deepEqual(item, resultRight) {
						return true, nil
					}
				}
				return false, nil
			}
			if m, ok := resultLeft.(map[string]any); ok {
				_, ok := m[toString(resultRight)]
				return ok, nil
			}
			if m, ok := resultLeft.(map[any]any); ok {
				_, ok := m[resultRight]
				return ok, nil
			}
			return strings.Contains(toString(resultLeft), toString(resultRight)), nil
		case NodeStartsWith:
			return strings.HasPrefix(toString(resultLeft), toString(resultRight)), nil
		case NodeEndsWith:
			return strings.HasSuffix(toString(resultLeft), toString(resultRight)), nil
		}
	case NodeNot:
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		right := toBool(resultRight)
		return !right, nil
	case NodeWhere:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		results := []any{}
		if resultLeft == nil {
			return nil, nil
		}
		if m, ok := resultLeft.(map[string]any); ok {
			resultLeft = mapValues(m)
		}
		if m, ok := resultLeft.(map[any]any); ok {
			values := make([]any, 0, len(m))
			for _, v := range m {
				values = append(values, v)
			}
			resultLeft = values
		}
		if leftSlice, ok := resultLeft.([]any); ok {
			for _, item := range leftSlice {
				// In an unquoted string scenario it makes no sense for the first/only
				// token after a `where` clause to be treated as a string. Instead we
				// treat a `where` the same as a field select `.` in this scenario.
				i.prevFieldSelect = true
				resultRight, err := i.run(ast.Right, item)
				if i.strict && err != nil {
					return nil, err
				}
				if toBool(resultRight) {
					results = append(results, item)
				}
			}
		}
		return results, nil
	case NodeFunctionCall:
		funcName := ast.Left.Value.(string)
		var fn any
		switch m := value.(type) {
		case map[string]any:
			fn = m[funcName]
		case map[any]any:
			fn = m[funcName]
		}
		if fn == nil {
			if i.strict {
				return nil, NewError(ast.Offset, ast.Length, "function %s not found", funcName)
			}
			return nil, nil
		}

		fnType := reflect.TypeOf(fn)
		if fnType == nil || fnType.Kind() != reflect.Func {
			return nil, NewError(ast.Offset, ast.Length, "%s is not a function", funcName)
		}
		if fnType.IsVariadic() || fnType.NumOut() != 1 {
			return nil, NewError(ast.Offset, ast.Length, "unsupported function type for %s", funcName)
		}

		params := ast.Value.([]Node)
		if len(params) != fnType.NumIn() {
			return nil, NewError(ast.Offset, ast.Length, "function %s expects %d parameter(s), got %d", funcName, fnType.NumIn(), len(params))
		}

		inputs := make([]reflect.Value, 0, len(params))
		for idx, param := range params {
			paramValue, err := i.run(&param, value)
			if err != nil {
				return nil, err
			}
			input, err := convertFunctionArg(ast, funcName, idx, paramValue, fnType.In(idx))
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, input)
		}

		result := reflect.ValueOf(fn).Call(inputs)[0]
		return result.Interface(), nil
	}
	return nil, nil
}

func convertFunctionArg(ast *Node, funcName string, idx int, value any, target reflect.Type) (reflect.Value, Error) {
	switch target.Kind() {
	case reflect.Bool:
		b, ok := value.(bool)
		if !ok {
			return reflect.Value{}, NewError(ast.Offset, ast.Length, "function %s parameter %d expects bool", funcName, idx+1)
		}
		return reflect.ValueOf(b).Convert(target), nil
	case reflect.String:
		if !isString(value) {
			return reflect.Value{}, NewError(ast.Offset, ast.Length, "function %s parameter %d expects string", funcName, idx+1)
		}
		return reflect.ValueOf(toString(value)).Convert(target), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := toNumber(ast, value)
		if err != nil {
			return reflect.Value{}, NewError(ast.Offset, ast.Length, "function %s parameter %d expects number", funcName, idx+1)
		}
		out := reflect.New(target).Elem()
		out.SetInt(int64(n))
		return out, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := toNumber(ast, value)
		if err != nil || n < 0 {
			return reflect.Value{}, NewError(ast.Offset, ast.Length, "function %s parameter %d expects number", funcName, idx+1)
		}
		out := reflect.New(target).Elem()
		out.SetUint(uint64(n))
		return out, nil
	case reflect.Float32, reflect.Float64:
		n, err := toNumber(ast, value)
		if err != nil {
			return reflect.Value{}, NewError(ast.Offset, ast.Length, "function %s parameter %d expects number", funcName, idx+1)
		}
		out := reflect.New(target).Elem()
		out.SetFloat(n)
		return out, nil
	}

	return reflect.Value{}, NewError(ast.Offset, ast.Length, "unsupported function type for %s", funcName)
}
