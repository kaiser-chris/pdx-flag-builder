package app

import (
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// pickerTarget is what a choice in the picker is for.
type pickerTarget int

const (
	pickNothing pickerTarget = iota
	pickPattern
	pickNewColoredEmblem
	pickNewTexturedEmblem
	pickNewSubFlag
	pickLayerTexture
	pickLayerParent
)

// picker is the modal that chooses a texture or a coat of arms.
type picker struct {
	target pickerTarget

	// layer is the layer whose texture or parent is being changed.
	layer int

	search string
}

// choose opens the picker.
func (a *App) choose(target pickerTarget, layer int) {
	a.state.picker = picker{target: target, layer: layer}
	a.state.popup = popupPicker
}

// title names the picker after what it is choosing.
func (p picker) title() string {
	switch p.target {
	case pickPattern:
		return "Choose a Pattern"
	case pickNewColoredEmblem, pickNewTexturedEmblem, pickLayerTexture:
		return "Choose an Emblem"
	case pickNewSubFlag, pickLayerParent:
		return "Choose a Coat of Arms"
	}

	return "Choose"
}

// choices lists the names the picker offers, already filtered by the search.
func (a *App) choices() []string {
	query := strings.ToLower(strings.TrimSpace(a.state.picker.search))
	library := &a.state.library

	var names []string

	add := func(name string) {
		if query == "" || strings.Contains(strings.ToLower(name), query) {
			names = append(names, name)
		}
	}

	switch a.state.picker.target {
	case pickNewSubFlag, pickLayerParent:
		for index := range library.flags {
			add(library.flags[index].Name)
		}

		return names
	}

	kind, ok := a.pickerTextureKind()
	if !ok {
		return nil
	}

	for index := range library.textures {
		if library.textures[index].Kind == kind {
			add(library.textures[index].Name)
		}
	}

	return names
}

// pickerTextureKind is the kind of texture the picker offers.
func (a *App) pickerTextureKind() (database.TextureKind, bool) {
	switch a.state.picker.target {
	case pickPattern:
		return database.PatternTexture, true

	case pickNewColoredEmblem:
		return database.ColoredEmblemTexture, true

	case pickNewTexturedEmblem:
		return database.TexturedEmblemTexture, true

	case pickLayerTexture:
		layer, ok := a.layerAt(a.state.picker.layer)
		if !ok {
			return 0, false
		}

		if _, textured := layer.(*pdx.TexturedEmblem); textured {
			return database.TexturedEmblemTexture, true
		}

		return database.ColoredEmblemTexture, true
	}

	return 0, false
}

func (a *App) pickerPopup() {
	imgui.SetNextWindowSizeV(gui.ScaledVec2(520, 520), imgui.CondAppearing)

	// The part after ### keeps the popup's id stable while its title changes.
	if !imgui.BeginPopupModalV(a.state.picker.title()+"###picker", nil, 0) {
		return
	}
	defer imgui.EndPopup()

	imgui.SetNextItemWidth(-1)
	gui.InputText("##picker-search", "Search", &a.state.picker.search)

	names := a.choices()
	imgui.TextDisabled(plural(len(names), "match", "matches"))

	chosen := ""

	listHeight := -imgui.FrameHeightWithSpacing()
	if imgui.BeginChildStrV("##choices", imgui.Vec2{Y: listHeight}, imgui.ChildFlagsBorders, 0) {
		clipper := imgui.NewListClipper()
		clipper.Begin(int32(len(names)))

		for clipper.Step() {
			for row := clipper.DisplayStart(); row < clipper.DisplayEnd(); row++ {
				if gui.Selectable(names[row], false, 0) {
					chosen = names[row]
				}
			}
		}

		clipper.End()
		clipper.Destroy()
	}
	imgui.EndChild()

	if gui.Button("Cancel") {
		imgui.CloseCurrentPopup()
	}

	if chosen != "" {
		a.applyChoice(chosen)
		imgui.CloseCurrentPopup()
	}
}

// applyChoice does what the picker was opened for.
func (a *App) applyChoice(name string) {
	flag := a.state.flag
	if flag == nil {
		return
	}

	switch a.state.picker.target {
	case pickPattern:
		flag.Pattern = name

	case pickNewColoredEmblem:
		a.addLayer(pdx.NewColoredEmblem(name))

	case pickNewTexturedEmblem:
		a.addLayer(pdx.NewTexturedEmblem(name))

	case pickNewSubFlag:
		a.addLayer(pdx.NewSubFlag(name))

	case pickLayerTexture:
		switch layer, _ := a.layerAt(a.state.picker.layer); typed := layer.(type) {
		case *pdx.ColoredEmblem:
			typed.Texture = name
		case *pdx.TexturedEmblem:
			typed.Texture = name
		}

	case pickLayerParent:
		if sub, ok := a.subFlagAt(a.state.picker.layer); ok {
			sub.Parent = name
		}

	default:
		return
	}

	a.changed()
	a.setStatus("%s: %s", a.state.picker.title(), name)
}

// addLayer puts a new layer on top of the others and selects it.
func (a *App) addLayer(layer pdx.Layer) {
	flag := a.state.flag
	flag.Layers = append(flag.Layers, layer)

	a.state.selectedLayer = len(flag.Layers) - 1
	a.state.showSelected = true
}

func (a *App) layerAt(index int) (pdx.Layer, bool) {
	if a.state.flag == nil || index < 0 || index >= len(a.state.flag.Layers) {
		return nil, false
	}

	return a.state.flag.Layers[index], true
}

func (a *App) subFlagAt(index int) (*pdx.SubFlag, bool) {
	layer, ok := a.layerAt(index)
	if !ok {
		return nil, false
	}

	sub, ok := layer.(*pdx.SubFlag)

	return sub, ok
}
