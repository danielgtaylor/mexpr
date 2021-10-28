package mexpr

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// TokenType defines the type of token produced by the lexer.
type TokenType string

// Token
const (
	TokenUnknown       TokenType = ""
	TokenIdentifier    TokenType = "identifier"
	TokenDot           TokenType = "dot"
	TokenNumber        TokenType = "number"
	TokenString        TokenType = "string"
	TokenLeftParen     TokenType = "left-paren"
	TokenRightParen    TokenType = "right-paren"
	TokenLeftBracket   TokenType = "left-bracket"
	TokenRightBracket  TokenType = "right-bracket"
	TokenSlice         TokenType = "slice"
	TokenAddSub        TokenType = "add-sub"
	TokenMulDiv        TokenType = "mul-div"
	TokenPower         TokenType = "power"
	TokenComparison    TokenType = "comparison"
	TokenAnd           TokenType = "and"
	TokenOr            TokenType = "or"
	TokenNot           TokenType = "not"
	TokenStringCompare TokenType = "in"
	TokenEOF           TokenType = "eof"
)

var basic = map[rune]TokenType{
	'.': TokenDot,
	'(': TokenLeftParen,
	')': TokenRightParen,
	'[': TokenLeftBracket,
	']': TokenRightBracket,
	':': TokenSlice,
	'+': TokenAddSub,
	'-': TokenAddSub,
	'*': TokenMulDiv,
	'/': TokenMulDiv,
	'%': TokenMulDiv,
	'^': TokenPower,
}

// Token describes a single token produced by the lexer.
type Token struct {
	Type   TokenType
	Value  string
	Offset int
}

func (t *Token) String() string {
	return fmt.Sprintf("%d (%s) %s", t.Offset, t.Type, t.Value)
}

// Lexer returns tokens from an input expression.
type Lexer interface {
	// Next returns the next token from the expression. The returned token may
	// be changed in-place on subsequent calls and should not be stored.
	Next() (*Token, Error)
}

// NewLexer creates a new lexer for the given expression.
func NewLexer(expression string) Lexer {
	return &lexer{
		expression: expression,
		pos:        0,
		lastWidth:  0,
		token:      &Token{},
	}
}

type lexer struct {
	expression string
	pos        int
	lastWidth  int

	// token is a cached token to prevent new tokens from being allocated.
	// It is re-used on each call to `Next()`.
	token *Token
}

// next returns the next rune in the expression at the current position.
func (l *lexer) next() rune {
	if l.pos >= len(l.expression) {
		l.lastWidth = 0
		return -1
	}
	r, w := utf8.DecodeRuneInString(l.expression[l.pos:])
	l.pos += w
	l.lastWidth = w
	return r
}

// back moves back one rune.
func (l *lexer) back() {
	l.pos -= l.lastWidth
}

// peek returns the next rune without moving the position forward.
func (l *lexer) peek() rune {
	r := l.next()
	l.back()
	return r
}

func (l *lexer) newToken(typ TokenType, value string) *Token {
	l.token.Type = typ
	l.token.Value = value
	l.token.Offset = l.pos - len(value)
	return l.token
}

// consumeNumber reads runes from the expression until a non-number or
// non-decimal is encountered.
func (l *lexer) consumeNumber() *Token {
	start := l.pos - l.lastWidth
	for {
		r := l.next()
		if r != '.' && (r < '0' || r > '9') {
			l.back()
			break
		}
	}
	return l.newToken(TokenNumber, l.expression[start:l.pos])
}

// consumeIdentifier reads runes from the expression until a non-identifier
// character is encountered. If the identifier is a known operator like `in`
// then that corresponding token is returned, otherwise a normal identifier.
func (l *lexer) consumeIdentifier() *Token {
	start := l.pos - l.lastWidth
	for {
		r := l.next()
		if r == -1 || basic[r] != TokenUnknown || r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '<' || r == '>' || r == '=' || r == '!' || r == '.' || r == '[' || r == '(' {
			l.back()
			break
		}
	}
	value := l.expression[start:l.pos]
	switch string(value) {
	case "and":
		return l.newToken(TokenAnd, value)
	case "or":
		return l.newToken(TokenOr, value)
	case "not":
		return l.newToken(TokenNot, value)
	case "in", "startsWith", "endsWith":
		return l.newToken(TokenStringCompare, value)
	}
	return l.newToken(TokenIdentifier, value)
}

// consumeString reads runes from the expression until a non-escaped double
// quote is encountered. Only double-quoted strings are supported.
func (l *lexer) consumeString() *Token {
	buf := bytes.NewBuffer(make([]byte, 0, 8))
	for {
		r := l.next()
		if r == '\\' && l.peek() == '"' {
			l.next()
			buf.WriteRune('"')
			continue
		}
		if r == -1 || r == '"' {
			break
		}
		buf.WriteRune(r)
	}
	return l.newToken(TokenString, buf.String())
}

func (l *lexer) Next() (*Token, Error) {
	r := l.next()
	for r == ' ' || r == '\t' || r == '\r' || r == '\n' {
		r = l.next()
	}
	switch {
	case r == -1:
		return l.newToken(TokenEOF, ""), nil
	case basic[r] != TokenUnknown:
		if r == '.' {
			n := l.peek()
			if n >= '0' && n <= '9' {
				return l.consumeNumber(), nil
			}
		}
		return l.newToken(basic[r], l.expression[l.pos-l.lastWidth:l.pos]), nil
	case r >= '0' && r <= '9':
		return l.consumeNumber(), nil
	case r == '<', r == '>', r == '!':
		eq := l.next()
		if eq == '=' {
			return l.newToken(TokenComparison, string(r)+"="), nil
		}
		l.back()
		return l.newToken(TokenComparison, string(r)), nil
	case r == '=':
		if l.peek() == '=' {
			l.next()
			return l.newToken(TokenComparison, "=="), nil
		}
		return nil, NewError(l.pos, "= should be ==")
	case r == '"':
		return l.consumeString(), nil
	}

	return l.consumeIdentifier(), nil
}
