package app

import (
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Panel and window titles.
//
// Dear ImGui identifies a window by its title and the saved docking layout
// refers to the same strings, so renaming one of these drops the user's stored
// position for that panel.
const (
	panelPreview  = "Flag Preview"
	panelLayers   = "Layers"
	panelSelected = "Selected Layer"

	windowSettings        = "Settings"
	windowFlagDatabase    = "Flag Database"
	windowTextureDatabase = "Texture Database"

	popupAbout   = "About"
	popupDiscard = "Unsaved Changes"
	popupPicker  = "Choose###picker"
)

// noLayer is what selectedLayer holds when nothing is selected.
const noLayer = -1

// databaseEntry is one row of the settings window's folder table. The settings
// window edits these copies and only writes them back when the user saves, so
// abandoning an edit cannot corrupt the stored configuration.
type databaseEntry struct {
	Name string
	Path string
}

// state is everything the interface remembers between frames that is not worth
// persisting to disk.
type state struct {
	// Panels that live in the docking layout.
	showLayers   bool
	showSelected bool

	// Windows that float above it.
	showSettings        bool
	showFlagDatabase    bool
	showTextureDatabase bool

	// focusedWindow is the closable window the user touched most recently.
	// Escape closes that one, matching how the Odin version unstacked windows.
	focusedWindow string

	// layoutBuilt tracks whether the default docking layout has been applied.
	// It starts true when a saved layout exists so that a user's arrangement is
	// not overwritten on startup.
	layoutBuilt bool

	// Settings window buffers.
	interfaceScale float32
	databases      []databaseEntry

	// Everything read from the configured folders.
	library library

	// The two database windows, and the rows their search boxes match.
	flagSearch    string
	flagRows      filteredRows
	textureSearch string
	textureRows   filteredRows

	// flag is the coat of arms open in the editor, a copy of the one in the
	// database so that editing it leaves the database alone. selectedLayer is
	// the index of the layer being edited, or noLayer for the coat of arms
	// itself.
	flag          *pdx.Flag
	selectedLayer int

	// history is the undo stack of the open flag, and modified says whether
	// it has changes that have not been saved.
	history  history
	modified bool

	// pendingFlag is a flag waiting to be opened until the user has decided
	// what happens to the unsaved changes of the one open now.
	pendingFlag *pdx.Flag

	// popup is a modal to open at the top of the next frame. Dear ImGui ties a
	// popup to the id stack it was opened from, so opening one from inside a
	// menu would leave it unreachable from the top level where it is drawn.
	popup string

	// picker chooses a texture or a coat of arms for the editor.
	picker picker

	// colorSearch is the search field of the named colour picker.
	colorSearch string

	// status is the message shown in the status bar.
	status string
}

func newState(settings *config.Settings) state {
	current := state{
		showLayers:    true,
		showSelected:  true,
		selectedLayer: noLayer,
		status:        "Ready",
	}

	current.loadFrom(settings)

	return current
}

// loadFrom resets the settings window buffers to the stored configuration.
func (s *state) loadFrom(settings *config.Settings) {
	s.interfaceScale = settings.InterfaceScale

	s.databases = make([]databaseEntry, 0, len(settings.Databases))
	for _, database := range settings.Databases {
		s.databases = append(s.databases, databaseEntry{Name: database.Name, Path: database.Path})
	}
}

// selectedLayerValue returns the layer being looked at.
func (s *state) selectedLayerValue() (pdx.Layer, bool) {
	if s.flag == nil || s.selectedLayer < 0 || s.selectedLayer >= len(s.flag.Layers) {
		return nil, false
	}

	return s.flag.Layers[s.selectedLayer], true
}
