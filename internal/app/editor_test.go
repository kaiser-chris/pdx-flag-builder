//go:build uitest

package app

import (
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// openFixture opens one of the fixture flags from the flag database, and
// closes the database again so that it does not cover the panels.
func openFixture(t *testing.T, application *App, driver *uitest.Driver, name string) {
	t.Helper()

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", name)

	if application.state.flag == nil || application.state.flag.Name != name {
		t.Fatalf("open flag = %v, want %s", application.state.flag, name)
	}

	application.state.showFlagDatabase = false
	driver.Frames(2)
}

// selectLayer clicks a layer in the layer list. Layers using the same texture
// read the same, so the row is picked among the ones with its label by how many
// alike layers come before it.
func selectLayer(t *testing.T, application *App, driver *uitest.Driver, index int) {
	t.Helper()

	layers := application.state.flag.Layers
	label := describeLayer(layers[index])

	alike := 0
	for _, earlier := range layers[:index] {
		if describeLayer(earlier) == label {
			alike++
		}
	}

	driver.ClickItem(driver.FindAll(panelLayers, label)[alike])

	if application.state.selectedLayer != index {
		t.Fatalf("selected layer = %d, want %d", application.state.selectedLayer, index)
	}
}

func TestAddLayerThroughThePicker(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")

	driver.Click(panelLayers, labelAddLayer)
	driver.Click("", "Colored Emblem...")
	driver.Click("", "ce_square.png")

	flag := application.state.flag

	if len(flag.Layers) != 1 {
		t.Fatalf("got %d layers, want the new one", len(flag.Layers))
	}

	emblem, ok := flag.Layers[0].(*pdx.ColoredEmblem)
	if !ok || emblem.Texture != "ce_square.png" {
		t.Fatalf("layer = %#v, want a coloured emblem with ce_square.png", flag.Layers[0])
	}

	if application.state.selectedLayer != 0 || !application.state.modified {
		t.Errorf("selected %d, modified %v; want the new layer selected and the flag modified",
			application.state.selectedLayer, application.state.modified)
	}

	// A new emblem covers the flag and borrows its first colour, so the white
	// lower half turns blue.
	waitForArtwork(t, application, driver, 2)

	if got := pixel(application, 384, 400); !near(got, fixtureBlue) {
		t.Errorf("lower half = %v, want the flag's first colour %v under the new emblem", got, fixtureBlue)
	}
}

func TestReorderAndRemoveLayers(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")

	// A second layer, from the texture database this time.
	driver.Menu("Databases", windowTextureDatabase)
	driver.ClickItem(rowButton(t, driver, "ce_square.png", labelAddAsLayer))
	application.state.showTextureDatabase = false
	driver.Frames(2)

	flag := application.state.flag
	if len(flag.Layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(flag.Layers))
	}

	first, second := flag.Layers[0], flag.Layers[1]

	// Move the first layer down: the two swap, and the selection follows.
	selectLayer(t, application, driver, 0)
	driver.ClickItem(driver.FindAll(panelLayers, labelMoveDown)[0])

	if flag.Layers[0] != second || flag.Layers[1] != first {
		t.Fatal("moving the first layer down did not swap the two layers")
	}

	if application.state.selectedLayer != 1 {
		t.Errorf("selection stayed at %d, want it to follow the layer to 1", application.state.selectedLayer)
	}

	// Remove the top one.
	driver.ClickItem(driver.FindAll(panelLayers, labelRemove)[1])

	if len(flag.Layers) != 1 || flag.Layers[0] != second {
		t.Fatalf("after removing the top layer got %d layers", len(flag.Layers))
	}

	if application.state.selectedLayer != noLayer {
		t.Errorf("selection = %d after its layer was removed, want the coat of arms", application.state.selectedLayer)
	}
}

func TestUndoAndRedo(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")

	driver.Click(panelLayers, labelRemove)

	if len(application.state.flag.Layers) != 0 {
		t.Fatal("the layer was not removed")
	}

	driver.Menu("Edit", "Undo")

	if len(application.state.flag.Layers) != 1 {
		t.Fatal("undo did not bring the layer back")
	}

	driver.Shortcut(imgui.ModCtrl, imgui.KeyY)

	if len(application.state.flag.Layers) != 0 {
		t.Fatal("Ctrl+Y did not remove the layer again")
	}

	driver.Shortcut(imgui.ModCtrl, imgui.KeyZ)

	if len(application.state.flag.Layers) != 1 {
		t.Fatal("Ctrl+Z did not bring the layer back")
	}
}

