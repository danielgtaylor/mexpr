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
	s, ok := getFunctionSchema(v)
	if !ok || len(s.parameters) != 0 {
		return nil, false
	}

	return reflect.ValueOf(v).Call(nil)[0].Interface(), true
}
