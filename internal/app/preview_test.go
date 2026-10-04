//go:build uitest

package app

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// previewOf is the window of one game.
func previewOf(t *testing.T, application *App, game *render.Game) *preview {
	t.Helper()

	for _, found := range application.previews {
		if found.game == game {
			return found
		}
	}

	t.Fatalf("there is no preview of %s", game.Name)

	return nil
}

// openPreview opens the window of a game with a flag in the editor.
func openPreview(t *testing.T, application *App, driver *uitest.Driver, game *render.Game, flag string) *preview {
	t.Helper()

	openFixture(t, application, driver, flag)
	waitForArtwork(t, application, driver, 1)

	driver.Menu(menuPreview, game.Name)
	driver.Frames(2)

	return previewOf(t, application, game)
}

// iconPixel reads one pixel of an icon of a preview.
func iconPixel(preview *preview, index, x, y int) color.RGBA {
	return preview.icons.Image(index).RGBAAt(x, y)
}

// flagArea is where the flag itself sits in an icon, which is in the middle
// of whatever the game draws around it.
func flagArea(size render.IconSize) (x, y int) {
	return int(size.Width-size.FlagWidth) / 2, int(size.Height-size.FlagHeight) / 2
}

// The preview shows the open flag at each of the sizes Victoria 3 draws a
// country's flag at.
func TestPreviewShowsTheFlagAtEverySize(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	if !preview.show {
		t.Fatal("the game's entry of the Preview menu did not open its window")
	}

	for index, size := range preview.game.Sizes {
		icon := preview.icons.Image(index)

		// The target is as large as the border, which is drawn around the
		// flag rather than inside it.
		if got := icon.Bounds(); got.Dx() != int(size.Width) || got.Dy() != int(size.Height) {
			t.Errorf("%s icon is %v, want %d by %d", size.Name, got, size.Width, size.Height)
		}

		// The upper half of the flag is the first of the fixture flag's two
		// colours, shaded by the overlay but still plainly blue.
		left, top := flagArea(size)
		x, y := left+int(size.FlagWidth)/2, top+int(size.FlagHeight)/4

		if got := icon.RGBAAt(x, y); got.B < 60 || got.R > 32 || got.G > 32 {
			t.Errorf("%s icon at %d,%d = %v, want the flag's blue", size.Name, x, y, got)
		}
	}
}

// The game shades every flag in its interface with an overlay it multiplies
// over it and frames it with the border of the country's rank. Both are the
// game's own artwork, bundled with the application.
func TestPreviewAppliesTheOverlayAndTheBorder(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	large := preview.game.Sizes[0]
	left, top := flagArea(large)

	// The overlay is multiplied over the flag, so the fixture's grey, which
	// is a little over half of full brightness, halves the flag's blue.
	blue := iconPixel(preview, 0, left+int(large.FlagWidth)/2, top+int(large.FlagHeight)/4)

	want := color.RGBA{B: uint8(int(interfaceGrey.B) * 255 / 255), A: 255}
	if !near(blue, want) {
		t.Errorf("the flag = %v, want its blue halved by the overlay to %v", blue, want)
	}

	// The border is drawn over the whole icon, the margin around the flag
	// included, which is where nothing else draws.
	if edge := iconPixel(preview, 0, int(large.Width)/2, 2); !near(edge, borderMarks[0]) {
		t.Errorf("the margin above the flag = %v, want the border of the first rank %v", edge, borderMarks[0])
	}

	// Every rank has a border of its own.
	if ranks := preview.icons.Ranks(); ranks != len(borderMarks) {
		t.Errorf("ranks = %d, want the %d the borders hold", ranks, len(borderMarks))
	}
}

// The ranks differ from one another, and the preview draws the one that is
// chosen.
func TestPreviewDrawsTheChosenRank(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	// The line of the border above the flag, which every rank draws in a
	// colour of its own.
	const line = 2

	middle := int(preview.game.Sizes[0].Width) / 2
	before := iconPixel(preview, 0, middle, line)

	// One button per rank, the chosen one pressed in.
	if !driver.Find(preview.title, "1").Checked || driver.Find(preview.title, "4").Checked {
		t.Error("the first rank is not the one shown as chosen")
	}

	driver.Click(preview.title, "4")

	if driver.Find(preview.title, "1").Checked || !driver.Find(preview.title, "4").Checked {
		t.Error("the button of the chosen rank is not the one pressed in")
	}
	driver.Frames(2)

	if preview.rank != 4 {
		t.Fatalf("rank = %d, want the one that was chosen", preview.rank)
	}

	after := iconPixel(preview, 0, middle, line)

	if !near(after, borderMarks[3]) {
		t.Errorf("the border = %v, want the fourth rank's %v, where it was %v", after, borderMarks[3], before)
	}
}

