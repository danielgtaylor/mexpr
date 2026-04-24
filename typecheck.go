package mexpr

import (
	"fmt"
	"sort"
	"strings"
)

type valueType string

const (
	typeUnknown  valueType = "unknown"
	typeBool     valueType = "boolean"
	typeNumber   valueType = "number"
	typeString   valueType = "string"
	typeArray    valueType = "array"
	typeObject   valueType = "object"
	typeFunction valueType = "function"
)

// mapKeys returns the keys of the map m.
// The keys will be in an indeterminate order.
func mapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

type schema struct {
	typeName   valueType
	items      *schema
	properties map[string]*schema
	parameters []*schema
	result     *schema
}

func (s *schema) String() string {
	if s.isArray() {
		return fmt.Sprintf("%s[%s]", s.typeName, s.items)
	}
	if s.isFunction() {
		params := make([]string, 0, len(s.parameters))
		for _, param := range s.parameters {
			params = append(params, param.String())
		}
		return fmt.Sprintf("%s(%s)->%s", s.typeName, strings.Join(params, ", "), s.result)
	}
	if s.isObject() {
		return fmt.Sprintf("%s{%v}", s.typeName, mapKeys(s.properties))
	}
	return string(s.typeName)
}

func (s *schema) isNumber() bool {
	return s != nil && s.typeName == typeNumber
}

func (s *schema) isString() bool {
	return s != nil && s.typeName == typeString
}

func (s *schema) isArray() bool {
	return s != nil && s.typeName == typeArray
}

func (s *schema) isObject() bool {
	return s != nil && s.typeName == typeObject
}

func (s *schema) isFunction() bool {
	return s != nil && s.typeName == typeFunction
}

var (
	schemaBool   = newSchema(typeBool)
	schemaNumber = newSchema(typeNumber)
	schemaString = newSchema(typeString)
)

func newSchema(t valueType) *schema {
	return &schema{typeName: t}
}

func mergeSchema(a, b *schema) *schema {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.typeName == typeUnknown || b.typeName == typeUnknown {
		return newSchema(typeUnknown)
	}
	if a.typeName != b.typeName {
		return newSchema(typeUnknown)
	}
	switch a.typeName {
	case typeArray:
		return &schema{
			typeName: typeArray,
			items:    mergeSchema(a.items, b.items),
		}
	case typeObject:
		merged := &schema{
			typeName:   typeObject,
			properties: map[string]*schema{},
		}
		for k, v := range a.properties {
			merged.properties[k] = v
		}
		for k, v := range b.properties {
			if existing, ok := merged.properties[k]; ok {
				merged.properties[k] = mergeSchema(existing, v)
				continue
			}
			merged.properties[k] = v
		}
		return merged
	default:
		return a
	}
}

func getSchema(v any) *schema {
	switch i := v.(type) {
	case bool:
		return schemaBool
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return schemaNumber
	case string, []byte:
		return schemaString
	case []any:
		s := newSchema(typeArray)
		for _, item := range i {
			s.items = mergeSchema(s.items, getSchema(item))
		}
		if s.items == nil {
			s.items = newSchema(typeUnknown)
		}
		return s
	case map[string]any:
		m := newSchema(typeObject)
		m.properties = make(map[string]*schema, len(i))
		for k, v := range i {
			m.properties[k] = getSchema(v)
		}
		return m
	case map[any]any:
		m := newSchema(typeObject)
		m.properties = make(map[string]*schema, len(i))
		for k, v := range i {
			m.properties[toString(k)] = getSchema(v)
		}
		return m
	}
	if fn, ok := getFunctionSchema(v); ok {
		if len(fn.parameters) == 0 {
			return fn.result
		}
		return fn
	}
	if isSlice(v) {
		s := newSchema(typeArray)
		iterateSlice(v, func(item any) bool {
			s.items = mergeSchema(s.items, getSchema(item))
			return true
		})
		if s.items == nil {
			s.items = newSchema(typeUnknown)
		}
		return s
	}
	return newSchema(typeUnknown)
}

// TypeChecker checks to ensure types used for operations will work.
type TypeChecker interface {
	Run(value any) Error
}

// NewTypeChecker returns a type checker for the given AST.
func NewTypeChecker(ast *Node, options ...InterpreterOption) TypeChecker {
	_, unquoted := parseInterpreterOptions(options)

	return &typeChecker{
		ast:      ast,
		unquoted: unquoted,
	}
}

