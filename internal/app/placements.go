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
	labelRemovePlacement = "##placement-remove"
)

// What the buttons of a placement, and the link between a pair of fields, are
// for.
const (
	tooltipPlacementUp   = "Move up, under the placement above it"
	tooltipPlacementDown = "Move down, over the placement below it"
	tooltipRemovePlaced  = "Remove this placement"
	tooltipLink          = "Link the two together"
	tooltipLinked        = "Linked: changing one changes the other by the same amount"
)

// pairFields are the names of a row that edits two numbers: what the row is
// called, the two fields, and the lock between them.
type pairFields struct {
	label string

	x, y, lock string
}

// The rows of a placement that edit a pair, and which lock each of them has.
// A sub flag's offset is a position by another name, so it takes the same
// lock as one.
var (
	positionFields = pairFields{"Position", "##position-x", "##position-y", "##position-lock"}
	offsetFields   = pairFields{"Offset", "##offset-x", "##offset-y", "##offset-lock"}
	scaleFields    = pairFields{"Scale", "##scale-x", "##scale-y", "##scale-lock"}
)

// minimumFieldWidth is how narrow a number field may become before it stops
// giving way to the panel around it.
const minimumFieldWidth = 32

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

		if a.lockedPair(index, positionFields, &instance.Position.X, &instance.Position.Y,
			&a.state.lockPosition, pdx.MinPosition, pdx.MaxPosition) {
			a.changed()
		}

		if a.lockedPair(index, scaleFields, &instance.Scale.X, &instance.Scale.Y,
			&a.state.lockScale, pdx.MinScale, pdx.MaxScale) {
			a.changed()
		}

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

		if a.lockedPair(index, offsetFields, &instance.Offset.X, &instance.Offset.Y,
			&a.state.lockPosition, pdx.MinPosition, pdx.MaxPosition) {
			a.changed()
		}

		if a.lockedPair(index, scaleFields, &instance.Scale.X, &instance.Scale.Y,
			&a.state.lockScale, pdx.MinScale, pdx.MaxScale) {
			a.changed()
		}

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

	buttons := height*3 + style.ItemSpacing().X*3

	action := placementKeep

	imgui.PushStyleVarVec2(imgui.StyleVarSelectableTextAlign, imgui.Vec2{Y: 0.5})

	label := fmt.Sprintf("Placement %d", index+1)
	if gui.SelectableSized(label, index == a.state.selectedPlacement, imgui.ContentRegionAvail().X-buttons, height) {
		a.state.selectedPlacement = index
	}

	imgui.PopStyleVar()

	imgui.SameLine()
	imgui.BeginDisabledV(index == 0)
	if gui.IconButton(labelPlacementUp, gui.IconUp) {
		action = placementUp
	}
	gui.Tooltip(tooltipPlacementUp)
	imgui.EndDisabled()

	imgui.SameLine()
	imgui.BeginDisabledV(index == count-1)
	if gui.IconButton(labelPlacementDown, gui.IconDown) {
		action = placementDown
	}
	gui.Tooltip(tooltipPlacementDown)
	imgui.EndDisabled()

	imgui.SameLine()
	if gui.IconButton(labelRemovePlacement, gui.IconDelete) {
		action = placementRemove
	}
	gui.Tooltip(tooltipRemovePlaced)

	return action
}

// lockedPair edits the two numbers of a placement's point, with a lock in
// between them.
//
// While the lock is closed the two move together: whichever one is changed,
// the other changes by the same amount, which is what keeps a square emblem
// square. A change that would take the other past its limit stops both of
// them, since moving one on its own is what the lock is there to prevent.
func (a *App) lockedPair(index int, fields pairFields, x, y *float32, locked *bool, low, high float32) bool {
	was := [2]float32{*x, *y}

	gui.Label(fields.label)

	// The two halves share what the lock between them leaves, down to a width
	// that still holds a number: a field given less than nothing would be laid
	// out from the right edge instead and come out backwards.
	spacing := imgui.CurrentStyle().ItemInnerSpacing().X
	width := max((imgui.ContentRegionAvail().X-imgui.FrameHeight()-spacing*2)/2, gui.Scaled(minimumFieldWidth))

	imgui.SetNextItemWidth(width)
	changed := gui.DragFloat(fields.x, x, dragFraction, low, high, "%.3f")
	a.focusPlacement(index)

	imgui.SameLineV(0, spacing)

	if gui.LockButton(fields.lock, *locked) {
		*locked = !*locked
	}

	lockTooltip(*locked)

	imgui.SameLineV(0, spacing)
	imgui.SetNextItemWidth(-1)

	changed = gui.DragFloat(fields.y, y, dragFraction, low, high, "%.3f") || changed
	a.focusPlacement(index)

	if !changed || !*locked {
		return changed
	}

	switch {
	case *x != was[0]:
		tandem(x, y, was[0], was[1], low, high)
	case *y != was[1]:
		tandem(y, x, was[1], was[0], low, high)
	}

	return changed
}

// tandem moves the other half of a locked pair by the same amount as the half
// that was changed, and holds both of them back where that would take it past
// a limit, so that the two keep the distance between them.
func tandem(changed, other *float32, was, otherWas, low, high float32) {
	delta := *changed - was

	switch {
	case otherWas+delta < low:
		delta = low - otherWas
	case otherWas+delta > high:
		delta = high - otherWas
	}

	*changed = was + delta
	*other = otherWas + delta
}

// lockTooltip says what the link between two fields does, which a chain on its
// own does not.
func lockTooltip(locked bool) {
	if locked {
		gui.Tooltip(tooltipLinked)

		return
	}

	gui.Tooltip(tooltipLink)
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
