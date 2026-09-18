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

func TestInstanceRect(t *testing.T) {
	tests := []struct {
		name     string
		instance pdx.Instance
		want     rl.Rectangle
	}{
		{
			// The implied placement covers the whole flag. The rectangle is
			// given by its centre, which is where the emblem turns around.
			name:     "default",
			instance: pdx.NewInstance(),
			want:     rl.Rectangle{X: 100 + 384, Y: 50 + 256, Width: 768, Height: 512},
		},
		{
			name:     "half size in the top left quarter",
			instance: pdx.Instance{Position: pdx.Vec2{X: 0.25, Y: 0.25}, Scale: pdx.Vec2{X: 0.5, Y: 0.5}},
			want:     rl.Rectangle{X: 100 + 192, Y: 50 + 128, Width: 384, Height: 256},
		},
		{
			// A negative scale mirrors the emblem; the rectangle keeps its
			// centre and turns its width around.
			name:     "mirrored",
			instance: pdx.Instance{Position: pdx.Vec2{X: 0.5, Y: 0.5}, Scale: pdx.Vec2{X: -1, Y: 1}},
			want:     rl.Rectangle{X: 100 + 384, Y: 50 + 256, Width: -768, Height: 512},
		},
	}

	for _, test := range tests {
		if got := instanceRect(test.instance, canvas); !sameRect(got, test.want) {
			t.Errorf("%s: instanceRect = %+v, want %+v", test.name, got, test.want)
		}
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

func TestRotationDistortion(t *testing.T) {
	tests := []struct {
		rotation        float32
		stretch, squish float32
	}{
		{0, 1, 1},
		{90, 1.5, 0.75},
		{180, 1, 1},
		{270, 1.5, 0.75},
		{-90, 1.5, 0.75},
		{45, 1.25, 0.875},
	}

	for _, test := range tests {
		stretch, squish := rotationDistortion(test.rotation)
		if !nearly(stretch, test.stretch) || !nearly(squish, test.squish) {
			t.Errorf("rotationDistortion(%v) = %v, %v, want %v, %v",
				test.rotation, stretch, squish, test.stretch, test.squish)
		}
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
	target := instanceRect(pdx.Instance{Position: pdx.Vec2{X: 0.25, Y: 0.75}, Scale: pdx.Vec2{X: 0.5, Y: 0.5}}, canvas)
	origin := rl.Vector2{X: target.Width / 2, Y: target.Height / 2}

	mask := maskFor(rl.Texture2D{}, color.RGBA{R: 255, A: 255}, canvas, target, origin, 0)

	if !nearly(mask.UVOffset[0], 0) || !nearly(mask.UVOffset[1], 0.5) {
		t.Errorf("UV offset = %v, want the emblem's top left corner at (0, 0.5) of the pattern", mask.UVOffset)
	}

	if !nearly(mask.UVAxisX[0], 0.5) || !nearly(mask.UVAxisX[1], 0) ||
		!nearly(mask.UVAxisY[0], 0) || !nearly(mask.UVAxisY[1], 0.5) {
		t.Errorf("UV axes = %v and %v, want half the pattern along each edge", mask.UVAxisX, mask.UVAxisY)
	}
}

// A quarter turn swaps the directions the emblem's edges run in.
func TestMaskForARotatedEmblem(t *testing.T) {
	target := rl.Rectangle{X: 100 + 384, Y: 50 + 256, Width: 200, Height: 100}
	origin := rl.Vector2{X: 100, Y: 50}

	mask := maskFor(rl.Texture2D{}, color.RGBA{}, canvas, target, origin, 90)

	// The emblem's top edge now runs down the flag, its left edge leftwards.
	if !nearly(mask.UVAxisX[0], 0) || !nearly(mask.UVAxisX[1], 200.0/512) {
		t.Errorf("UV axis along the top edge = %v, want straight down", mask.UVAxisX)
	}

	if !nearly(mask.UVAxisY[0], -100.0/768) || !nearly(mask.UVAxisY[1], 0) {
		t.Errorf("UV axis along the left edge = %v, want straight left", mask.UVAxisY)
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
