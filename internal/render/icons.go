package render

import (
	"image"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Icons draws a coat of arms the way Victoria 3's interface draws a country's
// flag: at each of the game's sizes, shaded by the overlay the game multiplies
// over it, and framed by the border of a rank.
//
// The overlay and the borders belong to the game rather than to any coat of
// arms, and are read from the configured folders like any other texture. A
// set of folders without them draws the flags on their own.
type Icons struct {
	game     *Game
	painter  *Painter
	textures *Textures
	targets  []rl.RenderTexture2D

	// canvas is the coat of arms at its own size, which every icon is a copy
	// of rather than a drawing of its own.
	canvas *Canvas
}

// NewIcons allocates one render target per size. It requires an active OpenGL
// context, so it must be called after the window exists.
//
// The targets keep their size for their whole lifetime, for the reason Preview
// gives: the interface caches texture references by raylib texture id.
func NewIcons(game *Game, painter *Painter, textures *Textures) *Icons {
	icons := &Icons{
		game:     game,
		painter:  painter,
		textures: textures,
		canvas:   NewCanvas(painter),
	}

	for _, size := range game.Sizes {
		target := rl.LoadRenderTexture(size.Width, size.Height)
		rl.SetTextureFilter(target.Texture, rl.FilterBilinear)

		icons.targets = append(icons.targets, target)
	}

	return icons
}

// Draw composes the icons of a coat of arms, one per size, in the border of
// the given rank, counted from one. Passing nil leaves them empty.
//
// It has to run while raylib drawing is active and before the interface
// samples the targets.
func (i *Icons) Draw(flag *pdx.Flag, rank int) {
	if flag != nil {
		i.canvas.Draw(*flag)
	}

	for index, size := range i.game.Sizes {
		rl.BeginTextureMode(i.targets[index])
		rl.ClearBackground(rl.Blank)

		if flag != nil {
			i.drawIcon(index, *flag, size, rank)
		}

		rl.EndTextureMode()
	}
}

// drawIcon draws one icon into the target that is already bound.
func (i *Icons) drawIcon(index int, flag pdx.Flag, size IconSize, rank int) {
	// The flag sits in the middle of the picture, which is as large as
	// whatever the game draws around it.
	area := rl.Rectangle{
		X:      float32(size.Width-size.FlagWidth) / 2,
		Y:      float32(size.Height-size.FlagHeight) / 2,
		Width:  float32(size.FlagWidth),
		Height: float32(size.FlagHeight),
	}

	whole := rl.Rectangle{Width: float32(size.Width), Height: float32(size.Height)}

	drawTexture(i.canvas.Texture(), i.canvas.Source(), area, rl.Vector2{}, 0, rl.White)

	// The game stretches the overlay over the flag and multiplies it in, which
	// is what gives every flag in its interface the same shading.
	if overlay, ok := i.textures.Get(i.game.Overlay); ok {
		rl.BeginBlendMode(rl.BlendMultiplied)
		drawTexture(overlay, wholeTexture(overlay), area, rl.Vector2{}, 0, rl.White)
		rl.EndBlendMode()
	}

	// A mask cuts the flag to the shape of the picture that is not see
	// through, which is how a round flag is drawn.
	if mask, ok := i.textures.Get(size.Mask); ok {
		rl.BeginBlendMode(rl.BlendCustom)
		rl.SetBlendFactors(rlZero, rlSourceAlpha, rlAdd)
		drawTexture(mask, wholeTexture(mask), whole, rl.Vector2{}, 0, rl.White)
		rl.EndBlendMode()
	}

	if border, ok := i.textures.Get(size.Border); ok {
		drawTexture(border, size.frame(border, rank), whole, rl.Vector2{}, 0, rl.White)
	}
}

// The blending a mask is drawn with: what is already there, kept only as far
// as the mask is solid. raylib has no name for it, so it is given the numbers
// OpenGL knows it by.
const (
	rlZero        = 0
	rlSourceAlpha = 0x0302
	rlAdd         = 0x8006
)

// frame is the part of a border picture holding the frame of a rank. The
// frames sit side by side, the first being rank one; a rank the file has no
// frame for takes the nearest it has. A border of a single frame is the whole
// picture.
func (s IconSize) frame(border rl.Texture2D, rank int) rl.Rectangle {
	if s.FrameWidth <= 0 {
		return wholeTexture(border)
	}

	count := max(border.Width/s.FrameWidth, 1)
	index := int32(min(max(rank, 1), int(count))) - 1

	return rl.Rectangle{
		X:      float32(index * s.FrameWidth),
		Width:  float32(s.FrameWidth),
		Height: float32(s.FrameHeight),
	}
}

// Ranks is how many ranks the borders hold a frame for, once one of them has
// been read. It is zero until then, and for a game whose borders hold one
// frame whatever the rank.
func (i *Icons) Ranks() int {
	for _, size := range i.game.Sizes {
		if size.FrameWidth <= 0 {
			continue
		}

		if border, ok := i.textures.Get(size.Border); ok {
			return int(max(border.Width/size.FrameWidth, 1))
		}
	}

	return 0
}

// Target is the icon of the size at an index of IconSizes.
func (i *Icons) Target(index int) rl.RenderTexture2D {
	return i.targets[index]
}

// Image reads an icon back from the GPU, the right way up. It needs the OpenGL
// context, so it has to be called from the goroutine that owns the window.
func (i *Icons) Image(index int) *image.RGBA {
	return readBack(i.targets[index].Texture, false)
}

// Unload releases the render targets. It requires a live OpenGL context, so it
// has to run before the window is destroyed.
func (i *Icons) Unload() {
	i.canvas.Unload()

	for _, target := range i.targets {
		rl.UnloadRenderTexture(target)
	}
}
