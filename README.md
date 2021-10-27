# MicroExpr

A small & fast dependency-free library for parsing micro expressions.

This library was originally built for use in templating languages (e.g. for-loop variable selection, if-statement evaluation) so is minimal in what it supports by design. If you need a more full-featured expression parser, check out [antonmedv/expr](https://github.com/antonmedv/expr) instead.

## Usage

```go
import "github.com/danielgtaylor/mexpr"

// Convenience for lexing/parsing/running in one step:
result, err := mexpr.Eval("a + b", map[string]interface{}{
	"a": 1,
	"b": 2,
})

// Manual method with type checking and fast AST re-use:
l := mexpr.NewLexer("a + b")
typeExamples = map[string]interface{}{
	"a": 1,
	"b": 1,
}
p := mexpr.NewParser(l, typeExamples)
ast, err := p.Parse()
result1, err := mexpr.Run(ast, map[string]interface{}{
	"a": 1,
	"b": 2,
})
result2, err := mexpr.Run(ast, map[string]interfae{}{
	"a": 150,
	"b": 30,
})
```

## Syntax

Literals:

- **strings** double quoted e.g. `"hello"`
- **numbers** e.g. `123`, `2.5`

Internally all numbers are treated as `float64`, which means fewer conversions/casts when taking arbitrary JSON/YAML inputs.

Accessing properties:

```py
foo.bar[0].value
```

Arithmetic operators:

- `+` (addition)
- `-` (subtration)
- `*` (multiplication)
- `/` (division)
- `%` (modulus)
- `^` (power)

```py
(1 + 2) * 3^2
```

Comparison operators:

- `==` (equal)
- `!=` (not equal)
- `<` (less than)
- `>` (greater than)
- `<=` (less than or equal to)
- `>=` (greater than or equal to)

```py
100 >= 42
```

Logical operators:

- `not` (negation)
- `and`
- `or`

```py
1 < 2 and 3 < 4
```

Non-boolean values are converted to booleans. The following result in `true`:

- numbers greater than zero
- non-empty string
- array with at least one item
- map with at least one key/value pair

String operators

- Indexing, e.g. `foo[0]`
- Slicing, e.g. `foo[1:2]`
- `.length` pseudo-property, e.g. `foo.length`
- `+` (concatenation)
- `in` e.g. `"f" in "foo"`
- `startsWith` e.g. `"foo" startsWith "f"`
- `endsWith` e.g. `"foo" endsWith "o"`

Slices indexes are mandatory and _inclusive_. Indexes can be negative, e.g. `foo[-1]` selects the last item in the array.

Any value concatenated with a string will result in a string. For example `"id" + 1` will result in `"id1"`.

Array/slice operators

- Indexing, e.g. `foo[1]`
- Slicing, e.g. `foo[1:2]`
- `.length` pseudo-property, e.g. `foo.length`
- `+` (concatenation)
- `in` (has item), e.g. `1 in foo`

Slices indexes are mandatory and _inclusive_. Indexes can be negative, e.g. `foo[-1]` selects the last item in the array.

Map operators

- `in` (has key), e.g. `"key" in foo`

## Performance

Performance compares favorably to [antonmedv/expr](https://github.com/antonmedv/expr) for both `Eval(...)` and cached program performance, which is expected given the more limited feature set. The example expression used is non-trivial: `foo.bar / 2 * (2 + 4 / 2) == 20 and "v" in baz`.

```
$ go test -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/danielgtaylor/mexpr
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMexpr-12            	  322903	      3592 ns/op	    2576 B/op	      53 allocs/op
BenchmarkMexprCached-12      	 7066062	       166.4 ns/op	      32 B/op	       4 allocs/op
BenchmarkLibExpr-12          	  110338	      9976 ns/op	    8146 B/op	      79 allocs/op
BenchmarkLibExprCached-12    	 2816659	       432.6 ns/op	      96 B/op	       6 allocs/op
```
