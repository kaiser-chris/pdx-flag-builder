//go:build uitest

package app

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// An icon is drawn into its button rather than set as its label, so that it is
// placed by its own ink. Dear ImGui cuts the fraction off the place it is told
// to draw text at rather than rounding it, which used to leave every icon up
// and to the left of the middle of its button.
//
// Only the icons that are the same either way round are measured, since what
// is left of the middle of an arrow is not what is right of it.
func TestIconButtonsHoldTheirIconsInTheMiddle(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)
	driver.Frames(3)

	captured := rl.LoadImageFromScreen()
	defer rl.UnloadImage(captured)

	colors := rl.LoadImageColors(captured)
	defer rl.UnloadImageColors(colors)

	width := int(captured.Width)

	buttons := map[string]uitest.Item{
		"the layer's delete":     driver.FindAll(panelLayers, labelRemove)[0],
		"the colour's delete":    driver.FindAll(panelSelected, labelRemoveColor)[0],
		"the layer's own delete": driver.Find(panelSelected, labelDeleteLayer),
	}

	for name, button := range buttons {
		// The ink of the icon is whatever differs from the button it is drawn
		// on, weighted by how much it differs: steadier than a threshold, and
		// it needs no guess at how faint an edge still counts.
		fill := colors[(int(button.Min.Y)+int(button.Max.Y))/2*width+int(button.Min.X)+1]

		var weight, across float64

		for y := int(button.Min.Y); y < int(button.Max.Y); y++ {
			for x := int(button.Min.X); x < int(button.Max.X); x++ {
				at := colors[y*width+x]

				difference := float64(distance(at.R, fill.R) + distance(at.G, fill.G) + distance(at.B, fill.B))
				if difference < 12 {
					continue
				}

				weight += difference
				across += difference * (float64(x) + 0.5)
			}
		}

		if weight == 0 {
			t.Errorf("%s has no icon on it", name)

			continue
		}

		middle := (float64(button.Min.X) + float64(button.Max.X)) / 2

		if off := across/weight - middle; off < -0.75 || off > 0.75 {
			t.Errorf("the icon on %s sits %+.2f pixels across from the middle of it", name, off)
		}
	}

	_ = application
}

// distance is how far apart two channels of a colour are.
func distance(got, want uint8) int {
	if got > want {
		return int(got) - int(want)
	}

	return int(want) - int(got)
}
