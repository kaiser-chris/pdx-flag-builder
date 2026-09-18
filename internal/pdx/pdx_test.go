package pdx

import (
	"image/color"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx/script"
)

func decode(t *testing.T, input string) ([]Flag, []Issue) {
	t.Helper()

	document, err := script.Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	return DecodeFlags(document, Origin{Database: "game", File: "test.txt"})
}

func TestDecodeFlag(t *testing.T) {
	flags, issues := decode(t, `
		ABS = {
			pattern = "pattern_solid.tga"
			color1 = "blue_light"
			color2 = "yellow"

			textured_emblem = {
				texture = "te_crow_star.dds"
				instance = { scale = { 0.70 0.70 } position = { 0.5 0.5 } }
			}
			colored_emblem = {
				texture = "ce_solid.dds"
				color1 = "red"
				color2 = color1
				mask = { 2 }
				instance = { position = { 0.5 0.9 } scale = { 1.0 0.12 } }
				instance = { position = { 0.5 0.1 } scale = { 1.0 0.12 } rotation = 90 }
			}
			sub = {
				parent = "sub_ENG_coa"
				instance = { scale = { 0.5 0.5 } offset = { 0.25 0 } }
			}
		}
	`)

	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}

	if len(flags) != 1 {
		t.Fatalf("got %d flags, want 1", len(flags))
	}

	flag := flags[0]

	if flag.Name != "ABS" {
		t.Errorf("name = %q, want ABS", flag.Name)
	}
	if flag.Pattern != "pattern_solid.tga" {
		t.Errorf("pattern = %q", flag.Pattern)
	}
	if flag.Origin.Database != "game" {
		t.Errorf("origin database = %q, want game", flag.Origin.Database)
	}

	if len(flag.Colors) != 2 {
		t.Fatalf("got %d colours, want 2", len(flag.Colors))
	}

	// Layers keep the order they were written in, because that is the order
	// they are drawn in.
	if len(flag.Layers) != 3 {
		t.Fatalf("got %d layers, want 3", len(flag.Layers))
	}

	textured, ok := flag.Layers[0].(*TexturedEmblem)
	if !ok {
		t.Fatalf("layer 0 is %T, want a TexturedEmblem", flag.Layers[0])
	}
	if textured.Texture != "te_crow_star.dds" {
		t.Errorf("textured emblem texture = %q", textured.Texture)
	}

	colored, ok := flag.Layers[1].(*ColoredEmblem)
	if !ok {
		t.Fatalf("layer 1 is %T, want a ColoredEmblem", flag.Layers[1])
	}
	if colored.Mask != 2 {
		t.Errorf("mask = %d, want 2", colored.Mask)
	}
	if len(colored.Instances) != 2 {
		t.Fatalf("got %d instances, want 2", len(colored.Instances))
	}
	if colored.Instances[1].Rotation != 90 {
		t.Errorf("rotation = %v, want 90", colored.Instances[1].Rotation)
	}
	if got := colored.Instances[0].Scale; got.X != 1 || got.Y != 0.12 {
		t.Errorf("scale = %+v, want {1 0.12}", got)
	}

	// An emblem slot pointing at a slot name is a reference, not a colour name.
	second, _ := colored.Colors.Get("color2")
	if _, ok := second.Value.(SlotColor); !ok {
		t.Errorf("color2 = %T, want a SlotColor", second.Value)
	}

	sub, ok := flag.Layers[2].(*SubFlag)
	if !ok {
		t.Fatalf("layer 2 is %T, want a SubFlag", flag.Layers[2])
	}
	if sub.Parent != "sub_ENG_coa" {
		t.Errorf("parent = %q", sub.Parent)
	}
	if got := sub.Instances[0].Offset; got.X != 0.25 {
		t.Errorf("offset = %+v, want an x of 0.25", got)
	}
}

func TestDecodeSkipsTemplateBlock(t *testing.T) {
	flags, _ := decode(t, `
		template = {
			template_charge = {
				pattern = "pattern_solid.tga"
				color1 = list "normal_colors"
			}
		}

		template_uruguay = {
			pattern = "pattern_solid.tga"
		}

		ABS = {
			pattern = "pattern_solid.tga"
		}
	`)

	// The root template block holds building blocks for the games' random
	// generation and is not editable, but a top level entry that merely starts
	// with the word is an ordinary coat of arms.
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		names = append(names, flag.Name)
	}

	if len(names) != 2 || names[0] != "template_uruguay" || names[1] != "ABS" {
		t.Errorf("decoded %v, want [template_uruguay ABS]", names)
	}
}

func TestDecodeInstanceDefaults(t *testing.T) {
	flags, _ := decode(t, `
		ABS = {
			colored_emblem = { texture = "ce_solid.dds" }
		}
	`)

	emblem := flags[0].Layers[0].(*ColoredEmblem)

	// Nothing is invented in the model, so that writing the flag back out does
	// not add an instance the author never wrote.
	if len(emblem.Instances) != 0 {
		t.Fatalf("got %d stored instances, want 0", len(emblem.Instances))
	}

	placements := Placements(emblem.Instances)
	if len(placements) != 1 {
		t.Fatalf("got %d placements, want 1", len(placements))
	}

	if placements[0].Position != DefaultPosition || placements[0].Scale != DefaultScale {
		t.Errorf("placement = %+v, want the defaults", placements[0])
	}
}

