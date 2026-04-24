package mexpr

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// TokenType defines the type of token produced by the lexer.
type TokenType uint8

// Token
const (
	TokenUnknown TokenType = iota
	TokenIdentifier
	TokenDot
	TokenNumber
	TokenString
	TokenLeftParen
	TokenRightParen
	TokenLeftBracket
	TokenRightBracket
	TokenSlice
	TokenAddSub
	TokenMulDiv
	TokenPower
	TokenComparison
	TokenAnd
	TokenOr
	TokenNot
	TokenStringCompare
	TokenWhere
	TokenEOF
	TokenComma
)

func (t TokenType) String() string {
	switch t {
	case TokenIdentifier:
		return "identifier"
	case TokenDot:
		return "dot"
	case TokenNumber:
		return "number"
	case TokenString:
		return "string"
	case TokenLeftParen:
		return "left-paren"
	case TokenRightParen:
		return "right-paren"
	case TokenLeftBracket:
		return "left-bracket"
	case TokenRightBracket:
		return "right-bracket"
	case TokenSlice:
		return "slice"
	case TokenAddSub:
		return "add-sub"
	case TokenMulDiv:
		return "mul-div"
	case TokenPower:
		return "power"
	case TokenComparison:
		return "comparison"
	case TokenAnd:
		return "and"
	case TokenOr:
		return "or"
	case TokenNot:
		return "not"
	case TokenStringCompare:
		return "string-compare"
	case TokenWhere:
		return "where"
	case TokenEOF:
		return "eof"
	case TokenComma:
		return "comma"
	}
	return "unknown"
}

func basic(input rune) TokenType {
	switch input {
	case '.':
		return TokenDot
	case '(':
		return TokenLeftParen
	case ')':
		return TokenRightParen
	case '[':
		return TokenLeftBracket
	case ']':
		return TokenRightBracket
	case ':':
		return TokenSlice
	case '+', '-':
		return TokenAddSub
	case '*', '/', '%':
		return TokenMulDiv
	case '^':
		return TokenPower
	case ',':
		return TokenComma
	}

	return TokenUnknown
}

// Token describes a single token produced by the lexer.
type Token struct {
	Type   TokenType
	Length uint8
	Offset uint16
	Value  string
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
		runePos:    0,
		lastWidth:  0,
	}
}

type lexer struct {
	expression string
	pos        uint16
	runePos    uint16
	lastWidth  uint16

	// token is a cached token to prevent new tokens from being allocated.
	// It is re-used on each call to `Next()`.
	token Token
}

// next returns the next rune in the expression at the current position.
func (l *lexer) next() rune {
	if l.pos >= uint16(len(l.expression)) {
		l.lastWidth = 0
		return -1
	}
	r, w := utf8.DecodeRuneInString(l.expression[l.pos:])
	l.pos += uint16(w)
	l.runePos++
	l.lastWidth = uint16(w)
	return r
}

// back moves back one rune.
func (l *lexer) back() {
	l.pos -= l.lastWidth
	if l.lastWidth > 0 {
		l.runePos--
	}
}

// peek returns the next rune without moving the position forward.
func (l *lexer) peek() rune {
	r := l.next()
	l.back()
	return r
}

func (l *lexer) newToken(typ TokenType, value string, offset, length uint16) *Token {
	l.token.Type = typ
	l.token.Value = value
	l.token.Offset = offset
	l.token.Length = uint8(length)
	if l.token.Length == 0 {
		l.token.Length = 1
	}
	return &l.token
}

// consumeNumber reads runes from the expression until a non-number or
// non-decimal is encountered.
func (l *lexer) consumeNumber() *Token {
	start := l.pos - l.lastWidth
	offset := l.runePos - 1
	for {
		r := l.next()
		if r != '.' && r != '_' && (r < '0' || r > '9') {
			l.back()
			break
		}
	}
	return l.newToken(TokenNumber, l.expression[start:l.pos], offset, l.runePos-offset)
}

