//go:build uitest

package render

import (
	"image"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"

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

// stripeTexture is a pattern of fine stripes in the first two marker colours,
// at the size of the canvas: eight pixels of one and eight of the other, which
// is finer than the smallest icon has pixels for.
func stripeTexture(textures *Textures, name string) {
	picture := image.NewRGBA(image.Rect(0, 0, FlagWidth, FlagHeight))

	for y := range FlagHeight {
		for x := range FlagWidth {
			marker := PatternSlotColors[0]
			if x/8%2 == 1 {
				marker = PatternSlotColors[1]
			}

			picture.Set(int(x), int(y), marker)
		}
	}

	loaded := rl.NewImageFromImage(picture)
	defer rl.UnloadImage(loaded)

	uploaded := rl.LoadTextureFromImage(loaded)
	rl.SetTextureFilter(uploaded, rl.FilterBilinear)

	textures.loaded[name] = uploaded
	textures.used[name] = textures.generation
}

// A flag drawn at the smallest size a game draws one at holds the whole of the
// flag, rather than every fiftieth pixel of it.
//
// The stripes of the pattern are finer than that icon has pixels for, so every
// pixel of it has to come out half of one colour and half of the other. Drawn
// straight from the artwork at that size, each would be wholly one or wholly
// the other, depending on which pixel of the pattern it happened to land on.
func TestSmallIconsHoldTheWholeFlag(t *testing.T) {
	withOpenGL(t)

	shader, err := LoadRecolor()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(shader.Unload)

	textures := NewTextures(func(string) (string, bool) { return "", false })
	t.Cleanup(textures.Unload)

	stripeTexture(textures, "stripes.png")

	game := &Game{
		Name:  "test",
		Sizes: []IconSize{{Name: "tiny", Width: 27, Height: 18, FlagWidth: 27, FlagHeight: 18}},
	}

	icons := NewIcons(game, NewPainter(shader, textures), textures)
	t.Cleanup(icons.Unload)

	flag := pdx.Flag{
		Pattern: "stripes.png",
		Colors: pdx.Colors{
			{Slot: "color1", Value: pdx.RGBColor{}},
			{Slot: "color2", Value: pdx.RGBColor{R: 255, G: 255, B: 255}},
		},
	}

	icons.Draw(&flag, 1)

	picture := icons.Image(0)

	// The edges of the icon take some of the background with them, so the
	// middle of it is what is measured.
	for y := 4; y < 14; y++ {
		for x := 4; x < 23; x++ {
			got := picture.RGBAAt(x, y)

			if got.R < 80 || got.R > 175 {
				t.Fatalf("the pixel at %d,%d is %v, want the stripes averaged into a grey", x, y, got)
			}
		}
	}
}
