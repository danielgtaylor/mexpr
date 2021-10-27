package mexpr

import "fmt"

// Error represents an error at a specific location.
type Error interface {
	Error() string

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

func (e *exprErr) Pretty(source string) string {
	msg := e.Error() + "\n" + source + "\n"
	for i := 0; i < e.offset; i++ {
		msg += "."
	}
	msg += "^"
	return msg
}

func NewError(offset int, format string, a ...interface{}) Error {
	return &exprErr{
		offset:  offset,
		message: fmt.Sprintf(format, a...),
	}
}
