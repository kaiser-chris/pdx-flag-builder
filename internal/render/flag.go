package render

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// PatternSlotColors are the colours a pattern texture is painted in, one per
// colour slot. A pattern is not artwork in its own colours: it is a map saying
// which slot each pixel takes its colour from.
var PatternSlotColors = []color.RGBA{
	{R: 255, G: 0, B: 0, A: 255},
	{R: 255, G: 255, B: 0, A: 255},
	{R: 255, G: 255, B: 255, A: 255},
}

// EmblemSlotColors are the same thing for a coloured emblem. Only red and green
// identify the slot here; the blue channel carries a per pixel brightness, so
// that an emblem can have shading and still be recoloured.
var EmblemSlotColors = []color.RGBA{
	{R: 0, G: 0, B: 128, A: 255},
	{R: 0, G: 255, B: 128, A: 255},
	{R: 255, G: 0, B: 128, A: 255},
}

// maxSubFlagDepth stops a coat of arms that includes itself, directly or round
// a longer loop, from recursing forever.
const maxSubFlagDepth = 8

// Painter draws coats of arms.
type Painter struct {
	shader   Recolor
	textures *Textures

	palette pdx.Palette

	// subFlag finds the coat of arms a sub flag layer refers to.
	subFlag func(name string) (pdx.Flag, bool)
}

// NewPainter builds a painter around a compiled shader and a texture store.
func NewPainter(shader Recolor, textures *Textures) *Painter {
	return &Painter{shader: shader, textures: textures, palette: pdx.Palette{}}
}

// SetPalette gives the painter the named colours to resolve against.
func (p *Painter) SetPalette(palette pdx.Palette) {
	if palette == nil {
		palette = pdx.Palette{}
	}

	p.palette = palette
}

// SetSubFlagLookup gives the painter a way to find the coats of arms that sub
// flag layers refer to.
func (p *Painter) SetSubFlagLookup(lookup func(name string) (pdx.Flag, bool)) {
	p.subFlag = lookup
}

// Draw paints a coat of arms into a rectangle.
func (p *Painter) Draw(flag pdx.Flag, destination rl.Rectangle) {
	p.draw(flag, destination, 0)
}

func (p *Painter) draw(flag pdx.Flag, destination rl.Rectangle, depth int) {
	if depth > maxSubFlagDepth {
		return
	}

	p.drawPattern(flag, destination)

	// Layers are drawn in the order they were written, each over the last.
	for _, layer := range flag.Layers {
		switch typed := layer.(type) {
		case *pdx.ColoredEmblem:
			p.drawColoredEmblem(typed, flag, destination)

		case *pdx.TexturedEmblem:
			p.drawTexturedEmblem(typed, destination)

		case *pdx.SubFlag:
			p.drawSubFlag(typed, destination, depth)
		}
	}
}

func (p *Painter) drawPattern(flag pdx.Flag, destination rl.Rectangle) {
	pattern, ok := p.textures.Get(flag.Pattern)
	if !ok {
		return
	}

	recolorings := p.recolorings(flag.Colors, flag.Colors, PatternSlotColors)

	p.shader.Draw(pattern, wholeTexture(pattern), destination, DrawOptions{
		Recolorings: recolorings,
	})
}

func (p *Painter) drawColoredEmblem(emblem *pdx.ColoredEmblem, flag pdx.Flag, destination rl.Rectangle) {
	texture, ok := p.textures.Get(emblem.Texture)
	if !ok {
		return
	}

	// An emblem slot may point back at a slot of the flag it sits on, so the
	// flag's own colours are what those references resolve against.
	recolorings := p.recolorings(emblem.Colors, flag.Colors, EmblemSlotColors)

	maskTexture, maskColor, masked := p.mask(emblem, flag)

	for _, instance := range pdx.Placements(emblem.Instances) {
		target := instanceRect(instance, destination)
		origin := rl.Vector2{X: target.Width / 2, Y: target.Height / 2}

		options := DrawOptions{
			Origin:             origin,
			Rotation:           instance.Rotation,
			Recolorings:        recolorings,
			BlueChannelShading: true,
		}

		if masked {
			mask := maskFor(maskTexture, maskColor, destination, target, origin, instance.Rotation)
			options.Mask = &mask
		}

		p.shader.Draw(texture, wholeTexture(texture), target, options)
	}
}

func (p *Painter) drawTexturedEmblem(emblem *pdx.TexturedEmblem, destination rl.Rectangle) {
	texture, ok := p.textures.Get(emblem.Texture)
	if !ok {
		return
	}

	// A textured emblem is already in its final colours, so it is drawn as it
	// is with no shader involved.
	for _, instance := range pdx.Placements(emblem.Instances) {
		target := instanceRect(instance, destination)
		origin := rl.Vector2{X: target.Width / 2, Y: target.Height / 2}

		rl.DrawTexturePro(texture, wholeTexture(texture), target, origin, instance.Rotation, rl.White)
	}
}

func (p *Painter) drawSubFlag(sub *pdx.SubFlag, destination rl.Rectangle, depth int) {
	if p.subFlag == nil {
		return
	}

	parent, ok := p.subFlag(sub.Parent)
	if !ok {
		return
	}

	for _, instance := range pdx.SubPlacements(sub.Instances) {
		p.draw(parent, subFlagRect(instance, destination), depth+1)
	}
}

