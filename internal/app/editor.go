package app

import (
	"fmt"
	"slices"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Labels of the layer list, which the interface tests click by name.
const (
	labelCoatOfArms = "Coat of Arms"
	labelAddLayer   = "Add Layer"
	labelMoveUp     = "##up"
	labelMoveDown   = "##down"
	labelRemove     = "X##remove"
)

// Labels of the buttons that act on the layer being edited.
const (
	labelDuplicateLayer = "Duplicate"
	labelDeleteLayer    = "Delete"
)

// layerAction is what the buttons beside a layer's name ask for. The list it
// belongs to is changed once the layer has been drawn, never while it is.
type layerAction int

const (
	noLayerAction layerAction = iota
	duplicateLayer
	deleteLayer
)

// How finely a drag changes a value, per pixel the pointer moves.
const (
	dragFraction = 0.002
	dragDegrees  = 0.5
)

// layersBody lists the coat of arms and its layers, and edits the list.
func (a *App) layersBody() {
	flag := a.state.flag

	if flag == nil {
		emptyState(
			"No flag open.",
			"Pick one in the flag database",
			"or start a new one from the File menu.",
		)

		return
	}

	title := flag.Name
	if a.state.modified {
		// The usual marker for a document with unsaved changes.
		title += " *"
	}

	gui.PushStrongFont()
	imgui.TextUnformatted(title)
	gui.PopFont()

	if flag.Origin.File != "" {
		imgui.TextDisabled(fmt.Sprintf("%s / %s", flag.Origin.Database, flag.Origin.File))
	} else {
		imgui.TextDisabled("not saved to a file yet")
	}

	imgui.Spacing()

	a.addLayerMenu()

	imgui.Separator()

	// Every row is as tall as the buttons beside the layers, the coat of arms
	// too, with its label in the middle, so that the list reads evenly.
	height := imgui.FrameHeight()

	imgui.PushStyleVarVec2(imgui.StyleVarSelectableTextAlign, imgui.Vec2{Y: 0.5})
	defer imgui.PopStyleVar()

	if gui.SelectableSized(labelCoatOfArms, a.state.selectedLayer == noLayer, 0, height) {
		a.selectLayer(noLayer)
	}

	a.layerRows(flag, height)
}

func (a *App) addLayerMenu() {
	if gui.Button(labelAddLayer) {
		imgui.OpenPopupStr("##add-layer")
	}

	if !imgui.BeginPopup("##add-layer") {
		return
	}
	defer imgui.EndPopup()

	if gui.MenuItem("Colored Emblem...", "", true) {
		a.choose(pickNewColoredEmblem, noLayer)
	}

	if gui.MenuItem("Textured Emblem...", "", true) {
		a.choose(pickNewTexturedEmblem, noLayer)
	}

	if gui.MenuItem("Sub Flag...", "", true) {
		a.choose(pickNewSubFlag, noLayer)
	}
}

// layerRows draws one row per layer, with buttons to move and remove it. The
// list is changed after it has been drawn, never while it is being walked.
func (a *App) layerRows(flag *pdx.Flag, height float32) {
	const noAction = -1

	moveFrom, moveBy, remove := noAction, 0, noAction

	style := imgui.CurrentStyle()
	buttons := imgui.FrameHeight()*3 + style.ItemSpacing().X*3 + imgui.CalcTextSize("X").X + style.FramePadding().X*2

	for index, layer := range flag.Layers {
		imgui.PushIDInt(int32(index))

		width := imgui.ContentRegionAvail().X - buttons
		if gui.SelectableSized(describeLayer(layer), index == a.state.selectedLayer, width, height) {
			a.selectLayer(index)
		}

		imgui.SameLine()
		imgui.BeginDisabledV(index == 0)
		if gui.ArrowButton(labelMoveUp, imgui.DirUp) {
			moveFrom, moveBy = index, -1
		}
		imgui.EndDisabled()

		imgui.SameLine()
		imgui.BeginDisabledV(index == len(flag.Layers)-1)
		if gui.ArrowButton(labelMoveDown, imgui.DirDown) {
			moveFrom, moveBy = index, 1
		}
		imgui.EndDisabled()

		imgui.SameLine()
		if gui.Button(labelRemove) {
			remove = index
		}

		imgui.PopID()
	}

	switch {
	case moveFrom != noAction:
		a.moveLayer(moveFrom, moveBy)
	case remove != noAction:
		a.removeLayer(remove)
	}
}

func (a *App) selectLayer(index int) {
	if index != a.state.selectedLayer {
		a.state.selectedPlacement = 0
	}

	a.state.selectedLayer = index
	a.state.showSelected = true
}

// moveLayer moves a layer up or down the stack, taking the selection with it.
func (a *App) moveLayer(index, delta int) {
	target, moved := pdx.MoveItem(a.state.flag.Layers, index, delta)
	if !moved {
		return
	}

	switch a.state.selectedLayer {
	case index:
		a.state.selectedLayer = target
	case target:
		a.state.selectedLayer = index
	}

	a.changed()
}

// duplicateLayer puts a copy of a layer directly over the one it was copied
// from, and selects the copy, since that is the one further changes are meant
// for.
func (a *App) duplicateLayer(index int) {
	flag := a.state.flag

	flag.Layers = slices.Insert(flag.Layers, index+1, pdx.CloneLayer(flag.Layers[index]))

	a.selectLayer(index + 1)
	a.changed()
}

// removeLayer drops a layer. There is no confirmation: undo brings it back.
func (a *App) removeLayer(index int) {
	flag := a.state.flag
	flag.Layers = pdx.RemoveItem(flag.Layers, index)

	switch {
	case a.state.selectedLayer == index:
		a.state.selectedLayer = noLayer
	case a.state.selectedLayer > index:
		a.state.selectedLayer--
	}

	a.changed()
}

// selectedLayerBody edits whatever is selected in the layer list.
func (a *App) selectedLayerBody() {
	flag := a.state.flag
	if flag == nil {
		emptyState("No flag open.")

		return
	}

	layer, ok := a.state.selectedLayerValue()
	if !ok {
		a.coatOfArmsEditor(flag)

		return
	}

	index := a.state.selectedLayer

	gui.PushStrongFont()
	imgui.TextUnformatted(layerKind(layer))
	gui.PopFont()

	action := layerActions()

	imgui.Spacing()

	switch typed := layer.(type) {
	case *pdx.ColoredEmblem:
		a.textureField("Texture", typed.Texture, pickLayerTexture)
		a.maskField(&typed.Mask)

		sectionHeader("Colours")
		a.colorEditor("emblem", &typed.Colors, flag.Colors, layerColors)

		a.placementEditor(&typed.Instances)

	case *pdx.TexturedEmblem:
		a.textureField("Texture", typed.Texture, pickLayerTexture)
		a.maskField(&typed.Mask)
		a.placementEditor(&typed.Instances)

	case *pdx.SubFlag:
		a.parentField(typed)

		// The colours a sub flag fills in are handed to the coat of arms it
		// draws, in place of that one's own.
		sectionHeader("Colours")
		a.colorEditor("sub", &typed.Colors, flag.Colors, layerColors)

		a.subPlacementEditor(&typed.Instances)
	}

	switch action {
	case duplicateLayer:
		a.duplicateLayer(index)
	case deleteLayer:
		a.removeLayer(index)
	}
}

// layerActions draws the buttons that act on the layer as a whole, level with
// its name and over at the right, clear of the fields below them. A panel too
// narrow to hold them beside the name keeps them next to it rather than
// letting them fall off the edge.
func layerActions() layerAction {
	style := imgui.CurrentStyle()

	width := buttonWidth(labelDuplicateLayer) + buttonWidth(labelDeleteLayer) + style.ItemSpacing().X

	imgui.SameLine()
	imgui.SetCursorPosX(max(imgui.CursorPosX(), imgui.CursorPosX()+imgui.ContentRegionAvail().X-width))

	action := noLayerAction

	if gui.Button(labelDuplicateLayer) {
		action = duplicateLayer
	}

	imgui.SameLine()

	if gui.Button(labelDeleteLayer) {
		action = deleteLayer
	}

	return action
}

// buttonWidth is how wide a button with this label comes out.
func buttonWidth(label string) float32 {
	return imgui.CalcTextSize(label).X + imgui.CurrentStyle().FramePadding().X*2
}

// coatOfArmsEditor edits what belongs to the coat of arms rather than a layer.
func (a *App) coatOfArmsEditor(flag *pdx.Flag) {
	gui.PushStrongFont()
	imgui.TextUnformatted(labelCoatOfArms)
	gui.PopFont()

	imgui.Spacing()

	// The name is the key the coat of arms is written under, which is also how
	// sub flags and the game refer to it.
	if gui.InputText("Name", "key the coat of arms is written under", &flag.Name) {
		a.changed()
	}

	a.textureField("Pattern", flag.Pattern, pickPattern)

	sectionHeader("Colours")
	a.colorEditor("flag", &flag.Colors, flag.Colors, flagColors)
}

// textureField shows a texture and offers to change it. It also says when the
// texture is not in any configured folder, since that is why a layer would
// not show up.
func (a *App) textureField(label, texture string, target pickerTarget) {
	gui.Label(label)

	if texture == "" {
		imgui.TextDisabled("none")
	} else {
		imgui.TextDisabled(texture)
	}

	imgui.SameLine()

	if gui.SmallButton("Change...##" + label) {
		a.choose(target, a.state.selectedLayer)
	}

	if texture != "" {
		if _, found := a.state.library.set.Texture(texture); !found {
			imgui.TextDisabled("This texture was not found in the configured folders.")
		}
	}
}

func (a *App) parentField(sub *pdx.SubFlag) {
	gui.Label("Parent")
	imgui.TextDisabled(or(sub.Parent, "none"))
	imgui.SameLine()

	if gui.SmallButton("Change...##parent") {
		a.choose(pickLayerParent, a.state.selectedLayer)
	}

	if sub.Parent != "" {
		if _, found := a.state.library.set.Flag(sub.Parent); !found {
			imgui.TextDisabled("This coat of arms was not found in the configured folders.")
		}
	}
}

// maskField restricts a coloured emblem to one colour of the pattern.
func (a *App) maskField(mask *int) {
	if !gui.BeginCombo("Mask", maskLabel(*mask)) {
		return
	}
	defer imgui.EndCombo()

	for value := range pdx.MaxMask + 1 {
		if gui.Selectable(maskLabel(value), value == *mask, 0) && value != *mask {
			*mask = value
			a.changed()
		}
	}
}

func maskLabel(mask int) string {
	if mask <= 0 {
		return "None"
	}

	return fmt.Sprintf("Pattern colour %d", mask)
}
