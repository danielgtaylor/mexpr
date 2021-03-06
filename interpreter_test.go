package mexpr

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterpreter(t *testing.T) {
	type test struct {
		expr   string
		input  string
		err    string
		output interface{}
	}
	cases := []test{
		// Add/sub
		{expr: "1 + 2 - 3", output: 0.0},
		{expr: "-1 + +3", output: 2.0},
		{expr: "-1 + -3 - -4", output: 0.0},
		{expr: `0.5 + 0.2`, output: 0.7},
		{expr: `.5 + .2`, output: 0.7},
		{expr: `1_000_000 + 1`, output: 1000001.0},
		// Mul/div
		{expr: "4 * 5 / 10", output: 2.0},
		{expr: `19 % x`, input: `{"x": 5}`, output: 4},
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
		{expr: "x.length == 3", input: `{"x": "abc"}`, output: true},
		{expr: `19 % 5 == 4`, output: true},
		// Boolean comparisons
		{expr: "1 < 2 and 1 > 2", output: false},
		{expr: "1 < 2 and 2 > 1", output: true},
		{expr: "1 < 2 or 1 > 2", output: true},
		{expr: "1 < 2 or 2 > 1", output: true},
		{expr: `1 and "a"`, output: true},
		// Negation
		{expr: "not (1 < 2)", output: false},
		{expr: "not (1 < 2) and (3 < 4)", output: false},
		{expr: "not foo.bar", input: `{"foo": {"bar": true}}`, output: false},
		{expr: "not foo[0].bar", input: `{"foo": [{"bar": true}]}`, output: false},
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
		{expr: "foo[2:]", input: `{"foo": [1, 2, 3]}`, output: []interface{}{3.0}},
		{expr: "foo[:-1]", input: `{"foo": [1, 2, 3]}`, output: []interface{}{1.0, 2.0, 3.0}},
		{expr: "foo[1 + 2 / 2]", input: `{"foo": [1, 2, 3]}`, output: 3.0},
		{expr: "foo[1:1 + 2]", input: `{"foo": [1, 2, 3, 4]}`, output: []interface{}{2.0, 3.0, 4.0}},
		{expr: "foo[foo[0]:bar.baz * 1^2]", input: `{"foo": [1, 2, 3, 4], "bar": {"baz": 3}}`, output: []interface{}{2.0, 3.0, 4.0}},
		{expr: "foo + bar", input: `{"foo": [1, 2], "bar": [3, 4]}`, output: []interface{}{1.0, 2.0, 3.0, 4.0}},
		{expr: "foo[bar]", input: `{"foo": [1, 2, 3], "bar": [0, 1]}`, output: []interface{}{1.0, 2.0}},
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
		{expr: `"foo".length`, output: 3},
		{expr: `str.length`, input: `{"str": "abcdef"}`, output: 6},
		{expr: `arr.length`, input: `{"arr": [1, 2]}`, output: 2},
		// Order of operations
		{expr: "1 + 2 + 3", output: 6.0},
		{expr: "1 + 2 * 3", output: 7.0},
		{expr: "(1 + 2) * 3", output: 9.0},
		{expr: "6 / 3 + 2 * 5", output: 12.0},
		// failure
		{expr: "foo + 1", input: `{}`, err: "no property foo"},
		{expr: "6 -", err: "incomplete expression"},
		{expr: `foo.bar + "baz"`, input: `{"foo": 1}`, err: "no property bar"},
		{expr: `foo + 1`, input: `{"foo": [1, 2]}`, err: "cannot operate on incompatible types"},
		{expr: `foo[1-]`, input: `{"foo": "hello"}`, err: "unexpected right-bracket"},
		{expr: `not (1- <= 5)`, err: "missing right operand"},
		{expr: `(1 >=)`, err: "unexpected right-paren"},
		{expr: `foo[foo[0] != bar]`, input: `{"foo": [1, 2, 3], "bar": true}`, err: "array index must be number or slice"},
		{expr: `1 < "foo"`, err: "unable to convert to number"},
		{expr: `1 <`, err: "incomplete expression"},
		{expr: `1 +`, err: "incomplete expression"},
		{expr: `1 ]`, err: "expected eof but found right-bracket"},
		{expr: `0.5 + 1"`, err: "expected eof but found string"},
		{expr: `0.5 > "some kind of string"`, err: "unable to convert to number"},
		{expr: `foo beginswith "bar"`, input: `{"foo": "bar"}`, err: "expected eof"},
		{expr: `1 / (foo * 1)`, input: `{"foo": 0}`, err: "cannot divide by zero"},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			var input map[string]interface{}
			if tc.input != "" {
				if err := json.Unmarshal([]byte(tc.input), &input); err != nil {
					t.Fatal(err)
				}
			}
			ast, err := Parse(tc.expr, input)

			if tc.err != "" {
				if err != nil {
					if strings.Contains(err.Error(), tc.err) {
						return
					}
					t.Fatal(err.Pretty(tc.expr))
				}
			} else {
				if err != nil {
					t.Fatal(err.Pretty(tc.expr))
				}
			}
			result, err := Run(ast, input)
			if tc.err != "" {
				if err == nil {
					t.Fatal("expected error but found none")
				}
				if strings.Contains(err.Error(), tc.err) {
					return
				}
				t.Fatal(err.Pretty(tc.expr))
			} else {
				if err != nil {
					t.Fatal(err.Pretty(tc.expr))
				}
				assert.Equal(t, tc.output, result)
			}
		})
	}
}