func TestDecodeColorForms(t *testing.T) {
	flags, issues := decode(t, `
		ABS = {
			color1 = "red"
			color2 = rgb { 1 0.4 0.6 }
			color3 = rgb { 255 0 0 }
			color4 = hsv360 { 230 80 30 }
			color5 = hsv { 0.5 0.8 0.3 }
			color6 = { 0 255 0 }
			color7 = list "normal_colors"
		}
	`)

	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}

	colors := flags[0].Colors

	named, _ := colors.Get("color1")
	if value, ok := named.Value.(NamedColor); !ok || value.Name != "red" {
		t.Errorf("color1 = %#v, want the named colour red", named.Value)
	}

	// Fractions and whole channels are told apart by their magnitude, which is
	// what the games do.
	fraction, _ := colors.Get("color2")
	if value, ok := fraction.Value.(RGBColor); !ok || value.R != 255 || value.G != 102 || value.B != 153 {
		t.Errorf("color2 = %#v, want rgb 255 102 153", fraction.Value)
	}

	whole, _ := colors.Get("color3")
	if value, ok := whole.Value.(RGBColor); !ok || value.R != 255 || value.G != 0 {
		t.Errorf("color3 = %#v, want rgb 255 0 0", whole.Value)
	}

	degrees, _ := colors.Get("color4")
	if value, ok := degrees.Value.(HSVColor); !ok || value.H != 230 || value.S != 0.8 || value.V != 0.3 {
		t.Errorf("color4 = %#v, want hue 230 with 0.8 and 0.3", degrees.Value)
	}

	fractions, _ := colors.Get("color5")
	if value, ok := fractions.Value.(HSVColor); !ok || value.H != 180 || value.S != 0.8 {
		t.Errorf("color5 = %#v, want hue 180 with 0.8", fractions.Value)
	}

	untagged, _ := colors.Get("color6")
	if value, ok := untagged.Value.(RGBColor); !ok || value.G != 255 {
		t.Errorf("color6 = %#v, want rgb 0 255 0", untagged.Value)
	}

	random, _ := colors.Get("color7")
	if value, ok := random.Value.(ListColor); !ok || value.List != "normal_colors" {
		t.Errorf("color7 = %#v, want a list colour", random.Value)
	}
}

func TestResolveColors(t *testing.T) {
	palette := Palette{"red": {R: 200, G: 30, B: 30, A: 255}}

	slots := Colors{
		{Slot: "color1", Value: NamedColor{Name: "red"}},
		{Slot: "color2", Value: SlotColor{Slot: "color1"}},
	}

	resolved, ok := (Color{Value: SlotColor{Slot: "color2"}}).Resolve(palette, slots)
	if !ok {
		t.Fatal("a chain of references did not resolve")
	}
	if resolved != (color.RGBA{R: 200, G: 30, B: 30, A: 255}) {
		t.Errorf("resolved to %+v", resolved)
	}

	if _, ok := (Color{Value: NamedColor{Name: "nope"}}).Resolve(palette, slots); ok {
		t.Error("an unknown colour name resolved")
	}

	// A file may point two slots at each other. Giving up beats looping.
	circular := Colors{
		{Slot: "color1", Value: SlotColor{Slot: "color2"}},
		{Slot: "color2", Value: SlotColor{Slot: "color1"}},
	}
	if _, ok := (Color{Value: SlotColor{Slot: "color1"}}).Resolve(palette, circular); ok {
		t.Error("a circular reference resolved")
	}
}

func TestDecodePalette(t *testing.T) {
	document, err := script.Parse(`
		colors = {
			black = hsv360 { 0 0 5 }
			todo_purple = rgb { 1 0.4 0.6 }
			same_as_black = black
		}
	`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	palette, issues := DecodePalette(document)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}

	black, ok := palette["black"]
	if !ok {
		t.Fatal("black is missing")
	}
	if black.R != 13 || black.G != 13 || black.B != 13 {
		t.Errorf("black = %+v, want a near black grey", black)
	}

	// A colour may be defined in terms of one that came before it.
	if palette["same_as_black"] != black {
		t.Errorf("same_as_black = %+v, want %+v", palette["same_as_black"], black)
	}
}

func TestHSVToRGB(t *testing.T) {
	cases := []struct {
		name    string
		h, s, v float32
		want    color.RGBA
	}{
		{"red", 0, 1, 1, color.RGBA{R: 255, A: 255}},
		{"green", 120, 1, 1, color.RGBA{G: 255, A: 255}},
		{"blue", 240, 1, 1, color.RGBA{B: 255, A: 255}},
		{"white", 0, 0, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"black", 0, 0, 0, color.RGBA{A: 255}},
		{"wrapped hue", 360, 1, 1, color.RGBA{R: 255, A: 255}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := HSVToRGB(test.h, test.s, test.v); got != test.want {
				t.Errorf("HSVToRGB(%v, %v, %v) = %+v, want %+v", test.h, test.s, test.v, got, test.want)
			}
		})
	}
}

func TestNextFreeSlot(t *testing.T) {
	colors := Colors{{Slot: "color1"}, {Slot: "color2"}}

	if got := colors.NextFreeSlot(); got != "color3" {
		t.Errorf("NextFreeSlot = %q, want color3", got)
	}
}

func TestDecodeSingleValueScale(t *testing.T) {
	// Sulu writes its hoist stripes with one number, meaning a narrow bar of
	// full height rather than a small square.
	flags, issues := decode(t, `
		SUL = {
			colored_emblem = {
				texture = "ce_solid.dds"
				instance = { position = { 0.035 0.5 } scale = { 0.07 } }
			}
		}
	`)

	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}

	instance := flags[0].Layers[0].(*ColoredEmblem).Instances[0]

	if instance.Scale.X != 0.07 {
		t.Errorf("scale x = %v, want 0.07", instance.Scale.X)
	}
	if instance.Scale.Y != DefaultScale.Y {
		t.Errorf("scale y = %v, want the default of %v", instance.Scale.Y, DefaultScale.Y)
	}
}
