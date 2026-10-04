package gui

import (
	"math"

	"github.com/AllenDang/cimgui-go/imgui"
)

// The icons of the interface come from its icon font. The eyedropper is drawn
// instead, because it is painted over a colour that decides its own.

// iconFill is how much of the box it is drawn in an icon takes up.
const iconFill = 0.9

// drawIcon draws an icon of the icon font in the middle of a box.
//
// It is placed by the icon's own ink rather than by the line of text it would
// otherwise sit on: an icon is a picture in a square, and a line of text has
// room above and below it for parts of letters this font has none of, which
// would leave the icon sitting high in its box.
func drawIcon(icon string, low, high imgui.Vec2, color imgui.Vec4) {
	font := imgui.CurrentFont()
	if font == nil || icon == "" {
		return
	}

	size := (high.Y - low.Y) * iconFill

	glyph := font.FontBaked(size).FindGlyph(imgui.Wchar([]rune(icon)[0]))
	if glyph == nil {
		return
	}

	// A disabled button is drawn faded, which Dear ImGui does by thinning
	// everything drawn while it is, and an icon drawn in a colour of its own
	// has to do the same to fade with the button it is on.
	color.W *= imgui.CurrentStyle().Alpha()

	// Where the ink of the glyph lands, relative to where the text is drawn
	// from, which is what centres it.
	//
	// The place it is drawn from is rounded to a whole pixel, because Dear
	// ImGui cuts the fraction off one rather than rounding it, which would
	// nudge every icon up and to the left of the middle it was given.
	at := imgui.Vec2{
		X: round((low.X+high.X)/2 - (glyph.X0()+glyph.X1())/2),
		Y: round((low.Y+high.Y)/2 - (glyph.Y0()+glyph.Y1())/2),
	}

	imgui.WindowDrawList().AddTextFontPtr(font, size, at, imgui.ColorU32Vec4(color), icon)
}

// drawPickerIcon draws an eyedropper into a square, the usual sign that
// clicking it opens a colour picker. It is drawn in black or white, whichever
// stands out against the colour behind it.
func drawPickerIcon(low, high imgui.Vec2, behind [3]float32) {
	size := high.X - low.X
	at := func(across, down float32) imgui.Vec2 {
		return imgui.Vec2{X: low.X + size*across, Y: low.Y + size*down}
	}

	color := imgui.ColorU32Vec4(imgui.Vec4{X: 1, Y: 1, Z: 1, W: 0.9})
	if luminance(behind) > 0.55 {
		color = imgui.ColorU32Vec4(imgui.Vec4{W: 0.75})
	}

	drawList := imgui.WindowDrawList()

	// The glass tube, from its tip at the lower left up to the collar.
	drawList.AddLineArgs(at(0.24, 0.76), at(0.56, 0.44), color, size*0.09)

	// The collar across the top of the tube.
	drawList.AddLineArgs(at(0.44, 0.36), at(0.64, 0.56), color, size*0.1)

	// The rubber bulb above it, with a rounded end.
	drawList.AddLineArgs(at(0.58, 0.42), at(0.72, 0.28), color, size*0.2)
	drawList.AddCircleFilled(at(0.72, 0.28), size*0.1, color)
}

// luminance is how light a colour looks, from zero to one.
func luminance(channels [3]float32) float32 {
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}

// round is the nearest whole number to a coordinate.
func round(value float32) float32 {
	return float32(math.Round(float64(value)))
}
