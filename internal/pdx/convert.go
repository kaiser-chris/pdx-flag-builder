package pdx

import (
	"github.com/kaiser-chris/pdx-parser-go/victoria3"
)

// FromCoatOfArms turns a coat of arms read by the parser library into the
// model the editor works on.
//
// The two models differ where editing needs them to: the editor holds its
// layers as pointers, so that changing one changes it where it sits, keeps
// its geometry in the float32 the renderer draws with, and remembers where
// the definition came from, so that it can be written back to its own file.
func FromCoatOfArms(arms *victoria3.CoatOfArms, origin Origin) Flag {
	flag := Flag{
		Name:    arms.Key,
		Pattern: arms.Pattern.File,
		Colors:  append(Colors(nil), arms.Colors...),
		Origin:  origin,
	}

	for _, layer := range arms.Layers {
		switch typed := layer.(type) {
		case *victoria3.ColoredEmblem:
			flag.Layers = append(flag.Layers, &ColoredEmblem{
				Texture:   typed.Texture.File,
				Colors:    append(Colors(nil), typed.Colors...),
				Mask:      mask(typed.Masks),
				Instances: instances(typed.Instances),
			})

		case *victoria3.TexturedEmblem:
			flag.Layers = append(flag.Layers, &TexturedEmblem{
				Texture:   typed.Texture.File,
				Instances: instances(typed.Instances),
			})

		case *victoria3.SubCoatOfArms:
			flag.Layers = append(flag.Layers, &SubFlag{
				Parent:    typed.Parent,
				Instances: subInstances(typed.Instances),
			})
		}
	}

	return flag
}

// instances converts the placements of an emblem. A layer the file gives no
// instance keeps none here as well, so that writing the flag back out does
// not invent one the author never wrote.
func instances(placements []victoria3.Instance) []Instance {
	if len(placements) == 0 {
		return nil
	}

	converted := make([]Instance, 0, len(placements))

	for _, placement := range placements {
		converted = append(converted, Instance{
			Position: point(placement.Position),
			Scale:    point(placement.Scale),
			Rotation: float32(placement.Rotation),
		})
	}

	return converted
}

// subInstances converts the placements of a sub flag, which is placed by an
// offset rather than by a centre point.
func subInstances(placements []victoria3.Instance) []SubInstance {
	if len(placements) == 0 {
		return nil
	}

	converted := make([]SubInstance, 0, len(placements))

	for _, placement := range placements {
		converted = append(converted, SubInstance{
			Scale:  point(placement.Scale),
			Offset: point(placement.Offset),
		})
	}

	return converted
}

// point converts a position, an offset or a scale into the float32 the
// renderer draws with.
func point(value victoria3.Vec2) Vec2 {
	return Vec2{X: float32(value.X), Y: float32(value.Y)}
}

// mask takes the pattern colour an emblem is restricted to. The files name
// one; the editor offers one.
func mask(masks []int) int {
	if len(masks) == 0 {
		return 0
	}

	return masks[0]
}
