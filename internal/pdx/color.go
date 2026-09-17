// Package pdx holds the coat of arms model and reads it from script files.
//
// A coat of arms is a pattern with a stack of layers drawn over it. Layers and
// the flag itself carry numbered colour slots, and a layer can refer back to a
// slot of the flag it belongs to, so colours are only resolved once everything
// has been read.
package pdx

import (
	"fmt"
	"image/color"
	"math"
)

// ColorValue is one of the ways a colour slot can be filled in.
type ColorValue interface {
	// Describe returns a short label for the interface to show.
	Describe() string

	colorValue()
}

// NamedColor refers to a colour defined in the game's named colour files.
type NamedColor struct {
	Name string
}

// SlotColor refers to another colour slot, which is how a layer borrows a
// colour from the flag it sits on, as in color1 = color2.
type SlotColor struct {
	Slot string
}

// RGBColor is a literal colour.
type RGBColor struct {
	R, G, B uint8
}

// HSVColor is a literal colour in hue, saturation and value. Hue is in degrees
// and the other two are fractions, whichever spelling the file used.
type HSVColor struct {
	H, S, V float32
}

// ListColor picks a colour at random from a named list. The games use it in
// their randomly generated coats of arms; there is no single colour behind it,
// so it renders as a placeholder.
type ListColor struct {
	List string
}

func (NamedColor) colorValue() {}
func (SlotColor) colorValue()  {}
func (RGBColor) colorValue()   {}
func (HSVColor) colorValue()   {}
func (ListColor) colorValue()  {}

func (c NamedColor) Describe() string { return c.Name }
func (c SlotColor) Describe() string  { return c.Slot }
func (c RGBColor) Describe() string   { return fmt.Sprintf("rgb %d %d %d", c.R, c.G, c.B) }
func (c HSVColor) Describe() string {
	return fmt.Sprintf("hsv %.0f %.0f%% %.0f%%", c.H, c.S*100, c.V*100)
}
func (c ListColor) Describe() string { return "list " + c.List }

// Color is a colour slot and what fills it.
type Color struct {
	// Slot is the name the file used, from color1 to color9.
	Slot  string
	Value ColorValue
}

// Colors is the ordered set of colour slots of a flag or a layer.
type Colors []Color

// Get returns the colour in the given slot.
func (c Colors) Get(slot string) (Color, bool) {
	for _, current := range c {
		if current.Slot == slot {
			return current, true
		}
	}

	return Color{}, false
}

// Set replaces the colour in a slot, or appends it when the slot is not filled
// in yet. Slots stay in the order they were first seen.
func (c *Colors) Set(color Color) {
	for index := range *c {
		if (*c)[index].Slot == color.Slot {
			(*c)[index] = color

			return
		}
	}

	*c = append(*c, color)
}

// Palette maps the names from the game's named colour files to real colours.
type Palette map[string]color.RGBA

// maxSlotDepth bounds how far a chain of slot references is followed. A file
// can point color1 at color2 and color2 back at color1, and the games cope with
// that by giving up rather than looping.
const maxSlotDepth = 8

// Fallback is what a colour resolves to when it cannot be worked out: an
// unknown name, an unresolvable reference, or a random list.
var Fallback = color.RGBA{R: 255, G: 0, B: 255, A: 255}

// Resolve works out the colour of a slot.
//
// slots is the set the value may refer back to, which for a layer is the colour
// slots of the flag the layer belongs to. It reports whether the colour could be
// determined; when it could not, the fallback colour is returned so that the
// flag still draws.
func (c Color) Resolve(palette Palette, slots Colors) (color.RGBA, bool) {
	return resolve(c.Value, palette, slots, 0)
}

func resolve(value ColorValue, palette Palette, slots Colors, depth int) (color.RGBA, bool) {
	if depth > maxSlotDepth {
		return Fallback, false
	}

	switch typed := value.(type) {
	case RGBColor:
		return color.RGBA{R: typed.R, G: typed.G, B: typed.B, A: 255}, true

	case HSVColor:
		return HSVToRGB(typed.H, typed.S, typed.V), true

	case NamedColor:
		if found, ok := palette[typed.Name]; ok {
			return found, true
		}

	case SlotColor:
		if referenced, ok := slots.Get(typed.Slot); ok {
			return resolve(referenced.Value, palette, slots, depth+1)
		}

	case ListColor:
		// A random pick has no single answer.
		return Fallback, false
	}

	return Fallback, false
}

// HSVToRGB converts a hue in degrees and a saturation and value between zero
// and one into a colour.
func HSVToRGB(hue, saturation, value float32) color.RGBA {
	hue = float32(math.Mod(float64(hue), 360))
	if hue < 0 {
		hue += 360
	}

	saturation = clamp(saturation, 0, 1)
	value = clamp(value, 0, 1)

	chroma := value * saturation
	sector := hue / 60
	second := chroma * (1 - float32(math.Abs(math.Mod(float64(sector), 2)-1)))
	match := value - chroma

	var red, green, blue float32

	switch int(sector) {
	case 0:
		red, green, blue = chroma, second, 0
	case 1:
		red, green, blue = second, chroma, 0
	case 2:
		red, green, blue = 0, chroma, second
	case 3:
		red, green, blue = 0, second, chroma
	case 4:
		red, green, blue = second, 0, chroma
	default:
		red, green, blue = chroma, 0, second
	}

	return color.RGBA{
		R: channel(red + match),
		G: channel(green + match),
		B: channel(blue + match),
		A: 255,
	}
}

func channel(value float32) uint8 {
	return uint8(clamp(value*255, 0, 255) + 0.5)
}

func clamp(value, low, high float32) float32 {
	return min(max(value, low), high)
}

// slotNames are the colour slots a flag or layer can fill in.
var slotNames = []string{
	"color1", "color2", "color3", "color4", "color5",
	"color6", "color7", "color8", "color9",
}

// isSlotName reports whether a key names a colour slot.
func isSlotName(key string) bool {
	for _, name := range slotNames {
		if key == name {
			return true
		}
	}

	return false
}

// NextFreeSlot returns the first colour slot not filled in yet, which is what a
// newly added colour takes. It returns an empty string when all slots are used.
func (c Colors) NextFreeSlot() string {
	for _, name := range slotNames {
		if _, taken := c.Get(name); !taken {
			return name
		}
	}

	return ""
}
