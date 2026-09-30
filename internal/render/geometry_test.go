package render

import (
	"image/color"
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// canvas is the flag at its own size, placed away from the origin so that a
// rectangle computed without the flag's offset shows up.
var canvas = rl.Rectangle{X: 100, Y: 50, Width: FlagWidth, Height: FlagHeight}

func nearly(got, want float32) bool {
	return math.Abs(float64(got-want)) < 1e-3
}

func sameRect(got, want rl.Rectangle) bool {
	return nearly(got.X, want.X) && nearly(got.Y, want.Y) &&
		nearly(got.Width, want.Width) && nearly(got.Height, want.Height)
}

// bounds is the rectangle a quad takes up.
func bounds(quad Quad) rl.Rectangle {
	left, right := quad[0].X, quad[0].X
	top, bottom := quad[0].Y, quad[0].Y

	for _, corner := range quad[1:] {
		left, right = min(left, corner.X), max(right, corner.X)
		top, bottom = min(top, corner.Y), max(bottom, corner.Y)
	}

	return rl.Rectangle{X: left, Y: top, Width: right - left, Height: bottom - top}
}

func sameQuad(got, want Quad) bool {
	for index, corner := range got {
		if !nearly(corner.X, want[index].X) || !nearly(corner.Y, want[index].Y) {
			return false
		}
	}

	return true
}

func TestInstanceQuad(t *testing.T) {
	// The corners of the flag itself, in the order a quad holds them.
	whole := Quad{{X: 100, Y: 50}, {X: 868, Y: 50}, {X: 868, Y: 562}, {X: 100, Y: 562}}

	tests := []struct {
		name     string
		instance pdx.Instance
		want     Quad
	}{
		{
			// The implied placement covers the whole flag.
			name:     "default",
			instance: pdx.NewInstance(),
			want:     whole,
		},
		{
			name:     "half size in the top left quarter",
			instance: pdx.Instance{Position: pdx.Vec2{X: 0.25, Y: 0.25}, Scale: pdx.Vec2{X: 0.5, Y: 0.5}},
			want:     Quad{{X: 100, Y: 50}, {X: 484, Y: 50}, {X: 484, Y: 306}, {X: 100, Y: 306}},
		},
		{
			// A negative scale mirrors the emblem, and its corners cross over.
			name:     "mirrored",
			instance: pdx.Instance{Position: pdx.Vec2{X: 0.5, Y: 0.5}, Scale: pdx.Vec2{X: -1, Y: 1}},
			want:     Quad{{X: 868, Y: 50}, {X: 100, Y: 50}, {X: 100, Y: 562}, {X: 868, Y: 562}},
		},
		{
			// Half a turn is the same rectangle upside down.
			name:     "half a turn",
			instance: pdx.Instance{Position: pdx.Vec2{X: 0.5, Y: 0.5}, Scale: pdx.Vec2{X: 1, Y: 1}, Rotation: 180},
			want:     Quad{{X: 868, Y: 562}, {X: 100, Y: 562}, {X: 100, Y: 50}, {X: 868, Y: 50}},
		},
		{
			// A quarter turn leaves the emblem in the rectangle it had, with the
			// texture lying on its side in it. That is why Catalonia's nine
			// stripes, turned upright, still cover the flag from edge to edge.
			name:     "a quarter turn covers the same rectangle",
			instance: pdx.Instance{Position: pdx.Vec2{X: 0.5, Y: 0.5}, Scale: pdx.Vec2{X: 1, Y: 1}, Rotation: 90},
			want:     Quad{{X: 868, Y: 50}, {X: 868, Y: 562}, {X: 100, Y: 562}, {X: 100, Y: 50}},
		},
		{
			// The canton of the Orange Free State: a tricolour turned upright
			// into exactly the rectangle the plain emblem under it fills.
			name: "a quarter turn of a canton",
			instance: pdx.Instance{
				Position: pdx.Vec2{X: 0.2, Y: 0.25}, Scale: pdx.Vec2{X: 0.4, Y: 0.5}, Rotation: 90,
			},
			want: Quad{{X: 407.2, Y: 50}, {X: 407.2, Y: 306}, {X: 100, Y: 306}, {X: 100, Y: 50}},
		},
	}

	for _, test := range tests {
		if got := instanceQuad(test.instance, canvas); !sameQuad(got, test.want) {
			t.Errorf("%s: instanceQuad = %+v, want %+v", test.name, got, test.want)
		}
	}
}

// The diamond of the fascist Hungarian and Polish flags: a square emblem given
// an eighth of a turn, whose corners land halfway along the sides of the space
// it takes up.
//
// Those flags write the scale so that the emblem is square in pixels, a third
// of the width by half the height, and the game draws a diamond 362 pixels
// across on its 768 by 512 canvas.
func TestInstanceQuadOfAnEighthTurn(t *testing.T) {
	instance := pdx.Instance{
		Position: pdx.Vec2{X: 0.5, Y: 0.5},
		Scale:    pdx.Vec2{X: 1.0 / 3, Y: 0.5},
		Rotation: 45,
	}

	quad := instanceQuad(instance, canvas)

	const across = 362.0387 // 256 * sqrt(2)

	if box := bounds(quad); !nearly(box.Width, across) || !nearly(box.Height, across) {
		t.Errorf("the diamond takes up %v by %v, want %v square", box.Width, box.Height, across)
	}

	centre := rl.Vector2{X: canvas.X + canvas.Width/2, Y: canvas.Y + canvas.Height/2}

	// Top, right, bottom and left, each halfway along its side.
	want := Quad{
		{X: centre.X, Y: centre.Y - across/2},
		{X: centre.X + across/2, Y: centre.Y},
		{X: centre.X, Y: centre.Y + across/2},
		{X: centre.X - across/2, Y: centre.Y},
	}

	if !sameQuad(quad, want) {
		t.Errorf("the diamond has corners %+v, want %+v", quad, want)
	}
}

func TestSubFlagRect(t *testing.T) {
	// A sub flag is placed by its top left corner.
	instance := pdx.SubInstance{Offset: pdx.Vec2{X: 0.5, Y: 0.25}, Scale: pdx.Vec2{X: 0.5, Y: 0.5}}
	want := rl.Rectangle{X: 100 + 384, Y: 50 + 128, Width: 384, Height: 256}

	if got := subFlagRect(instance, canvas); !sameRect(got, want) {
		t.Errorf("subFlagRect = %+v, want %+v", got, want)
	}

	if got := subFlagRect(pdx.NewSubInstance(), canvas); !sameRect(got, canvas) {
		t.Errorf("the implied sub flag placement = %+v, want the whole flag %+v", got, canvas)
	}
}

func TestFitRect(t *testing.T) {
	box := rl.Rectangle{X: 10, Y: 20, Width: ThumbnailWidth, Height: ThumbnailHeight}

	// A square texture is as tall as the box and centred across it.
	square := rl.Texture2D{Width: 256, Height: 256}
	if got, want := fitRect(square, box), (rl.Rectangle{X: 30, Y: 20, Width: 80, Height: 80}); !sameRect(got, want) {
		t.Errorf("fitRect of a square = %+v, want %+v", got, want)
	}

	// A wide one is as wide as the box and centred up and down.
	wide := rl.Texture2D{Width: 400, Height: 100}
	if got, want := fitRect(wide, box), (rl.Rectangle{X: 10, Y: 45, Width: 120, Height: 30}); !sameRect(got, want) {
		t.Errorf("fitRect of a wide texture = %+v, want %+v", got, want)
	}

	// A texture that failed to load has no size to keep.
	if got := fitRect(rl.Texture2D{}, box); !sameRect(got, box) {
		t.Errorf("fitRect of an empty texture = %+v, want the box", got)
	}
}

// An unrotated emblem maps straight onto the part of the pattern under it.
func TestMaskForAnUnrotatedEmblem(t *testing.T) {
	target := instanceQuad(pdx.Instance{Position: pdx.Vec2{X: 0.25, Y: 0.75}, Scale: pdx.Vec2{X: 0.5, Y: 0.5}}, canvas)

	mask := maskFor(rl.Texture2D{}, color.RGBA{R: 255, A: 255}, canvas, target)

	if !nearly(mask.UVOffset[0], 0) || !nearly(mask.UVOffset[1], 0.5) {
		t.Errorf("UV offset = %v, want the emblem's top left corner at (0, 0.5) of the pattern", mask.UVOffset)
	}

	if !nearly(mask.UVAxisX[0], 0.5) || !nearly(mask.UVAxisX[1], 0) ||
		!nearly(mask.UVAxisY[0], 0) || !nearly(mask.UVAxisY[1], 0.5) {
		t.Errorf("UV axes = %v and %v, want half the pattern along each edge", mask.UVAxisX, mask.UVAxisY)
	}
}

// A quarter turn swaps the directions the emblem's edges run in, and leaves it
// over the same part of the pattern as before.
func TestMaskForARotatedEmblem(t *testing.T) {
	instance := pdx.Instance{Position: pdx.Vec2{X: 0.25, Y: 0.75}, Scale: pdx.Vec2{X: 0.5, Y: 0.5}, Rotation: 90}

	mask := maskFor(rl.Texture2D{}, color.RGBA{}, canvas, instanceQuad(instance, canvas))

	// The texture now starts at the far corner of that part of the pattern.
	if !nearly(mask.UVOffset[0], 0.5) || !nearly(mask.UVOffset[1], 0.5) {
		t.Errorf("UV offset = %v, want the corner at (0.5, 0.5) of the pattern", mask.UVOffset)
	}

	// Its top edge runs down the pattern, its left edge leftwards.
	if !nearly(mask.UVAxisX[0], 0) || !nearly(mask.UVAxisX[1], 0.5) {
		t.Errorf("UV axis along the top edge = %v, want straight down", mask.UVAxisX)
	}

	if !nearly(mask.UVAxisY[0], -0.5) || !nearly(mask.UVAxisY[1], 0) {
		t.Errorf("UV axis along the left edge = %v, want straight left", mask.UVAxisY)
	}
}

// A mirrored quad has its corners the other way round, and is drawn from the
// other end so that the graphics card is still shown the front of it.
func TestMirroredQuad(t *testing.T) {
	plain := instanceQuad(pdx.NewInstance(), canvas)
	if plain.mirrored() {
		t.Error("a plain emblem counts as mirrored")
	}

	flipped := pdx.Instance{Position: pdx.Vec2{X: 0.5, Y: 0.5}, Scale: pdx.Vec2{X: -1, Y: 1}}
	if !instanceQuad(flipped, canvas).mirrored() {
		t.Error("an emblem mirrored across does not count as mirrored")
	}

	over := pdx.Instance{Position: pdx.Vec2{X: 0.5, Y: 0.5}, Scale: pdx.Vec2{X: -1, Y: -1}}
	if instanceQuad(over, canvas).mirrored() {
		t.Error("an emblem mirrored both ways counts as mirrored, but it is only turned over")
	}

	if got := quadOf(canvas); !sameQuad(got, plain) {
		t.Errorf("quadOf the flag = %+v, want the same as an emblem covering it %+v", got, plain)
	}
}

// span is the stretch a drawn texture covers across the flag, left to right,
// as raylib computes it when nothing is rotated.
func span(destination rl.Rectangle, origin rl.Vector2) (low, high float32) {
	start := destination.X - origin.X
	end := start + destination.Width

	return min(start, end), max(start, end)
}

func TestUnmirror(t *testing.T) {
	source := rl.Rectangle{Width: 64, Height: 32}

	tests := []struct {
		name        string
		destination rl.Rectangle
		origin      rl.Vector2
	}{
		// An emblem pivots on its middle.
		{"mirrored emblem", rl.Rectangle{X: 192, Y: 256, Width: -384, Height: 512}, rl.Vector2{X: -192, Y: 256}},
		// A pattern is drawn from its corner.
		{"mirrored pattern", rl.Rectangle{X: 768, Y: 0, Width: -768, Height: 512}, rl.Vector2{}},
		{"upside down", rl.Rectangle{X: 0, Y: 512, Width: 768, Height: -512}, rl.Vector2{}},
	}

	for _, test := range tests {
		gotSource, gotDestination, gotOrigin := unmirror(source, test.destination, test.origin)

		if gotDestination.Width < 0 || gotDestination.Height < 0 {
			t.Errorf("%s: destination %+v, want a positive size for raylib", test.name, gotDestination)
		}

		// The sign moved to the source, which is how raylib mirrors.
		if (gotSource.Width < 0) != (test.destination.Width < 0) || (gotSource.Height < 0) != (test.destination.Height < 0) {
			t.Errorf("%s: source %+v, want it negative where the destination was", test.name, gotSource)
		}

		// And the texture covers the same part of the flag as before.
		wantLow, wantHigh := span(test.destination, test.origin)
		gotLow, gotHigh := span(gotDestination, gotOrigin)

		if !nearly(gotLow, wantLow) || !nearly(gotHigh, wantHigh) {
			t.Errorf("%s: covers %v to %v, want %v to %v", test.name, gotLow, gotHigh, wantLow, wantHigh)
		}
	}

	// Nothing mirrored, nothing changed.
	plain := rl.Rectangle{X: 10, Y: 20, Width: 30, Height: 40}
	if gotSource, gotDestination, gotOrigin := unmirror(source, plain, rl.Vector2{X: 15, Y: 20}); gotSource != source ||
		gotDestination != plain || gotOrigin != (rl.Vector2{X: 15, Y: 20}) {
		t.Errorf("unmirror changed a plain rectangle: %+v %+v %+v", gotSource, gotDestination, gotOrigin)
	}
}
