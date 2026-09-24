package pdx

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// The keys and tags the writer spells a coat of arms with. The reader knows
// them too, in the parser library; they are repeated here because what this
// writes is a layout of its own, not a copy of what was read.
const (
	keyPattern        = "pattern"
	keyColoredEmblem  = "colored_emblem"
	keyTexturedEmblem = "textured_emblem"
	keySub            = "sub"
	keyTexture        = "texture"
	keyParent         = "parent"
	keyInstance       = "instance"
	keyMask           = "mask"
	keyPosition       = "position"
	keyScale          = "scale"
	keyRotation       = "rotation"
	keyOffset         = "offset"

	tagRGB    = "rgb"
	tagHSV360 = "hsv360"
	tagList   = "list"
)

// Script writes a coat of arms as script, laid out the way the games' own
// files lay one out: the pattern and the colours first, then one block per
// layer, and in it one block per placement. lineBreak is "\n" or "\r\n", to
// match the file the script goes into.
//
// What the file spelled with variables or expressions is written as the
// numbers they came to, since that is all that is left of them once read.
func Script(flag Flag, lineBreak string) string {
	w := &writer{lineBreak: lineBreak}

	w.line(0, flag.Name+" = {")

	if flag.Pattern != "" {
		w.line(1, keyPattern+" = "+quote(flag.Pattern))
	}

	w.colors(1, flag.Colors)

	for _, layer := range flag.Layers {
		w.blank()

		switch typed := layer.(type) {
		case *ColoredEmblem:
			w.coloredEmblem(typed)
		case *TexturedEmblem:
			w.texturedEmblem(typed)
		case *SubFlag:
			w.subFlag(typed)
		}
	}

	w.text.WriteString("}")

	return w.text.String()
}

type writer struct {
	text      strings.Builder
	lineBreak string
}

func (w *writer) line(depth int, text string) {
	w.text.WriteString(strings.Repeat("\t", depth))
	w.text.WriteString(text)
	w.text.WriteString(w.lineBreak)
}

func (w *writer) blank() {
	w.text.WriteString(w.lineBreak)
}

func (w *writer) coloredEmblem(emblem *ColoredEmblem) {
	w.line(1, keyColoredEmblem+" = {")
	w.line(2, keyTexture+" = "+quote(emblem.Texture))
	w.colors(2, emblem.Colors)

	if emblem.Mask > 0 {
		w.line(2, fmt.Sprintf("%s = { %d }", keyMask, emblem.Mask))
	}

	w.instances(emblem.Instances)
	w.line(1, "}")
}

func (w *writer) texturedEmblem(emblem *TexturedEmblem) {
	w.line(1, keyTexturedEmblem+" = {")
	w.line(2, keyTexture+" = "+quote(emblem.Texture))
	w.instances(emblem.Instances)
	w.line(1, "}")
}

func (w *writer) subFlag(sub *SubFlag) {
	w.line(1, keySub+" = {")
	w.line(2, keyParent+" = "+quote(sub.Parent))

	for _, instance := range sub.Instances {
		w.line(2, keyInstance+" = {")
		w.line(3, keyOffset+" = "+vector(instance.Offset))
		w.line(3, keyScale+" = "+vector(instance.Scale))
		w.line(2, "}")
	}

	w.line(1, "}")
}

// instances writes an emblem's placements, one block each with an attribute
// per line. A rotation of zero is left out, as the games' files do; position
// and scale are always written, so that a reader of the file does not have to
// know the defaults.
func (w *writer) instances(instances []Instance) {
	for _, instance := range instances {
		w.line(2, keyInstance+" = {")
		w.line(3, keyPosition+" = "+vector(instance.Position))
		w.line(3, keyScale+" = "+vector(instance.Scale))

		if instance.Rotation != 0 {
			w.line(3, keyRotation+" = "+number(instance.Rotation))
		}

		w.line(2, "}")
	}
}

// colors writes colour slots in the order of their numbers, whatever order
// they were added in.
func (w *writer) colors(depth int, colors Colors) {
	sorted := slices.Clone(colors)
	slices.SortStableFunc(sorted, func(first, second Color) int {
		return cmp.Compare(SlotIndex(first.Slot), SlotIndex(second.Slot))
	})

	for _, color := range sorted {
		if text, ok := colorScript(color.Value); ok {
			w.line(depth, color.Slot+" = "+text)
		}
	}
}

// colorScript spells a colour value the way the files do.
func colorScript(value ColorValue) (string, bool) {
	switch typed := value.(type) {
	case NamedColor:
		return quote(typed.Name), true

	case SlotColor:
		// Unquoted, which is what makes it a reference to the slot rather
		// than a colour of that name.
		return typed.Slot, true

	case RGBColor:
		return fmt.Sprintf("%s { %d %d %d }", tagRGB, typed.R, typed.G, typed.B), true

	case HSVColor:
		// hsv360 is the spelling a person can read: degrees and percentages.
		return fmt.Sprintf("%s { %s %s %s }", tagHSV360,
			number(typed.H), number(typed.S*100), number(typed.V*100)), true

	case ListColor:
		return tagList + " " + quote(typed.List), true
	}

	return "", false
}

func vector(value Vec2) string {
	return "{ " + number(value.X) + " " + number(value.Y) + " }"
}

// numberDecimals is how many decimals a number is written with at most: a
// ten thousandth of the canvas is well under a pixel.
const numberDecimals = 4

// number writes a number as short as it can be: 0.5 rather than 0.500000.
func number[T float32 | float64](value T) string {
	scale := math.Pow(10, numberDecimals)
	rounded := math.Round(float64(value)*scale) / scale

	if rounded == 0 {
		// Rounding a small negative number leaves minus zero behind.
		rounded = 0
	}

	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// quote writes a string in the quotes the files put around names.
func quote(text string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text)

	return `"` + escaped + `"`
}
