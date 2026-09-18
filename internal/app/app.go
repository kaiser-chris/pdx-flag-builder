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
	"time"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/assets"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
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
	store    config.Store
	settings *config.Settings
	state    state

	// input feeds extra input into Dear ImGui each frame. It is only ever set
	// by the interface tests.
	input func()

	// Drawing the flag: the recolouring shader, the artwork it needs, the
	// painter that puts them together and the target it paints into.
	shader   render.Recolor
	textures *render.Textures
	painter  *render.Painter
	preview  *render.Preview

	// thumbnails are the small previews in the lists, drawn into an atlas that
	// is handed to Dear ImGui once, the first time a list shows one.
	thumbnails     *render.Thumbnails
	thumbnailAtlas *imgui.TextureRef

	// dockWindowClass is handed to the dock space every frame. cimgui-go
	// dereferences that argument even when it is nil, so one default instance
	// is allocated for the lifetime of the application instead.
	dockWindowClass *imgui.WindowClass
}

// Options changes how the application starts. The zero value is what a user
// gets; the interface tests fill it in.
type Options struct {
	// ConfigDir holds the settings and the saved layout. Empty means the user's
	// configuration directory.
	ConfigDir string

	// Hidden runs the application without showing its window.
	Hidden bool
}

// Run starts the application and returns when its window has been closed.
func Run() error {
	application, err := New(Options{})
	if err != nil {
		return err
	}

	application.window.Run(application.frameSpec())

	return nil
}

// New creates the window and everything the application needs, and starts
// reading the configured folders. It must be called from the main goroutine.
func New(options Options) (*App, error) {
	store, err := openStore(options.ConfigDir)
	if err != nil {
		return nil, err
	}

	settings, err := store.Load()
	if err != nil {
		// A settings file that cannot be read should not keep the application
		// from starting: it comes up with defaults and says so in the status
		// bar, and the user can repair it from the settings window.
		warn(err)
	}

	application := &App{
		store:    store,
		settings: settings,
		state:    newState(settings),
	}
	if err != nil {
		application.state.status = "Settings could not be loaded, using defaults"
	}

	application.state.layoutBuilt = fileExists(store.LayoutPath())

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
		LayoutFile: store.LayoutPath(),
		Icon:       icon,
		Background: application.backgroundColor(),
		Hidden:     options.Hidden,
	})

	application.window.SizeForScale(defaultWindowWidth, defaultWindowHeight, application.interfaceScale())

	// Everything below needs the OpenGL context the window just created.
	shader, err := render.LoadRecolor()
	if err != nil {
		application.window.Close()

		// Without the shader nothing can be recoloured, which leaves an editor
		// that cannot show what it is editing.
		return nil, fmt.Errorf("load the recolouring shader: %w", err)
	}
	application.shader = shader

	application.textures = render.NewTextures(application.texturePath)
	application.painter = render.NewPainter(shader, application.textures)
	application.painter.SetSubFlagLookup(application.subFlag)

	application.preview = render.NewPreview(render.FlagWidth, render.FlagHeight, application.painter)
	application.thumbnails = render.NewThumbnails(shader, application.texturePath)
	application.thumbnails.SetSubFlagLookup(application.subFlag)
	application.dockWindowClass = imgui.NewWindowClass()

	application.window.OnShutdown(func() {
		application.preview.Unload()
		application.thumbnails.Unload()
		application.textures.Unload()
		application.shader.Unload()
		application.dockWindowClass.Destroy()
	})

	// Reading the configured folders starts right away and finishes in the
	// background, so the window is up while the game files are still being read.
	application.state.library.reload(settings.Databases)
	application.state.status = "Reading the configured folders"

	return application, nil
}

// Step runs one frame. The interface tests advance the application with it.
func (a *App) Step() {
	a.window.Step(a.frameSpec())
}

// Close shuts the application down and closes its window.
func (a *App) Close() {
	a.window.Close()
}

// SetInput installs a function that feeds input into Dear ImGui every frame,
// after the platform input and before the interface is built.
func (a *App) SetInput(input func()) {
	a.input = input
}

func (a *App) frameSpec() gui.Frame {
	return gui.Frame{
		Offscreen: a.drawOffscreen,
		Input:     a.input,
		UI:        a.frame,
	}
}

func openStore(dir string) (config.Store, error) {
	if dir == "" {
		return config.UserStore()
	}

	return config.NewStore(dir)
}

// drawOffscreen does the raylib drawing for a frame: it hands the GPU whatever
// artwork has finished loading, then paints the flag into the preview target
// and any newly wanted thumbnails into theirs, both of which the interface
// samples further down the same frame.
func (a *App) drawOffscreen() {
	a.textures.Upload()
	a.preview.Draw(a.state.flag)
	a.thumbnails.Draw()
}

// texturePath turns the file name a coat of arms refers to into a path on disk.
func (a *App) texturePath(name string) (string, bool) {
	found, ok := a.state.library.set.Texture(name)
	if !ok {
		return "", false
	}

	return found.Path, true
}

// subFlag finds the coat of arms a sub flag layer refers to.
func (a *App) subFlag(name string) (pdx.Flag, bool) {
	found, ok := a.state.library.set.Flag(name)
	if !ok {
		return pdx.Flag{}, false
	}

	return *found, true
}

// openFlag puts a copy of a coat of arms into the editor, replacing whatever
// was open without asking. requestOpen is the version that asks.
func (a *App) openFlag(flag pdx.Flag) {
	opened := flag.Clone()

	a.state.flag = &opened
	a.state.selectedLayer = noLayer
	a.state.showLayers = true
	a.state.modified = false
	a.state.history.reset(opened)

	if flag.Origin.Database != "" {
		a.setStatus("Opened %s from %s", flag.Name, flag.Origin.Database)
	} else {
		a.setStatus("Started %s", flag.Name)
	}
}

// reportLoad summarises a finished read of the configured folders.
func (a *App) reportLoad() {
	library := &a.state.library

	// A flag held open in the editor is a copy, so it survives the reload; what
	// is thrown away is the search, which now points at a different list, and
	// the artwork, which may now come from a different folder.
	a.state.flagRows.invalidate()
	a.state.textureRows.invalidate()

	a.painter.SetPalette(library.palette)
	a.textures.Forget()
	a.thumbnails.SetPalette(library.palette)
	a.thumbnails.Forget()

	if len(library.set) == 0 {
		a.setStatus("No folders configured")

		return
	}

	message := fmt.Sprintf("Read %s and %s in %s",
		plural(len(library.flags), "flag", "flags"),
		plural(len(library.textures), "texture", "textures"),
		library.took.Round(time.Millisecond))

	if count := len(library.problems); count > 0 {
		message += ", " + plural(count, "problem", "problems")
	}

	a.setStatus("%s", message)
}

// frame builds one frame of the interface.
//
// The order matters: the menu bar and the status bar claim their strip of the
// viewport first, so the dock space that follows covers exactly what is left.
func (a *App) frame() {
	// A finished folder read is picked up here, which is the only place the
	// data it produced crosses onto the interface goroutine.
	if a.state.library.poll(a.settings.Databases) {
		a.reportLoad()
	}

	// Checked every frame so that "Automatic" follows the window from one
	// monitor to another. Nothing happens unless the scale actually changes.
	a.window.SetScale(a.interfaceScale())

	a.openRequestedPopup()

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
	a.pickerPopup()
	a.discardPopup()

	a.handleShortcuts()

	// Last, once every widget has had its say about this frame.
	a.settleHistory()
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
