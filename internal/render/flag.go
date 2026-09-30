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

// Ready reports whether every texture a coat of arms needs, its sub flags'
// included, has arrived or failed for good, and asks for any that have not
// been wanted yet. Once it is ready, drawing it again would draw the same.
func (p *Painter) Ready(flag pdx.Flag) bool {
	return p.ready(flag, 0)
}

func (p *Painter) ready(flag pdx.Flag, depth int) bool {
	if depth > maxSubFlagDepth {
		return true
	}

	// Every texture is asked for before answering, rather than stopping at
	// the first one missing, so that they are all read at the same time.
	ready := p.textures.Settled(flag.Pattern)

	for _, layer := range flag.Layers {
		switch typed := layer.(type) {
		case *pdx.ColoredEmblem:
			ready = p.textures.Settled(typed.Texture) && ready

		case *pdx.TexturedEmblem:
			ready = p.textures.Settled(typed.Texture) && ready

		case *pdx.SubFlag:
			if p.subFlag == nil {
				continue
			}

			if parent, found := p.subFlag(typed.Parent); found {
				ready = p.ready(parent, depth+1) && ready
			}
		}
	}

	return ready
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
			p.drawTexturedEmblem(typed, flag, destination)

		case *pdx.SubFlag:
			p.drawSubFlag(typed, flag, destination, depth)
		}
	}
}

