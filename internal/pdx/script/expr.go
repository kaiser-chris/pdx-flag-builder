package script

import (
	"fmt"
	"strconv"
)

// evaluate works out a bracketed expression.
//
// The games allow arithmetic over the variables defined in a file, which coat
// of arms files use to keep proportions readable, as in ( 333 / 768 ) + 0.001.
// Variables are named without the sigil inside an expression.
func evaluate(expression string, variables map[string]float64) (float64, error) {
	reader := &expressionReader{input: expression, variables: variables}

	value, err := reader.readSum()
	if err != nil {
		return 0, err
	}

	reader.skipSpace()
	if reader.offset < len(reader.input) {
		return 0, fmt.Errorf("unexpected %q in expression %q", reader.input[reader.offset], expression)
	}

	return value, nil
}

type expressionReader struct {
	input     string
	offset    int
	variables map[string]float64
}

func (r *expressionReader) skipSpace() {
	for r.offset < len(r.input) {
		switch r.input[r.offset] {
		case ' ', '\t', '\n', '\r':
			r.offset++
		default:
			return
		}
	}
}

func (r *expressionReader) readSum() (float64, error) {
	value, err := r.readProduct()
	if err != nil {
		return 0, err
	}

	for {
		r.skipSpace()

		if r.offset >= len(r.input) {
			return value, nil
		}

		operator := r.input[r.offset]
		if operator != '+' && operator != '-' {
			return value, nil
		}
		r.offset++

		right, err := r.readProduct()
		if err != nil {
			return 0, err
		}

		if operator == '+' {
			value += right
		} else {
			value -= right
		}
	}
}

func (r *expressionReader) readProduct() (float64, error) {
	value, err := r.readTerm()
	if err != nil {
		return 0, err
	}

	for {
		r.skipSpace()

		if r.offset >= len(r.input) {
			return value, nil
		}

		operator := r.input[r.offset]
		if operator != '*' && operator != '/' {
			return value, nil
		}
		r.offset++

		right, err := r.readTerm()
		if err != nil {
			return 0, err
		}

		if operator == '/' {
			if right == 0 {
				return 0, fmt.Errorf("division by zero in expression %q", r.input)
			}

			value /= right
		} else {
			value *= right
		}
	}
}

func (r *expressionReader) readTerm() (float64, error) {
	r.skipSpace()

	if r.offset >= len(r.input) {
		return 0, fmt.Errorf("expression %q ends early", r.input)
	}

	char := r.input[r.offset]

	switch {
	case char == '(':
		r.offset++

		value, err := r.readSum()
		if err != nil {
			return 0, err
		}

		r.skipSpace()
		if r.offset >= len(r.input) || r.input[r.offset] != ')' {
			return 0, fmt.Errorf("missing closing bracket in expression %q", r.input)
		}
		r.offset++

		return value, nil

	case char == '-' || char == '+':
		r.offset++

		value, err := r.readTerm()
		if err != nil {
			return 0, err
		}

		if char == '-' {
			return -value, nil
		}

		return value, nil

	case isDigit(char) || char == '.':
		return r.readNumber()

	case isIdentifierStart(char) || char == '@':
		return r.readVariable()
	}

	return 0, fmt.Errorf("unexpected %q in expression %q", char, r.input)
}

func (r *expressionReader) readNumber() (float64, error) {
	begin := r.offset

	for r.offset < len(r.input) && (isDigit(r.input[r.offset]) || r.input[r.offset] == '.') {
		r.offset++
	}

	text := r.input[begin:r.offset]

	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed number %q in expression", text)
	}

	return value, nil
}

func (r *expressionReader) readVariable() (float64, error) {
	// Variables are normally written without the sigil inside an expression,
	// but accept it as well rather than failing over a stray character.
	if r.input[r.offset] == '@' {
		r.offset++
	}

	begin := r.offset
	for r.offset < len(r.input) && isExpressionNamePart(r.input[r.offset]) {
		r.offset++
	}

	name := r.input[begin:r.offset]

	value, defined := r.variables[name]
	if !defined {
		return 0, fmt.Errorf("undefined variable %q in expression", name)
	}

	return value, nil
}

// isExpressionNamePart is stricter than the rule for identifiers elsewhere in
// the script. Inside an expression the slash and the dash are operators, so a
// name written without spaces around them, as in third/2, has to end before
// them rather than swallow them.
func isExpressionNamePart(char byte) bool {
	return isIdentifierStart(char) || isDigit(char)
}
