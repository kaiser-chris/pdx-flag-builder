package pdx

import (
	"image/color"

	"github.com/kaiser-chris/pdx-parser-go/victoria3"
)

// The colours of a coat of arms are the parser library's, so that the editor
// works on the same values the files were read into.
//
// A colour slot holds a named colour, a reference to another slot, a literal
// colour in either spelling, or a random pick from a list, and resolving one
// needs the named colours of the folders and the slots it may refer back to.
// None of that is particular to this editor.
type (
	// ColorValue is one of the ways a colour slot can be filled in.
	ColorValue = victoria3.ColorValue

	// NamedColor refers to a colour from the game's named colour files.
	NamedColor = victoria3.NamedColor

	// SlotColor refers to another colour slot.
	SlotColor = victoria3.SlotColor

	// RGBColor is a literal colour.
	RGBColor = victoria3.RGBColor

	// HSVColor is a literal colour in hue, saturation and value.
	HSVColor = victoria3.HSVColor

	// ListColor picks a colour at random from a named list.
	ListColor = victoria3.ListColor

	// Color is a colour slot and what fills it.
	Color = victoria3.Color

	// Colors is the ordered set of colour slots of a flag or a layer.
	Colors = victoria3.Colors

	// Palette maps the names from the game's named colour files to colours.
	Palette = victoria3.Palette
)

// Fallback is what a colour resolves to when it cannot be worked out.
var Fallback = victoria3.FallbackColor

// SlotIndex returns the position of a colour slot, so that color1 is 0. It
// returns minus one for anything that is not a slot name.
func SlotIndex(slot string) int {
	return victoria3.ColorSlotIndex(slot)
}

// HSVToRGB converts a hue in degrees and a saturation and value between zero
// and one into a colour.
func HSVToRGB(hue, saturation, value float64) color.RGBA {
	return victoria3.HSVToRGB(hue, saturation, value)
}

// RGBToHSV is the inverse of HSVToRGB: hue in degrees, saturation and value
// between zero and one.
func RGBToHSV(value color.RGBA) (hue, saturation, brightness float64) {
	return victoria3.RGBToHSV(value)
}