// consumeIdentifier reads runes from the expression until a non-identifier
// character is encountered. If the identifier is a known operator like `in`
// then that corresponding token is returned, otherwise a normal identifier.
func (l *lexer) consumeIdentifier() *Token {
	start := l.pos - l.lastWidth
	offset := l.runePos - 1
	for {
		r := l.next()
		if r == -1 || basic(r) != TokenUnknown || r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '<' || r == '>' || r == '=' || r == '!' || r == '.' || r == '[' || r == '(' {
			l.back()
			break
		}
	}
	value := l.expression[start:l.pos]
	if l.token.Type != TokenDot {
		// Only parse special identifiers if the last token type was *not* an object
		// property selector, e.g. `foo.in.not` vs `foo in ...`. This enables
		// keywords to be used as properties without issue.
		switch string(value) {
		case "and":
			return l.newToken(TokenAnd, value, offset, l.runePos-offset)
		case "or":
			return l.newToken(TokenOr, value, offset, l.runePos-offset)
		case "not":
			return l.newToken(TokenNot, value, offset, l.runePos-offset)
		case "in", "contains", "startsWith", "endsWith", "before", "after":
			return l.newToken(TokenStringCompare, value, offset, l.runePos-offset)
		case "where":
			return l.newToken(TokenWhere, value, offset, l.runePos-offset)
		}
	}
	return l.newToken(TokenIdentifier, value, offset, l.runePos-offset)
}

// consumeString reads runes from the expression until a non-escaped double
// quote is encountered. Only double-quoted strings are supported.
func (l *lexer) consumeString() (*Token, Error) {
	offset := l.runePos - 1
	start := l.pos

	for {
		r := l.next()
		if r == -1 {
			return nil, NewError(offset, 1, "unterminated string")
		}
		if r == '"' {
			return l.newToken(TokenString, l.expression[start:l.pos-l.lastWidth], offset, l.runePos-offset), nil
		}
		if r != '\\' {
			continue
		}

		buf := bytes.NewBuffer(make([]byte, 0, int(l.pos-start)+8))
		buf.WriteString(l.expression[start : l.pos-l.lastWidth])
		if l.peek() == '"' {
			l.next()
			buf.WriteRune('"')
		} else {
			buf.WriteRune('\\')
		}

		for {
			r = l.next()
			if r == '\\' && l.peek() == '"' {
				l.next()
				buf.WriteRune('"')
				continue
			}
			if r == -1 {
				return nil, NewError(offset, 1, "unterminated string")
			}
			if r == '"' {
				return l.newToken(TokenString, buf.String(), offset, l.runePos-offset), nil
			}
			buf.WriteRune(r)
		}
	}
}

func (l *lexer) Next() (*Token, Error) {
	r := l.next()
	for r == ' ' || r == '\t' || r == '\r' || r == '\n' {
		r = l.next()
	}
	if r == -1 {
		return l.newToken(TokenEOF, "", l.runePos, 0), nil
	}

	b := basic(r)
	if b != TokenUnknown {
		if r == '.' {
			n := l.peek()
			if n >= '0' && n <= '9' {
				return l.consumeNumber(), nil
			}
		}
		if l.pos-l.lastWidth > uint16(len(l.expression)-1) {
			return l.newToken(TokenEOF, "", l.runePos, 0), nil
		}
		offset := l.runePos - 1
		return l.newToken(b, l.expression[l.pos-l.lastWidth:l.pos], offset, 1), nil
	}

	if r >= '0' && r <= '9' {
		return l.consumeNumber(), nil
	}

	if r == '<' || r == '>' || r == '!' {
		offset := l.runePos - 1
		eq := l.next()
		if eq == '=' {
			switch r {
			case '<':
				return l.newToken(TokenComparison, "<=", offset, 2), nil
			case '>':
				return l.newToken(TokenComparison, ">=", offset, 2), nil
			default:
				return l.newToken(TokenComparison, "!=", offset, 2), nil
			}
		}
		l.back()
		switch r {
		case '<':
			return l.newToken(TokenComparison, "<", offset, 1), nil
		case '>':
			return l.newToken(TokenComparison, ">", offset, 1), nil
		default:
			return l.newToken(TokenComparison, "!", offset, 1), nil
		}
	}

	if r == '=' {
		if l.peek() == '=' {
			l.next()
			return l.newToken(TokenComparison, "==", l.runePos-2, 2), nil
		}
		return nil, NewError(l.runePos-1, 1, "= should be ==")
	}

	if r == '"' {
		return l.consumeString()
	}

	return l.consumeIdentifier(), nil
}
