package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Labels of the unsaved changes dialog.
const (
	labelSaveFirst = "Save"
	labelDiscard   = "Discard Changes"
	labelCancel    = "Cancel"
)

// pendingAction is something that would throw away unsaved changes, held back
// until the user has said what should happen to them.
type pendingAction struct {
	// what says what the action is, to finish the sentence "... discards them".
	what string

	run func()
}

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
	a.confirmDiscard("Opening "+pending.Name, func() { a.openFlag(pending) })
}

// requestQuit closes the application, first asking what should happen to
// unsaved changes.
func (a *App) requestQuit() {
	if a.allowQuit() {
		a.window.RequestClose()
	}
}

// allowQuit is asked when the user closes the window. With unsaved changes it
// says no and asks about them instead; the close happens once they are dealt
// with.
func (a *App) allowQuit() bool {
	if a.state.flag == nil || !a.state.modified {
		return true
	}

	a.confirmDiscard("Closing "+applicationName, a.window.RequestClose)

	return false
}

// confirmDiscard holds an action back behind the unsaved changes dialog.
func (a *App) confirmDiscard(what string, run func()) {
	a.state.pending = &pendingAction{what: what, run: run}
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

	pending := a.state.pending
	if pending == nil || a.state.flag == nil {
		imgui.CloseCurrentPopup()

		return
	}

	imgui.TextWrapped(fmt.Sprintf("%s has changes that have not been saved. %s discards them.",
		a.state.flag.Name, pending.what))

	imgui.Spacing()

	// Saving right here only works for a flag that already has a file. A
	// new one would need the file dialog first, and the user can do that
	// from the File menu after cancelling.
	if a.state.flag.Origin.Path != "" {
		// Enter takes the choice that loses nothing. It never discards.
		if gui.Button(labelSaveFirst) || imgui.IsKeyPressedBool(imgui.KeyEnter) {
			a.save()

			// A save that failed says why in the status bar, and the
			// changes stay.
			if !a.state.modified {
				pending.run()
			}

			a.state.pending = nil
			imgui.CloseCurrentPopup()
		}

		imgui.SameLine()
	}

	if gui.Button(labelDiscard) {
		pending.run()
		a.state.pending = nil
		imgui.CloseCurrentPopup()
	}

	imgui.SameLine()

	if gui.Button(labelCancel) || imgui.IsKeyPressedBool(imgui.KeyEscape) {
		a.state.pending = nil
		imgui.CloseCurrentPopup()
	}
}
