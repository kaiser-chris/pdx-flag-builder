package app

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Columns of the flag database.
const (
	flagColumnPreview = iota
	flagColumnName
	flagColumnLayers
	flagColumnFolder
	flagColumnFile
	flagColumnUse
	flagColumns
)

// Columns of the texture database.
const (
	textureColumnPreview = iota
	textureColumnName
	textureColumnKind
	textureColumnFolder
	textureColumnUse
	textureColumns
)

// labelAddAsSubFlag is the flag database's row action.
const labelAddAsSubFlag = "Add as Sub Flag"

// flagDatabaseWindow lists every coat of arms found in the configured folders.
func (a *App) flagDatabaseWindow() {
	if !a.state.showFlagDatabase {
		return
	}

	imgui.SetNextWindowSizeV(gui.ScaledVec2(860, 560), imgui.CondFirstUseEver)

	if imgui.BeginV(windowFlagDatabase, &a.state.showFlagDatabase, 0) {
		a.trackFocus(windowFlagDatabase)
		a.flagDatabaseBody()
	}

	imgui.End()
}

func (a *App) flagDatabaseBody() {
	library := &a.state.library

	if library.loading {
		emptyState("Reading the configured folders...")

		return
	}

	if len(library.flags) == 0 {
		emptyState(
			"No flags found.",
			"Add a game or mod folder in the settings, then save.",
		)

		return
	}

	imgui.SetNextItemWidth(-1)
	gui.InputText("##flag-search", "Search by name, folder or file", &a.state.flagSearch)

	query := a.state.flagSearch
	rows := a.state.flagRows.get(
		query,
		library.version,
		len(library.flags),
		func(index int, query string) bool {
			flag := &library.flags[index]

			return containsFold(flag.Name, query) ||
				containsFold(flag.Origin.Database, query) ||
				containsFold(flag.Origin.File, query)
		},
	)

	imgui.TextDisabled(fmt.Sprintf("%d of %d flags", len(rows), len(library.flags)))

	if !imgui.BeginTableV("flags", flagColumns, tableFlags, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	setupThumbnailColumn()
	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthStretch|imgui.TableColumnFlagsDefaultSort, 0, 0)
	imgui.TableSetupColumnV("Layers", imgui.TableColumnFlagsWidthFixed, gui.Scaled(60), 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthFixed, gui.Scaled(110), 0)
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("##use", imgui.TableColumnFlagsWidthFixed|imgui.TableColumnFlagsNoSort, gui.Scaled(120), 0)
	imgui.TableSetupScrollFreeze(0, 1)
	gui.TableHeadersRow()

	rows = a.state.flagRows.inOrder(currentOrder(), func(first, second, column int) int {
		one, other := &library.flags[first], &library.flags[second]

		switch column {
		case flagColumnName:
			return compareFold(one.Name, other.Name)
		case flagColumnLayers:
			return cmp.Compare(len(one.Layers), len(other.Layers))
		case flagColumnFolder:
			return compareFold(one.Origin.Database, other.Origin.Database)
		case flagColumnFile:
			return compareFold(one.Origin.File, other.Origin.File)
		}

		return 0
	})

	height := rowHeight()

	// The list runs to thousands of rows, so only the visible ones are built.
	clipper := imgui.NewListClipper()
	defer clipper.Destroy()

	clipper.Begin(int32(len(rows)))

	for clipper.Step() {
		for row := clipper.DisplayStart(); row < clipper.DisplayEnd(); row++ {
			index := rows[row]
			flag := &library.flags[index]

			imgui.TableNextRow()
			imgui.PushIDInt(int32(index))

			imgui.TableSetColumnIndex(flagColumnPreview)
			a.flagThumbnail(flag)

			imgui.TableSetColumnIndex(flagColumnName)

			open := a.state.flag != nil &&
				a.state.flag.Name == flag.Name &&
				a.state.flag.Origin.Path == flag.Origin.Path

			// The whole row opens the flag, except for the button at its end.
			rowFlags := imgui.SelectableFlagsSpanAllColumns | imgui.SelectableFlagsAllowOverlap
			if gui.HighlightedSelectable(flag.Name, query, open, rowFlags, height) {
				a.requestOpen(*flag)
			}

			imgui.TableSetColumnIndex(flagColumnLayers)
			plainCell(strconv.Itoa(len(flag.Layers)), height)

			imgui.TableSetColumnIndex(flagColumnFolder)
			highlightedCell(flag.Origin.Database, query, height)

			imgui.TableSetColumnIndex(flagColumnFile)
			highlightedCell(flag.Origin.File, query, height)

			// Adding a sub flag needs a flag to add it to.
			imgui.TableSetColumnIndex(flagColumnUse)
			centredCell(height, imgui.TextLineHeight())
			imgui.BeginDisabledV(a.state.flag == nil)

			if gui.SmallButton(labelAddAsSubFlag) {
				a.addLayer(pdx.NewSubFlag(flag.Name))
				a.changed()
				a.setStatus("Added %s as a sub flag", flag.Name)
			}

			imgui.EndDisabled()
			imgui.PopID()
		}
	}

	clipper.End()
}

