// Package render draws flags with raylib.
//
// raylib is what the Go rewrite keeps from the Odin version: flags are composed
// on the GPU from pattern and emblem textures with a recolouring fragment
// shader, so that pipeline can be ported across largely unchanged. This file
// owns the offscreen target the interface samples; the layer compositing itself
// is ported in a later step and currently draws a placeholder.
package render

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// The flag canvas is a fixed size in both games, and the Odin version rendered
// it at exactly this resolution.
const (
	FlagWidth  = 768
	FlagHeight = 512
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
	target rl.RenderTexture2D
	width  int32
	height int32
}

// NewPreview allocates the render target. It requires an active OpenGL context,
// so it must be called after the window exists.
func NewPreview(width, height int32) *Preview {
	preview := &Preview{
		target: rl.LoadRenderTexture(width, height),
		width:  width,
		height: height,
	}

	// The preview is scaled to fit its panel, so it wants smooth minification.
	rl.SetTextureFilter(preview.target.Texture, rl.FilterBilinear)

	return preview
}

// Target is the render texture holding the most recently drawn frame.
func (p *Preview) Target() rl.RenderTexture2D {
	return p.target
}

// Size reports the resolution flags are composed at.
func (p *Preview) Size() (width, height int32) {
	return p.width, p.height
}

// Draw composes the current flag into the render target. It has to run while
// raylib drawing is active and before the interface samples the texture.
func (p *Preview) Draw() {
	rl.BeginTextureMode(p.target)
	defer rl.EndTextureMode()

	rl.ClearBackground(rl.Blank)

	p.drawCheckerboard()

	// TODO: replace with the ported layer compositing (pattern, coloured
	// emblems, textured emblems and sub flags) from the Odin renderer.
	p.drawPlaceholder()

	rl.DrawRectangleLines(0, 0, p.width, p.height, borderColor)
}

// drawCheckerboard fills the target with the pattern that marks transparency,
// so an empty flag does not look like a rendering failure.
func (p *Preview) drawCheckerboard() {
	for y := int32(0); y < p.height; y += checkerSize {
		for x := int32(0); x < p.width; x += checkerSize {
			color := checkerLight
			if (x/checkerSize+y/checkerSize)%2 == 0 {
				color = checkerDark
			}

			rl.DrawRectangle(x, y, checkerSize, checkerSize, color)
		}
	}
}

func (p *Preview) drawPlaceholder() {
	const fontSize = 20

	label := "No flag loaded"
	width := rl.MeasureText(label, fontSize)
	rl.DrawText(label, (p.width-width)/2, p.height/2-fontSize, fontSize, labelColor)

	hint := fmt.Sprintf("%d x %d", p.width, p.height)
	hintWidth := rl.MeasureText(hint, fontSize-6)
	rl.DrawText(hint, (p.width-hintWidth)/2, p.height/2+8, fontSize-6, labelColor)
}

// Unload releases the render target. It requires a live OpenGL context, so it
// has to run before the window is destroyed.
func (p *Preview) Unload() {
	rl.UnloadRenderTexture(p.target)
}
