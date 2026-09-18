package pdx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx/script"
)

func TestScriptLayout(t *testing.T) {
	flag := Flag{
		Name:    "TST",
		Pattern: "pattern_solid.tga",
		Colors: Colors{
			{Slot: "color2", Value: RGBColor{R: 10, G: 20, B: 30}},
			{Slot: "color1", Value: NamedColor{Name: "red"}},
		},
		Layers: []Layer{
			&ColoredEmblem{
				Texture: "ce_star.dds",
				Colors: Colors{
					{Slot: "color1", Value: SlotColor{Slot: "color2"}},
					{Slot: "color2", Value: HSVColor{H: 120, S: 0.5, V: 0.25}},
				},
				Mask: 2,
				Instances: []Instance{
					{Position: Vec2{X: 0.25, Y: 0.5}, Scale: Vec2{X: 0.5, Y: 0.5}},
					{Position: Vec2{X: 0.75, Y: 0.5}, Scale: Vec2{X: -0.5, Y: 0.5}, Rotation: 45},
				},
			},
			&TexturedEmblem{Texture: "te_crest.dds"},
			&SubFlag{
				Parent:    "GBR",
				Instances: []SubInstance{{Offset: Vec2{X: 0.5}, Scale: Vec2{X: 0.5, Y: 0.5}}},
			},
		},
	}

	want := strings.Join([]string{
		`TST = {`,
		`	pattern = "pattern_solid.tga"`,
		`	color1 = "red"`,
		`	color2 = rgb { 10 20 30 }`,
		``,
		`	colored_emblem = {`,
		`		texture = "ce_star.dds"`,
		`		color1 = color2`,
		`		color2 = hsv360 { 120 50 25 }`,
		`		mask = { 2 }`,
		`		instance = {`,
		`			position = { 0.25 0.5 }`,
		`			scale = { 0.5 0.5 }`,
		`		}`,
		`		instance = {`,
		`			position = { 0.75 0.5 }`,
		`			scale = { -0.5 0.5 }`,
		`			rotation = 45`,
		`		}`,
		`	}`,
		``,
		`	textured_emblem = {`,
		`		texture = "te_crest.dds"`,
		`	}`,
		``,
		`	sub = {`,
		`		parent = "GBR"`,
		`		instance = {`,
		`			offset = { 0.5 0 }`,
		`			scale = { 0.5 0.5 }`,
		`		}`,
		`	}`,
		`}`,
	}, "\r\n")

	if got := Script(flag, "\r\n"); got != want {
		t.Errorf("Script =\n%s\nwant\n%s", got, want)
	}
}

func TestNumbersAreWrittenShort(t *testing.T) {
	tests := map[float32]string{
		0.5:         "0.5",
		1:           "1",
		-0.25:       "-0.25",
		1.0 / 3:     "0.3333",
		-0.00001:    "0",
		359.99999:   "360",
		0.123456789: "0.1235",
	}

	for value, want := range tests {
		if got := number(value); got != want {
			t.Errorf("number(%v) = %q, want %q", value, got, want)
		}
	}
}

// TestScriptRoundTrip reads what Script wrote and expects the same coat of
// arms back, for every way a colour and a layer can be written.
func TestScriptRoundTrip(t *testing.T) {
	const source = `
@third = @[1/3]
TST = {
	pattern = "pattern_split.tga"
	color1 = "blue"
	color2 = rgb { 0.5 0.25 1 }
	color3 = hsv { 0.5 0.5 0.5 }
	color4 = hsv360 { 90 20 80 }
	color5 = list "normal_colors"
	color6 = color1
	colored_emblem = {
		texture = "ce_a.dds"
		color1 = color2
		mask = { 1 }
		instance = { scale = { @third @third } position = { @third 0.5 } rotation = 90 }
		instance = { scale = { 0.07 } }
	}
	textured_emblem = { texture = "te_b.dds" instance = { position = { 0.1 0.2 } } }
	sub = { parent = "OTHER" instance = { offset = { 0.5 0 } scale = { 0.5 0.5 } } }
	sub = { parent = "EMPTY" }
}`

	original := decodeOne(t, source)
	written := Script(original, "\n")
	reread := decodeOne(t, written)

	// Writing is lossless apart from rounding, so writing what was read back
	// in comes out the same, character for character.
	if again := Script(reread, "\n"); again != written {
		t.Fatalf("writing the reread flag gave\n%s\nthe first time it was\n%s", again, written)
	}

	if len(reread.Layers) != len(original.Layers) || len(reread.Colors) != len(original.Colors) {
		t.Fatalf("reread %d layers and %d colours, want %d and %d",
			len(reread.Layers), len(reread.Colors), len(original.Layers), len(original.Colors))
	}

	for index, color := range original.Colors {
		if got := reread.Colors[index]; got.Slot != color.Slot || kindName(got.Value) != kindName(color.Value) {
			t.Errorf("colour %d reread as %#v, want %#v", index, got, color)
		}
	}

	emblem := reread.Layers[0].(*ColoredEmblem)
	if emblem.Mask != 1 || len(emblem.Instances) != 2 || emblem.Instances[0].Rotation != 90 {
		t.Errorf("colored emblem reread as %#v", emblem)
	}

	// The single value scale of the file means a full height bar; written out,
	// both values say so.
	if got := emblem.Instances[1].Scale; got != (Vec2{X: 0.07, Y: 1}) {
		t.Errorf("single value scale reread as %v, want {0.07 1}", got)
	}
}

// TestScriptRoundTripInstalledGame writes every coat of arms of a real
// installation and reads it back. It is skipped unless PDX_GAME_DIR points at
// a game or mod folder.
func TestScriptRoundTripInstalledGame(t *testing.T) {
	root := os.Getenv("PDX_GAME_DIR")
	if root == "" {
		t.Skip("set PDX_GAME_DIR to a game or mod folder to run this test")
	}

	files, _ := filepath.Glob(filepath.Join(root, "common", "coat_of_arms", "coat_of_arms", "*.txt"))
	count := 0

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		document, err := script.Parse(string(data))
		if err != nil {
			t.Errorf("%s: %v", path, err)

			continue
		}

		flags, _ := DecodeFlags(document, Origin{})

		for _, flag := range flags {
			written := Script(flag, "\n")
			reread := decodeOne(t, written)

			if again := Script(reread, "\n"); again != written {
				t.Errorf("%s changed when written a second time:\n%s\nthen\n%s", flag.Name, written, again)
			}

			if len(reread.Layers) != len(flag.Layers) {
				t.Errorf("%s: reread %d layers, want %d", flag.Name, len(reread.Layers), len(flag.Layers))
			}

			count++
		}
	}

	t.Logf("wrote and reread %d coats of arms", count)
}

func decodeOne(t *testing.T, source string) Flag {
	t.Helper()

	document, err := script.Parse(source)
	if err != nil {
		t.Fatalf("parse:\n%s\n%v", source, err)
	}

	flags, issues := DecodeFlags(document, Origin{})
	if len(issues) > 0 {
		t.Fatalf("decode issues: %v", issues)
	}

	if len(flags) != 1 {
		t.Fatalf("decoded %d flags, want 1", len(flags))
	}

	return flags[0]
}

func kindName(value ColorValue) string {
	switch value.(type) {
	case NamedColor:
		return "named"
	case SlotColor:
		return "slot"
	case RGBColor:
		return "rgb"
	case HSVColor:
		return "hsv"
	case ListColor:
		return "list"
	}

	return "?"
}
