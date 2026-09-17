package app

import (
	"fmt"
	"strconv"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
)

// tableFlags are shared by both database listings: scrollable, striped, and
// with columns the user can resize.
const tableFlags = imgui.TableFlagsRowBg |
	imgui.TableFlagsBordersInnerV |
	imgui.TableFlagsScrollY |
	imgui.TableFlagsResizable |
	imgui.TableFlagsSizingStretchProp

// flagDatabaseWindow lists every coat of arms found in the configured folders.
func (a *App) flagDatabaseWindow() {
	if !a.state.showFlagDatabase {
		return
	}

	imgui.SetNextWindowSizeV(imgui.Vec2{X: 760, Y: 560}, imgui.CondFirstUseEver)

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
	imgui.InputTextWithHint("##flag-search", "Search by name, folder or file", &a.state.flagSearch, 0, nil)

	rows := a.state.flagRows.get(
		a.state.flagSearch,
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

	if !imgui.BeginTableV("flags", 4, tableFlags, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("Layers", imgui.TableColumnFlagsWidthFixed, 60, 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthFixed, 110, 0)
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	// The list runs to thousands of rows, so only the visible ones are built.
	clipper := imgui.NewListClipper()
	defer clipper.Destroy()

	clipper.Begin(int32(len(rows)))

	for clipper.Step() {
		for row := clipper.DisplayStart(); row < clipper.DisplayEnd(); row++ {
			flag := &library.flags[rows[row]]

			imgui.TableNextRow()

			imgui.TableSetColumnIndex(0)

			open := a.state.flag != nil &&
				a.state.flag.Name == flag.Name &&
				a.state.flag.Origin.Path == flag.Origin.Path

			if imgui.SelectableBoolV(flag.Name, open, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{}) {
				a.openFlag(*flag)
			}

			imgui.TableSetColumnIndex(1)
			imgui.TextUnformatted(strconv.Itoa(len(flag.Layers)))

			imgui.TableSetColumnIndex(2)
			imgui.TextUnformatted(flag.Origin.Database)

			imgui.TableSetColumnIndex(3)
			imgui.TextUnformatted(flag.Origin.File)
		}
	}

	clipper.End()
}

// textureDatabaseWindow lists the patterns and emblems available to build with.
func (a *App) textureDatabaseWindow() {
	if !a.state.showTextureDatabase {
		return
	}

	imgui.SetNextWindowSizeV(imgui.Vec2{X: 700, Y: 560}, imgui.CondFirstUseEver)

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
	imgui.InputTextWithHint("##texture-search", "Search by name or kind", &a.state.textureSearch, 0, nil)

	rows := a.state.textureRows.get(
		a.state.textureSearch,
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

	// TODO: show the image itself once the DDS and BC7 loading is ported.
	if !imgui.BeginTableV("textures", 3, tableFlags, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("Kind", imgui.TableColumnFlagsWidthFixed, 130, 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthFixed, 110, 0)
	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	clipper := imgui.NewListClipper()
	defer clipper.Destroy()

	clipper.Begin(int32(len(rows)))

	for clipper.Step() {
		for row := clipper.DisplayStart(); row < clipper.DisplayEnd(); row++ {
			texture := &library.textures[rows[row]]

			imgui.TableNextRow()

			imgui.TableSetColumnIndex(0)
			imgui.TextUnformatted(texture.Name)

			imgui.TableSetColumnIndex(1)
			imgui.TextUnformatted(texture.Kind.String())

			imgui.TableSetColumnIndex(2)
			imgui.TextUnformatted(texture.Database)
		}
	}

	clipper.End()
}

// describeDatabase is the one line summary shown for a configured folder.
func describeDatabase(entry *database.Database) string {
	return fmt.Sprintf("%s: %s, %d flags, %d textures",
		entry.Name, entry.Game, len(entry.Flags), len(entry.Textures))
}
