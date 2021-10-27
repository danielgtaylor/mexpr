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
		// Run through with the example values and see if we get any errors.
		i := NewInterpreter(ast)
		_, err := i.Run(types)
		if err != nil {
			return nil, err
		}
	}
	return ast, nil
}

// Run executes an AST with the given input and returns the output.
func Run(ast *Node, input map[string]interface{}) (interface{}, Error) {
	i := NewInterpreter(ast)
	return i.Run(input)
}

// Eval is a convenience function with lexes, parses, and executes an expression
// with the given input. If you plan to execute the expression multiple times
// consider caching the output of `Parse(...)` instead for a big speed
// improvement.
func Eval(expression string, input map[string]interface{}) (interface{}, Error) {
	// No need to type check because we are about to run with the input.
	ast, err := Parse(expression, nil)
	if err != nil {
		return nil, err
	}
	return Run(ast, input)
}
