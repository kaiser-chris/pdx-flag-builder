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

	// A new emblem covers the flag, in the marker colours of its own texture,
	// since it has no colours of its own yet.
	waitForArtwork(t, application, driver, 2)

	if got := pixel(application, 384, 400); !near(got, emblemFirst) {
		t.Errorf("lower half = %v, want the emblem's own marker colour %v", got, emblemFirst)
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

// A layer's colour may point at the flag's colour of the same number, which is
// what the games' files write more often than anything else. Pointing one
// colour of the layer at a slot must not take that slot away from the others.
func TestALayerColourMayReferToEveryColourOfItsFlag(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	// The emblem's own colour, switched from a named colour to a reference.
	driver.ClickItem(driver.FindAll(panelSelected, "##kind")[0])
	driver.Click("", "Slot")

	driver.ClickItem(driver.FindAll(panelSelected, "##slot")[0])

	// The flag has two colours, and the emblem's color1 may take either.
	for _, slot := range []string{"color1", "color2"} {
		if !driver.Exists("", slot) {
			t.Errorf("%s is not offered to refer to", slot)
		}
	}

	driver.Click("", "color2")

	emblem := application.state.flag.Layers[0].(*pdx.ColoredEmblem)
	if color, _ := emblem.Colors.Get("color1"); color.Value != (pdx.SlotColor{Slot: "color2"}) {
		t.Fatalf("color1 = %#v, want a reference to the flag's color2", color.Value)
	}

	driver.ClickItem(driver.FindAll(panelSelected, "##slot")[0])
	driver.Click("", "color1")

	if color, _ := emblem.Colors.Get("color1"); color.Value != (pdx.SlotColor{Slot: "color1"}) {
		t.Errorf("color1 = %#v, want a reference to the flag's color1", color.Value)
	}
}

// A layer the editor adds is new, and a new layer has no colours: the emblem
// shows the colours of its own texture until the user fills a slot in.
func TestANewLayerHasNoColours(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_split")

	driver.Click(panelLayers, labelAddLayer)
	driver.Click("", "Colored Emblem...")
	driver.Click("", "ce_square.png")

	added, ok := application.state.flag.Layers[0].(*pdx.ColoredEmblem)
	if !ok {
		t.Fatalf("layer = %#v, want a coloured emblem", application.state.flag.Layers[0])
	}

	if len(added.Colors) != 0 {
		t.Errorf("the new emblem starts with %+v, want no colours at all", added.Colors)
	}

	// From the texture database it is the same layer, added the same way.
	driver.Menu("Databases", windowTextureDatabase)
	driver.ClickItem(rowButton(t, driver, "ce_square.png", labelAddAsLayer))

	fromDatabase, ok := lastLayer(t, application).(*pdx.ColoredEmblem)
	if !ok {
		t.Fatalf("last layer = %#v, want a coloured emblem", lastLayer(t, application))
	}

	if len(fromDatabase.Colors) != 0 {
		t.Errorf("the emblem added from the database starts with %+v, want no colours at all", fromDatabase.Colors)
	}
}

// The buttons beside a layer's name copy it and throw it away.
func TestDuplicateAndDeleteTheSelectedLayer(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")

	selectLayer(t, application, driver, 0)

	flag := application.state.flag
	original, ok := flag.Layers[0].(*pdx.ColoredEmblem)
	if !ok {
		t.Fatalf("the fixture's layer is %T, want a coloured emblem", flag.Layers[0])
	}

	// They belong at the top of the panel, side by side and over at the right,
	// out of the way of the fields that edit the layer itself.
	duplicate := driver.Find(panelSelected, labelDuplicateLayer)
	remove := driver.Find(panelSelected, labelDeleteLayer)

	if duplicate.Min.Y != remove.Min.Y || duplicate.Max.X > remove.Min.X {
		t.Errorf("the buttons are at %v and %v, want them side by side in that order", duplicate.Min, remove.Min)
	}

	for _, item := range driver.Items() {
		if item.Window != panelSelected || item.Label == labelDuplicateLayer || item.Label == labelDeleteLayer {
			continue
		}

		if item.Min.Y < duplicate.Max.Y {
			t.Errorf("%q is level with the buttons at %v, want them alone at the top", item.Label, item.Min)
		}

		if item.Max.X > remove.Max.X {
			t.Errorf("%q reaches further right than the buttons, want them against the edge", item.Label)
		}
	}

	driver.Click(panelSelected, labelDuplicateLayer)

	if len(flag.Layers) != 2 {
		t.Fatalf("got %d layers after duplicating one, want 2", len(flag.Layers))
	}

	if flag.Layers[0] != original {
		t.Error("the copy was put under the layer it came from, want it over")
	}

	copied, ok := flag.Layers[1].(*pdx.ColoredEmblem)
	if !ok || copied == original {
		t.Fatalf("the copy is %#v, want a coloured emblem of its own", flag.Layers[1])
	}

	if application.state.selectedLayer != 1 {
		t.Errorf("selected layer = %d, want the copy at 1", application.state.selectedLayer)
	}

	if copied.Texture != original.Texture || len(copied.Colors) != len(original.Colors) {
		t.Errorf("the copy is %+v, want the same layer as %+v", copied, original)
	}

	// What the copy holds is its own: editing it leaves the original alone.
	copied.Instances[0].Position.X = 0.25
	copied.Colors[0].Slot = "color2"

	if original.Instances[0].Position.X == 0.25 || original.Colors[0].Slot == "color2" {
		t.Error("editing the copy changed the layer it came from")
	}

	driver.Click(panelSelected, labelDeleteLayer)

	if len(flag.Layers) != 1 || flag.Layers[0] != original {
		t.Fatalf("got %d layers after deleting the copy, want only the original", len(flag.Layers))
	}

	if application.state.selectedLayer != noLayer {
		t.Errorf("selection = %d after its layer was deleted, want the coat of arms", application.state.selectedLayer)
	}

	// Deleting it is a step of its own to undo, and so is duplicating it.
	driver.Menu("Edit", "Undo")

	if got := len(application.state.flag.Layers); got != 2 {
		t.Errorf("undo after deleting left %d layers, want the copy back", got)
	}

	driver.Menu("Edit", "Undo")

	if got := len(application.state.flag.Layers); got != 1 {
		t.Errorf("undo after duplicating left %d layers, want only the original", got)
	}
}
