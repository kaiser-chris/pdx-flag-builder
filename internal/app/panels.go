package app

import (
	"github.com/AllenDang/cimgui-go/imgui"
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
	}

	imgui.End()
}

func (a *App) drawPreviewImage() {
	width, height := a.preview.Size()
	available := imgui.ContentRegionAvail()
	if available.X <= 0 || available.Y <= 0 {
		return
	}

	scale := min(available.X/float32(width), available.Y/float32(height))
	if scale > 1 {
		// Showing the flag larger than it is rendered only blurs it.
		scale = 1
	}

	size := imgui.Vec2{X: float32(width) * scale, Y: float32(height) * scale}

	cursor := imgui.CursorPos()
	imgui.SetCursorPos(imgui.Vec2{
		X: cursor.X + (available.X-size.X)/2,
		Y: cursor.Y + (available.Y-size.Y)/2,
	})

	// OpenGL renders into a texture bottom up, so the image is flipped back.
	a.window.Backend().DrawImageRenderTexture(a.preview.Target(), size, true, false)
}

// layersPanel lists the layers of the flag being edited.
func (a *App) layersPanel() {
	if !a.state.showLayers {
		return
	}

	if imgui.BeginV(panelLayers, &a.state.showLayers, 0) {
		imgui.BeginDisabledV(true)
		imgui.Button("Add Layer")
		imgui.SameLine()
		imgui.Button("Add Sub Flag")
		imgui.EndDisabled()

		imgui.Separator()

		// TODO: replace with the layer list once the flag model is ported.
		emptyState(
			"No layers yet.",
			"Open a flag from the flag database",
			"or start a new one from the File menu.",
		)
	}

	imgui.End()
}

// selectedLayerPanel edits whichever layer or instance is selected.
func (a *App) selectedLayerPanel() {
	if !a.state.showSelected {
		return
	}

	if imgui.BeginV(panelSelected, &a.state.showSelected, 0) {
		// TODO: replace with the per layer property editor (colours, instances,
		// masks) once the flag model is ported.
		emptyState("Select a layer to edit it.")
	}

	imgui.End()
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