func Benchmark(b *testing.B) {
	benchmarks := []struct {
		name   string
		mexpr  string
		expr   string
		result interface{}
	}{
		{"field", `baz`, `baz`, "value"},
		{"comparison", `foo.bar > 1000`, `foo.bar > 1000`, true},
		{"logical", `1 > 2 or 3 > 4`, `1 > 2 or 3 > 4`, false},
		{"math", `foo.bar + 1`, `foo.bar + 1`, 1000000001.0},
		{"string", `baz startsWith "va"`, `baz startsWith "va"`, true},
		{"index", `arr[1]`, `arr[1]`, 2},
		{
			name:   "complex",
			mexpr:  `foo.bar / (1 * 1024 * 1024) >= 1.0 and "v" in baz and baz.length > 3 and arr[2:].length == 1`,
			expr:   `foo.bar / (1 * 1024 * 1024) >= 1.0 and baz contains "v" and len(baz) > 3 and len(arr[2:]) == 1`,
			result: true,
		},
	}

	var r interface{}
	input := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": 1000000000.0,
		},
		"baz": "value",
		"arr": []interface{}{1, 2, 3},
	}

	for _, bm := range benchmarks {
		b.Run("mexpr-"+bm.name+"-slow", func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				ast, _ := Parse(bm.mexpr, input)
				r, _ = Run(ast, input, StrictMode)
			}
			assert.Equal(b, bm.result, r)
		})

		// b.Run(" expr-"+bm.name+"-slow", func(b *testing.B) {
		// 	b.ReportAllocs()
		// 	for n := 0; n < b.N; n++ {
		// 		r, _ = expr.Eval(bm.expr, input)
		// 	}
		// 	assert.Equal(b, bm.result, r)
		// })
	}

	for _, bm := range benchmarks {
		b.Run("mexpr-"+bm.name+"-cached", func(b *testing.B) {
			b.ReportAllocs()
			ast, err := Parse(bm.mexpr, input)
			assert.NoError(b, err)
			i := NewInterpreter(ast)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				r, _ = i.Run(input)
			}
			assert.Equal(b, bm.result, r)
		})

		// b.Run(" expr-"+bm.name+"-cached", func(b *testing.B) {
		// 	b.ReportAllocs()
		// 	program, err := expr.Compile(bm.expr)
		// 	assert.NoError(b, err)
		// 	b.ResetTimer()
		// 	for n := 0; n < b.N; n++ {
		// 		r, _ = expr.Run(program, input)
		// 	}
		// 	assert.Equal(b, bm.result, r)
		// })
	}
}
