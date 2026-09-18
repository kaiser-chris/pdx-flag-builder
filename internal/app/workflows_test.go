//go:build uitest

package app

import (
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// rowButton finds the button of a table row, the one level with the row's
// name, since every row has a button of the same label.
func rowButton(t *testing.T, driver *uitest.Driver, row, button string) uitest.Item {
	t.Helper()

	name := driver.Find("", row)
	middle := (name.Min.Y + name.Max.Y) / 2

	for _, item := range driver.FindAll("", button) {
		if item.Min.Y <= middle && middle <= item.Max.Y {
			return item
		}
	}

	t.Fatalf("no %q button in the row of %q", button, row)

	return uitest.Item{}
}

func lastLayer(t *testing.T, application *App) pdx.Layer {
	t.Helper()

	layers := application.state.flag.Layers
	if len(layers) == 0 {
		t.Fatal("the flag has no layers")
	}

	return layers[len(layers)-1]
}

func TestAddAsSubFlagFromTheFlagDatabase(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_emblem")
	driver.ClickItem(rowButton(t, driver, "TST_split", labelAddAsSubFlag))

	sub, ok := lastLayer(t, application).(*pdx.SubFlag)
	if !ok || sub.Parent != "TST_split" {
		t.Fatalf("last layer = %#v, want a sub flag of TST_split", lastLayer(t, application))
	}

	if application.state.selectedLayer != len(application.state.flag.Layers)-1 || !application.state.modified {
		t.Error("the new sub flag is not selected, or the flag does not count as changed")
	}
}

func TestSetAsPatternFromTheTextureDatabase(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("File", "New Flag")
	driver.Menu("Databases", windowTextureDatabase)
	driver.Click("", labelSetPattern)

	if application.state.flag.Pattern != "pattern_split.png" {
		t.Errorf("pattern = %q, want the one set from the texture database", application.state.flag.Pattern)
	}
}

func TestSubFlagIsDrawn(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("File", "New Flag")
	driver.Click("", labelAddLayer)
	driver.Click("", "Sub Flag...")

	// The picker lists coats of arms here, not textures.
	driver.Click("", "TST_split")

	if sub, ok := lastLayer(t, application).(*pdx.SubFlag); !ok || sub.Parent != "TST_split" {
		t.Fatalf("last layer = %#v, want a sub flag of TST_split", lastLayer(t, application))
	}

	waitForArtwork(t, application, driver, 1)

	// With no placement of its own, the sub flag covers the whole flag.
	if got := pixel(application, 384, 100); !near(got, fixtureBlue) {
		t.Errorf("top = %v, want TST_split's first colour %v", got, fixtureBlue)
	}

	if got := pixel(application, 384, 400); !near(got, fixtureWhite) {
		t.Errorf("bottom = %v, want TST_split's second colour %v", got, fixtureWhite)
	}
}

func TestTexturedEmblemIsDrawnAsItIs(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("File", "New Flag")
	driver.Click("", labelAddLayer)
	driver.Click("", "Textured Emblem...")
	driver.Click("", "te_mark.png")

	if emblem, ok := lastLayer(t, application).(*pdx.TexturedEmblem); !ok || emblem.Texture != "te_mark.png" {
		t.Fatalf("last layer = %#v, want a textured emblem of te_mark.png", lastLayer(t, application))
	}

	waitForArtwork(t, application, driver, 1)

	// Nothing recolours a textured emblem.
	if got := pixel(application, 384, 256); !near(got, texturedMark) {
		t.Errorf("middle = %v, want the texture's own colour %v", got, texturedMark)
	}
}

func TestChooseANamedColourAndRemoveOne(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")

	// The first colour's named colour picker, narrowed down by its search.
	driver.ClickItem(driver.FindAll(panelSelected, "##named")[0])
	driver.Fill(driver.Find("", "##colour-search"), "gre")

	if driver.Exists("", "white") {
		t.Error("the colour search still lists white after searching for gre")
	}

	driver.Click("", "green")

	if got, _ := application.state.flag.Colors.Get("color1"); got.Value != (pdx.NamedColor{Name: "green"}) {
		t.Errorf("color1 = %#v, want the green picked", got.Value)
	}

	// Each colour row can go again.
	driver.ClickItem(driver.FindAll(panelSelected, "Remove")[1])

	if _, found := application.state.flag.Colors.Get("color2"); found || len(application.state.flag.Colors) != 1 {
		t.Errorf("colours after removing color2 = %#v", application.state.flag.Colors)
	}
}

func TestRemoveAFolder(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")
	driver.Click(windowSettings, "Add Folder")
	driver.ClickItem(driver.FindAll(windowSettings, "X")[0])

	if len(application.state.databases) != 1 || application.state.databases[0].Name != "" {
		t.Errorf("folders = %+v, want only the new, empty row left", application.state.databases)
	}
}

func TestResetLayoutReopensThePanels(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("View", panelLayers)

	if application.state.showLayers {
		t.Fatal("the layers panel did not close from the View menu")
	}

	driver.Menu("View", "Reset Layout")

	if !application.state.showLayers || !application.state.showSelected {
		t.Error("resetting the layout did not bring the panels back")
	}
}

// Where a flag is saved is not an edit: undoing past a save does not take the
// flag back to having no file.
func TestUndoKeepsWhereTheFlagIsSaved(t *testing.T) {
	application, driver := startApp(t)

	target := filepath.Join(t.TempDir(), "saved.txt")
	answerDialogs(application, target)

	driver.Menu("File", "New Flag")
	driver.Fill(driver.Find(panelSelected, "Name"), "renamed")
	driver.Menu("File", labelSave)
	driver.WaitFor("the save", func() bool { return application.state.flag.Origin.Path != "" })

	driver.Menu("Edit", "Undo")

	if application.state.flag.Name != "new_flag" {
		t.Errorf("name after undo = %q, want the rename undone", application.state.flag.Name)
	}

	if application.state.flag.Origin.Path != target {
		t.Errorf("file after undo = %q, want it to stay %q", application.state.flag.Origin.Path, target)
	}
}

// failingDialogs stands for a system without a working file dialog, such as a
// Linux desktop without zenity or kdialog. It counts how often it was asked.
type failingDialogs struct {
	asked atomic.Int32
}

func (f *failingDialogs) choose(fileRequest) (string, error) {
	f.asked.Add(1)

	return "", errors.New("no dialog program found")
}

func TestAFileDialogThatFailsIsReported(t *testing.T) {
	application, driver := startApp(t)
	dialogs := &failingDialogs{}
	application.dialogs = dialogs

	openFixture(t, application, driver, "TST_split")
	driver.Menu("File", labelSaveToFile)
	driver.WaitFor("the dialog's answer", func() bool { return application.state.dialog == nil })

	if !strings.Contains(application.state.status, "no dialog program found") {
		t.Errorf("status = %q, want it to say why no dialog came up", application.state.status)
	}

	// A failed dialog does not stay open in the application's eyes, which
	// would refuse every dialog after it.
	driver.Menu("File", labelSaveToFile)
	driver.WaitFor("the second answer", func() bool { return application.state.dialog == nil })

	if asked := dialogs.asked.Load(); asked != 2 {
		t.Errorf("the dialog was asked for %d times, want a second try to ask again", asked)
	}
}

// A negative scale mirrors an emblem where it stands, as the games do: the
// Byzantine flag's left emblems are its right ones turned around.
func TestNegativeScaleMirrorsAnEmblemInPlace(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("File", "New Flag")
	driver.Click("", labelAddLayer)
	driver.Click("", "Textured Emblem...")
	driver.Click("", "te_left.png")

	// Half the flag wide, centred on its left half, mirrored.
	emblem := lastLayer(t, application).(*pdx.TexturedEmblem)
	emblem.Instances = []pdx.Instance{{Position: pdx.Vec2{X: 0.25, Y: 0.5}, Scale: pdx.Vec2{X: -0.5, Y: 1}}}

	waitForArtwork(t, application, driver, 1)

	// The texture is coloured on its left half; mirrored, that half lands on
	// the right of the emblem, which covers the left half of the flag.
	for _, check := range []struct {
		x       int
		colored bool
		why     string
	}{
		{288, true, "the emblem's right quarter holds the texture's coloured half"},
		{96, false, "the emblem's left quarter holds the texture's empty half"},
		{480, false, "nothing is drawn right of the emblem"},
	} {
		if got := pixel(application, check.x, 256); near(got, texturedMark) != check.colored {
			t.Errorf("pixel at x %d = %v: %s", check.x, got, check.why)
		}
	}
}

// A sub flag with a negative scale is mirrored as a whole.
func TestNegativeScaleMirrorsASubFlag(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("File", "New Flag")
	driver.Click("", labelAddLayer)
	driver.Click("", "Sub Flag...")
	driver.Click("", "TST_split")

	// Upside down: placed from the bottom edge, growing upwards.
	sub := lastLayer(t, application).(*pdx.SubFlag)
	sub.Instances = []pdx.SubInstance{{Offset: pdx.Vec2{X: 0, Y: 1}, Scale: pdx.Vec2{X: 1, Y: -1}}}

	waitForArtwork(t, application, driver, 1)

	if got := pixel(application, 384, 100); !near(got, fixtureWhite) {
		t.Errorf("top = %v, want TST_split's bottom colour %v on top", got, fixtureWhite)
	}

	if got := pixel(application, 384, 400); !near(got, fixtureBlue) {
		t.Errorf("bottom = %v, want TST_split's top colour %v at the bottom", got, fixtureBlue)
	}
}
