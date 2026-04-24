package mexpr

import "reflect"

func schemaForScalarType(t reflect.Type) (*schema, bool) {
	switch t.Kind() {
	case reflect.Bool:
		return schemaBool, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return schemaNumber, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return schemaNumber, true
	case reflect.Float32, reflect.Float64:
		return schemaNumber, true
	case reflect.String:
		return schemaString, true
	}
	return nil, false
}

func getFunctionSchema(v any) (*schema, bool) {
	t := reflect.TypeOf(v)
	if t == nil || t.Kind() != reflect.Func || t.IsVariadic() || t.NumOut() != 1 {
		return nil, false
	}

	result, ok := schemaForScalarType(t.Out(0))
	if !ok {
		return nil, false
	}

	s := newSchema(typeFunction)
	s.result = result
	s.parameters = make([]*schema, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		param, ok := schemaForScalarType(t.In(i))
		if !ok {
			return nil, false
		}
		s.parameters[i] = param
	}

	return s, true
}

func resolveLazyValue(v any) (any, bool) {
	switch fn := v.(type) {
	case nil:
		return nil, false
	case bool, string, []byte:
		return nil, false
	case int, int8, int16, int32, int64:
		return nil, false
	case uint, uint8, uint16, uint32, uint64:
		return nil, false
	case float32, float64:
		return nil, false
	case []any, []int, []float64, []string:
		return nil, false
	case map[string]any, map[any]any:
		return nil, false
	case func() bool:
		return fn(), true
	case func() int:
		return fn(), true
	case func() int8:
		return fn(), true
	case func() int16:
		return fn(), true
	case func() int32:
		return fn(), true
	case func() int64:
		return fn(), true
	case func() uint:
		return fn(), true
	case func() uint8:
		return fn(), true
	case func() uint16:
		return fn(), true
	case func() uint32:
		return fn(), true
	case func() uint64:
		return fn(), true
	case func() float32:
		return fn(), true
	case func() float64:
		return fn(), true
	case func() string:
		return fn(), true
	}

	s, ok := getFunctionSchema(v)
	if !ok || len(s.parameters) != 0 {
		return nil, false
	}

	return reflect.ValueOf(v).Call(nil)[0].Interface(), true
}
