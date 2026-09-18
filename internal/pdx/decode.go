package pdx

import (
	"fmt"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx/script"
)

// Keys used by coat of arms files.
const (
	keyTemplate       = "template"
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
	keyColors         = "colors"
)

// Tags a colour can be written with.
const (
	tagRGB    = "rgb"
	tagHSV    = "hsv"
	tagHSV360 = "hsv360"
	tagList   = "list"
)

// Issue is something wrong with a file that did not stop it being read. A flag
// with one bad instance is still worth showing, so problems are collected and
// handed back rather than thrown.
type Issue struct {
	Line    int
	Message string
}

func (i Issue) String() string {
	return fmt.Sprintf("line %d: %s", i.Line, i.Message)
}

// DecodeFlags reads every coat of arms defined in a document.
//
// The root template block is skipped: its contents are building blocks for the
// games' random generation, not coats of arms this tool can edit.
func DecodeFlags(document *script.Document, origin Origin) ([]Flag, []Issue) {
	decoder := &decoder{}

	var flags []Flag

	for _, field := range document.Fields {
		if field.Key == keyTemplate {
			continue
		}

		if !field.Value.IsBlock() {
			continue
		}

		flagOrigin := origin
		flagOrigin.Line = field.Line
		flagOrigin.Key = field.Key

		flags = append(flags, decoder.flag(field.Key, field.Value, flagOrigin))
	}

	return flags, decoder.issues
}

// DecodePalette reads a named colour file, whose colours sit in a colors block.
//
// A colour may refer to one defined before it, so the palette is filled in as
// it is read.
func DecodePalette(document *script.Document) (Palette, []Issue) {
	decoder := &decoder{}
	palette := Palette{}

	block, ok := document.Get(keyColors)
	if !ok || block.Kind != script.KindBlock {
		return palette, nil
	}

	for _, field := range block.Fields {
		value, ok := decoder.colorValue(field.Value)
		if !ok {
			decoder.reportf(field.Line, "colour %q could not be read", field.Key)

			continue
		}

		resolved, ok := resolve(value, palette, nil, 0)
		if !ok {
			decoder.reportf(field.Line, "colour %q could not be resolved", field.Key)

			continue
		}

		palette[field.Key] = resolved
	}

	return palette, decoder.issues
}

type decoder struct {
	issues []Issue
}

func (d *decoder) reportf(line int, format string, args ...any) {
	d.issues = append(d.issues, Issue{Line: line, Message: fmt.Sprintf(format, args...)})
}

func (d *decoder) flag(name string, node script.Node, origin Origin) Flag {
	flag := Flag{Name: name, Origin: origin}

	// Fields are walked in order because the order of the layers is the order
	// they are drawn in.
	for _, field := range node.Fields {
		switch {
		case field.Key == keyPattern:
			if pattern, ok := field.Value.Str(); ok {
				flag.Pattern = pattern
			} else {
				d.reportf(field.Line, "pattern of %q is not a file name", name)
			}

		case isSlotName(field.Key):
			if color, ok := d.color(field.Key, field.Value); ok {
				flag.Colors.Set(color)
			}

		case field.Key == keyColoredEmblem:
			flag.Layers = append(flag.Layers, d.coloredEmblem(field.Value))

		case field.Key == keyTexturedEmblem:
			flag.Layers = append(flag.Layers, d.texturedEmblem(field.Value))

		case field.Key == keySub:
			flag.Layers = append(flag.Layers, d.subFlag(field.Value))
		}
	}

	return flag
}

func (d *decoder) coloredEmblem(node script.Node) *ColoredEmblem {
	emblem := &ColoredEmblem{}

	for _, field := range node.Fields {
		switch {
		case field.Key == keyTexture:
			emblem.Texture, _ = field.Value.Str()

		case isSlotName(field.Key):
			if color, ok := d.color(field.Key, field.Value); ok {
				emblem.Colors.Set(color)
			}

		case field.Key == keyMask:
			emblem.Mask = d.mask(field)

		case field.Key == keyInstance:
			emblem.Instances = append(emblem.Instances, d.instance(field))
		}
	}

	return emblem
}

func (d *decoder) texturedEmblem(node script.Node) *TexturedEmblem {
	emblem := &TexturedEmblem{}

	for _, field := range node.Fields {
		switch {
		case field.Key == keyTexture:
			emblem.Texture, _ = field.Value.Str()

		case field.Key == keyInstance:
			emblem.Instances = append(emblem.Instances, d.instance(field))
		}
	}

	return emblem
}

func (d *decoder) subFlag(node script.Node) *SubFlag {
	sub := &SubFlag{}

	for _, field := range node.Fields {
		switch {
		case field.Key == keyParent:
			sub.Parent, _ = field.Value.Str()

		case field.Key == keyInstance:
			sub.Instances = append(sub.Instances, d.subInstance(field))
		}
	}

	return sub
}

