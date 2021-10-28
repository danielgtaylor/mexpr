package mexpr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterpreter(t *testing.T) {
	type test struct {
		expr   string
		input  string
		output interface{}
	}
	cases := []test{
		// Add/sub
		{expr: "1 + 2 - 3", output: 0.0},
		{expr: "-1 + +3", output: 2.0},
		{expr: "-1 + -3 - -4", output: 0.0},
		{expr: `0.5 + 0.2"`, output: 0.7},
		{expr: `.5 + .2`, output: 0.7},
		// Mul/div
		{expr: "4 * 5 / 10", output: 2.0},
		// Power
		{expr: "2^3", output: 8.0},
		{expr: "2^3^2", output: 512.0},
		{expr: "16^.5", output: 4.0},
		// Parentheses
		{expr: "((1 + (2)) * 3)", output: 9.0},
		// Comparison
		{expr: "1 < 2", output: true},
		{expr: "1 > 2", output: false},
		{expr: "1 > 1", output: false},
		{expr: "1 >= 1", output: true},
		{expr: "1 < 1", output: false},
		{expr: "1 <= 1", output: true},
		{expr: "1 == 1", output: true},
		{expr: "1 == 2", output: false},
		{expr: "1 != 1", output: false},
		{expr: "1 != 2", output: true},
		// Boolean comparisons
		{expr: "1 < 2 and 1 > 2", output: false},
		{expr: "1 < 2 and 2 > 1", output: true},
		{expr: "1 < 2 or 1 > 2", output: true},
		{expr: "1 < 2 or 2 > 1", output: true},
		{expr: `1 and "a"`, output: true},
		// Negation
		{expr: "not (1 < 2)", output: false},
		{expr: "not (1 < 2) and (3 < 4)", output: false},
		// Strings
		{expr: `"foo" == "foo"`, output: true},
		{expr: `"foo" == "bar"`, output: false},
		{expr: `"foo\"bar"`, output: `foo"bar`},
		{expr: `"foo" + "bar" == "foobar"`, output: true},
		{expr: `foo + "a"`, input: `{"foo": 1}`, output: "1a"},
		{expr: `foo + bar`, input: `{"foo": "id", "bar": 1}`, output: "id1"},
		{expr: `foo[0]`, input: `{"foo": "hello"}`, output: "h"},
		{expr: `foo[-1]`, input: `{"foo": "hello"}`, output: "o"},
		{expr: `foo[0:-3]`, input: `{"foo": "hello"}`, output: "hel"},
		// Identifier / fields
		{expr: "foo", input: `{"foo": 1.0}`, output: 1.0},
		{expr: "foo.bar.baz", input: `{"foo": {"bar": {"baz": 1.0}}}`, output: 1.0},
		{expr: `foo == "foo"`, input: `{"foo": "foo"}`, output: true},
		// Arrays
		{expr: "foo[0]", input: `{"foo": [1, 2]}`, output: 1.0},
		{expr: "foo[-1]", input: `{"foo": [1, 2]}`, output: 2.0},
		{expr: "foo[:1]", input: `{"foo": [1, 2, 3]}`, output: []interface{}{1.0, 2.0}},
		// {expr: "foo[2:]", input: `{"foo": [1, 2, 3]}`, output: []interface{}{3.0}},
		{expr: "foo[1 + 2 / 2]", input: `{"foo": [1, 2, 3]}`, output: 3.0},
		{expr: "foo[1:1 + 2]", input: `{"foo": [1, 2, 3, 4]}`, output: []interface{}{2.0, 3.0, 4.0}},
		{expr: "foo[foo[0]:bar.baz * 1^2]", input: `{"foo": [1, 2, 3, 4], "bar": {"baz": 3}}`, output: []interface{}{2.0, 3.0, 4.0}},
		{expr: "foo + bar", input: `{"foo": [1, 2], "bar": [3, 4]}`, output: []interface{}{1.0, 2.0, 3.0, 4.0}},
		// In
		{expr: `"foo" in "foobar"`, output: true},
		{expr: `"foo" in bar`, input: `{"bar": ["foo", "other"]}`, output: true},
		{expr: `123 in 12345`, output: true},
		{expr: `1 in "best 1"`, output: true},
		{expr: `1 < 2 in "this is true"`, output: true},
		{expr: `1 < 2 in "this is false"`, output: false},
		{expr: `"bar" in foo`, input: `{"foo": {"bar": 1}}`, output: true},
		// Starts / ends with
		{expr: `"foo" startsWith "f"`, output: true},
		{expr: `"foo" startsWith "o"`, output: false},
		{expr: `foo startsWith "f"`, input: `{"foo": "foo"}`, output: true},
		{expr: `name startsWith "/groups/" + group`, input: `{"name": "/groups/foo/bar", "group": "foo"}`, output: true},
		{expr: `"foo" endsWith "f"`, output: false},
		{expr: `"foo" endsWith "o"`, output: true},
		{expr: `"id1" endsWith 1`, output: true},
		// Length
		{expr: `"foo".length`, output: 3.0},
		{expr: `str.length`, input: `{"str": "abcdef"}`, output: 6.0},
		{expr: `arr.length`, input: `{"arr": [1, 2]}`, output: 2.0},
		// Order of operations
		{expr: "1 + 2 + 3", output: 6.0},
		{expr: "1 + 2 * 3", output: 7.0},
		{expr: "(1 + 2) * 3", output: 9.0},
		{expr: "6 / 3 + 2 * 5", output: 12.0},
		// failure
		// {expr: "6 -"},
		// {expr: `foo.bar + "baz"`},
		// {expr: `foo + 1`, input: `{"foo": [1, 2]}`},
		// {expr: `foo[0]`, input: `{"foo": "hello"}`},
		// {expr: "foo + 1"},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			var input map[string]interface{}
			if tc.input != "" {
				if err := json.Unmarshal([]byte(tc.input), &input); err != nil {
					t.Fatal(err)
				}
			}
			result, err := Eval(tc.expr, input)

			if err != nil {
				t.Fatal(err.Pretty(tc.expr))
			}
			assert.Equal(t, tc.output, result)
		})
	}
}

