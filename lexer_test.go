package mexpr

import (
	"strings"
	"testing"
	"unicode/utf8"
)

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
	if tok.Length != uint8(utf8.RuneCountInString(expr)) {
		t.Fatalf("expected token length %d, got %d", utf8.RuneCountInString(expr), tok.Length)
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
	if ast.Length != uint8(utf8.RuneCountInString(expr)) {
		t.Fatalf("expected node length %d, got %d", utf8.RuneCountInString(expr), ast.Length)
	}
}

func TestPrettyErrorUsesRuneOffsets(t *testing.T) {
	_, err := Parse("é +", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	pretty := err.Pretty("é +")
	expected := "incomplete expression, EOF found\né +\n...^"
	if pretty != expected {
		t.Fatalf("expected %q but found %q", expected, pretty)
	}
}

func TestErrorAccessors(t *testing.T) {
	_, err := Parse("1 ]", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Offset() != 2 {
		t.Fatalf("expected offset 2, got %d", err.Offset())
	}
	if err.Length() != 1 {
		t.Fatalf("expected length 1, got %d", err.Length())
	}
}

func TestSingleEqualsUsesRuneOffsets(t *testing.T) {
	_, err := Parse("é = 1", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	pretty := err.Pretty("é = 1")
	expected := "= should be ==\né = 1\n..^"
	if pretty != expected {
		t.Fatalf("expected %q but found %q", expected, pretty)
	}
}

func TestTokenFormatting(t *testing.T) {
	l := NewLexer("foo")
	tok, err := l.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got := tok.String(); got != "0 (identifier) foo" {
		t.Fatalf("expected formatted token, got %q", got)
	}

	if TokenIdentifier.String() != "identifier" {
		t.Fatalf("expected identifier string, got %q", TokenIdentifier.String())
	}
	if TokenWhere.String() != "where" {
		t.Fatalf("expected where string, got %q", TokenWhere.String())
	}
	if TokenUnknown.String() != "unknown" {
		t.Fatalf("expected unknown string, got %q", TokenUnknown.String())
	}
}

func TestNodeFormatting(t *testing.T) {
	ast, err := Parse("a + 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ast == nil {
		t.Fatal("expected ast")
	}
	if got := ast.String(); got != "+" {
		t.Fatalf("expected root string +, got %q", got)
	}
	dot := ast.Dot("")
	if !strings.Contains(dot, `"+" [label="+"]`) {
		t.Fatalf("expected dot to contain root label, got %q", dot)
	}
	if !strings.Contains(dot, `"la" [label="a"]`) {
		t.Fatalf("expected dot to contain identifier child, got %q", dot)
	}
	if !strings.Contains(dot, `"r1" [label="1"]`) {
		t.Fatalf("expected dot to contain literal child, got %q", dot)
	}
}
