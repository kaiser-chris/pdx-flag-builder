package app

import (
	"strconv"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// arrowKeys are the keys that nudge a placement.
var arrowKeys = []imgui.Key{imgui.KeyLeftArrow, imgui.KeyRightArrow, imgui.KeyUpArrow, imgui.KeyDownArrow}

// nudge changes the selected placement of the selected layer with the arrow
// keys, the way the Odin version's instance editor did: on their own they
// move it, with Ctrl they scale it, and with Alt, left and right turn it.
// Shift makes every step bigger. It runs while the flag preview has focus,
// where the arrow keys have nothing else to do.
func (a *App) nudge() {
	a.state.nudging = false

	for _, key := range arrowKeys {
		if imgui.IsKeyDown(key) {
			a.state.nudging = true
		}
	}

	// Pressed rather than down, so that a tap is exactly one step, and a key
	// held down repeats at the rate the system repeats keys at.
	var across, down float32

	if imgui.IsKeyPressedBool(imgui.KeyLeftArrow) {
		across--
	}

	if imgui.IsKeyPressedBool(imgui.KeyRightArrow) {
		across++
	}

	if imgui.IsKeyPressedBool(imgui.KeyUpArrow) {
		down--
	}

	if imgui.IsKeyPressedBool(imgui.KeyDownArrow) {
		down++
	}

	if across == 0 && down == 0 {
		return
	}

	layer, ok := a.state.selectedLayerValue()
	if !ok {
		a.setStatus("Select a layer to move it with the arrow keys")

		return
	}

	io := imgui.CurrentIO()

	scale := float32(1)
	if io.KeyShift() {
		scale = nudgeFaster
	}

	switch typed := layer.(type) {
	case *pdx.ColoredEmblem:
		a.nudgeInstance(&typed.Instances, across*scale, down*scale, io.KeyCtrl(), io.KeyAlt())

	case *pdx.TexturedEmblem:
		a.nudgeInstance(&typed.Instances, across*scale, down*scale, io.KeyCtrl(), io.KeyAlt())

	case *pdx.SubFlag:
		// A sub flag cannot be turned, so Alt does nothing for it.
		if io.KeyAlt() {
			return
		}

		a.nudgeSubInstance(&typed.Instances, across*scale, down*scale, io.KeyCtrl())
	}
}

// nudgeInstance moves, scales or turns an emblem's selected placement by a
// number of steps. An emblem drawn at the implied default placement gets that
// placement written out first, since there is nothing else to move.
func (a *App) nudgeInstance(instances *[]pdx.Instance, across, down float32, scaling, turning bool) {
	if len(*instances) == 0 {
		*instances = append(*instances, pdx.NewInstance())
	}

	instance := &(*instances)[a.clampPlacement(len(*instances))]

	switch {
	case turning:
		if across == 0 {
			return
		}

		instance.Rotation = clampTo(instance.Rotation+across*nudgeDegrees, -pdx.MaxRotation, pdx.MaxRotation)

	case scaling:
		instance.Scale.X = clampTo(instance.Scale.X+across*nudgeStep, pdx.MinScale, pdx.MaxScale)
		instance.Scale.Y = clampTo(instance.Scale.Y+down*nudgeStep, pdx.MinScale, pdx.MaxScale)

	default:
		instance.Position.X = clampTo(instance.Position.X+across*nudgeStep, pdx.MinPosition, pdx.MaxPosition)
		instance.Position.Y = clampTo(instance.Position.Y+down*nudgeStep, pdx.MinPosition, pdx.MaxPosition)
	}

	a.changed()
}

// nudgeSubInstance moves or scales a sub flag's selected placement.
func (a *App) nudgeSubInstance(instances *[]pdx.SubInstance, across, down float32, scaling bool) {
	if len(*instances) == 0 {
		*instances = append(*instances, pdx.NewSubInstance())
	}

	instance := &(*instances)[a.clampPlacement(len(*instances))]

	if scaling {
		instance.Scale.X = clampTo(instance.Scale.X+across*nudgeStep, pdx.MinScale, pdx.MaxScale)
		instance.Scale.Y = clampTo(instance.Scale.Y+down*nudgeStep, pdx.MinScale, pdx.MaxScale)
	} else {
		instance.Offset.X = clampTo(instance.Offset.X+across*nudgeStep, pdx.MinPosition, pdx.MaxPosition)
		instance.Offset.Y = clampTo(instance.Offset.Y+down*nudgeStep, pdx.MinPosition, pdx.MaxPosition)
	}

	a.changed()
}

// clampPlacement keeps the selected placement within a list of placements,
// which undo or a removal can have made shorter.
func (a *App) clampPlacement(count int) int {
	a.state.selectedPlacement = min(max(a.state.selectedPlacement, 0), count-1)

	return a.state.selectedPlacement
}

func clampTo(value, low, high float32) float32 {
	return min(max(value, low), high)
}

// nudgeHint tells the user how to move the selected placement from the
// keyboard, or that the preview has to be clicked first. It is drawn over the
// bottom of the flag preview.
func (a *App) nudgeHint(focused bool) string {
	layer, ok := a.state.selectedLayerValue()
	if !ok {
		return ""
	}

	// The number the arrow keys would act on, which undo can have taken past
	// the end of a shorter list.
	placement := min(a.state.selectedPlacement, max(layer.InstanceCount()-1, 0)) + 1

	if !focused {
		return "Click the flag to move placement " + strconv.Itoa(placement) + " of the selected layer with the arrow keys"
	}

	if _, sub := layer.(*pdx.SubFlag); sub {
		return "Placement " + strconv.Itoa(placement) + ": arrows move, Ctrl + arrows scale, Shift for bigger steps"
	}

	return "Placement " + strconv.Itoa(placement) + ": arrows move, Ctrl + arrows scale, Alt + left and right turn, Shift for bigger steps"
}
