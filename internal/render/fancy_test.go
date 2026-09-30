//go:build uitest

package render

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/mesh/meshtest"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// clothFlag is a coat of arms the painter can draw without reading anything
// from disk: one pattern, which put has already handed to the GPU, in one
// colour.
func clothFlag() pdx.Flag {
	return pdx.Flag{
		Name:    "TST",
		Pattern: "pattern.png",
		Colors:  pdx.Colors{{Slot: "color1", Value: pdx.RGBColor{R: 255}}},
	}
}

// newFancy prepares the cloth with a painter that can draw that flag, and
// with a cloth of its own standing in for the game's.
func newFancy(t *testing.T) *Fancy {
	t.Helper()

	shader, err := LoadRecolor()
	if err != nil {
		t.Fatalf("LoadRecolor: %v", err)
	}

	t.Cleanup(shader.Unload)

	folder := t.TempDir()

	if err := os.WriteFile(filepath.Join(folder, ClothMesh), meshtest.Quad(9, 6), 0o644); err != nil {
		t.Fatal(err)
	}

	textures := NewTextures(func(string) (string, bool) { return "", false })
	t.Cleanup(textures.Unload)

	put(textures, "pattern.png")

	// The maps of the cloth's material. The game's are DDS files; what they
	// are written as here does not matter, since the loader goes by the name
	// it is given.
	for _, name := range []string{ClothDiffuse, ClothNormal, ClothProperties} {
		writePicture(t, filepath.Join(folder, name+".png"))
	}

	fancy, err := NewFancy(Victoria3.Cloth, NewPainter(shader, textures), func(name string) (string, bool) {
		if name == ClothMesh {
			return filepath.Join(folder, name), true
		}

		return filepath.Join(folder, name+".png"), true
	})
	if err != nil {
		t.Fatalf("NewFancy: %v", err)
	}

	t.Cleanup(fancy.Unload)

	return fancy
}

// writePicture writes a small white picture, which stands in for one of the
// cloth's material maps.
func writePicture(t *testing.T, path string) {
	t.Helper()

	picture := rl.GenImageColor(4, 4, rl.White)
	defer rl.UnloadImage(picture)

	if !rl.ExportImage(*picture, path) {
		t.Fatalf("write %s", path)
	}
}

// drawn is how many pixels of an image were drawn at all.
func drawn(picture *image.RGBA) int {
	count := 0

	bounds := picture.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if picture.RGBAAt(x, y).A > 0 {
				count++
			}
		}
	}

	return count
}

// The cloth is the game's own mesh, waved by the game's own maths, with the
// coat of arms drawn into it.
func TestFancyDrawsTheClothWithTheFlagOnIt(t *testing.T) {
	withOpenGL(t)

	fancy := newFancy(t)
	flag := clothFlag()

	fancy.Draw(&flag, 0)

	if !fancy.Ready() {
		t.Fatalf("the cloth was not read: %v", fancy.Problem())
	}

	cloth := fancy.Image()

	bounds := cloth.Bounds()
	if bounds.Dx() != int(Victoria3.Cloth.RenderWidth) || bounds.Dy() != int(Victoria3.Cloth.RenderHeight) {
		t.Fatalf("the cloth is %v, want the size the game draws it at", bounds)
	}

	// The cloth hangs in the middle of the target, covering much of it but
	// leaving the corners empty.
	covered := drawn(cloth)

	if covered < bounds.Dx()*bounds.Dy()/3 {
		t.Errorf("%d pixels of %d were drawn, want the cloth to fill much of it", covered, bounds.Dx()*bounds.Dy())
	}

	if corner := cloth.RGBAAt(2, 2); corner.A != 0 {
		t.Errorf("the corner of the target = %v, want it left empty", corner)
	}

	// The flag's colour is what the cloth is painted in, darkened by the
	// light and warmed by the adjustments the game ends on.
	middle := cloth.RGBAAt(bounds.Dx()/2, bounds.Dy()/2)

	if middle.A == 0 || middle.R <= middle.G || middle.R <= middle.B {
		t.Errorf("the middle of the cloth = %v, want the flag's red", middle)
	}
}

// The cloth waves: what it looks like depends on how long the window has been
// open.
func TestFancyWavesOverTime(t *testing.T) {
	withOpenGL(t)

	fancy := newFancy(t)
	flag := clothFlag()

	fancy.Draw(&flag, 0)
	first := fancy.Image()

	fancy.Draw(&flag, 0.7)
	second := fancy.Image()

	changed := 0

	bounds := first.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if first.RGBAAt(x, y) != second.RGBAAt(x, y) {
				changed++
			}
		}
	}

	if changed < bounds.Dx()*bounds.Dy()/50 {
		t.Errorf("%d pixels changed between two moments, want the cloth to have moved", changed)
	}
}

// A window with no flag open shows no cloth.
func TestFancyDrawsNothingWithoutAFlag(t *testing.T) {
	withOpenGL(t)

	fancy := newFancy(t)

	flag := clothFlag()
	fancy.Draw(&flag, 0)
	fancy.Draw(nil, 0)

	if covered := drawn(fancy.Image()); covered != 0 {
		t.Errorf("%d pixels were drawn without a flag, want none", covered)
	}
}
