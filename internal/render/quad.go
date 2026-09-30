package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Quad is where a texture is drawn: its four corners on the flag, in the order
// the texture's own corners come in, top left, top right, bottom right, bottom
// left.
//
// An emblem needs this rather than a rectangle and an angle, because a turned
// emblem is not a turned rectangle. The games turn the texture inside its own
// picture and stretch the turned picture onto the flag, and stretching a
// turned square by different amounts across and down leans it over. Only an
// emblem drawn square on the flag, or one turned by half a turn, comes out a
// rectangle again.
type Quad [4]rl.Vector2

// quadOf is a rectangle as a quad, its corners in the same order. A rectangle
// of negative size is mirrored, and its corners simply cross over.
func quadOf(rect rl.Rectangle) Quad {
	left, right := rect.X, rect.X+rect.Width
	top, bottom := rect.Y, rect.Y+rect.Height

	return Quad{
		{X: left, Y: top},
		{X: right, Y: top},
		{X: right, Y: bottom},
		{X: left, Y: bottom},
	}
}

// edges are the quad's own axes: how far the texture's x axis and y axis run
// across the flag, from its top left corner.
func (q Quad) edges() (alongX, alongY rl.Vector2) {
	return rl.Vector2{X: q[1].X - q[0].X, Y: q[1].Y - q[0].Y},
		rl.Vector2{X: q[3].X - q[0].X, Y: q[3].Y - q[0].Y}
}

// mirrored reports whether the quad is turned over, which a negative scale
// does. Its corners then come in the other order, and the graphics card, which
// throws away everything facing away from the viewer, would drop it.
func (q Quad) mirrored() bool {
	alongX, alongY := q.edges()

	return alongX.X*alongY.Y-alongX.Y*alongY.X < 0
}

// drawQuad draws part of a texture into a quad.
//
// raylib draws a texture into a rectangle it may turn about a point, which a
// leaning quad is not, so this hands the corners to the graphics card itself.
// They go in raylib's own order, top left, bottom left, bottom right, top
// right, and in the other direction for a mirrored quad, so that the face the
// card is shown is always the front one.
func drawQuad(texture rl.Texture2D, source rl.Rectangle, quad Quad, tint rl.Color) {
	if texture.ID == 0 || texture.Width == 0 || texture.Height == 0 {
		return
	}

	width, height := float32(texture.Width), float32(texture.Height)

	corners := [4]rl.Vector2{
		{X: source.X / width, Y: source.Y / height},
		{X: (source.X + source.Width) / width, Y: source.Y / height},
		{X: (source.X + source.Width) / width, Y: (source.Y + source.Height) / height},
		{X: source.X / width, Y: (source.Y + source.Height) / height},
	}

	order := [4]int{0, 3, 2, 1}
	if quad.mirrored() {
		order = [4]int{1, 2, 3, 0}
	}

	rl.SetTexture(texture.ID)
	rl.Begin(rl.Quads)

	rl.Color4ub(tint.R, tint.G, tint.B, tint.A)
	rl.Normal3f(0, 0, 1)

	for _, corner := range order {
		rl.TexCoord2f(corners[corner].X, corners[corner].Y)
		rl.Vertex2f(quad[corner].X, quad[corner].Y)
	}

	rl.End()
	rl.SetTexture(0)
}
