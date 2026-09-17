package app

import (
	"fmt"
	"strconv"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// swatchSize is the edge length of the little colour squares.
const swatchSize = 18

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

func (a *App) layersBody() {
	flag := a.state.flag

	if flag == nil {
		emptyState(
			"No flag open.",
			"Pick one in the flag database",
			"or start a new one from the File menu.",
		)

		return
	}

	gui.PushStrongFont()
	imgui.TextUnformatted(flag.Name)
	gui.PopFont()

	imgui.TextDisabled(fmt.Sprintf("%s / %s", flag.Origin.Database, flag.Origin.File))

	imgui.Spacing()

	if flag.Pattern != "" {
		imgui.TextUnformatted("Pattern")
		imgui.SameLine()
		imgui.TextDisabled(flag.Pattern)
	} else {
		imgui.TextDisabled("No pattern")
	}

	a.colorRow(flag.Colors, flag.Colors)

	imgui.Separator()

	if len(flag.Layers) == 0 {
		imgui.TextDisabled("This flag has no layers.")

		return
	}

	for index, layer := range flag.Layers {
		imgui.PushIDInt(int32(index))

		selected := index == a.state.selectedLayer
		if imgui.SelectableBoolV(describeLayer(layer), selected, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{}) {
			a.state.selectedLayer = index
			a.state.showSelected = true
		}

		imgui.PopID()
	}
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

func (a *App) selectedLayerBody() {
	layer, ok := a.state.selectedLayerValue()
	if !ok {
		emptyState("Select a layer to see it.")

		return
	}

	gui.PushStrongFont()
	imgui.TextUnformatted(layerKind(layer))
	gui.PopFont()

	imgui.Spacing()

	switch typed := layer.(type) {
	case pdx.ColoredEmblem:
		a.textureLine(typed.Texture)

		if typed.Mask > 0 {
			imgui.TextUnformatted("Mask")
			imgui.SameLine()
			imgui.TextDisabled(strconv.Itoa(typed.Mask))
		}

		// An emblem slot may point back at a slot of the flag, so the flag's
		// colours are what its references are resolved against.
		a.colorRow(typed.Colors, a.state.flag.Colors)
		a.instanceTable(pdx.Placements(typed.Instances), len(typed.Instances) == 0)

	case pdx.TexturedEmblem:
		a.textureLine(typed.Texture)
		a.instanceTable(pdx.Placements(typed.Instances), len(typed.Instances) == 0)

	case pdx.SubFlag:
		imgui.TextUnformatted("Parent")
		imgui.SameLine()
		imgui.TextDisabled(typed.Parent)

		if _, found := a.state.library.set.Flag(typed.Parent); !found {
			imgui.TextDisabled("This coat of arms was not found in the configured folders.")
		}

		a.subInstanceTable(pdx.SubPlacements(typed.Instances), len(typed.Instances) == 0)
	}
}

// textureLine names the texture a layer draws and says whether it was found.
func (a *App) textureLine(name string) {
	imgui.TextUnformatted("Texture")
	imgui.SameLine()

	if name == "" {
		imgui.TextDisabled("none")

		return
	}

	imgui.TextDisabled(name)

	if _, found := a.state.library.set.Texture(name); !found {
		imgui.TextDisabled("This texture was not found in the configured folders.")
	}
}

// colorRow draws the colour slots as swatches, resolved against the palette
// read from the configured folders.
func (a *App) colorRow(colors pdx.Colors, slots pdx.Colors) {
	if len(colors) == 0 {
		return
	}

	imgui.Spacing()

	for index, color := range colors {
		if index > 0 {
			imgui.SameLine()
		}

		resolved, known := color.Resolve(a.state.library.palette, slots)

		imgui.PushIDStr(color.Slot)
		imgui.ColorButtonV(
			"##swatch",
			imgui.Vec4{
				X: float32(resolved.R) / 255,
				Y: float32(resolved.G) / 255,
				Z: float32(resolved.B) / 255,
				W: 1,
			},
			imgui.ColorEditFlagsNoTooltip|imgui.ColorEditFlagsNoDragDrop,
			imgui.Vec2{X: swatchSize, Y: swatchSize},
		)
		imgui.PopID()

		if imgui.IsItemHovered() && imgui.BeginTooltip() {
			imgui.TextUnformatted(color.Slot + " = " + color.Value.Describe())

			if !known {
				imgui.TextDisabled("This colour could not be resolved.")
			}

			imgui.EndTooltip()
		}
	}
}

func (a *App) instanceTable(instances []pdx.Instance, implied bool) {
	imgui.Spacing()
	sectionHeader(instanceHeading(len(instances), implied))

	if !imgui.BeginTableV("instances", 4, imgui.TableFlagsRowBg|imgui.TableFlagsBordersInnerV, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("#", imgui.TableColumnFlagsWidthFixed, 24, 0)
	imgui.TableSetupColumnV("Position", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("Scale", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("Rotation", imgui.TableColumnFlagsWidthFixed, 70, 0)
	imgui.TableHeadersRow()

	for index, instance := range instances {
		imgui.TableNextRow()

		imgui.TableSetColumnIndex(0)
		imgui.TextDisabled(strconv.Itoa(index + 1))

		imgui.TableSetColumnIndex(1)
		imgui.TextUnformatted(describeVector(instance.Position))

		imgui.TableSetColumnIndex(2)
		imgui.TextUnformatted(describeVector(instance.Scale))

		imgui.TableSetColumnIndex(3)
		imgui.TextUnformatted(fmt.Sprintf("%g", instance.Rotation))
	}
}

func (a *App) subInstanceTable(instances []pdx.SubInstance, implied bool) {
	imgui.Spacing()
	sectionHeader(instanceHeading(len(instances), implied))

	if !imgui.BeginTableV("sub-instances", 3, imgui.TableFlagsRowBg|imgui.TableFlagsBordersInnerV, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("#", imgui.TableColumnFlagsWidthFixed, 24, 0)
	imgui.TableSetupColumnV("Offset", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableSetupColumnV("Scale", imgui.TableColumnFlagsWidthStretch, 0, 0)
	imgui.TableHeadersRow()

	for index, instance := range instances {
		imgui.TableNextRow()

		imgui.TableSetColumnIndex(0)
		imgui.TextDisabled(strconv.Itoa(index + 1))

		imgui.TableSetColumnIndex(1)
		imgui.TextUnformatted(describeVector(instance.Offset))

		imgui.TableSetColumnIndex(2)
		imgui.TextUnformatted(describeVector(instance.Scale))
	}
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
	case pdx.ColoredEmblem:
		return fmt.Sprintf("%s  %s", layerKind(layer), or(typed.Texture, "no texture"))

	case pdx.TexturedEmblem:
		return fmt.Sprintf("%s  %s", layerKind(layer), or(typed.Texture, "no texture"))

	case pdx.SubFlag:
		return fmt.Sprintf("%s  %s", layerKind(layer), or(typed.Parent, "no parent"))
	}

	return layerKind(layer)
}

func layerKind(layer pdx.Layer) string {
	switch layer.(type) {
	case pdx.ColoredEmblem:
		return "Colored Emblem"
	case pdx.TexturedEmblem:
		return "Textured Emblem"
	case pdx.SubFlag:
		return "Sub Flag"
	}

	return "Layer"
}

func describeVector(vector pdx.Vec2) string {
	return fmt.Sprintf("%g, %g", vector.X, vector.Y)
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
