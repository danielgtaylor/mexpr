package mexpr

import (
	"math"
	"strings"
)

// InterpreterOption passes configuration settings when creating a new
// interpreter instance.
type InterpreterOption int

const (
	// StrictMode does extra checks like making sure identifiers exist.
	StrictMode InterpreterOption = iota
)

// Interpreter executes expression AST programs.
type Interpreter interface {
	Run(value any) (any, Error)
}

// NewInterpreter returns an interpreter for the given AST.
func NewInterpreter(ast *Node, options ...InterpreterOption) Interpreter {
	strict := false

	for _, opt := range options {
		if opt == StrictMode {
			strict = true
		}
	}

	return &interpreter{
		ast:    ast,
		strict: strict,
	}
}

type interpreter struct {
	ast    *Node
	strict bool
}

func (i *interpreter) Run(value any) (any, Error) {
	return i.run(i.ast, value)
}

func (i *interpreter) run(ast *Node, value any) (any, Error) {
	switch ast.Type {
	case NodeIdentifier:
		switch ast.Value.(string) {
		case "@":
			return value, nil
		case "length":
			// Special pseudo-property to get the value's length.
			if s, ok := value.(string); ok {
				return len(s), nil
			}
			if a, ok := value.([]any); ok {
				return len(a), nil
			}
		case "lower":
			if s, ok := value.(string); ok {
				return strings.ToLower(s), nil
			}
		case "upper":
			if s, ok := value.(string); ok {
				return strings.ToUpper(s), nil
			}
		}
		if m, ok := value.(map[string]any); ok {
			if !i.strict {
				return m[ast.Value.(string)], nil
			}
			if v, ok := m[ast.Value.(string)]; ok {
				return v, nil
			}
		}
		if m, ok := value.(map[any]any); ok {
			if !i.strict {
				return m[ast.Value], nil
			}
			if v, ok := m[ast.Value]; ok {
				return v, nil
			}
		}
		return nil, NewError(ast.Offset, ast.Length, "cannot get %v from %v", ast.Value, value)
	case NodeFieldSelect:
		leftValue, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		return i.run(ast.Right, leftValue)
	case NodeArrayIndex:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		if !isSlice(resultLeft) && !isString(resultLeft) {
			return nil, NewError(ast.Offset, ast.Length, "can only index strings or arrays but got %v", resultLeft)
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		if isSlice(resultRight) {
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
				return left[int(start) : int(end)+1], nil
			}
			left := toString(resultLeft)
			if start < 0 {
				start += float64(len(left))
			}
			if end < 0 {
				end += float64(len(left))
			}
			return left[int(start) : int(end)+1], nil
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
				return left[int(idx)], nil
			}
			left := toString(resultLeft)
			if idx < 0 {
				idx += float64(len(left))
			}
			return string(left[int(idx)]), nil
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
		ast.Value.([]any)[0] = resultLeft
		ast.Value.([]any)[1] = resultRight
		return ast.Value, nil
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
				return int(left) % int(right), nil
			case NodePower:
				return math.Pow(left, right), nil
			}
		}
		return nil, NewError(ast.Offset, ast.Length, "cannot add incompatible types %v and %v", resultLeft, resultRight)
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
			return normalize(resultLeft) == normalize(resultRight), nil
		}
		if ast.Type == NodeNotEqual {
			return normalize(resultLeft) != normalize(resultRight), nil
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
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		left := toBool(resultLeft)
		right := toBool(resultRight)
		switch ast.Type {
		case NodeAnd:
			return left && right, nil
		case NodeOr:
			return left || right, nil
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
					if item == resultLeft {
						return true, nil
					}
				}
				return false, nil
			}
			if m, ok := resultRight.(map[string]any); ok {
				if m[toString(resultLeft)] != nil {
					return true, nil
				}
				return false, nil
			}
			if m, ok := resultRight.(map[any]any); ok {
				if m[resultLeft] != nil {
					return true, nil
				}
				return false, nil
			}
			return strings.Contains(toString(resultRight), toString(resultLeft)), nil
		case NodeContains:
			if a, ok := resultLeft.([]any); ok {
				for _, item := range a {
					if item == resultRight {
						return true, nil
					}
				}
				return false, nil
			}
			if m, ok := resultLeft.(map[string]any); ok {
				if m[toString(resultRight)] != nil {
					return true, nil
				}
				return false, nil
			}
			if m, ok := resultLeft.(map[any]any); ok {
				if m[resultRight] != nil {
					return true, nil
				}
				return false, nil
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
		for _, item := range resultLeft.([]any) {
			resultRight, _ := i.run(ast.Right, item)
			if i.strict && err != nil {
				return nil, err
			}
			if toBool(resultRight) {
				results = append(results, item)
			}
		}
		return results, nil
	}
	return nil, nil
}