func TestDraggingAPlacementIsOneUndoStep(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	emblem := application.state.flag.Layers[0].(*pdx.ColoredEmblem)
	before := emblem.Instances[0].Position.X

	driver.Drag(driver.Find(panelSelected, "Position"), 60, 0)

	after := application.state.flag.Layers[0].(*pdx.ColoredEmblem).Instances[0].Position.X
	if after <= before {
		t.Fatalf("dragging right moved the position from %v to %v, want it larger", before, after)
	}

	// The drag changed the value on many frames, but undoing it once has to
	// put the emblem back where the drag started.
	driver.Menu("Edit", "Undo")

	if got := application.state.flag.Layers[0].(*pdx.ColoredEmblem).Instances[0].Position.X; got != before {
		t.Errorf("one undo left the position at %v, want it back at %v", got, before)
	}
}

func TestMaskRestrictsTheEmblem(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	driver.Click(panelSelected, "Mask")
	driver.Click("", "Pattern colour 2")

	if mask := application.state.flag.Layers[0].(*pdx.ColoredEmblem).Mask; mask != 2 {
		t.Fatalf("mask = %d, want 2", mask)
	}

	waitForArtwork(t, application, driver, 2)

	// The half size emblem spans the middle of the flag. Masked to the lower
	// half of the pattern, it only shows below the middle.
	if got := pixel(application, 384, 200); !near(got, fixtureBlue) {
		t.Errorf("upper part of the emblem = %v, want the pattern %v showing through", got, fixtureBlue)
	}

	if got := pixel(application, 384, 320); !near(got, fixtureGreen) {
		t.Errorf("lower part of the emblem = %v, want the emblem colour %v", got, fixtureGreen)
	}
}

func TestSwitchColourToRGB(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")

	// The coat of arms is selected when a flag opens, so its colours show.
	driver.ClickItem(driver.FindAll(panelSelected, "##kind")[0])
	driver.Click("", "RGB")

	first, _ := application.state.flag.Colors.Get("color1")

	if value, ok := first.Value.(pdx.RGBColor); !ok || value != (pdx.RGBColor{B: 255}) {
		t.Errorf("color1 = %#v, want the same blue spelled as rgb", first.Value)
	}
}

func TestUnsavedChangesAreNotDiscardedSilently(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")

	driver.Click(panelSelected, "Add Colour")

	if !application.state.modified {
		t.Fatal("adding a colour did not mark the flag as modified")
	}

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_emblem")

	if !driver.Exists("", labelDiscard) {
		t.Fatal("opening another flag over unsaved changes did not ask first")
	}

	driver.Click("", labelCancel)

	if application.state.flag.Name != "TST_split" {
		t.Fatalf("cancelling still opened %s", application.state.flag.Name)
	}

	driver.Click("", "TST_emblem")
	driver.Click("", labelDiscard)

	if application.state.flag.Name != "TST_emblem" || application.state.modified {
		t.Errorf("after discarding, open flag = %s modified %v; want TST_emblem unmodified",
			application.state.flag.Name, application.state.modified)
	}
}

func TestNewFlag(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("File", "New Flag")

	if application.state.flag == nil || application.state.flag.Name != "new_flag" {
		t.Fatalf("open flag = %v, want a new one", application.state.flag)
	}

	driver.Click(panelSelected, "Change...##Pattern")
	driver.Click("", "pattern_split.png")

	if application.state.flag.Pattern != "pattern_split.png" {
		t.Errorf("pattern = %q, want the one picked", application.state.flag.Pattern)
	}
}

func TestAboutOpensFromTheHelpMenu(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Help", "About")

	if !driver.Exists("", "Close") {
		t.Error("the about dialog did not open")
	}
}

// Every control of the editing panels has to be reachable in the default
// window size. Rows that grow wider than their panel push their last buttons
// off the edge of the window, where no one can click them. A panel taller
// than the window is fine: it scrolls, but only up and down.
func TestEditingPanelsFitTheWindow(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")

	check := func(what string) {
		t.Helper()

		for _, item := range driver.OffScreen() {
			if width := imgui.CurrentIO().DisplaySize().X; item.Min.X >= 0 && item.Max.X <= width {
				continue
			}

			t.Errorf("%s: %q in %q reaches past the window at (%.0f,%.0f)-(%.0f,%.0f)",
				what, item.Label, item.Window, item.Min.X, item.Min.Y, item.Max.X, item.Max.Y)
		}
	}

	check("the coat of arms")

	selectLayer(t, application, driver, 0)
	check("a coloured emblem")
}
