package render

import (
	"image"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// IconSize is one of the sizes Victoria 3 draws a country's flag at in its
// interface, with the frame it draws around it.
//
// The numbers are the game's own, from gui/shared/flags.gui: the size of the
// flag, the size of the rank border, which is larger than the flag and centred
// on it, and the size of one frame of the border's texture, which holds one
// frame per rank side by side.
type IconSize struct {
	// Name is what the game calls the size.
	Name string

	// Width and Height are the flag itself.
	Width, Height int32

	// BorderWidth and BorderHeight are the frame drawn around it.
	BorderWidth, BorderHeight int32

	// Border is the file of the game's the frame is drawn from, which holds
	// one frame per rank side by side.
	Border string

	// FrameWidth and FrameHeight are one frame of that file.
	FrameWidth, FrameHeight int32
}

// IconSizes are the sizes of a country's flag in Victoria 3, largest first.
var IconSizes = []IconSize{
	{
		Name: "Large", Width: 96, Height: 64,
		BorderWidth: 114, BorderHeight: 82,
		Border:     "flag_power_large.dds",
		FrameWidth: 228, FrameHeight: 164,
	},
	{
		Name: "Normal", Width: 66, Height: 44,
		BorderWidth: 80, BorderHeight: 58,
		Border:     "flag_power_normal.dds",
		FrameWidth: 160, FrameHeight: 116,
	},
	{
		Name: "Small", Width: 48, Height: 32,
		BorderWidth: 62, BorderHeight: 46,
		Border:     "flag_power_small.dds",
		FrameWidth: 124, FrameHeight: 92,
	},
	{
		Name: "Tiny", Width: 27, Height: 18,
		BorderWidth: 33, BorderHeight: 24,
		Border:     "flag_power_tiny.dds",
		FrameWidth: 66, FrameHeight: 48,
	},
}

// InterfaceFlagFolder is where the game keeps the artwork its interface draws
// a flag with, below its game folder.
const InterfaceFlagFolder = "gfx/interface/flag"

// OverlayTexture is the file of the shading the game multiplies over every
// flag its interface draws.
const OverlayTexture = "flag_overlay.dds"

// Icons draws a coat of arms the way Victoria 3's interface draws a country's
// flag: at each of the game's sizes, shaded by the overlay the game multiplies
// over it, and framed by the border of a rank.
//
// The overlay and the borders belong to the game rather than to any coat of
// arms, and are read from the configured folders like any other texture. A
// set of folders without them draws the flags on their own.
type Icons struct {
	painter  *Painter
	textures *Textures
	targets  []rl.RenderTexture2D
}

// NewIcons allocates one render target per size. It requires an active OpenGL
// context, so it must be called after the window exists.
//
// The targets keep their size for their whole lifetime, for the reason Preview
// gives: the interface caches texture references by raylib texture id.
func NewIcons(painter *Painter, textures *Textures) *Icons {
	icons := &Icons{painter: painter, textures: textures}

	for _, size := range IconSizes {
		target := rl.LoadRenderTexture(size.BorderWidth, size.BorderHeight)
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
	for index, size := range IconSizes {
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
	// The flag sits in the middle of the target, which is as large as the
	// border drawn around it.
	area := rl.Rectangle{
		X:      float32(size.BorderWidth-size.Width) / 2,
		Y:      float32(size.BorderHeight-size.Height) / 2,
		Width:  float32(size.Width),
		Height: float32(size.Height),
	}

	i.painter.Draw(flag, area)

	// The game stretches the overlay over the flag and multiplies it in, which
	// is what gives every flag in its interface the same shading.
	if overlay, ok := i.textures.Get(OverlayTexture); ok {
		rl.BeginBlendMode(rl.BlendMultiplied)
		drawTexture(overlay, wholeTexture(overlay), area, rl.Vector2{}, 0, rl.White)
		rl.EndBlendMode()
	}

	border, ok := i.textures.Get(size.Border)
	if !ok {
		return
	}

	whole := rl.Rectangle{Width: float32(size.BorderWidth), Height: float32(size.BorderHeight)}

	drawTexture(border, size.frame(border, rank), whole, rl.Vector2{}, 0, rl.White)
}

// frame is the part of a border texture holding the frame of a rank. The
// frames sit side by side, the first being rank one; a rank the file has no
// frame for takes the nearest it has.
func (s IconSize) frame(border rl.Texture2D, rank int) rl.Rectangle {
	count := max(border.Width/s.FrameWidth, 1)
	index := int32(min(max(rank, 1), int(count))) - 1

	return rl.Rectangle{
		X:      float32(index * s.FrameWidth),
		Width:  float32(s.FrameWidth),
		Height: float32(s.FrameHeight),
	}
}

// Ranks is how many ranks the borders hold a frame for, once one of them has
// been read. It is zero until then, and for folders that have none.
func (i *Icons) Ranks() int {
	for _, size := range IconSizes {
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
	for _, target := range i.targets {
		rl.UnloadRenderTexture(target)
	}
}
