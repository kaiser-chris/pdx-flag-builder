package gui

import "testing"

func TestClampScale(t *testing.T) {
	tests := map[float32]float32{
		0.5:  MinScale,
		1:    1,
		1.5:  1.5,
		2.25: 2.25,
		4:    MaxScale,
	}

	for scale, want := range tests {
		if got := clampScale(scale); got != want {
			t.Errorf("clampScale(%v) = %v, want %v", scale, got, want)
		}
	}
}

// The eyedropper is drawn dark on light colours and light on dark ones.
func TestLuminanceTellsLightFromDark(t *testing.T) {
	light := [][3]float32{{1, 1, 1}, {0.8, 0.75, 0.55}, {1, 1, 0}}
	dark := [][3]float32{{0, 0, 0}, {0.1, 0.2, 0.1}, {0.47, 0.13, 0.11}, {0, 0, 1}}

	for _, color := range light {
		if luminance(color) <= 0.55 {
			t.Errorf("%v counts as dark, want light", color)
		}
	}

	for _, color := range dark {
		if luminance(color) > 0.55 {
			t.Errorf("%v counts as light, want dark", color)
		}
	}
}
