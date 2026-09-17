package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
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

	if imgui.BeginMenu("File") {
		if imgui.MenuItemBoolV("New Flag", "", false, true) {
			a.setStatus("New flag")
		}

		imgui.Separator()

		// Everything below waits on the flag model and the exporters, which are
		// ported in a later step. They are listed but disabled so the shape of
		// the application is visible.
		imgui.MenuItemBoolV("Save Changes", "Ctrl+S", false, false)

		if imgui.BeginMenu("Export") {
			imgui.MenuItemBoolV("To Image...", "", false, false)
			imgui.MenuItemBoolV("To Clipboard", "", false, false)
			imgui.MenuItemBoolV("As New Script File...", "", false, false)
			imgui.EndMenu()
		}

		imgui.Separator()

		if imgui.MenuItemBoolV("Exit", "Alt+F4", false, true) {
			a.window.RequestClose()
		}

		imgui.EndMenu()
	}

	if imgui.BeginMenu("Databases") {
		imgui.MenuItemBoolPtr(windowFlagDatabase, "", &a.state.showFlagDatabase)
		imgui.MenuItemBoolPtr(windowTextureDatabase, "", &a.state.showTextureDatabase)
		imgui.EndMenu()
	}

	if imgui.BeginMenu("View") {
		imgui.MenuItemBoolPtr(panelLayers, "", &a.state.showLayers)
		imgui.MenuItemBoolPtr(panelSelected, "", &a.state.showSelected)

		imgui.Separator()

		if imgui.MenuItemBoolV("Reset Layout", "", false, true) {
			a.state.layoutBuilt = false
			a.state.showLayers = true
			a.state.showSelected = true
			a.setStatus("Layout reset")
		}

		imgui.EndMenu()
	}

	if imgui.BeginMenu("Settings") {
		imgui.MenuItemBoolPtr("Open Settings", "Ctrl+,", &a.state.showSettings)
		imgui.EndMenu()
	}

	if imgui.BeginMenu("Help") {
		if imgui.MenuItemBoolV("About", "", false, true) {
			imgui.OpenPopupStr(popupAbout)
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

			width, height := a.preview.Size()
			right := fmt.Sprintf("%d x %d  |  %d folders  |  %.0f FPS",
				width, height, len(a.settings.Databases), imgui.CurrentIO().Framerate())

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
		a.setStatus("Saving is not available until the flag model is ported")
	}

	if io.KeyCtrl() && imgui.IsKeyPressedBool(imgui.KeyComma) {
		a.state.showSettings = !a.state.showSettings
	}

	// Escape closes the window on top. While a text field is being edited Dear
	// ImGui uses Escape to revert the edit, so leave it alone then.
	if imgui.IsKeyPressedBool(imgui.KeyEscape) && !io.WantTextInput() {
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
