// Package render draws flags with raylib.
//
// raylib is what the Go rewrite keeps from the Odin version. A coat of arms is
// composed on the GPU: a pattern is drawn first and the layers over it, each
// through a fragment shader that swaps the marker colours the textures are
// painted in for the colours the coat of arms asks for.
package render

import (
	"image"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// The flag canvas is a fixed size in both games, and the Odin version rendered
// it at exactly this resolution.
const (
	FlagWidth  = pdx.CanvasWidth
	FlagHeight = pdx.CanvasHeight
)

// checkerSize is the edge length of one square of the transparency checkerboard.
const checkerSize = 16

var (
	checkerLight = rl.NewColor(0x50, 0x54, 0x5B, 0xFF)
	checkerDark  = rl.NewColor(0x44, 0x48, 0x4E, 0xFF)
	borderColor  = rl.NewColor(0x33, 0x3A, 0x45, 0xFF)
	labelColor   = rl.NewColor(0x9A, 0xA3, 0xB0, 0xFF)
)

// Preview is the offscreen target a flag is composed into before the interface
// draws it as an image.
//
// The target keeps its size for its whole lifetime on purpose. The Dear ImGui
// backend caches texture references by raylib texture id, and raylib reuses ids
// after an unload, so recreating the target would hand the cache a reference to
// a freed texture.
type Preview struct {
	target  rl.RenderTexture2D
	width   int32
	height  int32
	painter *Painter

	// checker is one tile of the transparency pattern, repeated across the
	// target in a single draw.
	checker rl.Texture2D
}

// NewPreview allocates the render target. It requires an active OpenGL context,
// so it must be called after the window exists.
func NewPreview(width, height int32, painter *Painter) *Preview {
	preview := &Preview{
		target:  rl.LoadRenderTexture(width, height),
		width:   width,
		height:  height,
		painter: painter,
		checker: checkerTexture(),
	}

	// The preview is scaled to fit its panel, so it wants smooth minification.
	rl.SetTextureFilter(preview.target.Texture, rl.FilterBilinear)

	return preview
}

// checkerTexture builds the two by two tile the transparency pattern repeats.
func checkerTexture() rl.Texture2D {
	tile := rl.GenImageChecked(checkerSize*2, checkerSize*2, checkerSize, checkerSize, checkerDark, checkerLight)
	defer rl.UnloadImage(tile)

	texture := rl.LoadTextureFromImage(tile)

	// Repeating is what lets one quad cover the whole target, and the squares
	// want hard edges rather than a blur.
	rl.SetTextureWrap(texture, rl.WrapRepeat)
	rl.SetTextureFilter(texture, rl.FilterPoint)

	return texture
}

// Target is the render texture holding the most recently drawn frame.
func (p *Preview) Target() rl.RenderTexture2D {
	return p.target
}

// Size reports the resolution flags are composed at.
func (p *Preview) Size() (width, height int32) {
	return p.width, p.height
}

// Draw composes a coat of arms into the render target. Passing nil draws the
// empty state instead. It has to run while raylib drawing is active and before
// the interface samples the texture.
func (p *Preview) Draw(flag *pdx.Flag) {
	rl.BeginTextureMode(p.target)
	defer rl.EndTextureMode()

	rl.ClearBackground(rl.Blank)

	p.drawCheckerboard()

	if flag == nil {
		p.drawPlaceholder("No flag open")
	} else {
		p.painter.Draw(*flag, rl.Rectangle{Width: float32(p.width), Height: float32(p.height)})
	}

	rl.DrawRectangleLines(0, 0, p.width, p.height, borderColor)
}

// drawCheckerboard fills the target with the pattern that marks transparency,
// so that the see through parts of a flag look deliberate.
//
// It is one quad of a repeating texture rather than a grid of small rectangles.
// That is not only faster: filling the target with well over a thousand
// rectangles leaves so much queued in raylib's batch that the shader draws
// which follow come out clipped to a fraction of their size.
func (p *Preview) drawCheckerboard() {
	source := rl.Rectangle{Width: float32(p.width), Height: float32(p.height)}
	destination := source

	rl.DrawTexturePro(p.checker, source, destination, rl.Vector2{}, 0, rl.White)
}

func (p *Preview) drawPlaceholder(label string) {
	const fontSize = 20

	width := rl.MeasureText(label, fontSize)
	rl.DrawText(label, (p.width-width)/2, (p.height-fontSize)/2, fontSize, labelColor)
}

// Image reads the most recently drawn frame back from the GPU, the right way up.
// It needs the OpenGL context, so it has to be called from the goroutine that
// owns the window.
func (p *Preview) Image() *image.RGBA {
	return readBack(p.target.Texture, false)
}

// Unload releases the render target. It requires a live OpenGL context, so it
// has to run before the window is destroyed.
func (p *Preview) Unload() {
	rl.UnloadRenderTexture(p.target)
	rl.UnloadTexture(p.checker)
}
