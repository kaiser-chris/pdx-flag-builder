package pdx

import (
	"image/color"
	"slices"
	"testing"
)

func TestMoveItem(t *testing.T) {
	cases := []struct {
		name      string
		index     int
		delta     int
		want      []string
		wantIndex int
		wantMoved bool
	}{
		{"up", 2, -1, []string{"a", "c", "b", "d"}, 1, true},
		{"down", 1, 1, []string{"a", "c", "b", "d"}, 2, true},
		{"to the top", 3, -3, []string{"d", "a", "b", "c"}, 0, true},
		{"to the bottom", 0, 3, []string{"b", "c", "d", "a"}, 3, true},
		{"past the top", 0, -1, []string{"a", "b", "c", "d"}, 0, false},
		{"past the bottom", 3, 1, []string{"a", "b", "c", "d"}, 3, false},
		{"far past the end", 1, 10, []string{"a", "c", "d", "b"}, 3, true},
		{"out of range", 7, -1, []string{"a", "b", "c", "d"}, 7, false},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			items := []string{"a", "b", "c", "d"}

			index, moved := MoveItem(items, test.index, test.delta)

			if !slices.Equal(items, test.want) || index != test.wantIndex || moved != test.wantMoved {
				t.Errorf("MoveItem(%d, %d) = %v at %d moved %v, want %v at %d moved %v",
					test.index, test.delta, items, index, moved, test.want, test.wantIndex, test.wantMoved)
			}
		})
	}
}

func TestRemoveItem(t *testing.T) {
	if got := RemoveItem([]int{1, 2, 3}, 1); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("RemoveItem = %v, want [1 3]", got)
	}

	if got := RemoveItem([]int{1, 2, 3}, 3); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("RemoveItem out of range = %v, want the slice unchanged", got)
	}
}

func TestColorsRemove(t *testing.T) {
	colors := Colors{{Slot: "color1"}, {Slot: "color2"}, {Slot: "color3"}}

	if !colors.Remove("color2") {
		t.Fatal("removing a filled slot reported nothing removed")
	}

	if len(colors) != 2 || colors[0].Slot != "color1" || colors[1].Slot != "color3" {
		t.Errorf("colours = %+v, want color1 and color3 in order", colors)
	}

	if colors.Remove("color9") {
		t.Error("removing an empty slot reported something removed")
	}
}

func TestNewInstanceIsTheImpliedPlacement(t *testing.T) {
	// An emblem without placements is drawn once at the defaults. The editor
	// adds a first placement with NewInstance, which must be that same one so
	// that adding it does not make the emblem jump.
	if implied := Placements(nil); len(implied) != 1 || implied[0] != NewInstance() {
		t.Errorf("implied placement %v differs from a new instance %v", implied, NewInstance())
	}

	if implied := SubPlacements(nil); len(implied) != 1 || implied[0] != NewSubInstance() {
		t.Errorf("implied sub placement %v differs from a new one %v", implied, NewSubInstance())
	}
}

func TestNewColoredEmblemTakesTheFlagColours(t *testing.T) {
	flag := Flag{Colors: Colors{
		{Slot: "color1", Value: RGBColor{R: 10}},
		{Slot: "color2", Value: RGBColor{G: 20}},
	}}

	emblem := NewColoredEmblem("ce_solid.dds")

	first, _ := emblem.Colors.Get("color1")
	resolved, ok := first.Resolve(Palette{}, flag.Colors)

	if !ok || resolved.R != 10 {
		t.Errorf("a new emblem's first colour resolved to %v, want the flag's first colour", resolved)
	}
}

func TestCloneDoesNotShareLayers(t *testing.T) {
	emblem := NewColoredEmblem("ce_solid.dds")
	emblem.Instances = append(emblem.Instances, NewInstance())
	original := Flag{Layers: []Layer{emblem}}

	clone := original.Clone()

	edited := clone.Layers[0].(*ColoredEmblem)
	edited.Mask = 2
	edited.Instances[0].Rotation = 45
	edited.Colors.Remove("color1")

	source := original.Layers[0].(*ColoredEmblem)

	if source.Mask != 0 || source.Instances[0].Rotation != 0 || len(source.Colors) != 2 {
		t.Errorf("editing the clone changed the original: %+v", source)
	}
}

func TestRGBToHSVRoundTrip(t *testing.T) {
	for _, value := range []color.RGBA{
		{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255},
		{R: 200, G: 30, B: 90, A: 255}, {R: 12, G: 140, B: 60, A: 255},
		{A: 255}, {R: 255, G: 255, B: 255, A: 255}, {R: 128, G: 128, B: 128, A: 255},
	} {
		hue, saturation, brightness := RGBToHSV(value)
		back := HSVToRGB(hue, saturation, brightness)

		if back != value {
			t.Errorf("%v went to hsv %v %v %v and came back as %v", value, hue, saturation, brightness, back)
		}
	}
}

func TestPaletteNearest(t *testing.T) {
	palette := Palette{
		"red":   {R: 200, A: 255},
		"blue":  {B: 200, A: 255},
		"white": {R: 250, G: 250, B: 250, A: 255},
	}

	if name, _ := palette.Nearest(color.RGBA{R: 180, G: 20, A: 255}); name != "red" {
		t.Errorf("nearest to a dark red = %q, want red", name)
	}

	if _, ok := (Palette{}).Nearest(color.RGBA{}); ok {
		t.Error("an empty palette found a nearest colour")
	}
}
