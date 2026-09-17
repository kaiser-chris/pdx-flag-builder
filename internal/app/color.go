package app

import "image/color"

// colorFromFloats converts the 0..1 channels Dear ImGui's colour editor works
// with back into the 8 bit channels the settings file stores.
func colorFromFloats(channels [4]float32) color.RGBA {
	return color.RGBA{
		R: uint8(channels[0]*255 + 0.5),
		G: uint8(channels[1]*255 + 0.5),
		B: uint8(channels[2]*255 + 0.5),
		A: uint8(channels[3]*255 + 0.5),
	}
}
