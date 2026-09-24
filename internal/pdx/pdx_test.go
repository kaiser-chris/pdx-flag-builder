package pdx

import (
	"testing"

	"github.com/kaiser-chris/pdx-parser-go/database"
	"github.com/kaiser-chris/pdx-parser-go/folders"
	"github.com/kaiser-chris/pdx-parser-go/report"
	"github.com/kaiser-chris/pdx-parser-go/script"
	"github.com/kaiser-chris/pdx-parser-go/victoria3"
)

// readFlags reads the coats of arms of a file the way the database does, for
// the tests that write a flag out and read it back.
func readFlags(content string) []Flag {
	arms, _ := victoria3.ReadCoatOfArms(script.Parse(content), folders.File{}, &report.Collector{})

	var flags []Flag

	for key, definition := range arms.All() {
		flags = append(flags, FromCoatOfArms(definition, Origin{Key: key}))
	}

	return flags
}

// The parser library reads the files; what is tested here is the step from
// what it read to the model the editor works on. Reading itself, from the
// script language to colours and instances, is the library's own business and
// is tested there and, from end to end, in internal/database.

func TestFromCoatOfArms(t *testing.T) {
	arms := &victoria3.CoatOfArms{
		Definition: database.Definition{Key: "ABS"},
		Pattern:    victoria3.Texture{File: "pattern_solid.tga"},
		Colors: victoria3.Colors{
			{Slot: "color1", Value: victoria3.NamedColor{Name: "red"}},
			{Slot: "color2", Value: victoria3.SlotColor{Slot: "color1"}},
		},
		Layers: []victoria3.Layer{
			&victoria3.TexturedEmblem{Texture: victoria3.Texture{File: "te_crow.dds"}},
			&victoria3.ColoredEmblem{
				Texture: victoria3.Texture{File: "ce_solid.dds"},
				Colors:  victoria3.Colors{{Slot: "color1", Value: victoria3.NamedColor{Name: "yellow"}}},
				Masks:   []int{2},
				Instances: []victoria3.Instance{
					{Position: victoria3.Vec2{X: 0.5, Y: 0.9}, Scale: victoria3.Vec2{X: 1, Y: 0.12}},
					{Position: victoria3.Vec2{X: 0.5, Y: 0.1}, Scale: victoria3.Vec2{X: 1, Y: 0.12}, Rotation: 90},
				},
			},
			&victoria3.SubCoatOfArms{
				Parent:    "sub_ENG_coa",
				Instances: []victoria3.Instance{{Offset: victoria3.Vec2{X: 0.25}, Scale: victoria3.Vec2{X: 0.5, Y: 0.5}}},
			},
		},
	}

	origin := Origin{Database: "game", File: "01_flags.txt", Path: "game/01_flags.txt", Key: "ABS", Line: 12}

	flag := FromCoatOfArms(arms, origin)

	if flag.Name != "ABS" || flag.Pattern != "pattern_solid.tga" || flag.Origin != origin {
		t.Fatalf("flag = %+v", flag)
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
	if !ok || textured.Texture != "te_crow.dds" {
		t.Errorf("layer 0 = %#v, want the textured emblem", flag.Layers[0])
	}

	colored, ok := flag.Layers[1].(*ColoredEmblem)
	if !ok {
		t.Fatalf("layer 1 is %T, want a ColoredEmblem", flag.Layers[1])
	}

	// The files name the pattern colours an emblem is restricted to as a
	// list; the editor offers the one the files use.
	if colored.Mask != 2 {
		t.Errorf("mask = %d, want 2", colored.Mask)
	}

	if len(colored.Instances) != 2 || colored.Instances[1].Rotation != 90 {
		t.Fatalf("instances = %+v", colored.Instances)
	}

	if got := colored.Instances[0].Scale; got != (Vec2{X: 1, Y: 0.12}) {
		t.Errorf("scale = %+v, want {1 0.12}", got)
	}

	sub, ok := flag.Layers[2].(*SubFlag)
	if !ok || sub.Parent != "sub_ENG_coa" {
		t.Fatalf("layer 2 = %#v, want the sub flag", flag.Layers[2])
	}

	if got := sub.Instances[0]; got.Offset != (Vec2{X: 0.25}) || got.Scale != (Vec2{X: 0.5, Y: 0.5}) {
		t.Errorf("sub instance = %+v, want the right half", got)
	}
}

func TestFromCoatOfArmsKeepsNothingItWasNotGiven(t *testing.T) {
	arms := &victoria3.CoatOfArms{
		Definition: database.Definition{Key: "RND"},

		// A pattern picked at random is not a file the editor can draw.
		Pattern: victoria3.Texture{List: "pattern_anarchy"},
		Layers:  []victoria3.Layer{&victoria3.ColoredEmblem{Texture: victoria3.Texture{File: "ce_solid.dds"}}},
	}

	flag := FromCoatOfArms(arms, Origin{})

	if flag.Pattern != "" {
		t.Errorf("pattern = %q, want none for a random pick", flag.Pattern)
	}

	emblem := flag.Layers[0].(*ColoredEmblem)

	// Nothing is invented, so that writing the flag back out does not add an
	// instance the author never wrote.
	if len(emblem.Instances) != 0 {
		t.Errorf("instances = %+v, want none kept", emblem.Instances)
	}

	placements := Placements(emblem.Instances)
	if len(placements) != 1 || placements[0].Position != DefaultPosition || placements[0].Scale != DefaultScale {
		t.Errorf("placements = %+v, want the one default placement", placements)
	}
}

func TestFromCoatOfArmsCopiesTheColours(t *testing.T) {
	arms := &victoria3.CoatOfArms{
		Definition: database.Definition{Key: "ABS"},
		Colors:     victoria3.Colors{{Slot: "color1", Value: victoria3.NamedColor{Name: "red"}}},
	}

	flag := FromCoatOfArms(arms, Origin{})
	flag.Colors.Set(Color{Slot: "color1", Value: NamedColor{Name: "blue"}})

	if first, _ := arms.Colors.Get("color1"); first.Value.Describe() != "red" {
		t.Errorf("editing the flag changed what was read: %s", first.Value.Describe())
	}
}
