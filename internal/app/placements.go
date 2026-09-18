package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Labels of a placement's header, which the interface tests click by name.
const (
	labelPlacementUp     = "##placement-up"
	labelPlacementDown   = "##placement-down"
	labelRemovePlacement = "Remove##placement"
)

// How far one press of an arrow key changes a placement: a thousandth of the
// canvas, under a pixel of the flag, for lining things up exactly, and a
// degree of rotation. Shift makes a step nudgeFaster times bigger.
const (
	nudgeStep    = 0.001
	nudgeDegrees = 1
	nudgeFaster  = 10
)

// placementAction is what the header of a placement asked for.
type placementAction int

const (
	placementKeep placementAction = iota
	placementUp
	placementDown
	placementRemove
)

// placementEditor edits where an emblem is drawn.
func (a *App) placementEditor(instances *[]pdx.Instance) {
	sectionHeader(instanceHeading(len(*instances), len(*instances) == 0))

	if len(*instances) == 0 {
		dimmedWrapped("Drawn once at the default placement until a placement is added. " +
			"Moving it with the arrow keys adds it.")
	} else {
		placementHint()
	}

	moveFrom, action := -1, placementKeep

	for index := range *instances {
		instance := &(*instances)[index]

		imgui.PushIDInt(int32(index))

		card := gui.BeginCard()

		if chosen := a.placementHeader(index, len(*instances)); chosen != placementKeep {
			moveFrom, action = index, chosen
		}

		if gui.DragPair("Position", &instance.Position.X, &instance.Position.Y,
			dragFraction, pdx.MinPosition, pdx.MaxPosition, "%.3f") {
			a.changed()
		}
		a.focusPlacement(index)

		if gui.DragPair("Scale", &instance.Scale.X, &instance.Scale.Y,
			dragFraction, pdx.MinScale, pdx.MaxScale, "%.3f") {
			a.changed()
		}
		a.focusPlacement(index)

		if gui.DragFloat("Rotation", &instance.Rotation,
			dragDegrees, -pdx.MaxRotation, pdx.MaxRotation, "%.1f deg") {
			a.changed()
		}
		a.focusPlacement(index)

		card.End(index == a.state.selectedPlacement)

		imgui.PopID()
	}

	*instances = applyPlacementAction(a, *instances, moveFrom, action)

	if gui.Button("Add Placement") {
		*instances = append(*instances, pdx.NewInstance())
		a.state.selectedPlacement = len(*instances) - 1
		a.changed()
	}
}

// subPlacementEditor edits where a sub flag is drawn.
func (a *App) subPlacementEditor(instances *[]pdx.SubInstance) {
	sectionHeader(instanceHeading(len(*instances), len(*instances) == 0))

	if len(*instances) == 0 {
		dimmedWrapped("Drawn once over the whole flag until a placement is added. " +
			"Moving it with the arrow keys adds it.")
	} else {
		placementHint()
	}

	moveFrom, action := -1, placementKeep

	for index := range *instances {
		instance := &(*instances)[index]

		imgui.PushIDInt(int32(index))

		card := gui.BeginCard()

		if chosen := a.placementHeader(index, len(*instances)); chosen != placementKeep {
			moveFrom, action = index, chosen
		}

		if gui.DragPair("Offset", &instance.Offset.X, &instance.Offset.Y,
			dragFraction, pdx.MinPosition, pdx.MaxPosition, "%.3f") {
			a.changed()
		}
		a.focusPlacement(index)

		if gui.DragPair("Scale", &instance.Scale.X, &instance.Scale.Y,
			dragFraction, pdx.MinScale, pdx.MaxScale, "%.3f") {
			a.changed()
		}
		a.focusPlacement(index)

		card.End(index == a.state.selectedPlacement)

		imgui.PopID()
	}

	*instances = applyPlacementAction(a, *instances, moveFrom, action)

	if gui.Button("Add Placement") {
		*instances = append(*instances, pdx.NewSubInstance())
		a.state.selectedPlacement = len(*instances) - 1
		a.changed()
	}
}

// placementHeader is the row above a placement's fields: its name, which
// selects it as the one the arrow keys move, and buttons to move it up and
// down the list and to remove it. The order matters where placements overlap:
// later ones are drawn over earlier ones.
func (a *App) placementHeader(index, count int) placementAction {
	style := imgui.CurrentStyle()
	height := imgui.FrameHeight()

	removeWidth := imgui.CalcTextSizeV(labelRemovePlacement, true, -1).X + style.FramePadding().X*2
	buttons := height*2 + removeWidth + style.ItemSpacing().X*3

	action := placementKeep

	imgui.PushStyleVarVec2(imgui.StyleVarSelectableTextAlign, imgui.Vec2{Y: 0.5})

	label := fmt.Sprintf("Placement %d", index+1)
	if gui.SelectableSized(label, index == a.state.selectedPlacement, imgui.ContentRegionAvail().X-buttons, height) {
		a.state.selectedPlacement = index
	}

	imgui.PopStyleVar()

	imgui.SameLine()
	imgui.BeginDisabledV(index == 0)
	if gui.ArrowButton(labelPlacementUp, imgui.DirUp) {
		action = placementUp
	}
	imgui.EndDisabled()

	imgui.SameLine()
	imgui.BeginDisabledV(index == count-1)
	if gui.ArrowButton(labelPlacementDown, imgui.DirDown) {
		action = placementDown
	}
	imgui.EndDisabled()

	imgui.SameLine()
	if gui.Button(labelRemovePlacement) {
		action = placementRemove
	}

	return action
}

// focusPlacement selects a placement as the one the arrow keys move as soon
// as one of its fields is taken hold of.
func (a *App) focusPlacement(index int) {
	if imgui.IsItemActivated() {
		a.state.selectedPlacement = index
	}
}

// applyPlacementAction carries out what a placement's header asked for, after
// the list has been drawn, and keeps the selection on the same placement.
func applyPlacementAction[T any](a *App, items []T, index int, action placementAction) []T {
	switch action {
	case placementUp, placementDown:
		delta := -1
		if action == placementDown {
			delta = 1
		}

		target, moved := pdx.MoveItem(items, index, delta)
		if !moved {
			return items
		}

		switch a.state.selectedPlacement {
		case index:
			a.state.selectedPlacement = target
		case target:
			a.state.selectedPlacement = index
		}

	case placementRemove:
		items = pdx.RemoveItem(items, index)

		if a.state.selectedPlacement >= index && a.state.selectedPlacement > 0 {
			a.state.selectedPlacement--
		}

	default:
		return items
	}

	a.changed()

	return items
}

// placementHint explains the fields of a placement, which look like plain
// boxes but are changed by dragging across them, and the keyboard.
func placementHint() {
	dimmedWrapped("Drag a value sideways to change it, or double-click it to type a number; " +
		"Shift drags in bigger steps, Alt in finer ones. " +
		"Click the flag to move the selected placement with the arrow keys.")
}
