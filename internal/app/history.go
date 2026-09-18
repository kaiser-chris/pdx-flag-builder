package app

import (
	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// maxUndoSteps bounds the history. A coat of arms is a few kilobytes, so this
// is generous without being unbounded.
const maxUndoSteps = 200

// history is the undo and redo stack of the flag open in the editor.
//
// An undo step is one gesture rather than one change: dragging a position
// changes it every frame, but undoing it should put the emblem back where the
// drag started, not one frame earlier. So the flag is snapshotted whenever no
// widget is being worked, and the first change of a gesture pushes that
// snapshot. The gesture ends when Dear ImGui reports no active widget, which
// for a button is the frame it was released in.
type history struct {
	undo []pdx.Flag
	redo []pdx.Flag

	// stable is the flag as it was before the gesture under way began.
	stable pdx.Flag

	// recorded is set once the current gesture has pushed its undo step.
	recorded bool

	// dirty is set when the flag changed since stable was taken.
	dirty bool
}

// reset starts a fresh history for a newly opened flag.
func (h *history) reset(flag pdx.Flag) {
	h.undo = nil
	h.redo = nil
	h.stable = flag.Clone()
	h.recorded = false
	h.dirty = false
}

// changed notes that the open flag has just been edited.
func (a *App) changed() {
	history := &a.state.history

	if !history.recorded {
		history.undo = append(history.undo, history.stable)
		if len(history.undo) > maxUndoSteps {
			history.undo = history.undo[1:]
		}

		history.redo = nil
		history.recorded = true
	}

	history.dirty = true
	a.state.modified = true
}

// settleHistory closes the gesture under way once nothing is being worked on,
// and takes the snapshot the next gesture will return to. It runs at the end
// of every frame.
func (a *App) settleHistory() {
	if imgui.IsAnyItemActive() || a.state.flag == nil {
		return
	}

	history := &a.state.history
	history.recorded = false

	if history.dirty {
		history.stable = a.state.flag.Clone()
		history.dirty = false
	}
}

func (a *App) canUndo() bool { return a.state.flag != nil && len(a.state.history.undo) > 0 }
func (a *App) canRedo() bool { return a.state.flag != nil && len(a.state.history.redo) > 0 }

// undo steps the open flag back one gesture.
func (a *App) undo() {
	if !a.canUndo() {
		return
	}

	history := &a.state.history
	last := len(history.undo) - 1

	history.redo = append(history.redo, a.state.flag.Clone())
	a.restore(history.undo[last])
	history.undo = history.undo[:last]

	a.setStatus("Undone")
}

// redo steps forward again after an undo.
func (a *App) redo() {
	if !a.canRedo() {
		return
	}

	history := &a.state.history
	last := len(history.redo) - 1

	history.undo = append(history.undo, a.state.flag.Clone())
	a.restore(history.redo[last])
	history.redo = history.redo[:last]

	a.setStatus("Redone")
}

func (a *App) restore(flag pdx.Flag) {
	restored := flag.Clone()
	a.state.flag = &restored

	a.state.history.stable = flag.Clone()
	a.state.history.dirty = false
	a.state.modified = true

	// The selection may point past the end of a flag with fewer layers.
	if a.state.selectedLayer >= len(restored.Layers) {
		a.state.selectedLayer = noLayer
	}
}
