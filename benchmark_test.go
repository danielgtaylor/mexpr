package mexpr

import "testing"

func BenchmarkInternals(b *testing.B) {
	b.Run("lexer-complex", func(b *testing.B) {
		b.ReportAllocs()
		expression := `foo.bar / (1 * 1024 * 1024) >= 1.0 and "v" in baz and baz.length > 3 and arr[2:].length == 1`
		for n := 0; n < b.N; n++ {
			l := lexer{expression: expression}
			for {
				tok, err := l.Next()
				if err != nil {
					b.Fatal(err)
				}
				if tok.Type == TokenEOF {
					break
				}
			}
		}
	})

	b.Run("resolve-lazy-value-non-function", func(b *testing.B) {
		b.ReportAllocs()
		input := map[string]any{"foo": "bar"}
		for n := 0; n < b.N; n++ {
			if _, ok := resolveLazyValue(input); ok {
				b.Fatal("unexpected lazy value")
			}
		}
	})

	b.Run("resolve-lazy-value-number-func", func(b *testing.B) {
		b.ReportAllocs()
		input := func() int { return 42 }
		for n := 0; n < b.N; n++ {
			out, ok := resolveLazyValue(input)
			if !ok || out.(int) != 42 {
				b.Fatalf("unexpected lazy value result: %v %v", out, ok)
			}
		}
	})

	b.Run("deep-equal-number", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			if !deepEqual(1, 1.0) {
				b.Fatal("expected equal numbers")
			}
		}
	})
}
