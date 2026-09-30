//go:build uitest

package app

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// iconPixel reads one pixel of an icon of the Victoria 3 preview.
func iconPixel(application *App, index, x, y int) color.RGBA {
	return application.icons.Image(index).RGBAAt(x, y)
}

// flagArea is where the flag itself sits in an icon, which is in the middle of
// the border drawn around it.
func flagArea(size render.IconSize) (x, y int) {
	return int(size.BorderWidth-size.Width) / 2, int(size.BorderHeight-size.Height) / 2
}

// The preview shows the open flag at each of the sizes Victoria 3 draws a
// country's flag at.
func TestVictoria3PreviewShowsTheFlagAtEverySize(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, labelVictoria3)

	if !application.state.showVictoria3 {
		t.Fatal("the Victoria 3 entry of the Preview menu did not open the window")
	}

	driver.Frames(2)

	for index, size := range render.IconSizes {
		icon := application.icons.Image(index)

		// The target is as large as the border, which is drawn around the
		// flag rather than inside it.
		if got := icon.Bounds(); got.Dx() != int(size.BorderWidth) || got.Dy() != int(size.BorderHeight) {
			t.Errorf("%s icon is %v, want %d by %d", size.Name, got, size.BorderWidth, size.BorderHeight)
		}

		// The upper half of the flag is the first of the fixture flag's two
		// colours, shaded by the overlay but still plainly blue.
		left, top := flagArea(size)
		x, y := left+int(size.Width)/2, top+int(size.Height)/4

		if got := icon.RGBAAt(x, y); got.B < 60 || got.R > 32 || got.G > 32 {
			t.Errorf("%s icon at %d,%d = %v, want the flag's blue", size.Name, x, y, got)
		}
	}
}

// The game shades every flag in its interface with an overlay it multiplies
// over it and frames it with the border of the country's rank. Both are the
// game's own artwork, bundled with the application.
func TestVictoria3PreviewAppliesTheOverlayAndTheBorder(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, labelVictoria3)
	driver.Frames(2)

	large := render.IconSizes[0]
	left, top := flagArea(large)

	// The overlay is multiplied over the flag, so the fixture's grey, which
	// is a little over half of full brightness, halves the flag's blue.
	blue := iconPixel(application, 0, left+int(large.Width)/2, top+int(large.Height)/4)

	want := color.RGBA{B: uint8(int(interfaceGrey.B) * 255 / 255), A: 255}
	if !near(blue, want) {
		t.Errorf("the flag = %v, want its blue halved by the overlay to %v", blue, want)
	}

	// The border is drawn over the whole icon, the margin around the flag
	// included, which is where nothing else draws.
	if edge := iconPixel(application, 0, int(large.BorderWidth)/2, 2); !near(edge, borderMarks[0]) {
		t.Errorf("the margin above the flag = %v, want the border of the first rank %v", edge, borderMarks[0])
	}

	// Every rank has a border of its own.
	if ranks := application.icons.Ranks(); ranks != len(borderMarks) {
		t.Errorf("ranks = %d, want the %d the borders hold", ranks, len(borderMarks))
	}
}

// The ranks differ from one another, and the preview draws the one that is
// chosen.
func TestVictoria3PreviewDrawsTheChosenRank(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, labelVictoria3)
	driver.Frames(2)

	// The line of the border above the flag, which every rank draws in a
	// colour of its own.
	const line = 2

	before := iconPixel(application, 0, int(render.IconSizes[0].BorderWidth)/2, line)

	// One button per rank, the chosen one pressed in.
	if !driver.Find(windowVictoria3, "1").Checked || driver.Find(windowVictoria3, "4").Checked {
		t.Error("the first rank is not the one shown as chosen")
	}

	driver.Click(windowVictoria3, "4")

	if driver.Find(windowVictoria3, "1").Checked || !driver.Find(windowVictoria3, "4").Checked {
		t.Error("the button of the chosen rank is not the one pressed in")
	}
	driver.Frames(2)

	if application.state.previewRank != 4 {
		t.Fatalf("rank = %d, want the one that was chosen", application.state.previewRank)
	}

	after := iconPixel(application, 0, int(render.IconSizes[0].BorderWidth)/2, line)

	if !near(after, borderMarks[3]) {
		t.Errorf("the border = %v, want the fourth rank's %v, where it was %v", after, borderMarks[3], before)
	}
}

