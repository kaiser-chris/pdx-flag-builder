package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
)

// The Preview menu and its one game.
const (
	menuPreview    = "Preview"
	labelVictoria3 = "Victoria 3"
)

// labelRank names the choice of which rank's border the icons are drawn in,
// and labelFancy the waving cloth the game hangs a flag on.
const (
	labelRank          = "Rank Border"
	labelFancy         = "Fancy"
	labelWhatIsMissing = "What is missing"
)

// victoria3Window shows the open flag the way Victoria 3's interface shows a
// country's flag: at each of the sizes the game draws it at, under the shading
// it multiplies over every flag, and in the border of a rank.
//
// It is a window of its own rather than part of the preview panel, because it
// answers a different question: the panel shows the flag as the file describes
// it, this shows what it will look like in the game.
func (a *App) victoria3Window() {
	if !a.state.showVictoria3 {
		return
	}

	imgui.SetNextWindowSizeV(gui.ScaledVec2(360, 420), imgui.CondFirstUseEver)

	if imgui.BeginV(windowVictoria3, &a.state.showVictoria3, 0) {
		a.trackFocus(windowVictoria3)
		a.victoria3Body()
	}

	imgui.End()
}

func (a *App) victoria3Body() {
	if a.state.flag == nil {
		emptyState("No flag open")

		return
	}

	a.missingGameFiles()
	a.rankField()

	imgui.Spacing()

	// The game draws these at a size in pixels, so the previews follow the
	// interface scale rather than the space the window happens to have.
	scale := a.window.Scale()

	// The fancy flag is the widest of them, and the rest are centred under it,
	// so that they read as one flag at several sizes rather than a staircase.
	widest := max(float32(render.IconSizes[0].BorderWidth), render.FancyImageWidth) * scale

	if !imgui.BeginTableV("sizes", 2, imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("name", imgui.TableColumnFlagsWidthFixed, gui.Scaled(70), 0)
	imgui.TableSetupColumnV("flag", imgui.TableColumnFlagsWidthFixed, widest, 0)

	// The fancy flag first, which is how the game's own preview lists them:
	// the largest at the top.
	imgui.TableNextRow()

	imgui.TableNextColumn()
	imgui.SetCursorPosY(imgui.CursorPosY() + max((render.FancyImageHeight*scale-imgui.TextLineHeight())/2, 0))
	imgui.TextUnformatted(labelFancy)
	gui.Record(labelFancy)

	imgui.TableNextColumn()
	imgui.SetCursorPosX(imgui.CursorPosX() + max((widest-render.FancyImageWidth*scale)/2, 0))
	a.window.Backend().DrawImageRenderTexture(a.fancy.Target(),
		imgui.Vec2{X: render.FancyImageWidth * scale, Y: render.FancyImageHeight * scale}, true, false)
	gui.Record(labelFancy + " flag")

	sizeTooltip(labelFancy, render.FancyWidth, render.FancyHeight)

	for index, size := range render.IconSizes {
		width, height := float32(size.BorderWidth)*scale, float32(size.BorderHeight)*scale

		imgui.TableNextRow()

		// The name sits level with the middle of the flag beside it.
		imgui.TableNextColumn()
		imgui.SetCursorPosY(imgui.CursorPosY() + max((height-imgui.TextLineHeight())/2, 0))
		imgui.TextUnformatted(size.Name)
		gui.Record(size.Name)

		imgui.TableNextColumn()
		imgui.SetCursorPosX(imgui.CursorPosX() + max((widest-width)/2, 0))
		a.window.Backend().DrawImageRenderTexture(a.icons.Target(index), imgui.Vec2{X: width, Y: height}, true, false)
		gui.Record(size.Name + " flag")

		sizeTooltip(size.Name, size.Width, size.Height)
	}
}

// sizeTooltip names the size of the flag the pointer is over.
func sizeTooltip(name string, width, height int32) {
	if imgui.IsItemHovered() {
		imgui.SetTooltip(sizeLabel(name, width, height))
	}
}

// sizeLabel names a size the way the game's own files give it: what it is
// called, and how many pixels of a flag it draws.
func sizeLabel(name string, width, height int32) string {
	return fmt.Sprintf("%s: %dx%d", name, width, height)
}

// missingGameFiles says which of the game's own files the preview needs and
// has not found.
//
// The shading, the borders and the cloth belong to Victoria 3 rather than to
// any coat of arms, so they are read from the configured folders. A set of
// folders with only mods in it has none of them, and what the preview can
// show then is the flag itself.
func (a *App) missingGameFiles() {
	var missing []string

	for _, file := range render.GameFiles {
		if _, ok := a.state.library.set.GameArt(file.Name); !ok {
			missing = append(missing, file.Path)
		}
	}

	if problem := a.fancy.Problem(); problem != nil {
		dimmedWrapped(fmt.Sprintf("The cloth could not be read: %v", problem))
		imgui.Spacing()
	}

	if len(missing) == 0 {
		return
	}

	verb := "is"
	if len(missing) > 1 {
		verb = "are"
	}

	dimmedWrapped(fmt.Sprintf("%s of the game's own %s not in any configured folder, so the flag is shown "+
		"without what the game draws it with. Add the game's folder in the settings to see the rest.",
		plural(len(missing), "file", "files"), verb))

	open := imgui.TreeNodeExStrV(labelWhatIsMissing, imgui.TreeNodeFlagsSpanAvailWidth)
	gui.Record(labelWhatIsMissing)

	if open {
		for _, path := range missing {
			imgui.TextDisabled(path)
			gui.Record(path)
		}

		imgui.TreePop()
	}

	imgui.Spacing()
}

// rankField chooses which rank's border the icons are drawn in. The game picks
// it from the country's rank; here it is the user's to try out, one button per
// rank with the chosen one pressed in.
func (a *App) rankField() {
	ranks := a.icons.Ranks()
	if ranks <= 1 {
		return
	}

	if a.state.previewRank < 1 {
		a.state.previewRank = 1
	}

	gui.Label(labelRank)

	for rank := 1; rank <= ranks; rank++ {
		if rank > 1 {
			imgui.SameLine()
		}

		if gui.ToggleButton(fmt.Sprintf("%d", rank), rank == a.state.previewRank) {
			a.state.previewRank = rank
		}
	}
}
