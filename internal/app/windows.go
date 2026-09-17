package app

import (
	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
)

// applicationVersion is reported in the about dialog. The Odin version this
// rewrite follows was at 1.4.0.
const applicationVersion = "0.1.0-dev"

// settingsWindow edits the configured game and mod folders and the window
// background. Edits are made on copies and only written to disk on save.
func (a *App) settingsWindow() {
	if !a.state.showSettings {
		return
	}

	imgui.SetNextWindowSizeV(imgui.Vec2{X: 620, Y: 460}, imgui.CondFirstUseEver)

	if imgui.BeginV(windowSettings, &a.state.showSettings, 0) {
		a.trackFocus(windowSettings)

		imgui.SeparatorText("Appearance")

		if imgui.ColorEdit4("Background", &a.state.backgroundColor) {
			// Applied immediately so the choice can be judged, but only stored
			// when the user saves.
			a.window.SetBackground(colorFromFloats(a.state.backgroundColor))
		}

		imgui.SeparatorText("Game and Mod Folders")
		dimmedWrapped("Point these at a game folder or a mod folder to load its flags and textures.")

		a.databaseTable()

		if imgui.Button("Add Folder") {
			a.state.databases = append(a.state.databases, databaseEntry{})
		}

		imgui.Separator()

		if imgui.Button("Save") {
			a.saveSettings()
		}

		imgui.SameLine()

		if imgui.Button("Revert") {
			a.state.loadFrom(a.settings)
			a.window.SetBackground(a.backgroundColor())
			a.setStatus("Settings reverted")
		}
	}

	imgui.End()
}

func (a *App) databaseTable() {
	flags := imgui.TableFlagsBordersInnerH |
		imgui.TableFlagsRowBg |
		imgui.TableFlagsSizingStretchProp

	// Leave room below the table for the buttons that follow it.
	outerSize := imgui.Vec2{X: 0, Y: max(imgui.ContentRegionAvail().Y-imgui.FrameHeightWithSpacing()*3, imgui.FrameHeightWithSpacing()*2)}

	if !imgui.BeginTableV("databases", 3, flags, outerSize, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthFixed, 150, 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsWidthFixed, 28, 0)
	imgui.TableHeadersRow()

	remove := -1

	for index := range a.state.databases {
		entry := &a.state.databases[index]

		imgui.PushIDInt(int32(index))

		imgui.TableNextRow()

		imgui.TableSetColumnIndex(0)
		imgui.SetNextItemWidth(-1)
		imgui.InputTextWithHint("##name", "game", &entry.Name, 0, nil)

		imgui.TableSetColumnIndex(1)
		imgui.SetNextItemWidth(-1)
		// TODO: a Browse button needs a native folder picker. The Odin version
		// used nativefiledialog; the Go port has to pick a cross platform
		// replacement before this can be wired up.
		imgui.InputTextWithHint("##path", "path to a game or mod folder", &entry.Path, 0, nil)

		imgui.TableSetColumnIndex(2)
		if imgui.Button("X") {
			remove = index
		}

		imgui.PopID()
	}

	if remove >= 0 {
		a.state.databases = append(a.state.databases[:remove], a.state.databases[remove+1:]...)
	}
}

func (a *App) saveSettings() {
	background := colorFromFloats(a.state.backgroundColor)
	a.settings.BackgroundColor = config.Color{R: background.R, G: background.G, B: background.B, A: background.A}

	databases := make([]config.Database, 0, len(a.state.databases))
	for _, entry := range a.state.databases {
		if entry.Name == "" || entry.Path == "" {
			// A half filled row is an edit in progress, not a folder to store.
			continue
		}

		databases = append(databases, config.Database{Name: entry.Name, Path: entry.Path})
	}
	a.settings.Databases = databases

	if err := a.settings.Save(); err != nil {
		warn(err)
		a.setStatus("Settings could not be saved: %v", err)

		return
	}

	a.setStatus("Settings saved")
}

// flagDatabaseWindow will list every flag found in the configured folders.
func (a *App) flagDatabaseWindow() {
	if !a.state.showFlagDatabase {
		return
	}

	imgui.SetNextWindowSizeV(imgui.Vec2{X: 720, Y: 520}, imgui.CondFirstUseEver)

	if imgui.BeginV(windowFlagDatabase, &a.state.showFlagDatabase, 0) {
		a.trackFocus(windowFlagDatabase)

		// TODO: fill once the coat of arms parser is ported.
		emptyState(
			"No flags loaded.",
			"Flags are read from the folders configured in the settings",
			"once the Paradox script parser is ported.",
		)
	}

	imgui.End()
}

// textureDatabaseWindow will list the patterns and emblems available to build with.
func (a *App) textureDatabaseWindow() {
	if !a.state.showTextureDatabase {
		return
	}

	imgui.SetNextWindowSizeV(imgui.Vec2{X: 720, Y: 520}, imgui.CondFirstUseEver)

	if imgui.BeginV(windowTextureDatabase, &a.state.showTextureDatabase, 0) {
		a.trackFocus(windowTextureDatabase)

		// TODO: fill once the DDS and BC7 texture loading is ported.
		emptyState(
			"No textures loaded.",
			"Patterns and emblems appear here once the texture loader is ported.",
		)
	}

	imgui.End()
}

func (a *App) aboutPopup() {
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 380, Y: 0}, imgui.CondAlways)

	if !imgui.BeginPopupModalV(popupAbout, nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoSavedSettings) {
		return
	}
	defer imgui.EndPopup()

	imgui.TextUnformatted(applicationName + " " + applicationVersion)
	imgui.TextDisabled("Flag editor for Victoria 3 and Europa Universalis 5")

	imgui.Separator()

	imgui.TextDisabled("Rendering by raylib, interface by Dear ImGui.")

	imgui.Separator()

	if imgui.Button("Close") {
		imgui.CloseCurrentPopup()
	}
}