// textureDatabaseWindow lists the patterns and emblems available to build with.
func (a *App) textureDatabaseWindow() {
	if !a.state.showTextureDatabase {
		return
	}

	imgui.SetNextWindowSizeV(gui.ScaledVec2(760, 560), imgui.CondFirstUseEver)

	if imgui.BeginV(windowTextureDatabase, &a.state.showTextureDatabase, 0) {
		a.trackFocus(windowTextureDatabase)
		a.textureDatabaseBody()
	}

	imgui.End()
}

func (a *App) textureDatabaseBody() {
	library := &a.state.library

	if library.loading {
		emptyState("Reading the configured folders...")

		return
	}

	if len(library.textures) == 0 {
		emptyState(
			"No textures found.",
			"Add a game or mod folder in the settings, then save.",
		)

		return
	}

	imgui.SetNextItemWidth(-1)
	gui.InputText("##texture-search", "Search by name, kind or folder", &a.state.textureSearch)

	query := a.state.textureSearch
	rows := a.state.textureRows.get(
		query,
		library.version,
		len(library.textures),
		func(index int, query string) bool {
			texture := &library.textures[index]

			return containsFold(texture.Name, query) ||
				containsFold(texture.Kind.String(), query) ||
				containsFold(texture.Database, query)
		},
	)

	imgui.TextDisabled(fmt.Sprintf("%d of %d textures", len(rows), len(library.textures)))

	if !imgui.BeginTableV("textures", textureColumns, tableFlags, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	setupThumbnailColumn()
	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthStretch|imgui.TableColumnFlagsDefaultSort, 0, 0)
	imgui.TableSetupColumnV("Kind", imgui.TableColumnFlagsWidthFixed, gui.Scaled(130), 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthFixed, gui.Scaled(110), 0)
	imgui.TableSetupColumnV("##use", imgui.TableColumnFlagsWidthFixed|imgui.TableColumnFlagsNoSort, gui.Scaled(110), 0)
	imgui.TableSetupScrollFreeze(0, 1)
	gui.TableHeadersRow()

	rows = a.state.textureRows.inOrder(currentOrder(), func(first, second, column int) int {
		one, other := &library.textures[first], &library.textures[second]

		switch column {
		case textureColumnName:
			return compareFold(one.Name, other.Name)
		case textureColumnKind:
			return compareFold(one.Kind.String(), other.Kind.String())
		case textureColumnFolder:
			return compareFold(one.Database, other.Database)
		}

		return 0
	})

	height := rowHeight()

	clipper := imgui.NewListClipper()
	defer clipper.Destroy()

	clipper.Begin(int32(len(rows)))

	for clipper.Step() {
		for row := clipper.DisplayStart(); row < clipper.DisplayEnd(); row++ {
			index := rows[row]
			texture := &library.textures[index]

			imgui.TableNextRow()
			imgui.PushIDInt(int32(index))

			imgui.TableSetColumnIndex(textureColumnPreview)
			a.textureThumbnail(texture.Path)

			imgui.TableSetColumnIndex(textureColumnName)
			highlightedCell(texture.Name, query, height)

			imgui.TableSetColumnIndex(textureColumnKind)
			highlightedCell(texture.Kind.String(), query, height)

			imgui.TableSetColumnIndex(textureColumnFolder)
			highlightedCell(texture.Database, query, height)

			// Using a texture needs a flag to use it on.
			imgui.TableSetColumnIndex(textureColumnUse)
			centredCell(height, imgui.TextLineHeight())
			imgui.BeginDisabledV(a.state.flag == nil)

			if gui.SmallButton(textureAction(texture.Kind)) {
				a.useTexture(*texture)
			}

			imgui.EndDisabled()
			imgui.PopID()
		}
	}

	clipper.End()
}

// describeDatabase is the one line summary shown for a configured folder.
func describeDatabase(entry *database.Database) string {
	return fmt.Sprintf("%s: %s, %d flags, %d textures",
		entry.Name, entry.Game, len(entry.Flags), len(entry.Textures))
}

// Labels of the texture database's row actions.
const (
	labelSetPattern = "Set as Pattern"
	labelAddAsLayer = "Add as Layer"
)

func textureAction(kind database.TextureKind) string {
	if kind == database.PatternTexture {
		return labelSetPattern
	}

	return labelAddAsLayer
}

// useTexture puts a texture from the database to work on the open flag: a
// pattern replaces the flag's pattern, an emblem becomes a new layer.
func (a *App) useTexture(texture database.Texture) {
	flag := a.state.flag
	if flag == nil {
		return
	}

	switch texture.Kind {
	case database.PatternTexture:
		flag.Pattern = texture.Name
		a.setStatus("Pattern set to %s", texture.Name)

	case database.ColoredEmblemTexture:
		a.addLayer(pdx.NewColoredEmblem(texture.Name))
		a.setStatus("Added %s", texture.Name)

	case database.TexturedEmblemTexture:
		a.addLayer(pdx.NewTexturedEmblem(texture.Name))
		a.setStatus("Added %s", texture.Name)
	}

	a.changed()
}
