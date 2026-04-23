package mexpr

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Error represents an error at a specific location.
type Error interface {
	Error() string

	// Offset returns the rune offset of the error within the expression.
	Offset() uint16

	// Length returns the rune length after the offset where the error ends.
	Length() uint8

	// Pretty prints out a message with a pointer to the source location of the
	// error.
	Pretty(source string) string
}

type exprErr struct {
	offset  uint16
	length  uint8
	message string
}

func (e *exprErr) Error() string {
	return e.message
}

func (e *exprErr) Offset() uint16 {
	return e.offset
}

func (e *exprErr) Length() uint8 {
	return e.length
}

func (e *exprErr) Pretty(source string) string {
	var msg strings.Builder
	msg.WriteString(e.Error())
	msg.WriteByte('\n')
	msg.WriteString(source)
	msg.WriteByte('\n')
	for i := uint16(0); i < e.offset; i++ {
		msg.WriteByte('.')
	}
	length := e.length
	if length == 0 && utf8.RuneCountInString(source) > int(e.offset) {
		length = 1
	}
	for i := uint8(0); i < length; i++ {
		msg.WriteByte('^')
	}
	return msg.String()
}

// NewError creates a new error at a specific location.
func NewError(offset uint16, length uint8, format string, a ...any) Error {
	return &exprErr{
		offset:  offset,
		length:  length,
		message: fmt.Sprintf(format, a...),
	}
}