type typeChecker struct {
	ast             *Node
	prevFieldSelect bool
	unquoted        bool
}

func (i *typeChecker) Run(value any) Error {
	_, err := i.run(i.ast, value)
	return err
}

func (i *typeChecker) runBoth(ast *Node, value any) (*schema, *schema, Error) {
	leftType, err := i.run(ast.Left, value)
	if err != nil {
		return nil, nil, err
	}
	rightType, err := i.run(ast.Right, value)
	if err != nil {
		return nil, nil, err
	}
	return leftType, rightType, nil
}

func (i *typeChecker) run(ast *Node, value any) (*schema, Error) {
	fromSelect := i.prevFieldSelect
	i.prevFieldSelect = false

	switch ast.Type {
	case NodeIdentifier:
		switch ast.Value.(string) {
		case "@":
			if s, ok := value.(*schema); ok {
				return s, nil
			}
			return getSchema(value), nil
		case "length":
			return schemaNumber, nil
		case "lower", "upper":
			return schemaString, nil
		}
		errValue := value
		if s, ok := value.(*schema); ok {
			if s.typeName == typeUnknown {
				return newSchema(typeUnknown), nil
			}
			if v, ok := s.properties[ast.Value.(string)]; ok {
				return v, nil
			}
			keys := []string{}
			for k := range s.properties {
				keys = append(keys, k)
			}
			errValue = "map with keys [" + strings.Join(keys, ", ") + "]"
		}
		if m, ok := value.(map[string]any); ok {
			if v, ok := m[ast.Value.(string)]; ok {
				return getSchema(v), nil
			}
			keys := []string{}
			for k := range m {
				keys = append(keys, k)
			}
			errValue = "map with keys [" + strings.Join(keys, ", ") + "]"
		}
		if m, ok := value.(map[any]any); ok {
			if v, ok := m[ast.Value]; ok {
				return getSchema(v), nil
			}
			keys := []string{}
			for k := range m {
				keys = append(keys, toString(k))
			}
			errValue = "map with keys [" + strings.Join(keys, ", ") + "]"
		}
		if i.unquoted && !fromSelect {
			// Identifiers not found in the map are treated as strings, but only if
			// the previous item was not a `.` like `obj.field`.
			return schemaString, nil
		}
		return nil, NewError(ast.Offset, ast.Length, "no property %v in %v", ast.Value, errValue)
	case NodeFieldSelect:
		i.prevFieldSelect = true
		leftType, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		i.prevFieldSelect = true
		return i.run(ast.Right, leftType)
	case NodeArrayIndex:
		leftType, rightType, err := i.runBoth(ast, value)
		if err != nil {
			return nil, err
		}
		if leftType.typeName == typeUnknown || rightType.typeName == typeUnknown {
			return newSchema(typeUnknown), nil
		}
		if !(leftType.isString() || leftType.isArray()) {
			return nil, NewError(ast.Offset, ast.Length, "can only index strings or arrays but got %v", leftType)
		}
		if rightType.isArray() {
			// This is a slice!
			return leftType, nil
		}
		if rightType.isNumber() {
			if leftType.isString() {
				return leftType, nil
			}
			return leftType.items, nil
		}
		return nil, NewError(ast.Offset, ast.Length, "array index must be number or slice but found %v", rightType)
	case NodeSlice:
		leftType, rightType, err := i.runBoth(ast, value)
		if err != nil {
			return nil, err
		}
		if leftType.typeName == typeUnknown || rightType.typeName == typeUnknown {
			s := newSchema(typeArray)
			s.items = newSchema(typeUnknown)
			return s, nil
		}
		if !leftType.isNumber() {
			return nil, NewError(ast.Offset, ast.Length, "slice index must be a number but found %s", leftType)
		}
		if !rightType.isNumber() {
			return nil, NewError(ast.Offset, ast.Length, "slice index must be a number but found %s", rightType)
		}
		s := newSchema(typeArray)
		s.items = leftType
		return s, nil
	case NodeLiteral:
		return getSchema(ast.Value), nil
	case NodeSign:
		rightType, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		if !rightType.isNumber() {
			return nil, NewError(ast.Offset, ast.Length, "expected number but found %s", rightType)
		}
		return schemaNumber, nil
	case NodeAdd, NodeSubtract, NodeMultiply, NodeDivide, NodeModulus, NodePower:
		leftType, rightType, err := i.runBoth(ast, value)
		if err != nil {
			return nil, err
		}
		if leftType.typeName == typeUnknown || rightType.typeName == typeUnknown {
			return newSchema(typeUnknown), nil
		}
		if ast.Type == NodeAdd {
			if leftType.isString() || rightType.isString() {
				return schemaString, nil
			}
			if leftType.isArray() && rightType.isArray() {
				if leftType.items.typeName != rightType.items.typeName {
					return nil, NewError(ast.Offset, ast.Length, "array item types don't match: %s vs %s", leftType.items, rightType.items)
				}
				return leftType, nil
			}
		}
		if leftType.isNumber() && rightType.isNumber() {
			return leftType, nil
		}
		return nil, NewError(ast.Offset, ast.Length, "cannot operate on incompatible types %v and %v", leftType.typeName, rightType.typeName)
	case NodeLessThan, NodeLessThanEqual, NodeGreaterThan, NodeGreaterThanEqual:
		leftType, rightType, err := i.runBoth(ast, value)
		if err != nil {
			return nil, err
		}
		if leftType.typeName == typeUnknown || rightType.typeName == typeUnknown {
			return schemaBool, nil
		}
		if !leftType.isNumber() || !rightType.isNumber() {
			return nil, NewError(ast.Offset, ast.Length, "cannot compare %s with %s", leftType, rightType)
		}
		return schemaBool, nil
	case NodeEqual, NodeNotEqual, NodeAnd, NodeOr, NodeIn, NodeContains, NodeStartsWith, NodeEndsWith, NodeBefore, NodeAfter:
		_, _, err := i.runBoth(ast, value)
		if err != nil {
			return nil, err
		}
		return schemaBool, nil
	case NodeWhere:
		leftType, err := i.run(ast.Left, value)
		if err != nil {
			return nil, err
		}
		if leftType.isObject() {
			objectType := leftType
			keys := mapKeys(objectType.properties)
			sort.Strings(keys)
			leftType = newSchema(typeArray)
			if len(keys) > 0 {
				for _, key := range keys {
					leftType.items = mergeSchema(leftType.items, objectType.properties[key])
				}
			}
			if leftType.items == nil {
				leftType.items = newSchema(typeUnknown)
			}
		}
		if leftType.isArray() && leftType.items == nil {
			leftType.items = newSchema(typeUnknown)
		}
		if !leftType.isArray() {
			return nil, NewError(ast.Offset, ast.Length, "where clause requires an array or object, but found %s", leftType)
		}
		// In an unquoted string scenario it makes no sense for the first/only
		// token after a `where` clause to be treated as a string. Instead we
		// treat a `where` the same as a field select `.` in this scenario.
		i.prevFieldSelect = true
		_, err = i.run(ast.Right, leftType.items)
		if err != nil {
			return nil, err
		}
		return leftType, nil
	case NodeNot:
		_, err := i.run(ast.Right, value)
		if err != nil {
			return nil, err
		}
		return schemaBool, nil
	case NodeFunctionCall:
		funcName := ast.Left.Value.(string)
		var fn any
		switch m := value.(type) {
		case map[string]any:
			fn = m[funcName]
		case map[any]any:
			fn = m[funcName]
		default:
			return nil, NewError(ast.Offset, ast.Length, "function %s not found", funcName)
		}
		if fn == nil {
			return nil, NewError(ast.Offset, ast.Length, "function %s not found", funcName)
		}

		fnSchema, ok := getFunctionSchema(fn)
		if !ok {
			return nil, NewError(ast.Offset, ast.Length, "unsupported function type for %s", funcName)
		}

		params := ast.Value.([]Node)
		if len(params) != len(fnSchema.parameters) {
			return nil, NewError(ast.Offset, ast.Length, "function %s expects %d parameter(s), got %d", funcName, len(fnSchema.parameters), len(params))
		}

		for idx, param := range params {
			paramType, err := i.run(&param, value)
			if err != nil {
				return nil, err
			}
			if !compatibleSchemas(fnSchema.parameters[idx], paramType) {
				return nil, NewError(ast.Offset, ast.Length, "function %s parameter %d expects %s but found %s", funcName, idx+1, fnSchema.parameters[idx], paramType)
			}
		}

		return fnSchema.result, nil
	}
	return nil, NewError(ast.Offset, ast.Length, "unexpected node %v", ast)
}

func compatibleSchemas(expected, actual *schema) bool {
	if expected == nil || actual == nil {
		return false
	}
	if expected.typeName == actual.typeName {
		return true
	}
	return expected.isNumber() && actual.isNumber()
}
