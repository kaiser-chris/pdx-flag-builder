package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// previewPanel shows the flag. The image is the raylib render target filled in
// this frame's offscreen pass, scaled to fit the panel without ever being
// enlarged past its native resolution.
func (a *App) previewPanel() {
	// The preview fills its panel edge to edge, so it gets no padding.
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{})
	open := imgui.BeginV(panelPreview, nil, imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoScrollWithMouse)
	imgui.PopStyleVar()

	if open {
		a.drawPreviewImage()

		// Clicking the flag gives the panel focus, and with it the arrow keys.
		focused := imgui.IsWindowFocused()
		if focused {
			a.nudge()
		} else {
			a.state.nudging = false
		}

		if hint := a.nudgeHint(focused); hint != "" {
			previewCaption(hint)
		}
	} else {
		a.state.nudging = false
	}

	imgui.End()
}

// previewCaption writes a line of dimmed text along the bottom of the preview
// panel, on a band of the panel's colour so that it reads over the flag too.
func previewCaption(text string) {
	padding := gui.Scaled(8)
	position := imgui.WindowPos()
	size := imgui.WindowSize()

	textSize := imgui.CalcTextSizeV(text, false, -1)
	top := imgui.Vec2{X: position.X + padding, Y: position.Y + size.Y - textSize.Y - padding*2}

	drawList := imgui.WindowDrawList()
	drawList.AddRectFilledV(top,
		imgui.Vec2{X: top.X + textSize.X + padding*2, Y: top.Y + textSize.Y + padding},
		imgui.ColorU32ColV(imgui.ColWindowBg, 0.85), gui.Scaled(4), 0)
	// One line, clipped by the panel when it is too narrow. cimgui-go cannot
	// pass the null clip rectangle the wrapping variant takes.
	drawList.AddTextVec2(imgui.Vec2{X: top.X + padding, Y: top.Y + padding/2},
		imgui.ColorU32Col(imgui.ColTextDisabled), text)
}

func (a *App) drawPreviewImage() {
	width, height := a.preview.Size()
	available := imgui.ContentRegionAvail()
	if available.X <= 0 || available.Y <= 0 {
		return
	}

	// The flag grows with the interface scale, so that it keeps its size next
	// to everything else on a high resolution display, but no further: past
	// that, enlarging it only blurs it.
	scale := min(available.X/float32(width), available.Y/float32(height), a.window.Scale())

	size := imgui.Vec2{X: float32(width) * scale, Y: float32(height) * scale}

	cursor := imgui.CursorPos()
	imgui.SetCursorPos(imgui.Vec2{
		X: cursor.X + (available.X-size.X)/2,
		Y: cursor.Y + (available.Y-size.Y)/2,
	})

	// OpenGL renders into a texture bottom up, so the image is flipped back.
	a.window.Backend().DrawImageRenderTexture(a.preview.Target(), size, true, false)
}

// layersPanel lists the layers of the flag open in the editor.
func (a *App) layersPanel() {
	if !a.state.showLayers {
		return
	}

	if imgui.BeginV(panelLayers, &a.state.showLayers, 0) {
		a.layersBody()
	}

	imgui.End()
}

// selectedLayerPanel shows the layer picked in the layers panel.
func (a *App) selectedLayerPanel() {
	if !a.state.showSelected {
		return
	}

	if imgui.BeginV(panelSelected, &a.state.showSelected, 0) {
		a.selectedLayerBody()
	}

	imgui.End()
}

// instanceHeading says how many placements a layer has, and marks the case
// where the file left them out and the defaults are standing in.
func instanceHeading(count int, implied bool) string {
	if implied {
		return "Placement (default)"
	}

	if count == 1 {
		return "1 placement"
	}

	return fmt.Sprintf("%d placements", count)
}

func describeLayer(layer pdx.Layer) string {
	switch typed := layer.(type) {
	case *pdx.ColoredEmblem:
		return fmt.Sprintf("%s  %s", layerKind(layer), or(typed.Texture, "no texture"))

	case *pdx.TexturedEmblem:
		return fmt.Sprintf("%s  %s", layerKind(layer), or(typed.Texture, "no texture"))

	case *pdx.SubFlag:
		return fmt.Sprintf("%s  %s", layerKind(layer), or(typed.Parent, "no parent"))
	}

	return layerKind(layer)
}

func layerKind(layer pdx.Layer) string {
	switch layer.(type) {
	case *pdx.ColoredEmblem:
		return "Colored Emblem"
	case *pdx.TexturedEmblem:
		return "Textured Emblem"
	case *pdx.SubFlag:
		return "Sub Flag"
	}

	return "Layer"
}

func or(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

// problemSummary describes what went wrong while reading the folders, naming
// the first few so that a mod author has somewhere to start.
func problemSummary(problems []database.Problem, count int) string {
	const shown = 3

	summary := plural(count, "problem", "problems") + " while reading: "

	for index, problem := range problems {
		if index == shown {
			summary += fmt.Sprintf(" and %d more", count-shown)

			break
		}

		if index > 0 {
			summary += "; "
		}

		summary += problem.String()
	}

	return summary
}

// sectionHeader is a heading in the heavier weight.
func sectionHeader(title string) {
	gui.PushStrongFont()
	imgui.SeparatorText(title)
	gui.PopFont()
}

// emptyState draws dimmed, horizontally centred lines explaining why a panel
// has nothing in it.
func emptyState(lines ...string) {
	available := imgui.ContentRegionAvail()
	if available.Y > 0 {
		lineHeight := imgui.TextLineHeightWithSpacing()
		imgui.Dummy(imgui.Vec2{Y: max((available.Y-lineHeight*float32(len(lines)))/2, 0)})
	}

	for _, line := range lines {
		width := imgui.CalcTextSize(line).X
		imgui.SetCursorPosX(imgui.CursorPosX() + max((imgui.ContentRegionAvail().X-width)/2, 0))
		imgui.TextDisabled(line)
	}
}

// dimmedWrapped draws help text that wraps at the width of its window.
func dimmedWrapped(text string) {
	colors := imgui.CurrentStyle().Colors()

	imgui.PushStyleColorVec4(imgui.ColText, colors[imgui.ColTextDisabled])
	imgui.TextWrapped(text)
	imgui.PopStyleColor()
}

// plural writes a count with its noun, so that the interface does not say
// "1 problems".
func plural(count int, singular, pluralForm string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}

	return fmt.Sprintf("%d %s", count, pluralForm)
}
