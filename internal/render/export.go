package render

import (
	"image"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// OpenGL's blend factors and equation, which rlgl takes as plain numbers.
const (
	glOne              = 1
	glSrcAlpha         = 0x0302
	glOneMinusSrcAlpha = 0x0303
	glFuncAdd          = 0x8006
)

// Export draws a coat of arms on its own, on a transparent background, and
// reads it back as an image. It needs the OpenGL context, and every texture
// the flag uses should have arrived first; Painter.Ready says when.
//
// The usual blending multiplies alpha by itself, which is harmless on screen
// but would leave every soft emblem edge in the file partly see-through, even
// over an opaque pattern. So the layers are composed the way layers compose:
// the colour comes out premultiplied by its coverage, the coverage adds up,
// and the premultiplication is undone when the pixels are read back.
func Export(painter *Painter, flag pdx.Flag, width, height int32) *image.RGBA {
	target := rl.LoadRenderTexture(width, height)
	defer rl.UnloadRenderTexture(target)

	rl.BeginTextureMode(target)
	rl.ClearBackground(rl.Blank)

	rl.SetBlendFactorsSeparate(glSrcAlpha, glOneMinusSrcAlpha, glOne, glOneMinusSrcAlpha, glFuncAdd, glFuncAdd)
	rl.BeginBlendMode(rl.BlendCustomSeparate)

	// Emblems placed partly off the flag stay off the image.
	rl.BeginScissorMode(0, 0, width, height)
	painter.Draw(flag, rl.Rectangle{Width: float32(width), Height: float32(height)})
	rl.EndScissorMode()

	rl.EndBlendMode()
	rl.EndTextureMode()

	return readBack(target.Texture, true)
}

// readBack copies a render target's pixels into an image, the right way up.
// premultiplied says the colours were stored multiplied by their alpha, which
// an image file does not expect.
func readBack(texture rl.Texture2D, premultiplied bool) *image.RGBA {
	captured := rl.LoadImageFromTexture(texture)
	defer rl.UnloadImage(captured)

	// OpenGL fills a render target bottom up.
	rl.ImageFlipVertical(captured)

	colors := rl.LoadImageColors(captured)
	defer rl.UnloadImageColors(colors)

	width, height := int(captured.Width), int(captured.Height)
	picture := image.NewRGBA(image.Rect(0, 0, width, height))

	for index, value := range colors[:width*height] {
		if premultiplied && value.A > 0 && value.A < 255 {
			value.R = unpremultiply(value.R, value.A)
			value.G = unpremultiply(value.G, value.A)
			value.B = unpremultiply(value.B, value.A)
		}

		picture.SetRGBA(index%width, index/width, value)
	}

	return picture
}

func unpremultiply(channel, alpha uint8) uint8 {
	return uint8(min(255, (int(channel)*255+int(alpha)/2)/int(alpha)))
}
