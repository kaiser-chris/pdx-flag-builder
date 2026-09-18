package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
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

	imgui.SetNextWindowSizeV(gui.ScaledVec2(640, 520), imgui.CondFirstUseEver)

	if imgui.BeginV(windowSettings, &a.state.showSettings, 0) {
		a.trackFocus(windowSettings)

		sectionHeader("Appearance")

		if gui.ColorEdit("Background", &a.state.backgroundColor) {
			// Applied immediately so the choice can be judged, but only stored
			// when the user saves.
			a.window.SetBackground(colorFromFloats(a.state.backgroundColor))
		}

		a.scaleField()

		sectionHeader("Game and Mod Folders")
		dimmedWrapped("Point these at a game folder or a mod folder to load its flags and textures. " +
			"A folder listed further down overrides the ones above it, the way a mod overrides the game.")

		a.databaseTable()

		if gui.Button("Add Folder") {
			a.state.databases = append(a.state.databases, databaseEntry{})
		}

		imgui.SameLine()

		if gui.Button("Save") {
			a.saveSettings()
		}

		imgui.SameLine()

		if gui.Button("Revert") {
			a.state.loadFrom(a.settings)
			a.window.SetBackground(a.backgroundColor())
			a.setStatus("Settings reverted")
		}

		a.loadedFolders()
	}

	imgui.End()
}

func (a *App) databaseTable() {
	// Leave room below the table for the buttons and the summary that follow it.
	height := imgui.FrameHeightWithSpacing() * 6
	outerSize := imgui.Vec2{X: 0, Y: min(height, max(imgui.ContentRegionAvail().Y-height, height))}

	if !imgui.BeginTableV("databases", 3, imgui.TableFlagsBordersInnerH|imgui.TableFlagsRowBg, outerSize, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthFixed, gui.Scaled(150), 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsWidthFixed, gui.Scaled(28), 0)
	imgui.TableHeadersRow()

	remove := -1

	for index := range a.state.databases {
		entry := &a.state.databases[index]

		imgui.PushIDInt(int32(index))

		imgui.TableNextRow()

		imgui.TableSetColumnIndex(0)
		imgui.SetNextItemWidth(-1)
		gui.InputText("##name", "game", &entry.Name)

		imgui.TableSetColumnIndex(1)
		imgui.SetNextItemWidth(-1)
		// TODO: a Browse button needs a native folder picker. The Odin version
		// used nativefiledialog; the Go port has to pick a cross platform
		// replacement before this can be wired up.
		gui.InputText("##path", "path to a game or mod folder", &entry.Path)

		imgui.TableSetColumnIndex(2)
		if gui.Button("X") {
			remove = index
		}

		imgui.PopID()
	}

	if remove >= 0 {
		a.state.databases = append(a.state.databases[:remove], a.state.databases[remove+1:]...)
	}
}

// loadedFolders reports what the configured folders actually produced, which is
// the quickest way to see that a path is pointing somewhere useful.
func (a *App) loadedFolders() {
	library := &a.state.library

	sectionHeader("Loaded")

	if library.loading {
		imgui.TextDisabled("Reading...")

		return
	}

	if len(library.set) == 0 {
		imgui.TextDisabled("Nothing loaded yet.")

		return
	}

	for _, entry := range library.set {
		imgui.BulletText(describeDatabase(entry))
	}

	if count := len(library.problems); count > 0 {
		imgui.Spacing()
		dimmedWrapped(problemSummary(library.problems, count))
	}
}

func (a *App) saveSettings() {
	background := colorFromFloats(a.state.backgroundColor)
	a.settings.BackgroundColor = config.Color{R: background.R, G: background.G, B: background.B, A: background.A}
	a.settings.InterfaceScale = a.state.interfaceScale

	databases := make([]config.Database, 0, len(a.state.databases))
	for _, entry := range a.state.databases {
		if entry.Name == "" || entry.Path == "" {
			// A half filled row is an edit in progress, not a folder to store.
			continue
		}

		databases = append(databases, config.Database{Name: entry.Name, Path: entry.Path})
	}
	a.settings.Databases = databases

	if err := a.store.Save(a.settings); err != nil {
		warn(err)
		a.setStatus("Settings could not be saved: %v", err)

		return
	}

	// The folders may have changed, so what was read from them is now stale.
	a.state.library.reload(a.settings.Databases)
	a.setStatus("Settings saved, reading the configured folders")
}

func (a *App) aboutPopup() {
	imgui.SetNextWindowSizeV(gui.ScaledVec2(380, 0), imgui.CondAlways)

	if !imgui.BeginPopupModalV(popupAbout, nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoSavedSettings) {
		return
	}
	defer imgui.EndPopup()

	gui.PushStrongFont()
	imgui.TextUnformatted(applicationName + " " + applicationVersion)
	gui.PopFont()

	imgui.TextDisabled("Flag editor for Victoria 3 and Europa Universalis 5")

	imgui.Separator()

	imgui.TextDisabled("Rendering by raylib, interface by Dear ImGui.")

	imgui.Separator()

	if gui.Button("Close") {
		imgui.CloseCurrentPopup()
	}
}

// scaleChoices are the interface scales offered besides following the monitor.
var scaleChoices = []float32{1, 1.25, 1.5, 1.75, 2, 2.5, 3}

func scaleLabel(scale float32) string {
	return fmt.Sprintf("%.0f%%", scale*100)
}

// automaticScaleLabel names the choice that follows the monitor, with the
// scale that works out to right now.
func automaticScaleLabel() string {
	return fmt.Sprintf("Automatic (%s)", scaleLabel(gui.MonitorScale()))
}

// scaleField chooses how large the interface is drawn. Like the background, a
// choice shows straight away but is only stored when the settings are saved.
func (a *App) scaleField() {
	current := a.state.interfaceScale

	preview := automaticScaleLabel()
	if current > 0 {
		preview = scaleLabel(current)
	}

	imgui.SetNextItemWidth(imgui.FontSize() * 12)

	if !gui.BeginCombo("Interface Scale", preview) {
		return
	}
	defer imgui.EndCombo()

	if gui.Selectable(automaticScaleLabel(), current == 0, 0) {
		a.state.interfaceScale = 0
	}

	for _, choice := range scaleChoices {
		if gui.Selectable(scaleLabel(choice), current == choice, 0) {
			a.state.interfaceScale = choice
		}
	}
}

// interfaceScale is the scale the interface should be drawn at: the one chosen
// in the settings, or the monitor's.
func (a *App) interfaceScale() float32 {
	if a.state.interfaceScale > 0 {
		return a.state.interfaceScale
	}

	return gui.MonitorScale()
}
