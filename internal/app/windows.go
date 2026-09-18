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

// settingsWidth is how wide the settings window opens, in unscaled units.
const settingsWidth = 640

// settingsWindow edits the interface scale and the configured game and mod
// folders. Edits are made on copies and only written to disk on save.
func (a *App) settingsWindow() {
	if !a.state.showSettings {
		return
	}

	// The window keeps whatever width it is given but always fits its contents
	// in height: a height of zero asks Dear ImGui to fit it, and asking every
	// frame keeps it fitting as folders are added and removed.
	width := gui.Scaled(settingsWidth)
	if window, found := gui.FindWindow(windowSettings); found && window.SizeFull().X > 0 {
		width = window.SizeFull().X
	}

	imgui.SetNextWindowSizeV(imgui.Vec2{X: width}, imgui.CondAlways)

	if imgui.BeginV(windowSettings, &a.state.showSettings, 0) {
		a.trackFocus(windowSettings)

		sectionHeader("Appearance")

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
			a.setStatus("Settings reverted")
		}

		a.loadedFolders()
	}

	imgui.End()
}

func (a *App) databaseTable() {
	// The same breathing room around the header and between the rows that the
	// lists with previews have. Dear ImGui reads the padding row by row, so it
	// stays pushed until the table ends.
	padding := imgui.CurrentStyle().CellPadding()
	imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, imgui.Vec2{X: padding.X, Y: gui.Scaled(5)})
	defer imgui.PopStyleVar()

	// No height of its own: the table grows by a row for every folder added,
	// and the window grows with it. Without outer borders Dear ImGui leaves
	// the outer edges unpadded, which puts the first column flush against the
	// edge while every other one is indented, so the padding is asked for.
	flags := imgui.TableFlagsBordersInnerH | imgui.TableFlagsRowBg | imgui.TableFlagsPadOuterX
	if !imgui.BeginTableV("databases", 3, flags, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsWidthFixed, gui.Scaled(150), 0)
	imgui.TableSetupColumnV("Folder", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsWidthFixed, gui.Scaled(28), 0)
	gui.TableHeadersRow()

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

// scaleField chooses how large the interface is drawn. A choice shows straight
// away but is only stored when the settings are saved.
func (a *App) scaleField() {
	current := a.state.interfaceScale

	preview := automaticScaleLabel()
	if current > 0 {
		preview = scaleLabel(current)
	}

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
