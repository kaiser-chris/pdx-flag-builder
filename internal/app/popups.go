package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Labels of the unsaved changes dialog.
const (
	labelDiscard = "Discard Changes"
	labelCancel  = "Cancel"
)

// openRequestedPopup opens the modal asked for during the last frame.
//
// A popup belongs to the id stack it was opened from. Menus and panels each
// push their own ids, so a modal opened straight from a menu item would never
// be found by the BeginPopupModal at the top level that draws it. Requests are
// therefore collected in state and opened here, at the top level, instead.
func (a *App) openRequestedPopup() {
	if a.state.popup == "" {
		return
	}

	imgui.OpenPopupStr(a.state.popup)
	a.state.popup = ""
}

// requestOpen opens a coat of arms in the editor, first asking what should
// happen to unsaved changes to the one open now.
func (a *App) requestOpen(flag pdx.Flag) {
	if a.state.flag == nil || !a.state.modified {
		a.openFlag(flag)

		return
	}

	pending := flag.Clone()
	a.state.pendingFlag = &pending
	a.state.popup = popupDiscard
}

// newFlag is the coat of arms File > New Flag starts from: nothing but a name.
func newFlag() pdx.Flag {
	return pdx.Flag{Name: "new_flag"}
}

func (a *App) discardPopup() {
	imgui.SetNextWindowSizeV(gui.ScaledVec2(440, 0), imgui.CondAlways)

	if !imgui.BeginPopupModalV(popupDiscard, nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoSavedSettings) {
		return
	}
	defer imgui.EndPopup()

	pending := a.state.pendingFlag
	if pending == nil || a.state.flag == nil {
		imgui.CloseCurrentPopup()

		return
	}

	imgui.TextWrapped(fmt.Sprintf("%s has changes that have not been saved. Opening %s discards them.",
		a.state.flag.Name, pending.Name))

	imgui.Spacing()

	if gui.Button(labelDiscard) {
		a.openFlag(*pending)
		a.state.pendingFlag = nil
		imgui.CloseCurrentPopup()
	}

	imgui.SameLine()

	if gui.Button(labelCancel) {
		a.state.pendingFlag = nil
		imgui.CloseCurrentPopup()
	}
}
