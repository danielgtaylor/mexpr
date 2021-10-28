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
	Run(value interface{}) (interface{}, Error)
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

func (i *interpreter) Run(value interface{}) (interface{}, Error) {
	return i.run(i.ast, value)
}

func (i *interpreter) run(ast *Node, value interface{}) (interface{}, Error) {
	switch ast.Type {
	case NodeIdentifier:
		if ast.Value.(string) == "length" {
			// Special pseudo-property to get the value's length.
			if s, ok := value.(string); ok {
				return float64(len(s)), nil
			}
			if a, ok := value.([]interface{}); ok {
				return float64(len(a)), nil
			}
		}
		if m, ok := value.(map[string]interface{}); ok {
			if !i.strict {
				return m[ast.Value.(string)], nil
			}
			if v, ok := m[ast.Value.(string)]; ok {
				return v, nil
			}
		}
		return nil, NewError(ast.Offset, "cannot get %v from %v", ast.Value, value)
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
			return nil, NewError(ast.Offset, "can only index strings or arrays but got %v", resultLeft)
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		if isSlice(resultRight) {
			start, err := toNumber(ast, resultRight.([]interface{})[0])
			if err != nil {
				return nil, err
			}
			end, err := toNumber(ast, resultRight.([]interface{})[1])
			if err != nil {
				return nil, err
			}
			if left, ok := resultLeft.([]interface{}); ok {
				if start < 0 {
					start = float64(len(left) + int(start))
				}
				if end < 0 {
					end = float64(len(left) + int(end))
				}
				return left[int(start) : int(end)+1], nil
			}
			left := toString(resultLeft)
			if start < 0 {
				start = float64(len(left) + int(start))
			}
			if end < 0 {
				end = float64(len(left) + int(end))
			}
			return left[int(start) : int(end)+1], nil
		}
		if isNumber(resultRight) {
			idx, err := toNumber(ast, resultRight)
			if err != nil {
				return nil, err
			}
			if left, ok := resultLeft.([]interface{}); ok {
				if idx < 0 {
					idx = float64(len(left) + int(idx))
				}
				return left[int(idx)], nil
			}
			left := toString(resultLeft)
			if idx < 0 {
				idx = float64(len(left) + int(idx))
			}
			return string(left[int(idx)]), nil
		}
		return nil, NewError(ast.Offset, "array index must be number or slice %v", resultRight)
	case NodeSlice:
		resultLeft, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		resultRight, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		return []interface{}{resultLeft, resultRight}, nil
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
	case NodeAdd, NodeSubtract, NodeMultiply, NodeDivide, NodePower:
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
				tmp := append([]interface{}{}, resultLeft.([]interface{})...)
				return append(tmp, resultRight.([]interface{})...), nil
			}
		}
		if isNumber(resultLeft) && isNumber(resultRight) {
			left, err := toNumber(ast, resultLeft)
			if err != nil {
				return nil, err
			}
			right, err := toNumber(ast, resultRight)
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
				return left / right, nil
			case NodeModulus:
				return float64(int(left) % int(right)), nil
			case NodePower:
				return math.Pow(left, right), nil
			}
		}
		return nil, NewError(ast.Offset, "cannot add incompatible types %v and %v", resultLeft, resultRight)
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
			return resultLeft == resultRight, nil
		}
		if ast.Type == NodeNotEqual {
			return resultLeft != resultRight, nil
		}

		left, err := toNumber(ast, resultLeft)
		if err != nil {
			return nil, err
		}
		right, err := toNumber(ast, resultRight)
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
	case NodeIn, NodeStartsWith, NodeEndsWith:
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
			if a, ok := resultRight.([]interface{}); ok {
				for _, item := range a {
					if item == resultLeft {
						return true, nil
					}
				}
				return false, nil
			}
			if m, ok := resultRight.(map[string]interface{}); ok {
				if m[toString(resultLeft)] != nil {
					return true, nil
				}
				return false, nil
			}
			return strings.Contains(toString(resultRight), toString(resultLeft)), nil
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
	}
	return nil, nil
}
