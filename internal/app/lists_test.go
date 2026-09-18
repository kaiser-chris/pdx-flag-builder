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

	window := imgui.InternalFindWindowByName(panelSelected)
	if window == nil {
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
