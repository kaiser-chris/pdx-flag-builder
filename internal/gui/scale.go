package gui

import (
	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Limits of the interface scale. Below the one, text gets unreadable; above
// the other, the panels no longer fit on any screen.
const (
	MinScale = 0.75
	MaxScale = 3
)

// MonitorScale is the scale the operating system asks for on the monitor the
// window is on: 1.5 for a display set to 150%, for example.
//
// GLFW makes the application aware of the display scale on Windows, so the
// window is drawn in real pixels rather than stretched by the system. That is
// what keeps it sharp, and also why the interface has to scale itself.
func MonitorScale() float32 {
	scale := rl.GetWindowScaleDPI().X
	if scale <= 0 {
		return 1
	}

	return clampScale(scale)
}

func clampScale(scale float32) float32 {
	return min(max(scale, MinScale), MaxScale)
}

// SetScale asks for the interface to be drawn at a scale, where one is the
// size it was designed at. The change takes effect at the start of the next
// frame, because Dear ImGui's style cannot change halfway through one.
func (w *Window) SetScale(scale float32) {
	w.pendingScale = clampScale(scale)
}

// Scale is the scale the interface is drawn at.
func (w *Window) Scale() float32 {
	return w.scale
}

// captureBaseStyle keeps the themed style at its designed size.
//
// Scaling works from this copy every time. Dear ImGui's ScaleAllSizes rounds
// every size down as it scales, so scaling the live style again and again would
// make it drift: a padding of 10 goes to 12 at 125% and comes back as 9.
func (w *Window) captureBaseStyle() {
	w.baseStyle = imgui.NewStyle()
	*w.baseStyle.CData = *imgui.CurrentStyle().CData
	w.scale = 1
}

func (w *Window) applyPendingScale() {
	// The scale a session starts at is the one its saved layout was left at,
	// so windows are only resized for a change made while it runs.
	starting := !w.scaleSettled
	w.scaleSettled = true

	if w.pendingScale == 0 || w.pendingScale == w.scale {
		return
	}

	if !starting {
		rescaleWindows(w.pendingScale / w.scale)
	}

	style := imgui.CurrentStyle()

	*style.CData = *w.baseStyle.CData
	style.ScaleAllSizes(w.pendingScale)

	// Dear ImGui 1.92 rasterises glyphs at the size they are drawn at, so
	// scaling the font keeps the text sharp rather than stretching it.
	style.SetFontScaleMain(w.pendingScale)

	w.scale = w.pendingScale
}

// Scaled converts a size in the units the interface was designed in to the
// pixels it takes at the current scale. Style sizes and text scale by
// themselves; this is for the sizes the application picks, such as the size a
// window first opens at.
func Scaled(size float32) float32 {
	return size * imgui.CurrentStyle().FontScaleMain()
}

// ScaledVec2 is Scaled for both sides of a size.
func ScaledVec2(width, height float32) imgui.Vec2 {
	return imgui.Vec2{X: Scaled(width), Y: Scaled(height)}
}

// rescaleWindows resizes the free floating windows by a ratio, so that a
// window which fitted its contents before the scale changed still does after.
//
// Docked windows take their size from the dock space, which fills the
// application window whatever the scale, and every kind of window that sizes
// itself (menus, popups, tooltips, child windows) is left alone. A window that
// no longer fits on screen is shrunk and moved back onto it.
func rescaleWindows(ratio float32) {
	const selfSizing = imgui.WindowFlagsChildWindow | imgui.WindowFlagsPopup | imgui.WindowFlagsTooltip |
		imgui.WindowFlagsAlwaysAutoResize | imgui.WindowFlagsNoResize

	viewport := imgui.MainViewport()
	area, areaSize := viewport.WorkPos(), viewport.WorkSize()

	for _, window := range contextWindows() {
		if window.Flags()&selfSizing != 0 || window.DockIsActive() {
			continue
		}

		full := window.SizeFull()
		size := imgui.Vec2{
			X: min(full.X*ratio, areaSize.X),
			Y: min(full.Y*ratio, areaSize.Y),
		}

		position := window.Pos()
		position.X = max(min(position.X, area.X+areaSize.X-size.X), area.X)
		position.Y = max(min(position.Y, area.Y+areaSize.Y-size.Y), area.Y)

		imgui.InternalSetWindowSizeWindowPtrV(window, size, 0)
		imgui.InternalSetWindowPosWindowPtrV(window, position, 0)
	}
}

// largestShareOfMonitor keeps a window that grows for a scaled interface from
// covering the whole screen.
const largestShareOfMonitor = 0.9

// SizeForScale grows the window for a scaled interface. The default size is
// chosen for an unscaled one and is cramped at twice the size, so the window
// grows with the scale, up to most of the monitor, and is centred on it.
func (w *Window) SizeForScale(width, height int, scale float32) {
	if scale <= 1 {
		return
	}

	monitor := rl.GetCurrentMonitor()
	monitorWidth := rl.GetMonitorWidth(monitor)
	monitorHeight := rl.GetMonitorHeight(monitor)

	scaledWidth := min(int(float32(width)*scale), int(float32(monitorWidth)*largestShareOfMonitor))
	scaledHeight := min(int(float32(height)*scale), int(float32(monitorHeight)*largestShareOfMonitor))

	rl.SetWindowSize(scaledWidth, scaledHeight)

	origin := rl.GetMonitorPosition(monitor)
	rl.SetWindowPosition(
		int(origin.X)+(monitorWidth-scaledWidth)/2,
		int(origin.Y)+(monitorHeight-scaledHeight)/2,
	)
}
