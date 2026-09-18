//go:build uitest

package app

import (
	"image/color"
	"path/filepath"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

func TestFlagDatabaseShowsRenderedFlags(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)

	flag := libraryFlag(t, application, "TST_split")
	thumbnail := waitForThumbnail(t, driver, func() (render.Thumbnail, bool) {
		return application.thumbnails.Flag(flagThumbnailKey(flag), flag)
	})

	// Rendered and recoloured, the right way up: the first colour on top.
	if got := thumbnailPixel(application, thumbnail, 0.5, 0.25); !near(got, fixtureBlue) {
		t.Errorf("top of the thumbnail = %v, want the flag's first colour %v", got, fixtureBlue)
	}

	if got := thumbnailPixel(application, thumbnail, 0.5, 0.75); !near(got, fixtureWhite) {
		t.Errorf("bottom of the thumbnail = %v, want the flag's second colour %v", got, fixtureWhite)
	}
}

func TestTextureDatabaseShowsBaseTextures(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowTextureDatabase)

	texture, ok := application.state.library.set.Texture("pattern_split.png")
	if !ok {
		t.Fatal("the fixture pattern is not in the library")
	}

	thumbnail := waitForThumbnail(t, driver, func() (render.Thumbnail, bool) {
		return application.thumbnails.Texture(texture.Path)
	})

	// The texture as the file has it, marker colours and all.
	if got := thumbnailPixel(application, thumbnail, 0.5, 0.25); !near(got, patternFirst) {
		t.Errorf("top of the thumbnail = %v, want the first marker colour %v", got, patternFirst)
	}

	if got := thumbnailPixel(application, thumbnail, 0.5, 0.75); !near(got, patternSecond) {
		t.Errorf("bottom of the thumbnail = %v, want the second marker colour %v", got, patternSecond)
	}
}

func TestColumnHeadersSortTheFlagDatabase(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)

	above := func(first, second string) bool {
		return driver.Find("", first).Min.Y < driver.Find("", second).Min.Y
	}

	if !above("TST_emblem", "TST_split") {
		t.Fatal("the flags are not sorted by name to begin with")
	}

	// TST_split has no layers and TST_emblem one.
	driver.Click(windowFlagDatabase+"/", "Layers")

	if !above("TST_split", "TST_emblem") {
		t.Error("sorting by layers did not put the flag without layers first")
	}

	driver.Click(windowFlagDatabase+"/", "Layers")

	if !above("TST_emblem", "TST_split") {
		t.Error("a second click did not reverse the order")
	}
}

func TestPickerShowsWhereTexturesComeFrom(t *testing.T) {
	application, driver := startApp(t, func(settings *config.Settings) {
		settings.Databases = append(settings.Databases, config.Database{Name: "mod", Path: newFixtureMod(t)})
	})

	driver.WaitFor("both folders to be read", func() bool {
		return len(application.state.library.set) == 2
	})

	driver.Menu("File", "New Flag")
	driver.Click("", labelAddLayer)
	driver.Click("", "Colored Emblem...")

	// The mod replaces the game's emblem, and the picker lists both, each with
	// its folder.
	if got := len(driver.FindAll("", "ce_square.png")); got != 2 {
		t.Fatalf("ce_square.png is listed %d times, want once for the game and once for the mod", got)
	}

	driver.Fill(driver.Find("", "##picker-search"), "mod")

	choices := driver.FindAll("", "ce_square.png")
	if len(choices) != 1 {
		t.Fatalf("searching for the mod's folder left %d rows, want 1", len(choices))
	}

	driver.ClickItem(choices[0])

	if layers := application.state.flag.Layers; len(layers) != 1 {
		t.Fatalf("got %d layers after choosing, want 1", len(layers))
	}
}

func TestFormLabelsAreOnTheLeft(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")

	field := driver.Find(panelSelected, "Name")

	window, found := gui.FindWindow(panelSelected)
	if !found {
		t.Fatal("the selected layer panel is not open")
	}

	left := window.Pos().X + imgui.CurrentStyle().WindowPadding().X
	right := window.Pos().X + window.Size().X - imgui.CurrentStyle().WindowPadding().X

	if field.Min.X < left+gui.Scaled(100) {
		t.Errorf("the name field starts at %v, want it right of its label, which starts at %v", field.Min.X, left)
	}

	if right-field.Max.X > 1 {
		t.Errorf("the name field ends at %v, want it to reach the edge of the panel at %v", field.Max.X, right)
	}

	if application.state.flag.Name != "TST_split" {
		t.Fatalf("open flag = %q, want TST_split", application.state.flag.Name)
	}
}

// newFixtureMod lays out a mod folder that replaces the fixture game's emblem.
func newFixtureMod(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "mod")

	writePNG(t, filepath.Join(root, "gfx", "coat_of_arms", "colored_emblems", "ce_square.png"), 32, 32,
		func(int, int) color.RGBA { return emblemFirst })

	return root
}

