//go:build uitest

package app

import (
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// Marker colours the fixture textures are painted in, as the games paint them.
var (
	patternFirst  = color.RGBA{R: 255, A: 255}
	patternSecond = color.RGBA{R: 255, G: 255, A: 255}
	emblemFirst   = color.RGBA{B: 128, A: 255}

	// texturedMark is the colour of the fixture's textured emblem, which is
	// drawn as it is: nothing in a flag recolours it.
	texturedMark = color.RGBA{R: 200, G: 120, B: 40, A: 255}
)

// Colours the fixture's named colour file defines, deliberately pure so that a
// test can recognise them in the rendered flag.
var (
	fixtureBlue  = color.RGBA{B: 255, A: 255}
	fixtureWhite = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	fixtureGreen = color.RGBA{G: 255, A: 255}
)

const fixtureFlags = `
TST_split = {
	pattern = "pattern_split.png"
	color1 = "blue"
	color2 = "white"
}

TST_dxt = {
	pattern = "pattern_dxt.dds"
	color1 = "green"
}

TST_emblem = {
	pattern = "pattern_split.png"
	color1 = "blue"
	color2 = "blue"

	colored_emblem = {
		texture = "ce_square.png"
		color1 = "green"
		instance = { scale = { 0.5 0.5 } }
	}
}
`

const fixtureColors = `
colors = {
	blue = rgb { 0 0 255 }
	white = rgb { 255 255 255 }
	green = rgb { 0 255 0 }
}
`

// newFixtureGame lays out a tiny game folder: two coats of arms, three named
// colours, a pattern split into its two marker colours top and bottom, a
// square emblem in the first emblem marker colour and a textured emblem of one
// plain colour.
func newFixtureGame(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "game")

	writeFile(t, filepath.Join(root, "common", "coat_of_arms", "coat_of_arms", "00_test.txt"), fixtureFlags)
	writeFile(t, filepath.Join(root, "common", "named_colors", "00_colors.txt"), fixtureColors)

	writePNG(t, filepath.Join(root, "gfx", "coat_of_arms", "patterns", "pattern_split.png"), 64, 64,
		func(_, y int) color.RGBA {
			if y < 32 {
				return patternFirst
			}

			return patternSecond
		})

	writePNG(t, filepath.Join(root, "gfx", "coat_of_arms", "colored_emblems", "ce_square.png"), 32, 32,
		func(int, int) color.RGBA { return emblemFirst })

	writePNG(t, filepath.Join(root, "gfx", "coat_of_arms", "textured_emblems", "te_mark.png"), 32, 32,
		func(int, int) color.RGBA { return texturedMark })

	// The first pattern marker colour again, as DXT5 with a full chain of
	// mipmap levels, the way most of the base game's textures come.
	writeDXT5(t, filepath.Join(root, "gfx", "coat_of_arms", "patterns", "pattern_dxt.dds"), 768, 512, patternFirst)

	// Coloured on its left half only, so that a mirrored copy can be told from
	// one drawn the right way round.
	writePNG(t, filepath.Join(root, "gfx", "coat_of_arms", "textured_emblems", "te_left.png"), 32, 32,
		func(x, _ int) color.RGBA {
			if x < 16 {
				return texturedMark
			}

			return color.RGBA{}
		})

	return root
}

// startApp runs the real application in a hidden window, configured with the
// fixture game folder, and hands back a driver for it. configure can change
// the settings it starts with.
func startApp(t *testing.T, configure ...func(settings *config.Settings)) (*App, *uitest.Driver) {
	t.Helper()

	// raylib and OpenGL belong to the thread that created the window, and a
	// test runs on a goroutine the scheduler may move between threads.
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)

	configDir := filepath.Join(t.TempDir(), "config")

	settings := config.Default()
	settings.Databases = []config.Database{{Name: "game", Path: newFixtureGame(t)}}

	// A fixed scale, so that what the layout tests see does not depend on the
	// display of the machine running them.
	settings.InterfaceScale = 1

	for _, change := range configure {
		change(settings)
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("encode settings: %v", err)
	}
	writeFile(t, filepath.Join(configDir, "settings.json"), string(data))

	application, err := New(Options{ConfigDir: configDir, Hidden: true})
	if err != nil {
		t.Fatalf("start the application: %v", err)
	}
	t.Cleanup(application.Close)

	// A test never gets to see a real dialog: this one cancels whatever it is
	// asked, until a test hands it answers.
	application.dialogs = &fakeDialogs{}

	driver := uitest.New(t, application)

	driver.WaitFor("the fixture game to be read", func() bool {
		return !application.state.library.loading && len(application.state.library.flags) > 0
	})

	return application, driver
}

// waitForArtwork runs frames until every texture the open flag asked for has
// reached the GPU.
func waitForArtwork(t *testing.T, application *App, driver *uitest.Driver, count int) {
	t.Helper()

	driver.WaitFor("the flag's textures to load", func() bool {
		loaded, pending, _ := application.textures.Counts()

		return pending == 0 && loaded >= count
	})

	// One more frame so the flag is drawn with everything in place.
	driver.Frame()
}

// pixel reads one pixel of the rendered flag.
func pixel(application *App, x, y int) color.RGBA {
	return application.preview.Image().RGBAAt(x, y)
}

// near reports whether two colours are the same within a little filtering noise.
func near(got, want color.RGBA) bool {
	const slack = 6

	close := func(a, b uint8) bool {
		if a > b {
			return a-b <= slack
		}

		return b-a <= slack
	}

	return close(got.R, want.R) && close(got.G, want.G) && close(got.B, want.B)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func writePNG(t *testing.T, path string, width, height int, paint func(x, y int) color.RGBA) {
	t.Helper()

	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.SetRGBA(x, y, paint(x, y))
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()

	if err := png.Encode(file, picture); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}

// writeDXT5 writes a DXT5 compressed DDS file of one colour, with every
// mipmap level down to one pixel.
func writeDXT5(t *testing.T, path string, width, height int, fill color.RGBA) {
	t.Helper()

	// Both endpoints the same colour and every index zero: every pixel of
	// the block is the first endpoint.
	rgb565 := uint16(fill.R>>3)<<11 | uint16(fill.G>>2)<<5 | uint16(fill.B>>3)
	block := []byte{
		fill.A, fill.A, 0, 0, 0, 0, 0, 0, // alpha: both endpoints, all indices zero
		byte(rgb565), byte(rgb565 >> 8), byte(rgb565), byte(rgb565 >> 8), 0, 0, 0, 0,
	}

	levels := 1
	for size := max(width, height); size > 1; size /= 2 {
		levels++
	}

	header := make([]byte, 128)
	copy(header[0:4], "DDS ")
	binary.LittleEndian.PutUint32(header[4:], 124)
	binary.LittleEndian.PutUint32(header[8:], 0x000A1007) // caps, height, width, pixel format, mipmap count, linear size
	binary.LittleEndian.PutUint32(header[12:], uint32(height))
	binary.LittleEndian.PutUint32(header[16:], uint32(width))
	binary.LittleEndian.PutUint32(header[20:], uint32(width*height))
	binary.LittleEndian.PutUint32(header[28:], uint32(levels))
	binary.LittleEndian.PutUint32(header[76:], 32)
	binary.LittleEndian.PutUint32(header[80:], 0x4)
	copy(header[84:88], "DXT5")

	data := header

	for level := range levels {
		blocks := (max(width>>level, 1) + 3) / 4 * ((max(height>>level, 1) + 3) / 4)
		for range blocks {
			data = append(data, block...)
		}
	}

	writeFile(t, path, string(data))
}
