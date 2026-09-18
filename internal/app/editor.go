package app

import (
	"fmt"

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

	if gui.Selectable(labelCoatOfArms, a.state.selectedLayer == noLayer, 0) {
		a.selectLayer(noLayer)
	}

	a.layerRows(flag)
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
func (a *App) layerRows(flag *pdx.Flag) {
	const noAction = -1

	moveFrom, moveBy, remove := noAction, 0, noAction

	style := imgui.CurrentStyle()
	buttons := imgui.FrameHeight()*3 + style.ItemSpacing().X*3 + imgui.CalcTextSize("X").X + style.FramePadding().X*2

	for index, layer := range flag.Layers {
		imgui.PushIDInt(int32(index))

		width := imgui.ContentRegionAvail().X - buttons
		if gui.SelectableSized(describeLayer(layer), index == a.state.selectedLayer, width) {
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

	gui.PushStrongFont()
	imgui.TextUnformatted(layerKind(layer))
	gui.PopFont()

	imgui.Spacing()

	switch typed := layer.(type) {
	case *pdx.ColoredEmblem:
		a.textureField("Texture", typed.Texture, pickLayerTexture)
		a.maskField(typed)

		sectionHeader("Colours")
		a.colorEditor("emblem", &typed.Colors, flag.Colors)

		a.placementEditor(&typed.Instances)

	case *pdx.TexturedEmblem:
		a.textureField("Texture", typed.Texture, pickLayerTexture)
		a.placementEditor(&typed.Instances)

	case *pdx.SubFlag:
		a.parentField(typed)
		a.subPlacementEditor(&typed.Instances)
	}
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
	a.colorEditor("flag", &flag.Colors, flag.Colors)
}

// textureField shows a texture and offers to change it. It also says when the
// texture is not in any configured folder, since that is why a layer would
// not show up.
func (a *App) textureField(label, texture string, target pickerTarget) {
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(label)
	imgui.SameLine()

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
	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted("Parent")
	imgui.SameLine()
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
func (a *App) maskField(emblem *pdx.ColoredEmblem) {
	imgui.SetNextItemWidth(gui.Scaled(200))

	if !gui.BeginCombo("Mask", maskLabel(emblem.Mask)) {
		return
	}
	defer imgui.EndCombo()

	for value := range pdx.MaxMask + 1 {
		if gui.Selectable(maskLabel(value), value == emblem.Mask, 0) && value != emblem.Mask {
			emblem.Mask = value
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

// placementEditor edits where an emblem is drawn.
func (a *App) placementEditor(instances *[]pdx.Instance) {
	sectionHeader(instanceHeading(len(*instances), len(*instances) == 0))

	if len(*instances) == 0 {
		dimmedWrapped("Drawn once at the default placement until a placement is added.")
	}

	remove := -1

	for index := range *instances {
		instance := &(*instances)[index]

		imgui.PushIDInt(int32(index))

		imgui.AlignTextToFramePadding()
		imgui.TextDisabled(fmt.Sprintf("Placement %d", index+1))
		imgui.SameLine()

		if gui.SmallButton("Remove") {
			remove = index
		}

		if gui.DragPair("Position", &instance.Position.X, &instance.Position.Y,
			dragFraction, pdx.MinPosition, pdx.MaxPosition, "%.3f") {
			a.changed()
		}

		if gui.DragPair("Scale", &instance.Scale.X, &instance.Scale.Y,
			dragFraction, pdx.MinScale, pdx.MaxScale, "%.3f") {
			a.changed()
		}

		if gui.DragFloat("Rotation", &instance.Rotation,
			dragDegrees, -pdx.MaxRotation, pdx.MaxRotation, "%.1f deg") {
			a.changed()
		}

		imgui.PopID()
	}

	if remove >= 0 {
		*instances = pdx.RemoveItem(*instances, remove)
		a.changed()
	}

	if gui.Button("Add Placement") {
		*instances = append(*instances, pdx.NewInstance())
		a.changed()
	}
}

// subPlacementEditor edits where a sub flag is drawn.
func (a *App) subPlacementEditor(instances *[]pdx.SubInstance) {
	sectionHeader(instanceHeading(len(*instances), len(*instances) == 0))

	if len(*instances) == 0 {
		dimmedWrapped("Drawn once over the whole flag until a placement is added.")
	}

	remove := -1

	for index := range *instances {
		instance := &(*instances)[index]

		imgui.PushIDInt(int32(index))

		imgui.AlignTextToFramePadding()
		imgui.TextDisabled(fmt.Sprintf("Placement %d", index+1))
		imgui.SameLine()

		if gui.SmallButton("Remove") {
			remove = index
		}

		if gui.DragPair("Offset", &instance.Offset.X, &instance.Offset.Y,
			dragFraction, pdx.MinPosition, pdx.MaxPosition, "%.3f") {
			a.changed()
		}

		if gui.DragPair("Scale", &instance.Scale.X, &instance.Scale.Y,
			dragFraction, pdx.MinScale, pdx.MaxScale, "%.3f") {
			a.changed()
		}

		imgui.PopID()
	}

	if remove >= 0 {
		*instances = pdx.RemoveItem(*instances, remove)
		a.changed()
	}

	if gui.Button("Add Placement") {
		*instances = append(*instances, pdx.NewSubInstance())
		a.changed()
	}
}
