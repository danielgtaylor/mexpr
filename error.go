package mexpr

import "fmt"

// Error represents an error at a specific location.
type Error interface {
	Error() string

	// Offset returns the character offset of the error within the experssion.
	Offset() int

	// Pretty prints out a message with a pointer to the source location of the
	// error.
	Pretty(source string) string
}

type exprErr struct {
	offset  int
	message string
}

func (e *exprErr) Error() string {
	return e.message
}

func (e *exprErr) Offset() int {
	return e.offset
}

func (e *exprErr) Pretty(source string) string {
	msg := e.Error() + "\n" + source + "\n"
	for i := 0; i < e.offset; i++ {
		msg += "."
	}
	msg += "^"
	return msg
}

// NewError creates a new error at a specific location.
func NewError(offset int, format string, a ...interface{}) Error {
	return &exprErr{
		offset:  offset,
		message: fmt.Sprintf(format, a...),
	}
}
