package app

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
)

// The Preview menu, and what its windows call the parts of a flag.
const (
	menuPreview        = "Preview"
	labelRank          = "Rank Border"
	labelFancy         = "Fancy"
	labelWhatIsMissing = "What is missing"
)

// preview is one game's window: the flag as that game shows it.
type preview struct {
	game *render.Game

	// title is the window's, which is also how the docking layout knows it.
	title string

	icons *render.Icons
	fancy *render.Fancy

	show bool

	// rank is the rank whose border the icons are drawn in, counted from one
	// the way the games' files number them.
	rank int
}

// newPreviews prepares one preview per game. It requires an active OpenGL
// context, so it must be called after the window exists.
func newPreviews(painter *render.Painter, textures *render.Textures, resolve func(name string) (string, bool)) ([]*preview, error) {
	var previews []*preview

	for _, game := range render.Games {
		fancy, err := render.NewFancy(game.Cloth, painter, resolve)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", game.Name, err)
		}

		previews = append(previews, &preview{
			game:  game,
			title: game.Name + " Preview",
			icons: render.NewIcons(game, painter, textures),
			fancy: fancy,
			rank:  1,
		})
	}

	return previews, nil
}

// drawPreviews composes what the open previews show. It has to run while
// raylib drawing is active and before the interface samples the targets.
func (a *App) drawPreviews(seconds float32) {
	for _, preview := range a.previews {
		if !preview.show {
			continue
		}

		preview.icons.Draw(a.state.flag, preview.rank)
		preview.fancy.Draw(a.state.flag, seconds)
	}
}

// forgetPreviews drops what the previews read from the folders, since the
// folders have been read again and may hold something else now.
func (a *App) forgetPreviews() {
	for _, preview := range a.previews {
		preview.fancy.Forget()
	}
}

// unloadPreviews releases everything the previews hold on the graphics card.
func (a *App) unloadPreviews() {
	for _, preview := range a.previews {
		preview.icons.Unload()
		preview.fancy.Unload()
	}
}

// previewWindows draws the previews that are open.
func (a *App) previewWindows() {
	for _, preview := range a.previews {
		a.previewWindow(preview)
	}
}

// previewWindow shows the open flag the way one game shows a country's: as
// the cloth it hangs one on, and at each of the sizes it draws a flat one at,
// under the shading it lays over them and in the frame it draws around them.
//
// It is a window of its own rather than part of the preview panel, because it
// answers a different question: the panel shows the flag as the file
// describes it, this shows what it will look like in the game.
func (a *App) previewWindow(preview *preview) {
	if !preview.show {
		return
	}

	imgui.SetNextWindowSizeV(gui.ScaledVec2(400, 520), imgui.CondFirstUseEver)

	if imgui.BeginV(preview.title, &preview.show, 0) {
		a.trackFocus(preview.title)
		a.previewBody(preview)
	}

	imgui.End()
}

func (a *App) previewBody(preview *preview) {
	if a.state.flag == nil {
		emptyState("No flag open")

		return
	}

	a.missingGameFiles(preview)
	a.rankField(preview)

	imgui.Spacing()

	// The games draw these at a size in pixels, so the previews follow the
	// interface scale rather than the space the window happens to have.
	scale := a.window.Scale()

	// The widest of them sets the column, and the rest are centred under it,
	// so that they read as one flag at several sizes rather than a staircase.
	cloth := preview.game.Cloth
	clothWidth, clothHeight := float32(cloth.RenderWidth)/2*scale, float32(cloth.RenderHeight)/2*scale

	widest := clothWidth
	for _, size := range preview.game.Sizes {
		widest = max(widest, float32(size.Width)*scale)
	}

	if !imgui.BeginTableV("sizes", 2, imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.TableSetupColumnV("name", imgui.TableColumnFlagsWidthFixed, gui.Scaled(80), 0)
	imgui.TableSetupColumnV("flag", imgui.TableColumnFlagsWidthFixed, widest, 0)

	// The cloth first, which is how the games' own previews list them: the
	// largest at the top.
	a.previewRow(labelFancy, cloth.Width, cloth.Height, clothWidth, clothHeight, widest, preview.fancy.Target())

	for index, size := range preview.game.Sizes {
		a.previewRow(size.Name, size.FlagWidth, size.FlagHeight,
			float32(size.Width)*scale, float32(size.Height)*scale, widest, preview.icons.Target(index))
	}
}

// previewRow is one size: its name, level with the middle of the flag, and
// the flag itself, centred in the column.
func (a *App) previewRow(name string, width, height int32, drawnWidth, drawnHeight, column float32, target rl.RenderTexture2D) {
	imgui.TableNextRow()

	imgui.TableNextColumn()
	imgui.SetCursorPosY(imgui.CursorPosY() + max((drawnHeight-imgui.TextLineHeight())/2, 0))
	imgui.TextUnformatted(name)
	gui.Record(name)

	imgui.TableNextColumn()
	imgui.SetCursorPosX(imgui.CursorPosX() + max((column-drawnWidth)/2, 0))
	a.window.Backend().DrawImageRenderTexture(target, imgui.Vec2{X: drawnWidth, Y: drawnHeight}, true, false)
	gui.Record(name + " flag")

	sizeTooltip(name, width, height)
}

// missingGameFiles says which of a game's own files the preview needs and has
// not found.
//
// The shading, the frames and the cloth belong to the game rather than to any
// coat of arms, so they are read from the configured folders. A set of
// folders with only mods in it has none of them, and what the preview can
// show then is the flag itself.
func (a *App) missingGameFiles(preview *preview) {
	var missing []string

	for _, file := range preview.game.Files() {
		if _, ok := a.state.library.set.GameArt(file.Name); !ok {
			missing = append(missing, file.Path)
		}
	}

	if problem := preview.fancy.Problem(); problem != nil {
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

	dimmedWrapped(fmt.Sprintf("%s of %s %s not in any configured folder, so the flag is shown without what "+
		"the game draws it with. Add the game's folder in the settings to see the rest.",
		plural(len(missing), "file", "files"), preview.game.Name, verb))

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

// rankField chooses which rank's border the icons are drawn in. The game
// picks it from the country's rank; here it is the user's to try out, one
// button per rank with the chosen one pressed in. A game whose frames are the
// same whatever the rank has no such choice.
func (a *App) rankField(preview *preview) {
	ranks := preview.icons.Ranks()
	if ranks <= 1 {
		return
	}

	if preview.rank < 1 {
		preview.rank = 1
	}

	gui.Label(labelRank)

	for rank := 1; rank <= ranks; rank++ {
		if rank > 1 {
			imgui.SameLine()
		}

		if gui.ToggleButton(fmt.Sprintf("%d", rank), rank == preview.rank) {
			preview.rank = rank
		}
	}
}

// sizeTooltip names the size of the flag the pointer is over.
func sizeTooltip(name string, width, height int32) {
	if imgui.IsItemHovered() {
		imgui.SetTooltip(sizeLabel(name, width, height))
	}
}

// sizeLabel names a size the way the games' own files give it: what it is
// called, and how many pixels of a flag it draws.
func sizeLabel(name string, width, height int32) string {
	return fmt.Sprintf("%s: %dx%d", name, width, height)
}
