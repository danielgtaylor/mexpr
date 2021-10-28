// Package expr provides a simple expression parser.
package mexpr

// Parse an expression and return the abstract syntax tree. If `types` is
// passed, it should be a set of representative example values for the input
// which will be used to type check the expression against.
func Parse(expression string, types map[string]interface{}) (*Node, Error) {
	l := NewLexer(expression)
	p := NewParser(l)
	ast, err := p.Parse()
	if err != nil {
		return nil, err
	}
	if types != nil {
		if err := TypeCheck(ast, types); err != nil {
			return nil, err
		}
	}
	return ast, nil
}

// TypeCheck will take a parsed AST and type check against the given input
// structure with representative example values.
func TypeCheck(ast *Node, types map[string]interface{}) Error {
	i := NewInterpreter(ast, StrictMode)
	_, err := i.Run(types)
	if err != nil {
		return err
	}
	return nil
}

// Run executes an AST with the given input and returns the output.
func Run(ast *Node, input map[string]interface{}, options ...InterpreterOption) (interface{}, Error) {
	i := NewInterpreter(ast, options...)
	return i.Run(input)
}

// Eval is a convenience function which lexes, parses, and executes an
// expression with the given input. If you plan to execute the expression
// multiple times consider caching the output of `Parse(...)` instead for a
// big speed improvement.
func Eval(expression string, input map[string]interface{}, options ...InterpreterOption) (interface{}, Error) {
	// No need to type check because we are about to run with the input.
	ast, err := Parse(expression, nil)
	if err != nil {
		return nil, err
	}
	return Run(ast, input, options...)
}