// mask returns the pattern texture and the marker colour a masked emblem is
// restricted to, and whether the emblem is masked at all.
func (p *Painter) mask(emblem *pdx.ColoredEmblem, flag pdx.Flag) (rl.Texture2D, color.RGBA, bool) {
	if emblem.Mask <= 0 || emblem.Mask > len(PatternSlotColors) || flag.Pattern == "" {
		return rl.Texture2D{}, color.RGBA{}, false
	}

	pattern, ok := p.textures.Get(flag.Pattern)
	if !ok {
		return rl.Texture2D{}, color.RGBA{}, false
	}

	// A mask of one means the first slot, so the count is one ahead of the
	// index into the marker colours.
	return pattern, PatternSlotColors[emblem.Mask-1], true
}

func (p *Painter) recolorings(colors, parent pdx.Colors, markers []color.RGBA) []Recoloring {
	var recolorings []Recoloring

	for _, entry := range colors {
		slot := pdx.SlotIndex(entry.Slot)
		if slot < 0 || slot >= len(markers) {
			// The textures only carry marker colours for the first few slots,
			// so anything beyond them has nothing to replace.
			continue
		}

		replacement, ok := entry.Resolve(p.palette, parent)
		if !ok {
			// An unresolvable colour is left alone rather than painted over
			// with the fallback, so the marker colour shows what went wrong.
			continue
		}

		recolorings = append(recolorings, Recoloring{Source: markers[slot], Replacement: replacement})
	}

	return recolorings
}

func wholeTexture(texture rl.Texture2D) rl.Rectangle {
	return rl.Rectangle{Width: float32(texture.Width), Height: float32(texture.Height)}
}

// instanceRect works out where one placement of an emblem lands on the flag.
func instanceRect(instance pdx.Instance, flag rl.Rectangle) rl.Rectangle {
	stretch, squish := rotationDistortion(instance.Rotation)

	width := flag.Width * instance.Scale.X * squish
	height := flag.Height * instance.Scale.Y * stretch

	// A position of one half is the middle of the flag, and moving away from
	// it shifts the emblem by that fraction of the whole flag.
	return rl.Rectangle{
		X:      flag.X + (flag.Width-width)/2 + flag.Width*(instance.Position.X-0.5) + width/2,
		Y:      flag.Y + (flag.Height-height)/2 + flag.Height*(instance.Position.Y-0.5) + height/2,
		Width:  width,
		Height: height,
	}
}

// subFlagRect works out where one placement of a sub flag lands. Sub flags are
// placed by an offset from the top left rather than by a centre point.
func subFlagRect(instance pdx.SubInstance, flag rl.Rectangle) rl.Rectangle {
	return rl.Rectangle{
		X:      flag.X + flag.Width*instance.Offset.X,
		Y:      flag.Y + flag.Height*instance.Offset.Y,
		Width:  flag.Width * instance.Scale.X,
		Height: flag.Height * instance.Scale.Y,
	}
}

// rotationDistortion reproduces the way the games resize a rotated emblem.
//
// The canvas is half again as wide as it is tall while scales are fractions of
// it, so a quarter turn would squash an emblem unless its size changes with the
// angle. Both factors are strongest at a quarter turn and fade to nothing at no
// turn and at a half turn. The numbers are an approximation carried over from
// the Odin version, which is the only description of this behaviour there is;
// four instances in the base game rotate at all.
func rotationDistortion(rotation float32) (stretch, squish float32) {
	angle := float32(math.Mod(float64(rotation), 180))
	if angle < 0 {
		angle += 180
	}

	amount := 1 - float32(math.Abs(float64((angle-90)/90)))

	return 1 + 0.5*amount, 1 - 0.25*amount
}

// maskFor works out how the shader turns a point on the emblem into a point on
// the pattern, so that it can look up which slot that pixel of the pattern
// belongs to.
//
// The emblem is drawn as its own quad with its own position, scale and
// rotation, so the corner and the two edge directions of that quad are worked
// out in the flag's own space first, then divided through by the flag
// rectangle to land in the pattern's texture coordinates.
func maskFor(
	texture rl.Texture2D,
	markerColor color.RGBA,
	flag rl.Rectangle,
	target rl.Rectangle,
	origin rl.Vector2,
	rotation float32,
) Mask {
	var (
		topLeft    [2]float32
		alongEdgeX [2]float32
		alongEdgeY [2]float32
	)

	if rotation == 0 {
		topLeft = [2]float32{target.X - origin.X, target.Y - origin.Y}
		alongEdgeX = [2]float32{target.Width, 0}
		alongEdgeY = [2]float32{0, target.Height}
	} else {
		radians := float64(rotation) * math.Pi / 180
		sin := float32(math.Sin(radians))
		cos := float32(math.Cos(radians))

		offsetX := -origin.X
		offsetY := -origin.Y

		topLeft = [2]float32{
			target.X + offsetX*cos - offsetY*sin,
			target.Y + offsetX*sin + offsetY*cos,
		}
		alongEdgeX = [2]float32{target.Width * cos, target.Width * sin}
		alongEdgeY = [2]float32{-target.Height * sin, target.Height * cos}
	}

	return Mask{
		Texture: texture,
		Color:   markerColor,
		UVOffset: [2]float32{
			(topLeft[0] - flag.X) / flag.Width,
			(topLeft[1] - flag.Y) / flag.Height,
		},
		UVAxisX: [2]float32{
			alongEdgeX[0] / flag.Width,
			alongEdgeX[1] / flag.Height,
		},
		UVAxisY: [2]float32{
			alongEdgeY[0] / flag.Width,
			alongEdgeY[1] / flag.Height,
		},
	}
}
