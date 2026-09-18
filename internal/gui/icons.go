package gui

import "github.com/AllenDang/cimgui-go/imgui"

// The interface has no icon font, so the one icon it needs is drawn.

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
