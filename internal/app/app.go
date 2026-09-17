// Package app is the application shell: the window, its panels and the state
// they share.
//
// The interface is built with Dear ImGui through internal/gui, and the flag
// itself is drawn by raylib into an offscreen target that the preview panel
// samples. Panels are described from scratch every frame, so the whole
// interface is a function of the state in this package.
package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/assets"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
)

const (
	applicationName = "PDX Flag Builder"

	defaultWindowWidth  = 1440
	defaultWindowHeight = 900

	// The window may not shrink below the flag preview plus a usable sidebar.
	minimumSidebarWidth = 360
	minimumChromeHeight = 150
)

// App owns the window and the state its panels read and write.
type App struct {
	window   *gui.Window
	settings *config.Settings
	preview  *render.Preview
	state    state

	// dockWindowClass is handed to the dock space every frame. cimgui-go
	// dereferences that argument even when it is nil, so one default instance
	// is allocated for the lifetime of the application instead.
	dockWindowClass *imgui.WindowClass
}

// Run starts the application and returns when its window has been closed.
func Run() error {
	settings, err := config.Load()
	if err != nil {
		// A settings file that cannot be read should not keep the application
		// from starting: it comes up with defaults and says so in the status
		// bar, and the user can repair it from the settings window.
		warn(err)
	}

	application := &App{
		settings: settings,
		state:    newState(settings),
	}
	if err != nil {
		application.state.status = "Settings could not be loaded, using defaults"
	}

	layoutPath, err := config.LayoutPath()
	if err != nil {
		warn(err)
	}
	application.state.layoutBuilt = fileExists(layoutPath)

	icon, err := loadIcon()
	if err != nil {
		warn(err)
	}

	application.window = gui.NewWindow(gui.Config{
		Title:      applicationName,
		Width:      defaultWindowWidth,
		Height:     defaultWindowHeight,
		MinWidth:   render.FlagWidth + minimumSidebarWidth,
		MinHeight:  render.FlagHeight + minimumChromeHeight,
		LayoutFile: layoutPath,
		Icon:       icon,
		Background: application.backgroundColor(),
	})

	// The render target needs the OpenGL context the window just created.
	application.preview = render.NewPreview(render.FlagWidth, render.FlagHeight)
	application.dockWindowClass = imgui.NewWindowClass()

	application.window.OnShutdown(func() {
		application.preview.Unload()
		application.dockWindowClass.Destroy()
	})

	application.window.Run(gui.Frame{
		Offscreen: application.preview.Draw,
		UI:        application.frame,
	})

	return nil
}

// frame builds one frame of the interface.
//
// The order matters: the menu bar and the status bar claim their strip of the
// viewport first, so the dock space that follows covers exactly what is left.
func (a *App) frame() {
	a.menuBar()
	a.statusBar()
	a.dockSpace()

	a.previewPanel()
	a.layersPanel()
	a.selectedLayerPanel()

	a.settingsWindow()
	a.flagDatabaseWindow()
	a.textureDatabaseWindow()
	a.aboutPopup()

	a.handleShortcuts()
}

// backgroundColor is the window's clear colour as configured by the user.
func (a *App) backgroundColor() color.RGBA {
	return color.RGBA{
		R: a.settings.BackgroundColor.R,
		G: a.settings.BackgroundColor.G,
		B: a.settings.BackgroundColor.B,
		A: a.settings.BackgroundColor.A,
	}
}

// trackFocus records which closable window the user interacted with last so
// that Escape closes the one on top.
func (a *App) trackFocus(title string) {
	if imgui.IsWindowFocusedV(imgui.FocusedFlagsRootAndChildWindows) {
		a.state.focusedWindow = title
	}
}

// setStatus replaces the message shown in the status bar.
func (a *App) setStatus(format string, args ...any) {
	a.state.status = fmt.Sprintf(format, args...)
}

func loadIcon() (image.Image, error) {
	data, err := assets.Read(assets.Icon)
	if err != nil {
		return nil, err
	}

	icon, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode application icon: %w", err)
	}

	return icon, nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}

	_, err := os.Stat(path)

	return err == nil
}

// warn reports a non fatal problem. The application keeps running.
func warn(err error) {
	fmt.Fprintf(os.Stderr, "pdx-flag-builder: %v\n", err)
}
