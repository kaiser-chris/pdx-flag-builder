//go:build uitest

package app

import (
	"math"
	"strings"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

// focusPreview clicks the flag preview, which gives it the arrow keys.
func focusPreview(t *testing.T, driver *uitest.Driver) {
	t.Helper()

	window, found := gui.FindWindow(panelPreview)
	if !found {
		t.Fatal("the flag preview is not open")
	}

	position, size := window.Pos(), window.Size()
	driver.ClickAt(imgui.Vec2{X: position.X + size.X/2, Y: position.Y + size.Y/2})
}

func emblemInstance(t *testing.T, application *App, index int) pdx.Instance {
	t.Helper()

	emblem, ok := application.state.flag.Layers[0].(*pdx.ColoredEmblem)
	if !ok || index >= len(emblem.Instances) {
		t.Fatalf("layer 0 = %#v, want a coloured emblem with placement %d", application.state.flag.Layers[0], index+1)
	}

	return emblem.Instances[index]
}

func nearly(got, want float32) bool {
	return math.Abs(float64(got-want)) < 1e-5
}

func TestArrowKeysNudgeTheSelectedPlacement(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)
	focusPreview(t, driver)

	driver.Press(imgui.KeyRightArrow)

	if got := emblemInstance(t, application, 0).Position.X; !nearly(got, 0.5+nudgeStep) {
		t.Fatalf("position after Right = %v, want one step to the right of 0.5", got)
	}

	// Shift makes the step bigger, Ctrl scales, Alt turns.
	driver.Hold(imgui.ModShift)
	driver.Press(imgui.KeyDownArrow)
	driver.Release(imgui.ModShift)

	driver.Hold(imgui.ModCtrl)
	driver.Press(imgui.KeyDownArrow)
	driver.Release(imgui.ModCtrl)

	driver.Hold(imgui.ModAlt)
	driver.Press(imgui.KeyRightArrow)
	driver.Release(imgui.ModAlt)

	instance := emblemInstance(t, application, 0)

	if !nearly(instance.Position.Y, 0.5+nudgeStep*nudgeFaster) {
		t.Errorf("position after Shift+Down = %v, want ten steps down from 0.5", instance.Position.Y)
	}

	if !nearly(instance.Scale.Y, 0.5+nudgeStep) {
		t.Errorf("scale after Ctrl+Down = %v, want one step taller than 0.5", instance.Scale.Y)
	}

	if instance.Rotation != nudgeDegrees {
		t.Errorf("rotation after Alt+Right = %v, want %v", instance.Rotation, nudgeDegrees)
	}

	// Each press was a gesture of its own.
	driver.Shortcut(imgui.ModCtrl, imgui.KeyZ)

	if got := emblemInstance(t, application, 0).Rotation; got != 0 {
		t.Errorf("rotation after one undo = %v, want the turn undone", got)
	}
}

func TestHoldingAnArrowKeyIsOneUndoStep(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)
	focusPreview(t, driver)

	// Repeat right away rather than after the system's delay, which would make
	// the test wait on the clock.
	io := imgui.CurrentIO()
	delay, rate := io.KeyRepeatDelay(), io.KeyRepeatRate()
	io.SetKeyRepeatDelay(0)
	io.SetKeyRepeatRate(0.0001)
	t.Cleanup(func() {
		io.SetKeyRepeatDelay(delay)
		io.SetKeyRepeatRate(rate)
	})

	driver.Hold(imgui.KeyRightArrow)
	driver.WaitFor("the key to repeat", func() bool {
		return emblemInstance(t, application, 0).Position.X > 0.5+nudgeStep*2.5
	})
	driver.Release(imgui.KeyRightArrow)
	driver.Frame()

	driver.Shortcut(imgui.ModCtrl, imgui.KeyZ)

	if got := emblemInstance(t, application, 0).Position.X; !nearly(got, 0.5) {
		t.Errorf("position after one undo = %v, want the whole hold undone back to 0.5", got)
	}
}

func TestArrowKeysLeaveTheFlagAloneWithoutFocus(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	driver.Press(imgui.KeyRightArrow)

	if got := emblemInstance(t, application, 0).Position.X; got != 0.5 {
		t.Errorf("position = %v after an arrow key outside the preview, want it untouched", got)
	}
}

func TestReorderPlacements(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	driver.Click(panelSelected, "Add Placement")

	if application.state.selectedPlacement != 1 {
		t.Fatalf("selected placement = %d after adding one, want the new one", application.state.selectedPlacement)
	}

	// The new placement goes first, and stays the selected one.
	driver.ClickItem(driver.FindAll(panelSelected, labelPlacementUp)[1])

	if got := emblemInstance(t, application, 0); got != pdx.NewInstance() {
		t.Errorf("first placement = %+v, want the new default one moved up", got)
	}

	if application.state.selectedPlacement != 0 {
		t.Errorf("selected placement = %d, want it to follow the placement it moved with", application.state.selectedPlacement)
	}

	// The arrow keys act on the selected placement, not the first.
	driver.Click(panelSelected, "Placement 2")
	focusPreview(t, driver)
	driver.Press(imgui.KeyLeftArrow)

	if got := emblemInstance(t, application, 1).Position.X; !nearly(got, 0.5-nudgeStep) {
		t.Errorf("second placement = %v, want it moved by the arrow key", got)
	}
}

