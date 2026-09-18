package script

import (
	"math"
	"testing"
)

func TestParseFieldsAndValues(t *testing.T) {
	document, err := Parse(`
		# a comment
		pattern = "pattern_solid.tga"
		color1 = red
		size = 3
		offset = -0.5
	`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	pattern, ok := document.Get("pattern")
	if !ok {
		t.Fatal("pattern is missing")
	}
	if text, _ := pattern.Str(); text != "pattern_solid.tga" {
		t.Errorf("pattern = %q, want pattern_solid.tga", text)
	}
	if !pattern.Quoted {
		t.Error("pattern should be marked as quoted")
	}

	// Whether a string was quoted is what separates a named colour from a
	// reference to another colour slot, so it has to survive parsing.
	color, _ := document.Get("color1")
	if color.Quoted {
		t.Error("an unquoted identifier should not be marked as quoted")
	}

	size, _ := document.Get("size")
	if number, _ := size.Num(); number != 3 {
		t.Errorf("size = %v, want 3", number)
	}

	offset, _ := document.Get("offset")
	if number, _ := offset.Num(); number != -0.5 {
		t.Errorf("offset = %v, want -0.5", number)
	}
}

func TestParseByteOrderMark(t *testing.T) {
	document, err := Parse(byteOrderMark + "key = 1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if _, ok := document.Get("key"); !ok {
		t.Error("a leading byte order mark hid the first field")
	}
}

func TestParseRepeatedKeys(t *testing.T) {
	document, err := Parse(`
		colored_emblem = { texture = "a.dds" }
		colored_emblem = { texture = "b.dds" }
		color1 = "red"
		color1 = "blue"
	`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	emblems := document.All("colored_emblem")
	if len(emblems) != 2 {
		t.Fatalf("got %d emblems, want 2", len(emblems))
	}

	// A repeated scalar is an override, so the last one is the one that counts.
	color, _ := document.Get("color1")
	if text, _ := color.Str(); text != "blue" {
		t.Errorf("color1 = %q, want blue", text)
	}
}

func TestParseListsAndBlocks(t *testing.T) {
	document, err := Parse(`
		instance = { scale = { 0.7 0.7 } position = { 0.5 0.48 } }
		mask = { 2 }
		empty = {}
	`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	instance, _ := document.Get("instance")
	if instance.Kind != KindBlock {
		t.Fatalf("instance kind = %v, want a block", instance.Kind)
	}

	scale, _ := instance.Get("scale")
	numbers, ok := scale.Numbers()
	if !ok {
		t.Fatalf("scale is not a list of numbers: %+v", scale)
	}
	if len(numbers) != 2 || numbers[0] != 0.7 || numbers[1] != 0.7 {
		t.Errorf("scale = %v, want [0.7 0.7]", numbers)
	}

	mask, _ := document.Get("mask")
	if numbers, _ := mask.Numbers(); len(numbers) != 1 || numbers[0] != 2 {
		t.Errorf("mask = %v, want [2]", numbers)
	}

	empty, _ := document.Get("empty")
	if numbers, ok := empty.Numbers(); !ok || len(numbers) != 0 {
		t.Errorf("empty = %v, want an empty list", numbers)
	}
}

func TestParseTaggedValue(t *testing.T) {
	document, err := Parse(`
		colors = {
			black = hsv360 { 0 0 5 }
			todo_purple = rgb { 1 0.4 0.6 }
		}
	`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	colors, _ := document.Get("colors")

	black, _ := colors.Get("black")
	if black.Kind != KindTagged || black.Tag != "hsv360" {
		t.Fatalf("black = %+v, want a value tagged hsv360", black)
	}

	numbers, ok := black.Value.Numbers()
	if !ok || len(numbers) != 3 || numbers[2] != 5 {
		t.Errorf("black = %v, want [0 0 5]", numbers)
	}

	purple, _ := colors.Get("todo_purple")
	if purple.Tag != "rgb" {
		t.Errorf("todo_purple tag = %q, want rgb", purple.Tag)
	}
}

func TestParseVariablesAndExpressions(t *testing.T) {
	document, err := Parse(`
		@third = @[1/3]
		@offset = 0.25
		@canton_x = @[ ( 333 / 768 ) + 0.001 ]

		a = @third
		b = @[0.5 + offset]
		c = @canton_x
		d = @[ -offset * 2 ]
	`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	cases := []struct {
		key  string
		want float64
	}{
		{"a", 1.0 / 3.0},
		{"b", 0.75},
		{"c", (333.0 / 768.0) + 0.001},
		{"d", -0.5},
	}

	for _, test := range cases {
		value, ok := document.Get(test.key)
		if !ok {
			t.Errorf("%s is missing", test.key)

			continue
		}

		number, ok := value.Num()
		if !ok {
			t.Errorf("%s is not a number: %+v", test.key, value)

			continue
		}

		if math.Abs(number-test.want) > 1e-9 {
			t.Errorf("%s = %v, want %v", test.key, number, test.want)
		}
	}

	// A definition is an instruction to the parser, not part of the document.
	if _, ok := document.Get("third"); ok {
		t.Error("a variable definition leaked into the document")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"unclosed block", `a = { b = 1`},
		{"unclosed list", `a = { 1 2`},
		{"stray closing brace", `a = 1 }`},
		{"missing value", `a =`},
		{"missing equals", `a 1`},
		{"unterminated string", `a = "abc`},
		{"undefined variable", `a = @nope`},
		{"undefined in expression", `a = @[ nope + 1 ]`},
		{"division by zero", `a = @[ 1 / 0 ]`},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse(test.input); err == nil {
				t.Errorf("Parse(%q) succeeded, want an error", test.input)
			}
		})
	}
}

func TestSyntaxErrorReportsPosition(t *testing.T) {
	_, err := Parse("a = 1\nb = }\n")

	syntaxError, ok := err.(*SyntaxError)
	if !ok {
		t.Fatalf("error is %T, want a *SyntaxError", err)
	}

	if syntaxError.Line != 2 {
		t.Errorf("line = %d, want 2", syntaxError.Line)
	}
}

func TestFieldSpans(t *testing.T) {
	input := byteOrderMark + "@a = 1\nfirst = { x = { 1 2 } }\r\nsecond = \"text\" # comment\nthird = rgb { 1 @a 1 }"

	document, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := map[string]string{
		"first":  "first = { x = { 1 2 } }",
		"second": `second = "text"`,
		"third":  "third = rgb { 1 @a 1 }",
	}

	for _, field := range document.Fields {
		if got := input[field.Start:field.End]; got != want[field.Key] {
			t.Errorf("span of %s = %q, want %q", field.Key, got, want[field.Key])
		}
	}

	inner := document.Fields[0].Value.Fields[0]
	if got := input[inner.Start:inner.End]; got != "x = { 1 2 }" {
		t.Errorf("span of a nested field = %q, want %q", got, "x = { 1 2 }")
	}
}

func TestIsKey(t *testing.T) {
	for text, want := range map[string]bool{
		"GBR": true, "ABU_subject_GBR": true, "_x": true, "new_flag": true,
		"": false, "1GBR": false, "two words": false, "quote\"": false, "a{": false,
	} {
		if got := IsKey(text); got != want {
			t.Errorf("IsKey(%q) = %v, want %v", text, got, want)
		}
	}
}
