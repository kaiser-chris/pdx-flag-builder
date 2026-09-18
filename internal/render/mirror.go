package render

import rl "github.com/gen2brain/raylib-go/raylib"

// drawTexture draws a texture into a rectangle whose size may be negative,
// which mirrors it, the way a negative scale in a coat of arms mirrors an
// emblem, or a whole sub flag.
func drawTexture(texture rl.Texture2D, source, destination rl.Rectangle, origin rl.Vector2, rotation float32, tint rl.Color) {
	source, destination, origin = unmirror(source, destination, origin)
	rl.DrawTexturePro(texture, source, destination, origin, rotation, tint)
}

// unmirror turns a mirrored rectangle into the form raylib draws mirrored.
//
// raylib mirrors a texture when the source rectangle has a negative size. A
// negative destination size it simply drops the sign of, while the origin,
// usually half the size, keeps it: the texture comes out unmirrored and moved
// by its own width. So the sign moves to the source, the destination becomes
// positive, and the origin moves to keep the drawing where the signed
// rectangle put it: for an emblem pivoting on its middle it stays in the
// middle, for a pattern drawn from its corner it moves to the far corner.
//
// The corners of the signed rectangle, relative to the point it turns about,
// are -origin and -origin+size. Mirrored, the same span runs from
// -(origin-size) to -(origin-size)+|size|, which is where the new origin comes
// from.
func unmirror(source, destination rl.Rectangle, origin rl.Vector2) (rl.Rectangle, rl.Rectangle, rl.Vector2) {
	if destination.Width < 0 {
		source.Width = -source.Width
		origin.X -= destination.Width
		destination.Width = -destination.Width
	}

	if destination.Height < 0 {
		source.Height = -source.Height
		origin.Y -= destination.Height
		destination.Height = -destination.Height
	}

	return source, destination, origin
}