func TestSettingsWarnAboutUnsavedChanges(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")

	if driver.Exists(windowSettings, labelUnsavedSettings) {
		t.Fatal("the settings warn about changes before anything was changed")
	}

	driver.Click(windowSettings, "Add Folder")

	if !driver.Exists(windowSettings, labelUnsavedSettings) {
		t.Error("adding a folder did not warn that it is not saved yet")
	}

	driver.Click(windowSettings, "Revert")

	if driver.Exists(windowSettings, labelUnsavedSettings) {
		t.Error("the warning stayed after reverting")
	}
}

func TestTextureDatabaseShowsSizes(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowTextureDatabase)

	texture, _ := application.state.library.set.Texture("pattern_split.png")

	driver.WaitFor("the texture's size", func() bool { return application.textureSize(texture.Path) != "" })

	if got := application.textureSize(texture.Path); got != "64 × 64" {
		t.Errorf("size = %q, want the fixture's 64 × 64", got)
	}
}

func TestEscapeAndEnterAnswerTheUnsavedChangesPrompt(t *testing.T) {
	application, driver := startApp(t)

	openFixture(t, application, driver, "TST_split")
	driver.Fill(driver.Find(panelSelected, "Name"), "TST_kept")

	driver.Menu("Settings", "Open Settings")
	driver.Click(windowSettings, "Add Folder")
	driver.Menu("File", "New Flag")

	// Escape cancels the prompt, and only the prompt.
	driver.Press(imgui.KeyEscape)

	if application.state.flag.Name != "TST_kept" || application.state.pending != nil {
		t.Fatalf("Escape went on with the new flag: open flag %q", application.state.flag.Name)
	}

	if !application.state.showSettings {
		t.Error("Escape closed the settings behind the prompt as well")
	}

	// Enter saves, then goes on. The flag was renamed, so a second Enter
	// answers the question that raises: it is added beside the one it came
	// from rather than taking its place.
	driver.Menu("File", "New Flag")
	driver.Press(imgui.KeyEnter)
	driver.Press(imgui.KeyEnter)

	if application.state.flag.Name != "new_flag" {
		t.Errorf("open flag = %q after Enter, want the new flag", application.state.flag.Name)
	}

	if saved := readText(t, fixtureFile(application)); !strings.Contains(saved, "TST_kept = {") {
		t.Errorf("Enter did not save the changes first:\n%s", saved)
	}
}

// With the lock closed, changing one half of a pair changes the other by the
// same amount, and the two stay as far apart as they were.
func TestLockedPairMovesTogether(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	emblem := application.state.flag.Layers[0].(*pdx.ColoredEmblem)
	emblem.Instances[0].Scale = pdx.Vec2{X: 0.5, Y: 0.75}
	driver.Frames(2)

	// Unlocked, the other half stays where it is.
	driver.Drag(driver.Find(panelSelected, scaleFields.x), 40, 0)

	scale := emblem.Instances[0].Scale
	if scale.X <= 0.5 || scale.Y != 0.75 {
		t.Fatalf("dragging one half of an unlocked pair gave %v, want only the first changed", scale)
	}

	emblem.Instances[0].Scale = pdx.Vec2{X: 0.5, Y: 0.75}
	driver.Frames(2)

	driver.Click(panelSelected, scaleFields.lock)

	if !application.state.lockScale {
		t.Fatal("the lock button did not close the lock")
	}

	// Locked, dragging the first half takes the second with it.
	driver.Drag(driver.Find(panelSelected, scaleFields.x), 40, 0)

	scale = emblem.Instances[0].Scale
	if scale.X <= 0.5 {
		t.Fatalf("dragging right gave %v, want the first half larger", scale)
	}

	if difference := scale.Y - scale.X; difference < 0.2499 || difference > 0.2501 {
		t.Errorf("the pair is %v, want the two of them a quarter apart as they started", scale)
	}

	// And dragging the second half takes the first with it, the other way.
	before := scale

	driver.Drag(driver.Find(panelSelected, scaleFields.y), -40, 0)

	scale = emblem.Instances[0].Scale
	if scale.Y >= before.Y || scale.X >= before.X {
		t.Errorf("dragging the second half left gave %v, want both smaller than %v", scale, before)
	}

	// The lock holds for the position as well, and each has its own.
	if application.state.lockPosition {
		t.Error("closing the lock on the scale closed the one on the position too")
	}
}

// Neither half of a locked pair may be taken past its limit, since that would
// leave the two of them apart.
func TestLockedPairStopsAtTheLimit(t *testing.T) {
	application, driver := startApp(t)
	openFixture(t, application, driver, "TST_emblem")
	selectLayer(t, application, driver, 0)

	application.state.lockScale = true

	emblem := application.state.flag.Layers[0].(*pdx.ColoredEmblem)

	// The second half is a hair short of its limit, so there is almost no room
	// for either of them to move.
	before := pdx.Vec2{X: 1, Y: pdx.MaxScale - 0.05}
	emblem.Instances[0].Scale = before
	driver.Frames(2)

	// Far enough to the right to take it past that limit several times over.
	driver.Drag(driver.Find(panelSelected, scaleFields.x), 400, 0)

	scale := emblem.Instances[0].Scale

	if scale.Y != pdx.MaxScale {
		t.Errorf("the second half is %v, want it stopped at its limit %v", scale.Y, pdx.MaxScale)
	}

	gap, wanted := scale.Y-scale.X, before.Y-before.X
	if math.Abs(float64(gap-wanted)) > 0.001 {
		t.Errorf("the pair is %v, want the two of them %v apart as they started", scale, wanted)
	}

	if scale.X <= before.X {
		t.Errorf("the first half is %v, want it to have taken the room the second had", scale.X)
	}
}
