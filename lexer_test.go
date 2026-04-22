package mexpr

import "testing"

func TestEscapedStringTokenOffsets(t *testing.T) {
	expr := `"a\"b"`

	l := NewLexer(expr)
	tok, err := l.Next()
	if err != nil {
		t.Fatal(err)
	}

	if tok.Type != TokenString {
		t.Fatalf("expected string token, got %v", tok.Type)
	}
	if tok.Value != `a"b` {
		t.Fatalf("expected decoded string value %q, got %q", `a"b`, tok.Value)
	}
	if tok.Offset != 0 {
		t.Fatalf("expected token offset 0, got %d", tok.Offset)
	}
	if tok.Length != uint8(len(expr)) {
		t.Fatalf("expected token length %d, got %d", len(expr), tok.Length)
	}

	ast, err := Parse(expr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ast == nil {
		t.Fatal("expected ast, got nil")
	}
	if ast.Offset != 0 {
		t.Fatalf("expected node offset 0, got %d", ast.Offset)
	}
	if ast.Length != uint8(len(expr)) {
		t.Fatalf("expected node length %d, got %d", len(expr), ast.Length)
	}
}
