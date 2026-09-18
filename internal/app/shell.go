package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
)

// dockNodeFlagsDockSpace is ImGuiDockNodeFlags_DockSpace from imgui_internal.h.
// cimgui-go only generates the public flags, but building a default layout
// needs this one: it marks the root node as able to host a dock space.
const dockNodeFlagsDockSpace imgui.DockNodeFlags = 1 << 10

// sidebarRatio is the share of the window the sidebar takes when the default
// layout is built. The Odin version defaulted to a fixed 350 pixel sidebar;
// a ratio keeps the proportion sensible on wide monitors.
const sidebarRatio = 0.26

// dockSpace covers the remaining viewport with a dock space so every panel can
// be rearranged, tabbed or torn off by the user.
func (a *App) dockSpace() {
	// PassthruCentralNode keeps the central node transparent when no panel is
	// docked into it, so the window background stays visible.
	id := imgui.DockSpaceOverViewportV(0, imgui.MainViewport(), imgui.DockNodeFlagsPassthruCentralNode, a.dockWindowClass)

	if !a.state.layoutBuilt {
		a.state.layoutBuilt = true
		a.buildDefaultLayout(id)
	}
}

// buildDefaultLayout arranges the panels the way the Odin version did: the flag
// preview fills the window and the two sidebar panels sit stacked on the right.
// It only runs when there is no saved layout, or when the user resets it.
func (a *App) buildDefaultLayout(dockSpaceID imgui.ID) {
	imgui.InternalDockBuilderRemoveNode(dockSpaceID)
	imgui.InternalDockBuilderAddNodeV(dockSpaceID, dockNodeFlagsDockSpace)
	imgui.InternalDockBuilderSetNodeSize(dockSpaceID, imgui.MainViewport().WorkSize())

	var sidebar, center imgui.ID
	imgui.InternalDockBuilderSplitNode(dockSpaceID, imgui.DirRight, sidebarRatio, &sidebar, &center)

	var sidebarTop, sidebarBottom imgui.ID
	imgui.InternalDockBuilderSplitNode(sidebar, imgui.DirUp, 0.55, &sidebarTop, &sidebarBottom)

	imgui.InternalDockBuilderDockWindow(panelPreview, center)
	imgui.InternalDockBuilderDockWindow(panelLayers, sidebarTop)
	imgui.InternalDockBuilderDockWindow(panelSelected, sidebarBottom)

	imgui.InternalDockBuilderFinish(dockSpaceID)
}

func (a *App) menuBar() {
	if !imgui.BeginMainMenuBar() {
		return
	}
	defer imgui.EndMainMenuBar()

	if gui.BeginMenu("File") {
		if gui.MenuItem("New Flag", "", true) {
			a.requestOpen(newFlag())
		}

		imgui.Separator()

		open := a.state.flag != nil

		if gui.MenuItem(labelSave, "Ctrl+S", open) {
			a.save()
		}

		if gui.MenuItem(labelSaveToFile, "Ctrl+Shift+S", open) {
			a.saveToFile()
		}

		imgui.Separator()

		if gui.MenuItem(labelCopyScript, "", open) {
			a.copyScript()
		}

		if gui.MenuItem(labelExportImage, "", open) {
			a.exportImage()
		}

		imgui.Separator()

		if gui.MenuItem("Exit", "Alt+F4", true) {
			a.requestQuit()
		}

		imgui.EndMenu()
	}

	if gui.BeginMenu("Edit") {
		if gui.MenuItem("Undo", "Ctrl+Z", a.canUndo()) {
			a.undo()
		}

		if gui.MenuItem("Redo", "Ctrl+Y", a.canRedo()) {
			a.redo()
		}

		imgui.EndMenu()
	}

	if gui.BeginMenu("Databases") {
		gui.MenuToggle(windowFlagDatabase, "", &a.state.showFlagDatabase)
		gui.MenuToggle(windowTextureDatabase, "", &a.state.showTextureDatabase)
		imgui.EndMenu()
	}

	if gui.BeginMenu("View") {
		gui.MenuToggle(panelLayers, "", &a.state.showLayers)
		gui.MenuToggle(panelSelected, "", &a.state.showSelected)

		imgui.Separator()

		if gui.MenuItem("Reset Layout", "", true) {
			a.state.layoutBuilt = false
			a.state.showLayers = true
			a.state.showSelected = true
			a.setStatus("Layout reset")
		}

		imgui.EndMenu()
	}

	if gui.BeginMenu("Settings") {
		gui.MenuToggle("Open Settings", "Ctrl+,", &a.state.showSettings)
		imgui.EndMenu()
	}

	if gui.BeginMenu("Help") {
		if gui.MenuItem("About", "", true) {
			a.state.popup = popupAbout
		}
		imgui.EndMenu()
	}
}

