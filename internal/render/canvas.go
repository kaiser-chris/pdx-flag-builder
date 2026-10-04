package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Canvas is a coat of arms drawn at the size its artwork is made for, which
// everything smaller is then a copy of.
//
// A coat of arms is never composed at a smaller size than this. Composing it
// smaller would have the graphics card read the artwork from the smaller
// copies it keeps of each texture, and those are blends: of the marker colours
// that say which colour slot a pixel belongs to, which the shader has to
// recognise one of, and of the solid and see through parts of an emblem, which
// decide where it is drawn at all. A flag of 27 pixels built that way comes out
// as whatever those blends happened to mean, which is how the games' own
// stripes used to lose a stripe.
//
// Drawn once at its own size and copied from the smaller copies of that, every
// pixel of the artwork is accounted for, and the copy is what the games
// themselves draw their small flags from.
type Canvas struct {
	painter *Painter
	target  rl.RenderTexture2D
}

// NewCanvas allocates the target a coat of arms is composed into. It requires
// an active OpenGL context, so it must be called after the window exists.
func NewCanvas(painter *Painter) *Canvas {
	canvas := &Canvas{
		painter: painter,
		target:  rl.LoadRenderTexture(FlagWidth, FlagHeight),
	}

	// The smaller copies are made again after every drawing; the filter that
	// reads them is set once, and only once there are some to read.
	rl.GenTextureMipmaps(&canvas.target.Texture)
	rl.SetTextureFilter(canvas.target.Texture, rl.FilterTrilinear)

	return canvas
}

// Draw composes a coat of arms and has the graphics card make the smaller
// copies of it.
//
// It has to run while raylib drawing is active, and not while another target
// is being drawn into, since raylib draws into one at a time.
func (c *Canvas) Draw(flag pdx.Flag) {
	rl.BeginTextureMode(c.target)
	rl.ClearBackground(rl.Blank)

	c.painter.Draw(flag, rl.Rectangle{Width: FlagWidth, Height: FlagHeight})

	rl.EndTextureMode()

	rl.GenTextureMipmaps(&c.target.Texture)
}

// Texture is what was drawn, for copying into whatever shows it.
func (c *Canvas) Texture() rl.Texture2D {
	return c.target.Texture
}

// Source is the whole of it as a source rectangle. A render target is filled
// from the bottom up, so it is read back the other way round.
func (c *Canvas) Source() rl.Rectangle {
	return rl.Rectangle{Width: FlagWidth, Height: -FlagHeight}
}

// Unload releases the target. It requires a live OpenGL context, so it has to
// run before the window is destroyed.
func (c *Canvas) Unload() {
	rl.UnloadRenderTexture(c.target)
}
