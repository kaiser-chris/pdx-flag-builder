package script

import "fmt"

// Parse reads a script document.
func Parse(input string) (*Document, error) {
	lexer := newLexer(input)

	tokens, err := lexer.tokenize()
	if err != nil {
		return nil, err
	}

	parser := &parser{tokens: tokens, variables: map[string]float64{}}

	fields, err := parser.parseFields(false)
	if err != nil {
		return nil, err
	}

	return &Document{Fields: fields, Warnings: lexer.warnings}, nil
}

type parser struct {
	tokens []token
	index  int

	// variables collects the @name definitions of the document. The games
	// resolve them while reading, and so does this parser, so a definition has
	// to appear before it is used. One flat table per document is enough:
	// definitions inside a block stay visible to everything that follows.
	variables map[string]float64
}

func (p *parser) peek() token {
	return p.tokens[p.index]
}

func (p *parser) peekAt(offset int) token {
	index := p.index + offset
	if index >= len(p.tokens) {
		index = len(p.tokens) - 1
	}

	return p.tokens[index]
}

func (p *parser) take() token {
	current := p.tokens[p.index]
	if current.kind != tokenEOF {
		p.index++
	}

	return current
}

func (p *parser) errorAt(current token, format string, args ...any) error {
	return &SyntaxError{
		Message: fmt.Sprintf(format, args...),
		Line:    current.line,
		Column:  current.column,
	}
}

// parseFields reads key and value pairs, either to the end of the input or to
// the closing brace of the block they belong to.
func (p *parser) parseFields(insideBlock bool) ([]Field, error) {
	var fields []Field

	for {
		current := p.peek()

		switch current.kind {
		case tokenEOF:
			if insideBlock {
				return nil, p.errorAt(current, "unexpected end of file, a block was left open")
			}

			return fields, nil

		case tokenCloseBrace:
			if !insideBlock {
				return nil, p.errorAt(current, "unexpected closing brace")
			}

			p.take()

			return fields, nil

		case tokenVariable:
			if err := p.parseVariableDefinition(); err != nil {
				return nil, err
			}

		case tokenIdentifier, tokenString, tokenNumber:
			field, err := p.parseField()
			if err != nil {
				return nil, err
			}

			fields = append(fields, field)

		default:
			return nil, p.errorAt(current, "expected a key, found %q", current.text)
		}
	}
}

func (p *parser) parseField() (Field, error) {
	key := p.take()

	if next := p.peek(); next.kind != tokenEquals {
		return Field{}, p.errorAt(next, "expected = after %q", key.text)
	}
	p.take()

	value, err := p.parseValue()
	if err != nil {
		return Field{}, err
	}

	return Field{Key: key.text, Value: value, Line: key.line}, nil
}

// parseVariableDefinition records a variable definition. The definition itself
// is not part of the tree: by the time a caller sees the document, every use of
// a variable has been replaced by its value.
func (p *parser) parseVariableDefinition() error {
	name := p.take()

	if next := p.peek(); next.kind != tokenEquals {
		return p.errorAt(next, "expected = after variable %q", name.text)
	}
	p.take()

	value, err := p.parseValue()
	if err != nil {
		return err
	}

	if value.Kind != KindNumber {
		return p.errorAt(name, "variable %q is not a number", name.text)
	}

	p.variables[name.text] = value.Number

	return nil
}

func (p *parser) parseValue() (Node, error) {
	current := p.peek()

	switch current.kind {
	case tokenString:
		p.take()

		return Node{Kind: KindString, Text: current.text, Quoted: true, Line: current.line}, nil

	case tokenNumber:
		p.take()

		return Node{Kind: KindNumber, Text: current.text, Number: current.number, Line: current.line}, nil

	case tokenVariable:
		p.take()

		value, defined := p.variables[current.text]
		if !defined {
			return Node{}, p.errorAt(current, "undefined variable %q", current.text)
		}

		return Node{Kind: KindNumber, Text: current.text, Number: value, Line: current.line}, nil

	case tokenExpression:
		p.take()

		value, err := evaluate(current.text, p.variables)
		if err != nil {
			return Node{}, p.errorAt(current, "%s", err)
		}

		return Node{Kind: KindNumber, Text: current.text, Number: value, Line: current.line}, nil

	case tokenIdentifier:
		// An identifier can tag the value that follows it: a colour written
		// rgb { 1 0 0 }, or a random pick written list "normal_colors".
		//
		// Only a brace or a quoted string counts as the tagged value. A bare
		// identifier following another bare identifier is two separate values,
		// which is what keeps lists of plain names from collapsing into one.
		if next := p.peekAt(1); next.kind == tokenOpenBrace || next.kind == tokenString {
			p.take()

			tagged, err := p.parseValue()
			if err != nil {
				return Node{}, err
			}

			return Node{Kind: KindTagged, Tag: current.text, Value: &tagged, Line: current.line}, nil
		}

		p.take()

		return Node{Kind: KindString, Text: current.text, Line: current.line}, nil

	case tokenOpenBrace:
		return p.parseBraced()
	}

	return Node{}, p.errorAt(current, "expected a value, found %q", current.text)
}

// parseBraced reads a braced group, which is either a list of values or a block
// of fields. Which one it is can be seen from the first two tokens inside: a
// key followed by an equals sign makes it a block.
func (p *parser) parseBraced() (Node, error) {
	open := p.take()

	if p.peekAt(1).kind == tokenEquals {
		fields, err := p.parseFields(true)
		if err != nil {
			return Node{}, err
		}

		return Node{Kind: KindBlock, Fields: fields, Line: open.line}, nil
	}

	var items []Node

	for {
		current := p.peek()

		if current.kind == tokenCloseBrace {
			p.take()

			return Node{Kind: KindList, Items: items, Line: open.line}, nil
		}

		if current.kind == tokenEOF {
			return Node{}, p.errorAt(current, "unexpected end of file, a list was left open")
		}

		item, err := p.parseValue()
		if err != nil {
			return Node{}, err
		}

		items = append(items, item)
	}
}
