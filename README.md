# MicroExpr

[![Go Reference](https://pkg.go.dev/badge/github.com/danielgtaylor/mexpr.svg)](https://pkg.go.dev/github.com/danielgtaylor/mexpr) [![Go Report Card](https://goreportcard.com/badge/github.com/danielgtaylor/mexpr)](https://goreportcard.com/report/github.com/danielgtaylor/mexpr)

A small & fast dependency-free library for parsing micro expressions.

This library was originally built for use in templating languages (e.g. for-loop variable selection, if-statement evaluation) so is minimal in what it supports by design. If you need a more full-featured expression parser, check out [antonmedv/expr](https://github.com/antonmedv/expr) instead.

Features:

- Fast, low-allocation parser and runtime
- Simple
  - Easy to learn
  - Easy to read
  - No hiding complex branching logic in expressions
- Intuitive, e.g. `"id" + 1` => `"id1"`
- Useful error messages

## Usage

Try it out on the [Go Playground](https://play.golang.org/p/Z0UcEBgfxu_r)!

```go
import "github.com/danielgtaylor/mexpr"

// Convenience for lexing/parsing/running in one step:
result, err := mexpr.Eval("a + b", map[string]interface{}{
	"a": 1,
	"b": 2,
})

// Manual method with type checking and fast AST re-use. Error handling is
// omitted for brevity.
l := mexpr.NewLexer("a + b")
p := mexpr.NewParser(l)
ast, err := mexpr.Parse()
typeExamples = map[string]interface{}{
	"a": 1,
	"b": 1,
}
err := mexpr.TypeCheck(ast, typeExamples)
interpreter := mexpr.NewInterpreter(ast)
result1, err := interpreter.Run(map[string]interface{}{
	"a": 1,
	"b": 2,
})
result2, err := interpreter.Run(map[string]interfae{}{
	"a": 150,
	"b": 30,
})
```

## Syntax

### Literals

- **strings** double quoted e.g. `"hello"`
- **numbers** e.g. `123`, `2.5`

Internally all numbers are treated as `float64`, which means fewer conversions/casts when taking arbitrary JSON/YAML inputs.

### Accessing properties

```py
foo.bar[0].value
```

### Arithmetic operators

- `+` (addition)
- `-` (subtration)
- `*` (multiplication)
- `/` (division)
- `%` (modulus)
- `^` (power)

```py
(1 + 2) * 3^2
```

Math operations between constants are precomputed when possible, so it is efficient to write meaningful operations like `size <= 4 * 1024 * 1024`. The interpreter will see this as `size <= 4194304`.

### Comparison operators

- `==` (equal)
- `!=` (not equal)
- `<` (less than)
- `>` (greater than)
- `<=` (less than or equal to)
- `>=` (greater than or equal to)

```py
100 >= 42
```

### Logical operators

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

### String operators

- Indexing, e.g. `foo[0]`
- Slicing, e.g. `foo[1:2]`
- `.length` pseudo-property, e.g. `foo.length`
- `+` (concatenation)
- `in` e.g. `"f" in "foo"`
- `startsWith` e.g. `"foo" startsWith "f"`
- `endsWith` e.g. `"foo" endsWith "o"`

Slice indexes are mandatory and _inclusive_. Indexes can be negative, e.g. `foo[-1]` selects the last item in the array.

Any value concatenated with a string will result in a string. For example `"id" + 1` will result in `"id1"`.

There is no distinction between strings, bytes, or runes. Everything is treated as a string.

### Array/slice operators

- Indexing, e.g. `foo[1]`
- Slicing, e.g. `foo[1:2]`
- `.length` pseudo-property, e.g. `foo.length`
- `+` (concatenation)
- `in` (has item), e.g. `1 in foo`

Slice indexes are mandatory and _inclusive_. Indexes can be negative, e.g. `foo[-1]` selects the last item in the array.

### Map operators

- `in` (has key), e.g. `"key" in foo`

## Performance

Performance compares favorably to [antonmedv/expr](https://github.com/antonmedv/expr) for both `Eval(...)` and cached program performance, which is expected given the more limited feature set. The example expression used is non-trivial: `foo.bar / 2 * (2 + 4 / 2) == 20 and "v" in baz`.

```
$ go test -bench=. -benchtime=5s
goos: darwin
goarch: amd64
pkg: github.com/danielgtaylor/mexpr
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMexpr-12            2250564      2641 ns/op    1064 B/op    33 allocs/op
BenchmarkMexprCached-12     47554875     123.5 ns/op      16 B/op     2 allocs/op
BenchmarkLibExpr-12           621049      9300 ns/op    7474 B/op    75 allocs/op
BenchmarkLibExprCached-12   14324178     412.1 ns/op      96 B/op     6 allocs/op
```

## References

These were a big help in understanding how Pratt parsers work:

- https://dev.to/jrop/pratt-parsing
- https://journal.stuffwithstuff.com/2011/03/19/pratt-parsers-expression-parsing-made-easy/
- https://matklad.github.io/2020/04/13/simple-but-powerful-pratt-parsing.html
- https://www.oilshell.org/blog/2017/03/31.html
