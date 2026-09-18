package pdx

// The canvas a flag is composed on. Positions and scales in the script are
// fractions of it.
const (
	CanvasWidth  = 768
	CanvasHeight = 512
)

// Defaults for the parts of an instance a file leaves out.
var (
	DefaultPosition = Vec2{X: 0.5, Y: 0.5}
	DefaultScale    = Vec2{X: 1, Y: 1}
	DefaultOffset   = Vec2{}
)

// Vec2 is a position or a scale, in fractions of the canvas.
type Vec2 struct {
	X, Y float32
}

// Origin records where a flag was read from, so that the interface can show it
// and the exporter can write it back to the right file.
type Origin struct {
	// Database is the name of the configured folder the flag came from.
	Database string

	// File is the file name on its own, Path the whole path.
	File string
	Path string

	// Line is where the definition started.
	Line int
}

// Flag is one coat of arms: a pattern with layers drawn over it.
type Flag struct {
	// Name is the key the coat of arms was defined under, which is also how
	// sub flags refer to it.
	Name string

	// Pattern is the file name of the background texture. It may be empty.
	Pattern string

	// Colors are the slots the pattern is recoloured with, and the slots a
	// layer can refer back to.
	Colors Colors

	Layers []Layer

	Origin Origin
}

// Layer is one entry in a flag's stack of layers: a *ColoredEmblem, a
// *TexturedEmblem or a *SubFlag.
//
// Layers are pointers so that the editor can change one where it sits. Clone
// copies them, so a cloned flag never shares a layer with the original.
type Layer interface {
	// InstanceCount is how many placements the file gives the layer.
	InstanceCount() int

	layer()
}

// ColoredEmblem is an emblem whose texture is recoloured through the flag's
// colour slots.
type ColoredEmblem struct {
	Texture string

	// Colors are the emblem's own slots. A slot may refer back to a slot of
	// the flag the emblem belongs to.
	Colors Colors

	// Mask restricts the emblem to the part of the pattern matching one of the
	// pattern's colours. Zero means the emblem is not masked.
	Mask int

	Instances []Instance
}

// TexturedEmblem is an emblem drawn as it is, without recolouring.
type TexturedEmblem struct {
	Texture   string
	Instances []Instance
}

// SubFlag draws another coat of arms as a layer of this one.
type SubFlag struct {
	// Parent is the name of the coat of arms to draw.
	Parent    string
	Instances []SubInstance
}

func (*ColoredEmblem) layer()  {}
func (*TexturedEmblem) layer() {}
func (*SubFlag) layer()        {}

func (l *ColoredEmblem) InstanceCount() int  { return len(l.Instances) }
func (l *TexturedEmblem) InstanceCount() int { return len(l.Instances) }
func (l *SubFlag) InstanceCount() int        { return len(l.Instances) }

// Instance is one placement of an emblem on the flag.
type Instance struct {
	Position Vec2
	Scale    Vec2

	// Rotation is in degrees, clockwise.
	Rotation float32
}

// SubInstance is one placement of a sub flag. Sub flags are positioned by an
// offset rather than by a centre point.
type SubInstance struct {
	Scale  Vec2
	Offset Vec2
}

// NewInstance returns an instance with the defaults a file leaves out.
func NewInstance() Instance {
	return Instance{Position: DefaultPosition, Scale: DefaultScale}
}

// NewSubInstance returns a sub flag instance with the defaults a file leaves out.
func NewSubInstance() SubInstance {
	return SubInstance{Scale: DefaultScale, Offset: DefaultOffset}
}

// Placements returns where an emblem is drawn.
//
// An emblem without any instance is drawn once at the default position and
// scale, which is how the games read it. Keeping the stored list empty rather
// than filling in that default means writing the flag back out does not invent
// an instance the author never wrote.
func Placements(instances []Instance) []Instance {
	if len(instances) == 0 {
		return []Instance{NewInstance()}
	}

	return instances
}

// SubPlacements is Placements for sub flags.
func SubPlacements(instances []SubInstance) []SubInstance {
	if len(instances) == 0 {
		return []SubInstance{NewSubInstance()}
	}

	return instances
}

// Texture returns the texture a layer draws, and whether it has one. A sub flag
// draws another coat of arms instead, so it has none.
func Texture(layer Layer) (string, bool) {
	switch typed := layer.(type) {
	case *ColoredEmblem:
		return typed.Texture, true
	case *TexturedEmblem:
		return typed.Texture, true
	}

	return "", false
}

// Clone returns a deep copy.
//
// The flag open in the editor is a copy of the one that was read from disk, so
// that editing it does not change what the database still shows. Colour values
// are immutable, so only the slices need copying.
func (f Flag) Clone() Flag {
	clone := f
	clone.Colors = append(Colors(nil), f.Colors...)

	clone.Layers = make([]Layer, 0, len(f.Layers))
	for _, layer := range f.Layers {
		clone.Layers = append(clone.Layers, cloneLayer(layer))
	}

	return clone
}

// cloneLayer copies a layer, including the slices it holds.
func cloneLayer(layer Layer) Layer {
	switch typed := layer.(type) {
	case *ColoredEmblem:
		clone := *typed
		clone.Colors = append(Colors(nil), typed.Colors...)
		clone.Instances = append([]Instance(nil), typed.Instances...)

		return &clone

	case *TexturedEmblem:
		clone := *typed
		clone.Instances = append([]Instance(nil), typed.Instances...)

		return &clone

	case *SubFlag:
		clone := *typed
		clone.Instances = append([]SubInstance(nil), typed.Instances...)

		return &clone
	}

	return layer
}
