package script

import (
	"fmt"
	"strconv"
	"strings"
)

type tokenKind uint8

const (
	tokenEOF tokenKind = iota
	tokenIdentifier
	tokenString
	tokenNumber
	tokenOpenBrace
	tokenCloseBrace
	tokenEquals

	// tokenVariable is a variable such as @third, either being defined or used.
	tokenVariable

	// tokenExpression is an arithmetic expression such as @[ 1 / 3 ]. Its text
	// is whatever stood between the brackets.
	tokenExpression
)

// byteOrderMark is what the game tools put at the start of some files. It is
// not part of the script and has to be skipped before lexing.
const byteOrderMark = "\xef\xbb\xbf"

type token struct {
	kind   tokenKind
	text   string
	number float64
	line   int
	column int

	// start and end are byte offsets into the original input, byte order mark
	// included: where the token begins and just past where it ends.
	start, end int
}

// SyntaxError reports where a file stopped making sense.
type SyntaxError struct {
	Message string
	Line    int
	Column  int
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("line %d column %d: %s", e.Line, e.Column, e.Message)
}

type lexer struct {
	input  string
	offset int

	// base is how many bytes were cut from the front of the input, so that
	// offsets can be reported into the input as it was handed over.
	base int

	// tokenStart is where the token being read began.
	tokenStart int

	line     int
	column   int
	warnings []Warning
}

func newLexer(input string) *lexer {
	trimmed := strings.TrimPrefix(input, byteOrderMark)

	return &lexer{input: trimmed, base: len(input) - len(trimmed), line: 1, column: 1}
}

// tokenize reads the whole input up front. Coat of arms files are small, and
// having every token available lets the parser look ahead far enough to tell a
// list from a block without any backtracking.
func (l *lexer) tokenize() ([]token, error) {
	var tokens []token

	for {
		next, err := l.next()
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, next)

		if next.kind == tokenEOF {
			return tokens, nil
		}
	}
}

func (l *lexer) next() (token, error) {
	l.tokenStart = l.offset

	lexed, err := l.lexToken()
	if err != nil {
		return token{}, err
	}

	lexed.start = l.base + l.tokenStart
	lexed.end = l.base + l.offset

	return lexed, nil
}

func (l *lexer) lexToken() (token, error) {
	// Characters that belong to no token are skipped rather than treated as a
	// failure. The shipped game files contain the odd typo, such as a stray
	// bracket in the middle of a position, and the games read those files
	// anyway. Refusing the file would cost every flag in it, so the character
	// is dropped and recorded instead.
	for {
		l.skipSpaceAndComments()

		if l.offset >= len(l.input) {
			return token{kind: tokenEOF, line: l.line, column: l.column}, nil
		}

		start := l.position()
		l.tokenStart = l.offset
		char := l.input[l.offset]

		switch {
		case char == '{':
			l.advance()

			return token{kind: tokenOpenBrace, text: "{", line: start.line, column: start.column}, nil

		case char == '}':
			l.advance()

			return token{kind: tokenCloseBrace, text: "}", line: start.line, column: start.column}, nil

		case char == '=':
			l.advance()

			return token{kind: tokenEquals, text: "=", line: start.line, column: start.column}, nil

		case char == '"':
			return l.lexString(start)

		case char == '@':
			return l.lexVariable(start)

		case isNumberStart(char) && l.looksLikeNumber():
			return l.lexNumber(start)

		case isIdentifierStart(char):
			return l.lexIdentifier(start)
		}

		l.warn(start, "ignored unexpected character %q", char)
		l.advance()
	}
}

func (l *lexer) warn(where position, format string, args ...any) {
	l.warnings = append(l.warnings, Warning{
		Message: fmt.Sprintf(format, args...),
		Line:    where.line,
		Column:  where.column,
	})
}

type position struct {
	line   int
	column int
}

func (l *lexer) position() position {
	return position{line: l.line, column: l.column}
}

func (l *lexer) advance() {
	if l.input[l.offset] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	l.offset++
}

func (l *lexer) skipSpaceAndComments() {
	for l.offset < len(l.input) {
		switch char := l.input[l.offset]; {
		case char == ' ' || char == '\t' || char == '\r' || char == '\n':
			l.advance()

		case char == '#':
			for l.offset < len(l.input) && l.input[l.offset] != '\n' {
				l.advance()
			}

		default:
			return
		}
	}
}

