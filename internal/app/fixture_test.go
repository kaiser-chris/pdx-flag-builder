//go:build uitest

package app

import (
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
