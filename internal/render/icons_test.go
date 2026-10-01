//go:build uitest

package render

import (
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// An icon is a flag in the middle of a larger picture, with room around it for
// whatever the game draws there. An emblem placed partly outside the flag has
// to stop at its edge all the same, since the game draws its icons from a coat
// of arms that stops there.
func TestIconsKeepEmblemsInsideTheFlag(t *testing.T) {
	withOpenGL(t)

	shader, err := LoadRecolor()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(shader.Unload)

	textures := NewTextures(func(string) (string, bool) { return "", false })
	t.Cleanup(textures.Unload)

	put(textures, "emblem.png")

	game := &Game{
		Name:  "test",
		Sizes: []IconSize{{Name: "one", Width: 40, Height: 40, FlagWidth: 20, FlagHeight: 20}},
	}

	icons := NewIcons(game, NewPainter(shader, textures), textures)
	t.Cleanup(icons.Unload)

	// An emblem three times the size of the flag, which without clipping
	// would cover the whole picture.
	flag := pdx.Flag{
		Layers: []pdx.Layer{
			&pdx.TexturedEmblem{
				Texture: "emblem.png",
				Instances: []pdx.Instance{{
					Position: pdx.Vec2{X: 0.5, Y: 0.5},
					Scale:    pdx.Vec2{X: 3, Y: 3},
				}},
			},
		},
	}

	icons.Draw(&flag, 1)

	picture := icons.Image(0)

	// The flag sits in the middle of the picture, ten pixels in on each side.
	const inset = 10

	for y := range 40 {
		for x := range 40 {
			inside := x >= inset && x < 40-inset && y >= inset && y < 40-inset

			if got := picture.RGBAAt(x, y); inside != (got.A > 0) {
				if inside {
					t.Fatalf("the pixel at %d,%d is empty, want the flag drawn there", x, y)
				}

				t.Fatalf("the pixel at %d,%d is %v, want nothing outside the flag", x, y, got)
			}
		}
	}
}