// The four sizes read as one flag at four sizes: each name level with the
// middle of its flag, and the smaller flags under the middle of the largest.
func TestPreviewLinesTheSizesUp(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	var middle float32

	for index, size := range preview.game.Sizes {
		name := driver.Find(preview.title, size.Name)
		flag := driver.Find(preview.title, size.Name+" flag")

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
func TestPreviewShowsTheCloth(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	driver.WaitFor("the cloth to be read from the folders", preview.fancy[0].Ready)
	driver.Frame()

	largest := preview.game.Cloths[0]

	name := driver.Find(preview.title, largest.Name)
	cloth := driver.Find(preview.title, largest.Name+" flag")

	if !within(centreY(name), centreY(cloth), 1) {
		t.Errorf("the name sits at %v, want it level with the middle of the cloth at %v", centreY(name), centreY(cloth))
	}

	if largest := driver.Find(preview.title, preview.game.Sizes[0].Name+" flag"); !within(centreX(cloth), centreX(largest), 1) {
		t.Errorf("the cloth is centred on %v and the largest flat flag on %v, want them lined up",
			centreX(cloth), centreX(largest))
	}

	// The cloth carries the flag: its upper half is the first of the fixture
	// flag's two colours, blue, however the light falls on it.
	picture := preview.fancy[0].Image()
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
func TestPreviewSaysWhichGameFilesAreMissing(t *testing.T) {
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

	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	if !driver.Exists(preview.title, labelWhatIsMissing) {
		driver.Dump()
		t.Fatal("the preview does not say that the game's own files are missing")
	}

	// It names them where they belong in a game folder, which is where
	// someone has to go and look for them.
	driver.Click(preview.title, labelWhatIsMissing)

	for _, file := range preview.game.Files() {
		if !driver.Exists(preview.title, file.Path) {
			driver.Dump()
			t.Fatalf("%s is missing but not listed under its path", file.Name)
		}
	}

	// What it can show is the flag itself, unshaded and unframed.
	large := preview.game.Sizes[0]
	left, top := flagArea(large)

	if got := iconPixel(preview, 0, left+int(large.FlagWidth)/2, top+int(large.FlagHeight)/4); !near(got, fixtureBlue) {
		t.Errorf("the flag = %v, want it drawn in its own colours %v without the shading", got, fixtureBlue)
	}

	if edge := iconPixel(preview, 0, int(large.Width)/2, 2); edge.A != 0 {
		t.Errorf("the margin around the flag = %v, want nothing drawn there without a border", edge)
	}

	// And nothing at all of the cloth, which is the game's from end to end.
	if preview.fancy[0].Ready() {
		t.Error("the cloth was drawn although the game's files are not there")
	}
}

// Europa Universalis 5 draws a flag the other way round from Victoria 3: the
// frame is the size of the widget and the flag is drawn inside it, at the
// size of the button that holds it.
func TestEuropaUniversalis5DrawsTheFlagInsideItsFrame(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.EuropaUniversalis5, "TST_split")

	country := preview.game.Sizes[0]

	if country.Width != 180 || country.Height != 120 {
		t.Fatalf("the largest size is %dx%d, want the game's 180x120", country.Width, country.Height)
	}

	// The flag fills most of the widget but not all of it, which is what
	// leaves room for the frame around it.
	if country.FlagWidth != 145 || country.FlagHeight != 97 {
		t.Errorf("the flag inside it is %dx%d, want most of the widget", country.FlagWidth, country.FlagHeight)
	}

	left, top := flagArea(country)

	// The flag is there, shaded by the overlay.
	if got := iconPixel(preview, 0, left+int(country.FlagWidth)/2, top+int(country.FlagHeight)/4); got.B < 60 || got.R > 32 {
		t.Errorf("the flag = %v, want its blue", got)
	}

	// The frame is drawn over the whole widget, which is outside the flag.
	if edge := iconPixel(preview, 0, int(country.Width)/2, 1); !near(edge, borderMarks[0]) {
		t.Errorf("the top of the widget = %v, want the frame %v", edge, borderMarks[0])
	}

	// Its frames are the same whatever a country's rank, so there is nothing
	// to choose between.
	if ranks := preview.icons.Ranks(); ranks != 0 {
		t.Errorf("ranks = %d, want none to choose from", ranks)
	}

	if driver.Exists(preview.title, labelRank) {
		t.Error("the rank border can be chosen, although every frame is the same")
	}
}

// Its round flag is cut to a mask instead of being framed.
func TestEuropaUniversalis5CutsItsRoundFlagToItsMask(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.EuropaUniversalis5, "TST_split")

	var round render.IconSize

	index := 0
	for at, size := range preview.game.Sizes {
		if size.Mask != "" && size.Border == "" {
			round, index = size, at
		}
	}

	if round.Name == "" {
		t.Fatal("the game has no round flag")
	}

	// The fixture's mask is solid, so the flag shows through all of it, and
	// nothing frames it.
	left, top := flagArea(round)

	if got := iconPixel(preview, index, left+int(round.FlagWidth)/2, top+int(round.FlagHeight)/4); got.B < 60 {
		t.Errorf("the round flag = %v, want the flag showing through its mask", got)
	}

	if corner := iconPixel(preview, index, 0, 0); corner.A != 0 {
		t.Errorf("the corner of the round flag = %v, want nothing outside the flag", corner)
	}
}

// Both games hang a flag on the same cloth, at sizes of their own.
func TestEachGameShowsItsOwnCloth(t *testing.T) {
	application, driver := startApp(t)

	for _, game := range render.Games {
		preview := openPreview(t, application, driver, game, "TST_split")

		driver.WaitFor("the cloth of "+game.Name, preview.fancy[0].Ready)
		driver.Frame()

		picture := preview.fancy[0].Image()

		largest := game.Cloths[0]

		if got := picture.Bounds(); got.Dx() != int(largest.RenderWidth) || got.Dy() != int(largest.RenderHeight) {
			t.Errorf("%s draws its cloth %v, want %dx%d", game.Name, got, largest.RenderWidth, largest.RenderHeight)
		}

		bounds := picture.Bounds()

		if upper := picture.RGBAAt(bounds.Dx()/2, bounds.Dy()/3); upper.A == 0 || upper.B <= upper.R {
			t.Errorf("%s: the upper half of the cloth = %v, want the flag's blue", game.Name, upper)
		}
	}
}

// Victoria 3 hangs a flag on the same cloth at two sizes, a large one and a
// normal one, each drawn from a picture of its own and shown at the fraction
// of it the game draws it down to.
func TestVictoriaShowsTheClothAtTwoSizes(t *testing.T) {
	application, driver := startApp(t)
	preview := openPreview(t, application, driver, &render.Victoria3, "TST_split")

	cloths := render.Victoria3.Cloths

	if len(cloths) != 2 || len(preview.fancy) != len(cloths) {
		t.Fatalf("Victoria 3 has %d cloths and the preview %d, want two of each", len(cloths), len(preview.fancy))
	}

	if cloths[0].Name != "Fancy Large" || cloths[1].Name != "Fancy Normal" {
		t.Errorf("the cloths are called %q and %q, want the large one first", cloths[0].Name, cloths[1].Name)
	}

	var rows []uitest.Item

	for index, cloth := range cloths {
		driver.WaitFor("the "+cloth.Name+" cloth", preview.fancy[index].Ready)
		driver.Frame()

		// Each is drawn into a picture of its own size.
		if got := preview.fancy[index].Image().Bounds(); got.Dx() != int(cloth.RenderWidth) || got.Dy() != int(cloth.RenderHeight) {
			t.Errorf("%s is drawn %v, want %dx%d", cloth.Name, got, cloth.RenderWidth, cloth.RenderHeight)
		}

		// And shown at the size the game scales that picture down to.
		row := driver.Find(preview.title, cloth.Name+" flag")
		rows = append(rows, row)

		if width := row.Max.X - row.Min.X; !within(width, float32(cloth.Width), 1) {
			t.Errorf("%s is shown %v wide, want %d", cloth.Name, width, cloth.Width)
		}

		if height := row.Max.Y - row.Min.Y; !within(height, float32(cloth.Height), 1) {
			t.Errorf("%s is shown %v tall, want %d", cloth.Name, height, cloth.Height)
		}
	}

	// The larger one is above the other, and both are centred with the rest.
	if rows[0].Min.Y >= rows[1].Min.Y {
		t.Error("the large cloth is below the normal one, want the largest at the top")
	}

	if !within(centreX(rows[0]), centreX(rows[1]), 1) {
		t.Errorf("the cloths are centred on %v and %v, want them lined up", centreX(rows[0]), centreX(rows[1]))
	}
}

// The window opens at the size of what is in it, so that every flag it holds
// can be seen without resizing it or scrolling.
func TestPreviewWindowShowsEverythingItHolds(t *testing.T) {
	application, driver := startApp(t)

	for _, game := range render.Games {
		preview := openPreview(t, application, driver, game, "TST_split")

		window, found := gui.FindWindow(preview.title)
		if !found {
			t.Fatalf("the preview of %s is not open", game.Name)
		}

		if scroll := window.ScrollMax().Y; scroll > 0 {
			t.Errorf("%s: the window scrolls by %v, want it tall enough for everything in it", game.Name, scroll)
		}

		low, high := window.Pos(), imgui.Vec2{
			X: window.Pos().X + window.Size().X,
			Y: window.Pos().Y + window.Size().Y,
		}

		for _, item := range driver.Items() {
			if item.Window != preview.title {
				continue
			}

			if item.Min.X < low.X || item.Min.Y < low.Y || item.Max.X > high.X || item.Max.Y > high.Y {
				t.Errorf("%s: %q lies at %v-%v, outside the window at %v-%v",
					game.Name, item.Label, item.Min, item.Max, low, high)
			}
		}

		// The cloths and the flat sizes are all of them there.
		for _, cloth := range game.Cloths {
			driver.Find(preview.title, cloth.Name+" flag")
		}

		for _, size := range game.Sizes {
			driver.Find(preview.title, size.Name+" flag")
		}

		preview.show = false
		driver.Frames(2)
	}
}