func libraryFlag(t *testing.T, application *App, name string) *pdx.Flag {
	t.Helper()

	for index := range application.state.library.flags {
		if flag := &application.state.library.flags[index]; flag.Name == name {
			return flag
		}
	}

	t.Fatalf("%s is not in the library", name)

	return nil
}

func waitForThumbnail(t *testing.T, driver *uitest.Driver, get func() (render.Thumbnail, bool)) render.Thumbnail {
	t.Helper()

	var thumbnail render.Thumbnail

	driver.WaitFor("the thumbnail to be drawn", func() bool {
		found, ok := get()
		thumbnail = found

		return ok
	})

	return thumbnail
}

// thumbnailPixel reads one pixel of a thumbnail, at a fraction of its width
// and height from its top left corner, the way the lists sample it.
func thumbnailPixel(application *App, thumbnail render.Thumbnail, across, down float32) color.RGBA {
	captured := rl.LoadImageFromTexture(application.thumbnails.Atlas().Texture)
	defer rl.UnloadImage(captured)

	// The image comes back in texture coordinates: the first row is V zero.
	u := thumbnail.U0 + (thumbnail.U1-thumbnail.U0)*across
	v := thumbnail.V0 + (thumbnail.V1-thumbnail.V0)*down

	return rl.GetImageColor(*captured, int32(u*float32(captured.Width)), int32(v*float32(captured.Height)))
}

func TestLayerRowsAreEvenlySized(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_emblem")

	coatOfArms := driver.Find(panelLayers, labelCoatOfArms)
	layer := driver.Find(panelLayers, describeLayer(application.state.flag.Layers[0]))

	height := func(item uitest.Item) float32 { return item.Max.Y - item.Min.Y }

	// The layer has buttons beside it and the coat of arms has none, which
	// must not make the rows differ.
	if height(coatOfArms) != height(layer) {
		t.Errorf("the coat of arms row is %v tall and the layer row %v, want them the same",
			height(coatOfArms), height(layer))
	}
}

func TestSettingsWindowFitsItsContents(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")
	driver.Frames(3)

	windowHeight := func() float32 {
		window, found := gui.FindWindow(windowSettings)
		if !found {
			t.Fatal("the settings window is not open")
		}

		return window.SizeFull().Y
	}

	assertAllVisible := func(when string) {
		t.Helper()

		for _, item := range driver.Items() {
			if item.Window == windowSettings && !item.Reachable(item.ClickPoint()) {
				t.Errorf("%s: %q is outside the visible part of the settings window", when, item.Label)
			}
		}
	}

	assertAllVisible("on opening")

	before, button := windowHeight(), driver.Find(windowSettings, "Add Folder")

	driver.Click(windowSettings, "Add Folder")
	driver.Frames(3)

	// The table has no height of its own: a new row pushes everything below it
	// down, and the window grows by as much instead of scrolling.
	moved := driver.Find(windowSettings, "Add Folder").Min.Y - button.Min.Y
	grew := windowHeight() - before

	if moved <= 0 {
		t.Fatalf("Add Folder stayed at %v after adding a folder, want the table to grow above it", button.Min.Y)
	}

	if grew < moved-1 || grew > moved+1 {
		t.Errorf("the window grew by %v while the table grew by %v, want the same", grew, moved)
	}

	assertAllVisible("after adding a folder")
}

func TestFolderColumnsLineUpWithTheirHeaders(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")

	// Every column's field is as far in from its header as every other's: the
	// first one is not pushed against the edge of the table.
	nameInset := driver.Find(windowSettings, "##name").Min.X - driver.Find(windowSettings, "Name").Min.X
	folderInset := driver.Find(windowSettings, "##path").Min.X - driver.Find(windowSettings, "Folder").Min.X

	if nameInset <= 0 || nameInset != folderInset {
		t.Errorf("the name field is %v in from its header and the folder field %v, want the same, above zero",
			nameInset, folderInset)
	}
}

func TestSettingsWindowCanBeWidened(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")
	driver.Frames(3)

	window, found := gui.FindWindow(windowSettings)
	if !found {
		t.Fatal("the settings window is not open")
	}

	before := window.SizeFull()
	corner := imgui.Vec2{X: window.Pos().X + before.X - 3, Y: window.Pos().Y + before.Y - 3}

	// The resize grip in the corner, dragged outwards both ways.
	driver.DragAt(corner, 150, 120)
	driver.Frames(2)

	after := window.SizeFull()

	if after.X < before.X+140 {
		t.Errorf("the window is %v wide after dragging its corner 150 to the right, was %v", after.X, before.X)
	}

	// The height stays what the contents need.
	if after.Y != before.Y {
		t.Errorf("the window is %v tall after the drag, want it to stay %v", after.Y, before.Y)
	}
}
