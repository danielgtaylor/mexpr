package mexpr

import (
	"fmt"
	"strconv"
)

// NodeType defines the type of the abstract syntax tree node.
type NodeType int

const (
	NodeUnknown = iota
	NodeIdentifier
	NodeLiteral
	NodeArithmetic
	NodeComparison
	NodeBooleanComparison
	NodeNot
	NodeFieldSelect
	NodeArrayIndex
	NodeSlice
	NodeSign
	NodeStringCompare
)

// Node is a unit of the binary tree that makes up the abstract syntax tree.
type Node struct {
	Type  NodeType
	Token *Token
	Value interface{}
	Left  *Node
	Right *Node
}

// Print will print out a tree for debugging.
func (n *Node) Print(prefix string) {
	fmt.Printf("%s%s %s\n", prefix, n.Token.Type, n.Token.Value)
	if n.Left != nil {
		n.Left.Print(prefix + "L: ")
	}
	if n.Right != nil {
		n.Right.Print(prefix + "R: ")
	}
}

// bindingPowers for different tokens. Not listed means zero. The higher the
// number, the higher the token is in the order of operations.
var bindingPowers = map[TokenType]int{
	TokenOr:            1,
	TokenAnd:           2,
	TokenStringCompare: 3,
	TokenComparison:    5,
	TokenSlice:         5,
	TokenAddSub:        10,
	TokenMulDiv:        15,
	TokenDot:           40,
	TokenNot:           45,
	TokenPower:         50,
	TokenLeftBracket:   60,
	TokenLeftParen:     70,
}

// Parser takes a lexer and parses its tokens into an abstract syntax tree.
type Parser interface {
	// Parse the expression and return the root node.
	Parse() (*Node, Error)
}

// NewParser creates a new parser that uses the given lexer to get and process
// tokens into an abstract syntax tree.
func NewParser(lexer Lexer) Parser {
	return &parser{
		lexer: lexer,
	}
}

// parser is an implementation of a Pratt or top-down operator precedence parser
type parser struct {
	lexer Lexer
	token *Token
	back  bool
}

func (p *parser) advance() Error {
	if p.back {
		return nil
	}

	t, err := p.lexer.Next()
	if err != nil {
		return err
	}
	p.token = t
	return nil
}

func (p *parser) parse(bindingPower int) (*Node, Error) {
	leftToken := p.token
	if err := p.advance(); err != nil {
		return nil, err
	}
	leftNode, err := p.nud(leftToken)
	if err != nil {
		return nil, err
	}
	currentToken := p.token
	for bindingPower < bindingPowers[currentToken.Type] {
		if err := p.advance(); err != nil {
			return nil, err
		}
		leftNode, err = p.led(currentToken, leftNode)
		if err != nil {
			return nil, err
		}
		currentToken = p.token
	}
	return leftNode, nil
}

// ensure the current token is `typ`, returning the `result` unless `err` is
// set or some other error occurs. Advances past the expected token type.
func (p *parser) ensure(result *Node, err Error, typ TokenType) (*Node, Error) {
	if err != nil {
		return nil, err
	}
	if p.token.Type == typ {
		if err := p.advance(); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, NewError(p.token.Offset, "expected %s but found %s", typ, p.token.Type)
}

// nud: null denotation. These nodes have no left context and only
// consume to the right. Examples: identifiers, numbers, unary operators like
// minus.
func (p *parser) nud(t *Token) (*Node, Error) {
	switch t.Type {
	case TokenIdentifier:
		return &Node{Type: NodeIdentifier, Token: t}, nil
	case TokenNumber:
		f, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			return nil, NewError(p.token.Offset, err.Error())
		}
		return &Node{Type: NodeLiteral, Token: t, Value: f}, nil
	case TokenString:
		return &Node{Type: NodeLiteral, Token: t, Value: t.Value}, nil
	case TokenLeftParen:
		result, err := p.parse(0)
		return p.ensure(result, err, TokenRightParen)
	case TokenNot:
		result, err := p.parse(bindingPowers[t.Type])
		if err != nil {
			return nil, err
		}
		return &Node{Type: NodeNot, Token: t, Right: result}, nil
	case TokenAddSub:
		result, err := p.parse(bindingPowers[t.Type])
		if err != nil {
			return nil, err
		}
		return &Node{Type: NodeSign, Token: t, Right: result}, nil
	case TokenSlice:
		result, err := p.parse(bindingPowers[t.Type])
		if err != nil {
			return nil, err
		}
		// Create a dummy left node with value 0, the start of the slice.
		return &Node{Type: NodeSlice, Token: t, Left: &Node{Type: NodeLiteral, Token: t, Value: 0.0}, Right: result}, nil
	case TokenEOF:
		return nil, NewError(p.token.Offset, "incomplete expression, EOF found")
	}
	return nil, nil
}

// newNodeParseRight creates a new node with the right tree set to the
// output of recursively parsing until a lower binding power is encountered.
func (p *parser) newNodeParseRight(left *Node, t *Token, typ NodeType, bindingPower int) (*Node, Error) {
	right, err := p.parse(bindingPower)
	if err != nil {
		return nil, err
	}
	return &Node{Type: typ, Token: t, Left: left, Right: right}, nil
}

// led: left denotation. These tokens produce nodes that operate on two operands
// a left and a right. Examples: addition, multiplication, etc.
func (p *parser) led(t *Token, n *Node) (*Node, Error) {
	switch t.Type {
	case TokenAddSub, TokenMulDiv:
		return p.newNodeParseRight(n, t, NodeArithmetic, bindingPowers[t.Type])
	case TokenPower:
		return p.newNodeParseRight(n, t, NodeArithmetic, bindingPowers[t.Type]-1)
	case TokenComparison:
		return p.newNodeParseRight(n, t, NodeComparison, bindingPowers[t.Type])
	case TokenAnd, TokenOr:
		return p.newNodeParseRight(n, t, NodeBooleanComparison, bindingPowers[t.Type])
	case TokenStringCompare:
		return p.newNodeParseRight(n, t, NodeStringCompare, bindingPowers[t.Type])
	case TokenDot:
		return p.newNodeParseRight(n, t, NodeFieldSelect, bindingPowers[t.Type])
	case TokenLeftBracket:
		n, err := p.newNodeParseRight(n, t, NodeArrayIndex, 0)
		return p.ensure(n, err, TokenRightBracket)
	case TokenSlice:
		return p.newNodeParseRight(n, t, NodeSlice, bindingPowers[t.Type])
	}
	return nil, nil
}

func (p *parser) Parse() (*Node, Error) {
	p.advance()
	return p.parse(0)
}
