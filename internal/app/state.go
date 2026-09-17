package app

import "github.com/kaiser-chris/pdx-flag-builder-go/internal/config"

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

	popupAbout = "About"
)

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
	backgroundColor [4]float32
	databases       []databaseEntry

	// status is the message shown in the status bar.
	status string
}

func newState(settings *config.Settings) state {
	current := state{
		showLayers:   true,
		showSelected: true,
		status:       "Ready",
	}

	current.loadFrom(settings)

	return current
}

// loadFrom resets the settings window buffers to the stored configuration.
func (s *state) loadFrom(settings *config.Settings) {
	s.backgroundColor = [4]float32{
		float32(settings.BackgroundColor.R) / 255,
		float32(settings.BackgroundColor.G) / 255,
		float32(settings.BackgroundColor.B) / 255,
		float32(settings.BackgroundColor.A) / 255,
	}

	s.databases = make([]databaseEntry, 0, len(settings.Databases))
	for _, database := range settings.Databases {
		s.databases = append(s.databases, databaseEntry{Name: database.Name, Path: database.Path})
	}
}