func (d *decoder) instance(field script.Field) Instance {
	instance := NewInstance()

	for _, attribute := range field.Value.Fields {
		switch attribute.Key {
		case keyPosition:
			if vector, ok := d.vector(attribute, instance.Position); ok {
				instance.Position = vector
			}

		case keyScale:
			if vector, ok := d.vector(attribute, instance.Scale); ok {
				instance.Scale = vector
			}

		case keyRotation:
			if rotation, ok := attribute.Value.Num(); ok {
				instance.Rotation = float32(rotation)
			} else {
				d.reportf(attribute.Line, "rotation is not a number")
			}
		}
	}

	return instance
}

func (d *decoder) subInstance(field script.Field) SubInstance {
	instance := NewSubInstance()

	for _, attribute := range field.Value.Fields {
		switch attribute.Key {
		case keyScale:
			if vector, ok := d.vector(attribute, instance.Scale); ok {
				instance.Scale = vector
			}

		case keyOffset, keyPosition:
			if vector, ok := d.vector(attribute, instance.Offset); ok {
				instance.Offset = vector
			}
		}
	}

	return instance
}

// vector reads a position or a scale, such as { 0.5 0.5 }.
//
// A single value sets the horizontal component and leaves the vertical one at
// its default. That is not a mistake in the files: the hoist stripes of Sulu
// are written scale = { 0.07 } and are meant to come out as full height bars a
// fourteenth of the flag wide.
func (d *decoder) vector(field script.Field, fallback Vec2) (Vec2, bool) {
	numbers, ok := field.Value.Numbers()
	if !ok {
		d.reportf(field.Line, "%s is not a list of numbers", field.Key)

		return Vec2{}, false
	}

	if len(numbers) == 0 {
		d.reportf(field.Line, "%s is empty", field.Key)

		return Vec2{}, false
	}

	if len(numbers) > 2 {
		d.reportf(field.Line, "%s has %d values, only the first two are used", field.Key, len(numbers))
	}

	vector := fallback
	vector.X = float32(numbers[0])

	if len(numbers) > 1 {
		vector.Y = float32(numbers[1])
	}

	return vector, true
}

func (d *decoder) mask(field script.Field) int {
	numbers, ok := field.Value.Numbers()
	if !ok || len(numbers) == 0 {
		d.reportf(field.Line, "mask is not a number")

		return 0
	}

	if len(numbers) > 1 {
		d.reportf(field.Line, "mask has %d values, only the first is used", len(numbers))
	}

	return int(numbers[0])
}

func (d *decoder) color(slot string, node script.Node) (Color, bool) {
	value, ok := d.colorValue(node)
	if !ok {
		d.reportf(node.Line, "%s could not be read", slot)

		return Color{}, false
	}

	return Color{Slot: slot, Value: value}, true
}

func (d *decoder) colorValue(node script.Node) (ColorValue, bool) {
	switch node.Kind {
	case script.KindString:
		// A quoted string always names a colour. An unquoted one naming a slot
		// is a reference to that slot, which is how a layer borrows a colour
		// from the flag it sits on.
		if !node.Quoted && isSlotName(node.Text) {
			return SlotColor{Slot: node.Text}, true
		}

		return NamedColor{Name: node.Text}, true

	case script.KindList:
		// An untagged triple is a literal colour.
		return d.channels(tagRGB, node)

	case script.KindTagged:
		if node.Value == nil {
			return nil, false
		}

		if node.Tag == tagList {
			name, ok := node.Value.Str()
			if !ok {
				return nil, false
			}

			return ListColor{List: name}, true
		}

		return d.channels(node.Tag, *node.Value)
	}

	return nil, false
}

// channels turns a triple of numbers into a colour, in whichever of the three
// spellings the file used.
func (d *decoder) channels(tag string, node script.Node) (ColorValue, bool) {
	numbers, ok := node.Numbers()
	if !ok || len(numbers) != 3 {
		return nil, false
	}

	first, second, third := numbers[0], numbers[1], numbers[2]

	switch tag {
	case tagRGB:
		// Channels are written either as fractions or as whole numbers up to
		// 255, and which one it is can only be told from the values.
		if first <= 1 && second <= 1 && third <= 1 {
			return RGBColor{
				R: uint8(first*255 + 0.5),
				G: uint8(second*255 + 0.5),
				B: uint8(third*255 + 0.5),
			}, true
		}

		return RGBColor{R: channelOf(first), G: channelOf(second), B: channelOf(third)}, true

	case tagHSV360:
		// Degrees, then two percentages.
		return HSVColor{H: float32(first), S: float32(second / 100), V: float32(third / 100)}, true

	case tagHSV:
		// Three fractions.
		return HSVColor{H: float32(first * 360), S: float32(second), V: float32(third)}, true
	}

	return nil, false
}

func channelOf(value float64) uint8 {
	return uint8(clamp(float32(value), 0, 255))
}
