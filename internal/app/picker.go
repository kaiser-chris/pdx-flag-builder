package app

import (
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

	// rows are the flags or textures on offer that the search matches.
	rows filteredRows
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

// listsFlags reports whether the picker offers coats of arms rather than
// textures.
func (p picker) listsFlags() bool {
	return p.target == pickNewSubFlag || p.target == pickLayerParent
}

// Columns of the picker.
const (
	pickerColumnPreview = iota
	pickerColumnName
	pickerColumnFolder
	pickerColumns
)

func (a *App) pickerPopup() {
	imgui.SetNextWindowSizeV(gui.ScaledVec2(600, 560), imgui.CondAppearing)

	// The part after ### keeps the popup's id stable while its title changes.
	if !imgui.BeginPopupModalV(a.state.picker.title()+"###picker", nil, 0) {
		return
	}
	defer imgui.EndPopup()

	imgui.SetNextItemWidth(-1)
	gui.InputText("##picker-search", "Search by name or folder", &a.state.picker.search)

	query := a.state.picker.search
	listsFlags := a.state.picker.listsFlags()
	rows := a.pickerRows(query, listsFlags)

	imgui.TextDisabled(plural(len(rows), "match", "matches"))

	chosen := ""

	// The list takes the height the Cancel button below leaves it.
	size := imgui.Vec2{Y: -imgui.FrameHeightWithSpacing()}

	if imgui.BeginTableV("##choices", pickerColumns, tableFlags, size, 0) {
		setupThumbnailColumn()
		imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthStretch|imgui.TableColumnFlagsDefaultSort, 0, 0)
		imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthFixed, gui.Scaled(110), 0)
		imgui.TableSetupScrollFreeze(0, 1)
		gui.TableHeadersRow()

		rows = a.state.picker.rows.inOrder(currentOrder(), func(first, second, column int) int {
			firstName, firstFolder := a.pickerChoice(first, listsFlags)
			secondName, secondFolder := a.pickerChoice(second, listsFlags)

			if column == pickerColumnFolder {
				return compareFold(firstFolder, secondFolder)
			}

			return compareFold(firstName, secondName)
		})

		chosen = a.pickerTable(rows, query, listsFlags)

		imgui.EndTable()
	}

	if gui.Button("Cancel") {
		imgui.CloseCurrentPopup()
	}

	if chosen != "" {
		a.applyChoice(chosen)
		imgui.CloseCurrentPopup()
	}
}

// pickerRows lists the flags or textures on offer that the search matches, as
// indexes into the library.
func (a *App) pickerRows(query string, listsFlags bool) []int {
	library := &a.state.library

	if listsFlags {
		return a.state.picker.rows.get(query, library.version, len(library.flags),
			func(index int, query string) bool {
				flag := &library.flags[index]

				return containsFold(flag.Name, query) || containsFold(flag.Origin.Database, query)
			})
	}

	kind, ok := a.pickerTextureKind()

	return a.state.picker.rows.get(query, library.version, len(library.textures),
		func(index int, query string) bool {
			texture := &library.textures[index]

			return ok && texture.Kind == kind &&
				(containsFold(texture.Name, query) || containsFold(texture.Database, query))
		})
}

// pickerChoice is the name and the folder of one of the picker's rows.
func (a *App) pickerChoice(index int, listsFlags bool) (name, folder string) {
	library := &a.state.library

	if listsFlags {
		return library.flags[index].Name, library.flags[index].Origin.Database
	}

	return library.textures[index].Name, library.textures[index].Database
}

// pickerTable fills in the picker's rows and returns the name of the one
// clicked, if any. A texture a mod replaces is listed once for the game and
// once for the mod, so that the folder column shows where each comes from;
// either way it is the name that is chosen.
func (a *App) pickerTable(rows []int, query string, listsFlags bool) string {
	library := &a.state.library
	height := rowHeight()
	chosen := ""

	clipper := imgui.NewListClipper()
	defer clipper.Destroy()

	clipper.Begin(int32(len(rows)))

	for clipper.Step() {
		for row := clipper.DisplayStart(); row < clipper.DisplayEnd(); row++ {
			index := rows[row]
			name, folder := a.pickerChoice(index, listsFlags)

			imgui.TableNextRow()
			imgui.PushIDInt(int32(index))

			imgui.TableSetColumnIndex(pickerColumnPreview)

			if listsFlags {
				a.flagThumbnail(&library.flags[index])
			} else {
				a.textureThumbnail(library.textures[index].Path)
			}

			imgui.TableSetColumnIndex(pickerColumnName)

			if gui.HighlightedSelectable(name, query, false, imgui.SelectableFlagsSpanAllColumns, height) {
				chosen = name
			}

			imgui.TableSetColumnIndex(pickerColumnFolder)
			highlightedCell(folder, query, height)

			imgui.PopID()
		}
	}

	clipper.End()

	return chosen
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
	a.state.selectedPlacement = 0
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