func BenchmarkMexpr(b *testing.B) {
	b.ReportAllocs()
	var r interface{}
	input := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": 10.0,
		},
		"baz": "value",
	}
	for n := 0; n < b.N; n++ {
		r, _ = Eval(`foo.bar / 2 * (2 + 4 / 2) == 20 and "v" in baz`, input)
	}
	assert.Equal(b, true, r)
}

func BenchmarkMexprCached(b *testing.B) {
	b.ReportAllocs()
	var r interface{}
	ast, err := Parse(`foo.bar / 2 * (2 + 4 / 2) == 20 and "v" in baz`, nil)
	assert.NoError(b, err)
	i := NewInterpreter(ast)
	input := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": 10.0,
		},
		"baz": "value",
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r, _ = i.Run(input)
	}
	assert.Equal(b, true, r)
}

// func BenchmarkLibExpr(b *testing.B) {
// 	b.ReportAllocs()
// 	var r interface{}
// 	input := map[string]interface{}{
// 		"foo": map[string]interface{}{
// 			"bar": 10.0,
// 		},
// 		"baz": "value",
// 	}
// 	for n := 0; n < b.N; n++ {
// 		r, _ = expr.Eval(`foo.bar / 2 * (2 + 4 / 2) == 20.0 && baz contains "v"`, input)
// 	}
// 	assert.Equal(b, true, r)
// }

// func BenchmarkLibExprCached(b *testing.B) {
// 	b.ReportAllocs()
// 	var r interface{}
// 	program, err := expr.Compile(`foo.bar / 2 * (2 + 4 / 2) == 20.0 && baz contains "v"`)
// 	assert.NoError(b, err)
// 	input := map[string]interface{}{
// 		"foo": map[string]interface{}{
// 			"bar": 10.0,
// 		},
// 		"baz": "value",
// 	}
// 	b.ResetTimer()
// 	for n := 0; n < b.N; n++ {
// 		r, err = expr.Run(program, input)
// 		assert.NoError(b, err)
// 	}
// 	assert.Equal(b, true, r)
// }