func (l *lexer) lexString(start position) (token, error) {
	l.advance() // opening quote

	var text strings.Builder

	for l.offset < len(l.input) {
		switch char := l.input[l.offset]; char {
		case '"':
			l.advance()

			return token{kind: tokenString, text: text.String(), line: start.line, column: start.column}, nil

		case '\n':
			return token{}, &SyntaxError{Message: "unterminated string", Line: start.line, Column: start.column}

		case backslash:
			l.advance()

			if l.offset >= len(l.input) {
				return token{}, &SyntaxError{Message: "unterminated string", Line: start.line, Column: start.column}
			}

			text.WriteByte(l.input[l.offset])
			l.advance()

		default:
			text.WriteByte(char)
			l.advance()
		}
	}

	return token{}, &SyntaxError{Message: "unterminated string", Line: start.line, Column: start.column}
}

// backslash escapes the next character inside a quoted string.
const backslash = 92

// looksLikeNumber decides whether a leading sign starts a number or something
// else. A sign on its own is neither, so it has to be followed by a digit or a
// decimal point.
func (l *lexer) looksLikeNumber() bool {
	if isDigit(l.input[l.offset]) {
		return true
	}

	if l.offset+1 >= len(l.input) {
		return false
	}

	next := l.input[l.offset+1]

	return isDigit(next) || next == '.'
}

func (l *lexer) lexNumber(start position) (token, error) {
	begin := l.offset

	if l.input[l.offset] == '+' || l.input[l.offset] == '-' {
		l.advance()
	}

	for l.offset < len(l.input) && (isDigit(l.input[l.offset]) || l.input[l.offset] == '.') {
		l.advance()
	}

	text := l.input[begin:l.offset]

	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return token{}, &SyntaxError{
			Message: fmt.Sprintf("malformed number %q", text),
			Line:    start.line,
			Column:  start.column,
		}
	}

	return token{kind: tokenNumber, text: text, number: value, line: start.line, column: start.column}, nil
}

func (l *lexer) lexIdentifier(start position) (token, error) {
	begin := l.offset

	for l.offset < len(l.input) && isIdentifierPart(l.input[l.offset]) {
		l.advance()
	}

	return token{
		kind:   tokenIdentifier,
		text:   l.input[begin:l.offset],
		line:   start.line,
		column: start.column,
	}, nil
}

// lexVariable reads either a variable name, @third, or an expression, @[ 1/3 ].
func (l *lexer) lexVariable(start position) (token, error) {
	l.advance() // the sigil

	if l.offset < len(l.input) && l.input[l.offset] == '[' {
		l.advance()

		begin := l.offset
		for l.offset < len(l.input) && l.input[l.offset] != ']' {
			l.advance()
		}

		if l.offset >= len(l.input) {
			return token{}, &SyntaxError{Message: "unterminated expression", Line: start.line, Column: start.column}
		}

		text := l.input[begin:l.offset]
		l.advance() // the closing bracket

		return token{kind: tokenExpression, text: text, line: start.line, column: start.column}, nil
	}

	begin := l.offset
	for l.offset < len(l.input) && isIdentifierPart(l.input[l.offset]) {
		l.advance()
	}

	if begin == l.offset {
		return token{}, &SyntaxError{
			Message: "expected a variable name",
			Line:    start.line,
			Column:  start.column,
		}
	}

	return token{
		kind:   tokenVariable,
		text:   l.input[begin:l.offset],
		line:   start.line,
		column: start.column,
	}, nil
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isNumberStart(char byte) bool {
	return isDigit(char) || char == '-' || char == '+'
}

func isIdentifierStart(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char == '_'
}

// isIdentifierPart is deliberately generous: unquoted values in these files
// include file names such as pattern_solid.tga and paths such as gfx/flags.
func isIdentifierPart(char byte) bool {
	return isIdentifierStart(char) || isDigit(char) ||
		char == '.' || char == '/' || char == '-' || char == ':'
}

// IsKey reports whether text reads back as a key when written without quotes,
// which is how the names of coats of arms are written.
func IsKey(text string) bool {
	if text == "" || !isIdentifierStart(text[0]) {
		return false
	}

	for index := 1; index < len(text); index++ {
		if !isIdentifierPart(text[index]) {
			return false
		}
	}

	return true
}