func (p *Painter) drawPattern(flag pdx.Flag, destination rl.Rectangle) {
	pattern, ok := p.textures.Get(flag.Pattern)
	if !ok {
		return
	}

	recolorings := p.recolorings(flag.Colors, flag.Colors, PatternSlotColors)

	p.shader.Draw(pattern, wholeTexture(pattern), quadOf(destination), DrawOptions{
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

	maskTexture, maskColor, masked := p.mask(emblem.Mask, flag)

	for _, instance := range pdx.Placements(emblem.Instances) {
		target := instanceQuad(instance, destination)

		options := DrawOptions{
			Recolorings:        recolorings,
			BlueChannelShading: true,
		}

		if masked {
			mask := maskFor(maskTexture, maskColor, destination, target)
			options.Mask = &mask
		}

		p.shader.Draw(texture, wholeTexture(texture), target, options)
	}
}

func (p *Painter) drawTexturedEmblem(emblem *pdx.TexturedEmblem, flag pdx.Flag, destination rl.Rectangle) {
	texture, ok := p.textures.Get(emblem.Texture)
	if !ok {
		return
	}

	maskTexture, maskColor, masked := p.mask(emblem.Mask, flag)

	for _, instance := range pdx.Placements(emblem.Instances) {
		target := instanceQuad(instance, destination)

		// A textured emblem is already in its final colours, so it is drawn
		// as it is. Only a mask needs the shader, which replaces no colour
		// and cuts the emblem to the part of the pattern it belongs to.
		if !masked {
			drawQuad(texture, wholeTexture(texture), target, rl.White)

			continue
		}

		mask := maskFor(maskTexture, maskColor, destination, target)

		p.shader.Draw(texture, wholeTexture(texture), target, DrawOptions{Mask: &mask})
	}
}

func (p *Painter) drawSubFlag(sub *pdx.SubFlag, flag pdx.Flag, destination rl.Rectangle, depth int) {
	if p.subFlag == nil {
		return
	}

	parent, ok := p.subFlag(sub.Parent)
	if !ok {
		return
	}

	parent = p.handOver(parent, sub.Colors, flag.Colors)

	for _, instance := range pdx.SubPlacements(sub.Instances) {
		p.draw(parent, subFlagRect(instance, destination), depth+1)
	}
}

// handOver gives a coat of arms the colours the sub flag layer hands it, in
// place of its own.
//
// What is handed over is worked out here rather than passed down, because a
// colour of the layer may refer back to a slot of the flag the layer sits on,
// which the coat of arms being drawn knows nothing about.
func (p *Painter) handOver(parent pdx.Flag, handed, flag pdx.Colors) pdx.Flag {
	if len(handed) == 0 {
		return parent
	}

	parent.Colors = append(pdx.Colors(nil), parent.Colors...)

	for _, entry := range handed {
		resolved, ok := entry.Resolve(p.palette, flag)
		if !ok {
			// A colour that cannot be worked out is left to the coat of arms
			// being drawn, which has one of its own.
			continue
		}

		parent.Colors.Set(pdx.Color{Slot: entry.Slot, Value: pdx.RGBColor{R: resolved.R, G: resolved.G, B: resolved.B}})
	}

	return parent
}

// mask returns the pattern texture and the marker colour a masked emblem is
// restricted to, and whether the emblem is masked at all.
func (p *Painter) mask(mask int, flag pdx.Flag) (rl.Texture2D, color.RGBA, bool) {
	if mask <= 0 || mask > len(PatternSlotColors) || flag.Pattern == "" {
		return rl.Texture2D{}, color.RGBA{}, false
	}

	pattern, ok := p.textures.Get(flag.Pattern)
	if !ok {
		return rl.Texture2D{}, color.RGBA{}, false
	}

	// A mask of one means the first slot, so the count is one ahead of the
	// index into the marker colours.
	return pattern, PatternSlotColors[mask-1], true
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

// instanceQuad works out where one placement of an emblem lands on the flag.
//
// The games turn an emblem inside its own picture rather than on the flag: the
// texture is turned as the square it is drawn as, and that turned picture is
// then stretched to the size the scale asks for, across by the width of the
// flag and down by its height. On a canvas half again as wide as it is tall
// the two orders are not the same, and the games' own flags show which one
// they use. A quarter turn leaves an emblem in exactly the rectangle it had,
// with the texture lying on its side in it: that is why Catalonia's nine
// stripes, turned upright, still cover the flag from edge to edge, and why the
// Orange Free State's canton is filled by a tricolour turned into it. Turned
// by anything else the picture leans over, and at an eighth of a turn it comes
// out the diamond the fascist Hungarian and Polish flags are drawn with.
func instanceQuad(instance pdx.Instance, flag rl.Rectangle) Quad {
	halfWidth := flag.Width * instance.Scale.X / 2
	halfHeight := flag.Height * instance.Scale.Y / 2

	sin, cos := math.Sincos(float64(instance.Rotation) * math.Pi / 180)

	// Where the texture's own axes end up: turned first, and each part of the
	// turned axis then stretched by the flag's own width and height.
	alongX := rl.Vector2{X: halfWidth * float32(cos), Y: halfHeight * float32(sin)}
	alongY := rl.Vector2{X: -halfWidth * float32(sin), Y: halfHeight * float32(cos)}

	// A position of one half is the middle of the flag, and moving away from
	// it shifts the emblem by that fraction of the whole flag.
	centre := rl.Vector2{
		X: flag.X + flag.Width*instance.Position.X,
		Y: flag.Y + flag.Height*instance.Position.Y,
	}

	return Quad{
		{X: centre.X - alongX.X - alongY.X, Y: centre.Y - alongX.Y - alongY.Y},
		{X: centre.X + alongX.X - alongY.X, Y: centre.Y + alongX.Y - alongY.Y},
		{X: centre.X + alongX.X + alongY.X, Y: centre.Y + alongX.Y + alongY.Y},
		{X: centre.X - alongX.X + alongY.X, Y: centre.Y - alongX.Y + alongY.Y},
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

// maskFor works out how the shader turns a point on the emblem into a point on
// the pattern, so that it can look up which slot that pixel of the pattern
// belongs to.
//
// The emblem is drawn as its own quad with its own position, scale and
// rotation, so the corner the texture starts at and the two edges it runs
// along are worked out in the flag's own space first, then divided through by
// the flag rectangle to land in the pattern's texture coordinates.
func maskFor(texture rl.Texture2D, markerColor color.RGBA, flag rl.Rectangle, target Quad) Mask {
	alongX, alongY := target.edges()

	return Mask{
		Texture: texture,
		Color:   markerColor,
		UVOffset: [2]float32{
			(target[0].X - flag.X) / flag.Width,
			(target[0].Y - flag.Y) / flag.Height,
		},
		UVAxisX: [2]float32{alongX.X / flag.Width, alongX.Y / flag.Height},
		UVAxisY: [2]float32{alongY.X / flag.Width, alongY.Y / flag.Height},
	}
}
