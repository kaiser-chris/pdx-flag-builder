package app

import (
	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
)

// tableFlags are shared by every listing: scrollable, striped, with columns
// the user can resize, and sorted by clicking a column's header.
const tableFlags = imgui.TableFlagsRowBg |
	imgui.TableFlagsBordersInnerV |
	imgui.TableFlagsScrollY |
	imgui.TableFlagsResizable |
	imgui.TableFlagsSortable |
	imgui.TableFlagsSizingStretchProp

// The size thumbnails are shown at in the lists, in the units the interface
// was designed in: the proportions of a flag, as the Odin version showed them.
const (
	thumbnailWidth  = 60
	thumbnailHeight = 40
)

// rowHeight is the height of a row with a thumbnail in it.
func rowHeight() float32 {
	return gui.Scaled(thumbnailHeight)
}

// setupThumbnailColumn adds the column thumbnails are shown in. There is
// nothing in it to sort by.
func setupThumbnailColumn() {
	imgui.TableSetupColumnV("##preview",
		imgui.TableColumnFlagsWidthFixed|imgui.TableColumnFlagsNoSort|imgui.TableColumnFlagsNoResize,
		gui.Scaled(thumbnailWidth), 0)
}

// tableOrder is how the user sorted a table: by which column, and which way.
type tableOrder struct {
	column     int
	descending bool
}

// unsorted is the order of a table that has none: the order rows come in.
var unsorted = tableOrder{column: -1}

// currentOrder reads how the table being built is sorted. It has to be called
// after the table's columns are set up.
func currentOrder() tableOrder {
	specs := imgui.TableGetSortSpecs()
	if specs == nil || specs.SpecsCount() == 0 {
		return unsorted
	}

	// Without TableFlagsSortMulti there is never more than one.
	spec := specs.Specs()

	return tableOrder{
		column:     int(spec.ColumnIndex()),
		descending: spec.SortDirection() == imgui.SortDirectionDescending,
	}
}

// centredCell lines the next item up with the middle of a row of the given
// height, for a cell next to one holding a thumbnail. lineHeight is the height
// of the item that follows.
func centredCell(height, lineHeight float32) {
	imgui.SetCursorPosY(imgui.CursorPosY() + max(height-lineHeight, 0)/2)
}

// highlightedCell is a cell of text, centred in its row, with the parts that
// match the search marked.
func highlightedCell(text, query string, height float32) {
	centredCell(height, imgui.TextLineHeight())
	gui.HighlightedText(text, query)
}

// plainCell is a cell of text, centred in its row.
func plainCell(text string, height float32) {
	centredCell(height, imgui.TextLineHeight())
	imgui.TextUnformatted(text)
}

// flagThumbnail shows a rendered coat of arms.
func (a *App) flagThumbnail(flag *pdx.Flag) {
	a.showThumbnail(a.thumbnails.Flag(flagThumbnailKey(flag), flag))
}

// flagThumbnailKey tells apart coats of arms of the same name from different
// files, which a list shows side by side.
func flagThumbnailKey(flag *pdx.Flag) string {
	return flag.Origin.Path + "|" + flag.Name
}

// textureThumbnail shows a texture file as it is, marker colours and all.
func (a *App) textureThumbnail(path string) {
	a.showThumbnail(a.thumbnails.Texture(path))
}

// showThumbnail draws a thumbnail, or leaves its space empty while it is still
// being drawn.
func (a *App) showThumbnail(thumbnail render.Thumbnail, ready bool) {
	size := gui.ScaledVec2(thumbnailWidth, thumbnailHeight)

	if !ready {
		imgui.Dummy(size)

		return
	}

	if a.thumbnailAtlas == nil {
		// The atlas lives as long as the application, so it is registered with
		// the backend once. Registering it again every frame would grow the
		// backend's texture cache without end.
		atlas := a.window.Backend().CreateTextureFromTexture2D(thumbnail.Texture)
		a.thumbnailAtlas = &atlas
	}

	imgui.ImageV(*a.thumbnailAtlas, size,
		imgui.Vec2{X: thumbnail.U0, Y: thumbnail.V0},
		imgui.Vec2{X: thumbnail.U1, Y: thumbnail.V1})
}
