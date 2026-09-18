// Package gui bootstraps the Dear ImGui user interface on top of raylib.
//
// raylib owns the window, the OpenGL context and the input queue; Dear ImGui
// draws through it using the raylib backend that ships with cimgui-go. Two
// windowing stacks cannot share one process, so this arrangement is what lets
// the flag renderer keep using raylib and its shaders while the interface is
// built from a full featured widget toolkit instead of a hand rolled one.
//
// Everything in this package runs on the goroutine that created the window:
// raylib and Dear ImGui are both single threaded.
package gui

import (
	"image"

	"github.com/AllenDang/cimgui-go/backend/raylibbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Config describes the application window.
type Config struct {
	Title  string
	Width  int
	Height int

	// MinWidth and MinHeight keep the window large enough for the flag preview
	// and the sidebar to be usable. Zero leaves the size unconstrained.
	MinWidth  int
	MinHeight int

	// LayoutFile is where Dear ImGui persists the docking layout between runs.
	LayoutFile string

	// Icon is shown in the title bar and the task bar. Nil keeps the default.
	Icon image.Image

	// Hidden creates the window without showing it and lets frames run as fast
	// as they can. It is how the interface tests drive the real application
	// without a window appearing on the desktop.
	Hidden bool
}

// Frame is the per frame work the application wants done, split by when it has
// to happen relative to the interface.
type Frame struct {
	// Offscreen runs first, with raylib drawing already begun. Render targets
	// filled here can be sampled by the interface in the same frame, which is
	// how the flag preview gets its texture.
	Offscreen func()

	// Input runs after the platform input has been handed to Dear ImGui and
	// before the interface is built, so whatever it feeds in takes precedence.
	// The interface tests use it to click and type.
	Input func()

	// UI builds the interface for this frame.
	UI func()
}

// Window is the application window and the Dear ImGui context that draws into it.
type Window struct {
	backend  *raylibbackend.RaylibBackend
	shutdown func()
	closing  bool

	// The interface scale: the one in effect, the one asked for, and the
	// unscaled style both are worked out from.
	scale        float32
	pendingScale float32
	baseStyle    *imgui.Style

	// scaleSettled is set once the first frame has picked its scale. Every
	// later change is one made while the application runs.
	scaleSettled bool
}

// NewWindow creates the window and the Dear ImGui context. It must be called
// from the main goroutine, which the caller is expected to have locked to its
// operating system thread.
func NewWindow(cfg Config) *Window {
	back := raylibbackend.NewRaylibBackend()

	flags := []raylibbackend.RaylibBackendFlags{
		raylibbackend.RaylibBackendFlagsResizable,
		raylibbackend.RaylibBackendFlagsMSAA4X,
	}
	if cfg.Hidden {
		flags = append(flags, raylibbackend.RaylibBackendFlagsHidden)
	} else {
		flags = append(flags, raylibbackend.RaylibBackendFlagsVsyncHint)
	}

	back.SetConfigFlags(flags...)

	// CreateWindow also creates the Dear ImGui context, so anything touching
	// imgui state has to come after it.
	back.CreateWindow(cfg.Title, cfg.Width, cfg.Height)

	if cfg.Hidden {
		// A test waits on frames, not on the display.
		rl.SetTargetFPS(0)
	}

	// Escape closes the focused panel rather than the application; raylib would
	// otherwise treat it as a quit request.
	back.SetExitKey(0)

	if cfg.MinWidth > 0 && cfg.MinHeight > 0 {
		// A zero maximum means "no upper bound" to GLFW.
		back.SetWindowSizeLimits(cfg.MinWidth, cfg.MinHeight, 0, 0)
	}

	if cfg.Icon != nil {
		back.SetIcons(cfg.Icon)
	}

	io := imgui.CurrentIO()
	io.SetConfigFlags(io.ConfigFlags() |
		imgui.ConfigFlagsDockingEnable |
		imgui.ConfigFlagsNavEnableKeyboard)
	io.SetIniFilename(cfg.LayoutFile)

	configureFonts()
	ApplyTheme()

	window := &Window{backend: back}
	window.captureBaseStyle()

	return window
}

// Backend exposes the raylib backend for the few places that need it, such as
// drawing a render texture as an image inside a panel.
func (w *Window) Backend() *raylibbackend.RaylibBackend {
	return w.backend
}

// OnShutdown registers work to run when the window closes, while the OpenGL
// context is still alive, which is the only point where GPU resources can be
// released.
func (w *Window) OnShutdown(fn func()) {
	w.shutdown = fn
}

// RequestClose ends the frame loop after the current frame.
func (w *Window) RequestClose() {
	w.closing = true
}

// ShouldClose reports whether the window has been asked to close, by the user
// or by the application.
func (w *Window) ShouldClose() bool {
	return w.closing || rl.WindowShouldClose()
}

// Run drives the frame loop until the window is closed, then closes it.
func (w *Window) Run(frame Frame) {
	for !w.ShouldClose() {
		w.Step(frame)
	}

	w.Close()
}

// Step runs exactly one frame.
//
// The application loop is nothing more than Step until the window should
// close. Having it separate is what lets a test advance the application one
// frame at a time and look at the result in between.
func (w *Window) Step(frame Frame) {
	rl.BeginDrawing()
	defer rl.EndDrawing()

	// Whatever no panel covers shows the same colour as the panels themselves.
	rl.ClearBackground(clearColor())

	if frame.Offscreen != nil {
		frame.Offscreen()

		// Flush the offscreen drawing so it is not interleaved with the
		// interface's own batch.
		rl.DrawRenderBatchActive()
	}

	w.applyPendingScale()
	w.backend.NewFrame()

	if frame.Input != nil {
		frame.Input()
	}

	imgui.NewFrame()

	if frame.UI != nil {
		frame.UI()
	}

	imgui.Render()

	if drawData := imgui.CurrentDrawData(); drawData != nil {
		// See textures.go: the backend's own texture handling breaks as
		// soon as Dear ImGui has more than one texture.
		restore := w.serviceTextures(drawData)
		w.backend.Render(*drawData)
		restore()
	}
}

// Close releases the application's GPU resources and closes the window.
func (w *Window) Close() {
	if w.shutdown != nil {
		w.shutdown()
		w.shutdown = nil
	}

	if w.baseStyle != nil {
		w.baseStyle.Destroy()
		w.baseStyle = nil
	}

	// Dear ImGui writes the docking layout out when its context goes away, so
	// that has to happen before the process ends.
	imgui.DestroyContext()
	rl.CloseWindow()
}