// statusBar claims a strip along the bottom of the viewport. It is submitted
// before the dock space so that panels do not overlap it.
func (a *App) statusBar() {
	flags := imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsMenuBar

	if imgui.InternalBeginViewportSideBar("##StatusBar", imgui.MainViewport(), imgui.DirDown, imgui.FrameHeight(), flags) {
		if imgui.BeginMenuBar() {
			imgui.TextUnformatted(a.state.status)

			right := a.diagnostics()

			// Right align the diagnostics.
			imgui.SameLine()
			imgui.SetCursorPosX(imgui.ContentRegionAvail().X - imgui.CalcTextSize(right).X)
			imgui.TextDisabled(right)

			imgui.EndMenuBar()
		}

		imgui.End()
	}
}

// handleShortcuts implements the keyboard shortcuts that are not attached to a
// single panel.
func (a *App) handleShortcuts() {
	io := imgui.CurrentIO()

	if io.KeyCtrl() && imgui.IsKeyPressedBool(imgui.KeyS) {
		if io.KeyShift() {
			a.saveToFile()
		} else {
			a.save()
		}
	}

	if io.KeyCtrl() && imgui.IsKeyPressedBool(imgui.KeyComma) {
		a.state.showSettings = !a.state.showSettings
	}

	// A text field being edited has an undo of its own, which is the one
	// Ctrl+Z should reach while typing.
	if io.KeyCtrl() && !io.WantTextInput() {
		switch {
		case imgui.IsKeyPressedBool(imgui.KeyZ) && io.KeyShift(), imgui.IsKeyPressedBool(imgui.KeyY):
			a.redo()
		case imgui.IsKeyPressedBool(imgui.KeyZ):
			a.undo()
		}
	}

	// Escape closes the window on top. While a text field is being edited Dear
	// ImGui uses Escape to revert the edit, so leave it alone then.
	// A dialog that was open answers Escape itself. It has closed by now, so
	// what counts is whether one was open when the frame began.
	if imgui.IsKeyPressedBool(imgui.KeyEscape) && !io.WantTextInput() && !a.state.popupWasOpen {
		a.closeFocusedWindow()
	}
}

func (a *App) closeFocusedWindow() {
	switch a.state.focusedWindow {
	case windowSettings:
		a.state.showSettings = false
	case windowFlagDatabase:
		a.state.showFlagDatabase = false
	case windowTextureDatabase:
		a.state.showTextureDatabase = false
	default:
		return
	}

	a.state.focusedWindow = ""
}

// diagnostics is the right hand side of the status bar: what has been read, and
// how the interface itself is doing.
func (a *App) diagnostics() string {
	library := &a.state.library

	if library.loading {
		return fmt.Sprintf("reading folders...  |  %.0f FPS", imgui.CurrentIO().Framerate())
	}

	summary := fmt.Sprintf("%d flags  |  %d textures", len(library.flags), len(library.textures))

	// Artwork that could not be read shows up as a missing layer, so say so
	// rather than leave the user wondering.
	if _, _, failed := a.textures.Counts(); failed > 0 {
		summary += fmt.Sprintf(" (%d unreadable)", failed)
	}

	return fmt.Sprintf("%s  |  %.0f FPS", summary, imgui.CurrentIO().Framerate())
}
