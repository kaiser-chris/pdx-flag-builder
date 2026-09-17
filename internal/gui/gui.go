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
	"image/color"

	"github.com/AllenDang/cimgui-go/backend/raylibbackend"
	"github.com/AllenDang/cimgui-go/imgui"
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
	// An empty path disables persistence.
	LayoutFile string

	// Icon is shown in the title bar and the task bar. Nil keeps the default.
	Icon image.Image

	// Background is cleared at the start of every frame. It sits behind both
	// the raylib drawing and the interface.
	Background color.RGBA
}

// Frame is the per frame work the application wants done, split by when it has
// to happen relative to the interface.
type Frame struct {
	// Offscreen runs first, with raylib drawing already begun. Render targets
	// filled here can be sampled by the interface in the same frame, which is
	// how the flag preview gets its texture.
	Offscreen func()

	// UI builds the interface for this frame.
	UI func()
}

// Window is the application window and the Dear ImGui context that draws into it.
type Window struct {
	backend *raylibbackend.RaylibBackend
}

// NewWindow creates the window and the Dear ImGui context. It must be called
// from the main goroutine, which the caller is expected to have locked to its
// operating system thread.
func NewWindow(cfg Config) *Window {
	back := raylibbackend.NewRaylibBackend()
	back.SetConfigFlags(
		raylibbackend.RaylibBackendFlagsResizable,
		raylibbackend.RaylibBackendFlagsVsyncHint,
		raylibbackend.RaylibBackendFlagsMSAA4X,
	)

	// CreateWindow also creates the Dear ImGui context, so anything touching
	// imgui state has to come after it.
	back.CreateWindow(cfg.Title, cfg.Width, cfg.Height)

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

	back.SetBgColor(vec4FromColor(cfg.Background))

	io := imgui.CurrentIO()
	io.SetConfigFlags(io.ConfigFlags() |
		imgui.ConfigFlagsDockingEnable |
		imgui.ConfigFlagsNavEnableKeyboard)
	io.SetIniFilename(cfg.LayoutFile)

	configureFonts()
	ApplyTheme()

	return &Window{backend: back}
}

// Backend exposes the raylib backend for the few places that need it, such as
// drawing a render texture as an image inside a panel.
func (w *Window) Backend() *raylibbackend.RaylibBackend {
	return w.backend
}

// SetBackground changes the colour cleared at the start of every frame.
func (w *Window) SetBackground(c color.RGBA) {
	w.backend.SetBgColor(vec4FromColor(c))
}

// OnShutdown registers work to run after the loop ends while the OpenGL context
// is still alive, which is the only point where GPU resources can be released.
func (w *Window) OnShutdown(fn func()) {
	w.backend.SetBeforeDestroyContextHook(fn)
}

// RequestClose ends the frame loop after the current frame.
func (w *Window) RequestClose() {
	w.backend.SetShouldClose(true)
}

// Run drives the frame loop until the window is closed. It returns once the
// window and the Dear ImGui context have been torn down.
func (w *Window) Run(frame Frame) {
	if frame.Offscreen != nil {
		w.backend.SetBeforeImGuiRenderHook(frame.Offscreen)
	}

	ui := frame.UI
	if ui == nil {
		ui = func() {}
	}

	w.backend.Run(ui)
	w.backend.Dispose()
}

func vec4FromColor(c color.RGBA) imgui.Vec4 {
	return imgui.Vec4{
		X: float32(c.R) / 255,
		Y: float32(c.G) / 255,
		Z: float32(c.B) / 255,
		W: float32(c.A) / 255,
	}
}