// The four sizes read as one flag at four sizes: each name level with the
// middle of its flag, and the smaller flags under the middle of the largest.
func TestVictoria3PreviewLinesTheSizesUp(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, labelVictoria3)
	driver.Frames(2)

	var middle float32

	for index, size := range render.IconSizes {
		name := driver.Find(windowVictoria3, size.Name)
		flag := driver.Find(windowVictoria3, size.Name+" flag")

		if got, want := centreY(name), centreY(flag); !within(got, want, 1) {
			t.Errorf("the name %s sits at %v, want it level with the middle of its flag at %v", size.Name, got, want)
		}

		if index == 0 {
			middle = centreX(flag)

			continue
		}

		if got := centreX(flag); !within(got, middle, 1) {
			t.Errorf("the %s flag is centred on %v, want the %v of the largest", size.Name, got, middle)
		}
	}
}

func centreX(item uitest.Item) float32 { return (item.Min.X + item.Max.X) / 2 }
func centreY(item uitest.Item) float32 { return (item.Min.Y + item.Max.Y) / 2 }

func within(got, want, slack float32) bool {
	return got-want <= slack && want-got <= slack
}

// The fancy flag, the waving cloth the game hangs a flag on, sits above the
// flat sizes and is centred with them.
func TestVictoria3PreviewShowsTheFancyFlag(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, labelVictoria3)
	driver.WaitFor("the cloth to be read from the folders", application.fancy.Ready)
	driver.Frame()

	name := driver.Find(windowVictoria3, labelFancy)
	cloth := driver.Find(windowVictoria3, labelFancy+" flag")

	if !within(centreY(name), centreY(cloth), 1) {
		t.Errorf("the name sits at %v, want it level with the middle of the cloth at %v", centreY(name), centreY(cloth))
	}

	if largest := driver.Find(windowVictoria3, render.IconSizes[0].Name+" flag"); !within(centreX(cloth), centreX(largest), 1) {
		t.Errorf("the cloth is centred on %v and the largest flat flag on %v, want them lined up",
			centreX(cloth), centreX(largest))
	}

	// The cloth carries the flag: its upper half is the first of the fixture
	// flag's two colours, blue, however the light falls on it.
	picture := application.fancy.Image()
	bounds := picture.Bounds()

	upper := picture.RGBAAt(bounds.Dx()/2, bounds.Dy()/3)
	if upper.A == 0 || upper.B <= upper.R || upper.B <= upper.G {
		t.Errorf("the upper half of the cloth = %v, want the flag's blue", upper)
	}

	// And its lower half the second, white, which comes out bright.
	lower := picture.RGBAAt(bounds.Dx()/2, 2*bounds.Dy()/3)
	if lower.A == 0 || lower.R < 120 || lower.G < 100 || lower.B < 100 {
		t.Errorf("the lower half of the cloth = %v, want the flag's white", lower)
	}
}

// The shading, the borders and the cloth are the game's own files. A set of
// folders with only mods in it has none of them, and the preview says so
// rather than pretending.
func TestVictoria3PreviewSaysWhichGameFilesAreMissing(t *testing.T) {
	application, driver := startApp(t)

	// The fixture folder without the game's own artwork, which is what a mod
	// folder looks like.
	root := application.settings.Databases[0].Path
	for _, folder := range []string{filepath.Join("gfx", "interface"), filepath.Join("gfx", "models")} {
		if err := os.RemoveAll(filepath.Join(root, folder)); err != nil {
			t.Fatal(err)
		}
	}

	driver.Menu("Databases", labelReload)
	driver.WaitFor("the folders to be read again", func() bool { return !application.state.library.loading })

	openFixture(t, application, driver, "TST_split")
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, labelVictoria3)
	driver.Frames(2)

	if !driver.Exists(windowVictoria3, labelWhatIsMissing) {
		driver.Dump()
		t.Fatal("the preview does not say that the game's own files are missing")
	}

	// What it can show is the flag itself, unshaded and unframed.
	large := render.IconSizes[0]
	left, top := flagArea(large)

	if got := iconPixel(application, 0, left+int(large.Width)/2, top+int(large.Height)/4); !near(got, fixtureBlue) {
		t.Errorf("the flag = %v, want it drawn in its own colours %v without the shading", got, fixtureBlue)
	}

	if edge := iconPixel(application, 0, int(large.BorderWidth)/2, 2); edge.A != 0 {
		t.Errorf("the margin around the flag = %v, want nothing drawn there without a border", edge)
	}

	// And nothing at all of the cloth, which is the game's from end to end.
	if application.fancy.Ready() {
		t.Error("the cloth was drawn although the game's files are not there")
	}
}
