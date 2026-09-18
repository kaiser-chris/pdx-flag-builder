package render

import (
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// The size of one thumbnail in the atlas, in pixels. Flags have the proportions
// of the canvas; textures are fitted into the same box. The lists show them
// at 60 by 40 units, so this stays sharp up to twice the interface scale.
const (
	ThumbnailWidth  = 120
	ThumbnailHeight = 80
)

const (
	atlasSize = 2048

	// thumbnailGutter keeps neighbouring thumbnails apart in the atlas, so that
	// sampling one at its edge never blends in a pixel of the next.
	thumbnailGutter = 2

	// thumbnailsPerFrame caps the drawing done in one frame, so that scrolling
	// a list into a page of new rows never costs a visible stall.
	thumbnailsPerFrame = 8

	// thumbnailTextures is how many textures are kept around for drawing
	// thumbnails once they are no longer needed. Patterns in particular are
	// shared by many flags, so keeping some saves reading them again.
	thumbnailTextures = 48
)

// Thumbnail is where a finished thumbnail is in the atlas.
type Thumbnail struct {
	Texture rl.Texture2D

	// U0, V0, U1 and V1 are the texture coordinates of its top left and bottom
	// right corners, already flipped for the render target being stored bottom
	// up.
	U0, V0, U1, V1 float32
}

// Thumbnails draws small previews of flags and textures for the lists, once
// each, into cells of one large render target.
//
// Drawing every visible flag again every frame would be a lot of work for
// images that never change, and keeping every texture of every flag a user
// scrolls past on the GPU would be a lot of memory. So a thumbnail is drawn
// once, when everything it needs has arrived, and the textures that went into
// it can be dropped again. Cells that have not been shown for the longest are
// reused when the atlas is full.
//
// The thumbnails keep their own textures and painter, apart from the ones the
// editor draws with, so that scrolling a list never unloads what the open flag
// is using.
type Thumbnails struct {
	atlas    rl.RenderTexture2D
	textures *Textures
	painter  *Painter

	columns int

	// cells maps each drawn thumbnail to its cell, free lists the unused cells.
	cells map[string]*thumbnailCell
	free  []int

	// wanted are the thumbnails asked for since the last Draw that are not
	// drawn yet. Anything no longer asked for is forgotten, so rows that were
	// scrolled past quickly are never drawn.
	wanted map[string]thumbnailJob

	frame uint64
}

type thumbnailCell struct {
	index int

	// shown is the frame the thumbnail was last asked for in.
	shown uint64
}

// thumbnailJob is what a thumbnail shows: a flag, or else a texture as it is.
type thumbnailJob struct {
	flag    *pdx.Flag
	texture string
}

// NewThumbnails allocates the atlas. It requires an active OpenGL context.
func NewThumbnails(shader Recolor, resolve func(name string) (string, bool)) *Thumbnails {
	textures := NewTextures(func(name string) (string, bool) {
		if path, ok := strings.CutPrefix(name, filePrefix); ok {
			return path, true
		}

		return resolve(name)
	})

	stride := ThumbnailWidth + thumbnailGutter
	columns := atlasSize / stride
	count := columns * (atlasSize / (ThumbnailHeight + thumbnailGutter))

	thumbnails := &Thumbnails{
		atlas:    rl.LoadRenderTexture(atlasSize, atlasSize),
		textures: textures,
		painter:  NewPainter(shader, textures),
		columns:  columns,
		cells:    map[string]*thumbnailCell{},
		free:     make([]int, 0, count),
		wanted:   map[string]thumbnailJob{},
	}

	// Filled from the back, so that cells are handed out from the top left.
	for index := count - 1; index >= 0; index-- {
		thumbnails.free = append(thumbnails.free, index)
	}

	rl.SetTextureFilter(thumbnails.atlas.Texture, rl.FilterBilinear)

	// Cleared once, so that a cell never shows what the GPU memory held before.
	rl.BeginTextureMode(thumbnails.atlas)
	rl.ClearBackground(rl.Blank)
	rl.EndTextureMode()

	return thumbnails
}

// SetPalette gives the painter the named colours flags are drawn with.
func (t *Thumbnails) SetPalette(palette pdx.Palette) {
	t.painter.SetPalette(palette)
}

// SetSubFlagLookup gives the painter a way to find the coats of arms that sub
// flag layers refer to.
func (t *Thumbnails) SetSubFlagLookup(lookup func(name string) (pdx.Flag, bool)) {
	t.painter.SetSubFlagLookup(lookup)
}

// Flag returns the thumbnail of a coat of arms, or asks for it to be drawn.
// key has to tell apart coats of arms of the same name from different folders.
// The flag is read when the thumbnail is drawn, so it has to stay unchanged
// until Forget.
func (t *Thumbnails) Flag(key string, flag *pdx.Flag) (Thumbnail, bool) {
	return t.get("flag:"+key, thumbnailJob{flag: flag})
}

// filePrefix marks a texture asked for by its path rather than by the name
// coats of arms refer to it by.
const filePrefix = "file:"

// Texture returns the thumbnail of a texture file as it is, or asks for it to
// be drawn. It takes a path rather than a name because a game and a mod can
// both have a texture of the same name, and a list shows both.
func (t *Thumbnails) Texture(path string) (Thumbnail, bool) {
	return t.get("texture:"+path, thumbnailJob{texture: filePrefix + path})
}

func (t *Thumbnails) get(key string, job thumbnailJob) (Thumbnail, bool) {
	if cell, ok := t.cells[key]; ok {
		cell.shown = t.frame

		return t.thumbnail(cell.index), true
	}

	t.wanted[key] = job

	return Thumbnail{}, false
}

// Draw draws the thumbnails that were asked for and whose textures have all
// arrived. It has to run while raylib drawing is active, before the interface
// that shows them is built.
func (t *Thumbnails) Draw() {
	t.frame++
	t.textures.Upload()

	if len(t.wanted) == 0 {
		t.textures.Trim(thumbnailTextures)

		return
	}

	drawn := 0

	rl.BeginTextureMode(t.atlas)

	for key, job := range t.wanted {
		if drawn == thumbnailsPerFrame {
			break
		}

		if !t.ready(job) {
			continue
		}

		index, ok := t.allocate()
		if !ok {
			break
		}

		t.drawCell(index, job)
		t.cells[key] = &thumbnailCell{index: index, shown: t.frame}
		drawn++
	}

	rl.EndTextureMode()

	// Whatever is still wanted will be asked for again by the rows showing it.
	clear(t.wanted)

	t.textures.Trim(thumbnailTextures)
}

func (t *Thumbnails) ready(job thumbnailJob) bool {
	if job.flag != nil {
		return t.painter.Ready(*job.flag)
	}

	return t.textures.Settled(job.texture)
}

// allocate hands out a free cell, or else the one shown least recently. A cell
// shown in the last frame is on screen and is never taken.
func (t *Thumbnails) allocate() (int, bool) {
	if count := len(t.free); count > 0 {
		index := t.free[count-1]
		t.free = t.free[:count-1]

		return index, true
	}

	oldestKey := ""

	var oldest *thumbnailCell

	for key, cell := range t.cells {
		if cell.shown+1 < t.frame && (oldest == nil || cell.shown < oldest.shown) {
			oldestKey, oldest = key, cell
		}
	}

	if oldest == nil {
		return 0, false
	}

	delete(t.cells, oldestKey)

	return oldest.index, true
}

func (t *Thumbnails) drawCell(index int, job thumbnailJob) {
	cell := t.cellRect(index)

	// Everything drawn is clipped to the cell: an emblem placed partly off its
	// flag must not spill into the neighbouring thumbnail.
	rl.BeginScissorMode(int32(cell.X), int32(cell.Y), int32(cell.Width), int32(cell.Height))
	defer rl.EndScissorMode()

	// Clearing respects the scissor, so this empties just the one cell.
	rl.ClearBackground(rl.Blank)

	if job.flag != nil {
		t.painter.Draw(*job.flag, cell)

		return
	}

	texture, ok := t.textures.Get(job.texture)
	if !ok {
		// Not found or not readable: the cell stays empty.
		return
	}

	rl.DrawTexturePro(texture, wholeTexture(texture), fitRect(texture, cell), rl.Vector2{}, 0, rl.White)
}

// fitRect is the largest rectangle with the texture's proportions that fits in
// the box, centred in it.
func fitRect(texture rl.Texture2D, box rl.Rectangle) rl.Rectangle {
	if texture.Width <= 0 || texture.Height <= 0 {
		return box
	}

	scale := min(box.Width/float32(texture.Width), box.Height/float32(texture.Height))
	width, height := float32(texture.Width)*scale, float32(texture.Height)*scale

	return rl.Rectangle{
		X:      box.X + (box.Width-width)/2,
		Y:      box.Y + (box.Height-height)/2,
		Width:  width,
		Height: height,
	}
}

func (t *Thumbnails) cellRect(index int) rl.Rectangle {
	column, row := index%t.columns, index/t.columns

	return rl.Rectangle{
		X:      float32(column*(ThumbnailWidth+thumbnailGutter) + thumbnailGutter/2),
		Y:      float32(row*(ThumbnailHeight+thumbnailGutter) + thumbnailGutter/2),
		Width:  ThumbnailWidth,
		Height: ThumbnailHeight,
	}
}

func (t *Thumbnails) thumbnail(index int) Thumbnail {
	cell := t.cellRect(index)

	// The render target is stored bottom up, so the top of the cell is the
	// larger V.
	return Thumbnail{
		Texture: t.atlas.Texture,
		U0:      cell.X / atlasSize,
		V0:      1 - cell.Y/atlasSize,
		U1:      (cell.X + cell.Width) / atlasSize,
		V1:      1 - (cell.Y+cell.Height)/atlasSize,
	}
}

// Atlas is the render target every thumbnail is drawn into. It stays the same
// for the lifetime of the thumbnails.
func (t *Thumbnails) Atlas() rl.RenderTexture2D {
	return t.atlas
}

// Forget throws every thumbnail away, so that each is drawn again the next
// time it is asked for. It is how a change to the configured folders, and so
// to the flags and textures behind the thumbnails, takes effect.
func (t *Thumbnails) Forget() {
	for _, cell := range t.cells {
		t.free = append(t.free, cell.index)
	}

	clear(t.cells)
	clear(t.wanted)
	t.textures.Forget()
}

// Unload releases the atlas and the textures. It requires a live OpenGL
// context.
func (t *Thumbnails) Unload() {
	t.textures.Unload()
	rl.UnloadRenderTexture(t.atlas)
}
