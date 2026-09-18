// Package script parses the script language Paradox games use for their data
// files, as far as coat of arms definitions need it.
//
// A document is a sequence of "key = value" fields. A value is a string, a
// number, a list, a block of further fields, or a tagged value such as
// "hsv360 { 0 0 5 }". Keys repeat: a coat of arms carries several
// colored_emblem fields, and an emblem several instance fields.
//
// The language also has variables. "@name = 0.25" defines one, "@name" uses it
// and "@[ 1 / 3 ]" evaluates an expression over them. Definitions are resolved
// while parsing, so the tree handed back contains plain numbers only.
package script

import "fmt"

// Kind is the sort of value a Node holds.
type Kind uint8

const (
	// KindString is a quoted string or a bare identifier. Quoted records which
	// of the two it was, which is what separates a named colour from a
	// reference to another colour slot.
	KindString Kind = iota + 1

	// KindNumber is a number. Everything is kept as a float64 because the
	// script does not distinguish integers from decimals.
	KindNumber

	// KindList is a brace enclosed sequence of values, such as { 0.5 0.5 }.
	KindList

	// KindBlock is a brace enclosed sequence of fields.
	KindBlock

	// KindTagged is a value preceded by a tag, such as rgb { 1 0 0 }.
	KindTagged
)

// Node is one value in a document.
type Node struct {
	Kind Kind

	// Text holds the contents of a string and the original spelling of a
	// number.
	Text string

	// Quoted reports whether a string was written in quotes.
	Quoted bool

	// Number holds the value of a number.
	Number float64

	// Items holds the elements of a list.
	Items []Node

	// Fields holds the fields of a block.
	Fields []Field

	// Tag and Value hold the two halves of a tagged value.
	Tag   string
	Value *Node

	// Line is where the value started, for error messages.
	Line int
}

// Field is a key and the value assigned to it.
type Field struct {
	Key   string
	Value Node
	Line  int

	// Start and End are byte offsets into the input the document was parsed
	// from: the first byte of the key and just past the last byte of the
	// value. They are what lets a definition be replaced in its file without
	// touching anything around it.
	Start, End int
}

// Document is a parsed file.
type Document struct {
	Fields []Field

	// Warnings lists what the parser had to overlook to get through the file.
	// Nothing here stops a document from being usable, but the interface can
	// show it so that a mod author can see a typo in their own file.
	Warnings []Warning
}

// Warning is something the parser ignored rather than failed over.
type Warning struct {
	Message string
	Line    int
	Column  int
}

func (w Warning) String() string {
	return fmt.Sprintf("line %d column %d: %s", w.Line, w.Column, w.Message)
}

// Get returns the value of the last field with the given key.
//
// The last one wins because that is how the games read their own files: a key
// repeated in the same block overrides the earlier value. Keys that are meant
// to repeat, such as colored_emblem, are read with All instead.
func (n Node) Get(key string) (Node, bool) {
	return lastField(n.Fields, key)
}

// All returns the values of every field with the given key, in order.
func (n Node) All(key string) []Node {
	return allFields(n.Fields, key)
}

// Get returns the value of the last top level field with the given key.
func (d *Document) Get(key string) (Node, bool) {
	return lastField(d.Fields, key)
}

// All returns the values of every top level field with the given key.
func (d *Document) All(key string) []Node {
	return allFields(d.Fields, key)
}

func lastField(fields []Field, key string) (Node, bool) {
	for index := len(fields) - 1; index >= 0; index-- {
		if fields[index].Key == key {
			return fields[index].Value, true
		}
	}

	return Node{}, false
}

func allFields(fields []Field, key string) []Node {
	var found []Node

	for _, field := range fields {
		if field.Key == key {
			found = append(found, field.Value)
		}
	}

	return found
}

// Str returns the value of a string node.
func (n Node) Str() (string, bool) {
	if n.Kind != KindString {
		return "", false
	}

	return n.Text, true
}

// Num returns the value of a number node.
func (n Node) Num() (float64, bool) {
	if n.Kind != KindNumber {
		return 0, false
	}

	return n.Number, true
}

// Numbers returns the values of a list that holds nothing but numbers. Lists in
// coat of arms files are always short and always numeric: positions, scales and
// colour channels.
func (n Node) Numbers() ([]float64, bool) {
	if n.Kind != KindList {
		return nil, false
	}

	numbers := make([]float64, 0, len(n.Items))
	for _, item := range n.Items {
		if item.Kind != KindNumber {
			return nil, false
		}

		numbers = append(numbers, item.Number)
	}

	return numbers, true
}

// IsBlock reports whether a value can be read as a block of fields. An empty
// pair of braces is as much an empty block as an empty list, and the file has
// no way of saying which it meant, so it counts as both.
func (n Node) IsBlock() bool {
	return n.Kind == KindBlock || n.Kind == KindList && len(n.Items) == 0
}
