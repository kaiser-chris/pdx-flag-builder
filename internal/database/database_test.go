package database

import (
	"os"
	"path/filepath"
	"testing"
)

// write creates a file and every folder leading to it.
func write(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// victoriaFolder builds the folder layout Victoria 3 uses.
func victoriaFolder(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	write(t, filepath.Join(root, "common", "named_colors", "00_colors.txt"), `
		colors = {
			red = rgb { 255 0 0 }
		}
	`)

	write(t, filepath.Join(root, "common", "coat_of_arms", "coat_of_arms", "01_flags.txt"), `
		ABS = {
			pattern = "pattern_solid.tga"
			color1 = "red"
			colored_emblem = { texture = "ce_solid.dds" }
		}
		ZZZ = {
			pattern = "pattern_solid.tga"
		}
	`)

	write(t, filepath.Join(root, "gfx", "coat_of_arms", "patterns", "pattern_solid.tga"), "")
	write(t, filepath.Join(root, "gfx", "coat_of_arms", "colored_emblems", "ce_solid.dds"), "")
	write(t, filepath.Join(root, "gfx", "coat_of_arms", "colored_emblems", "notes.txt"), "")

	return root
}

func TestDetect(t *testing.T) {
	victoria := victoriaFolder(t)
	if got := Detect(victoria); got != GameVictoria3 {
		t.Errorf("Detect(victoria) = %v, want Victoria 3", got)
	}

	europa := t.TempDir()
	write(t, filepath.Join(europa, "main_menu", "common", "named_colors", "colors.txt"), "colors = {}")
	if got := Detect(europa); got != GameEuropaUniversalis5 {
		t.Errorf("Detect(europa) = %v, want Europa Universalis 5", got)
	}

	if got := Detect(t.TempDir()); got != GameUnknown {
		t.Errorf("Detect(empty) = %v, want Unknown", got)
	}
}

func TestLoad(t *testing.T) {
	database, err := Load(Folder{Name: "game", Path: victoriaFolder(t)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(database.Problems) != 0 {
		t.Errorf("unexpected problems: %v", database.Problems)
	}

	if len(database.Flags) != 2 {
		t.Fatalf("got %d flags, want 2", len(database.Flags))
	}

	// Flags come back sorted, and each one knows where it came from.
	if database.Flags[0].Name != "ABS" || database.Flags[1].Name != "ZZZ" {
		t.Errorf("flags = %q %q, want ABS ZZZ", database.Flags[0].Name, database.Flags[1].Name)
	}
	if database.Flags[0].Origin.Database != "game" || database.Flags[0].Origin.File != "01_flags.txt" {
		t.Errorf("origin = %+v", database.Flags[0].Origin)
	}

	if _, ok := database.Palette["red"]; !ok {
		t.Error("the named colour red is missing")
	}

	// Only image files count as textures.
	if len(database.Textures) != 2 {
		t.Fatalf("got %d textures, want 2: %+v", len(database.Textures), database.Textures)
	}

	pattern, ok := findTexture(database.Textures, "pattern_solid.tga")
	if !ok || pattern.Kind != PatternTexture {
		t.Errorf("pattern_solid.tga = %+v, want a pattern", pattern)
	}

	emblem, ok := findTexture(database.Textures, "ce_solid.dds")
	if !ok || emblem.Kind != ColoredEmblemTexture {
		t.Errorf("ce_solid.dds = %+v, want a coloured emblem", emblem)
	}
}

func TestLoadMissingFolder(t *testing.T) {
	if _, err := Load(Folder{Name: "gone", Path: filepath.Join(t.TempDir(), "nope")}); err == nil {
		t.Error("loading a folder that does not exist succeeded")
	}
}

func TestLoadReportsBrokenFile(t *testing.T) {
	root := victoriaFolder(t)
	write(t, filepath.Join(root, "common", "coat_of_arms", "coat_of_arms", "02_broken.txt"), `BAD = { pattern =`)

	database, err := Load(Folder{Name: "game", Path: root})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(database.Problems) == 0 {
		t.Error("a broken file was not reported")
	}

	// One bad file must not cost the flags in the good ones.
	if len(database.Flags) != 2 {
		t.Errorf("got %d flags, want the 2 from the readable file", len(database.Flags))
	}
}

func TestSetPrecedence(t *testing.T) {
	game := victoriaFolder(t)

	mod := t.TempDir()
	write(t, filepath.Join(mod, "common", "coat_of_arms", "coat_of_arms", "01_flags.txt"), `
		ABS = { pattern = "pattern_overridden.tga" }
	`)

	set, problems := LoadAll([]Folder{{Name: "game", Path: game}, {Name: "mod", Path: mod}})
	if len(problems) != 0 {
		t.Errorf("unexpected problems: %v", problems)
	}

	if len(set) != 2 {
		t.Fatalf("got %d databases, want 2", len(set))
	}

	// A folder configured later wins, which is how a mod overrides the game.
	flag, ok := set.Flag("ABS")
	if !ok {
		t.Fatal("ABS is missing")
	}
	if flag.Pattern != "pattern_overridden.tga" {
		t.Errorf("pattern = %q, want the one from the mod", flag.Pattern)
	}

	if set.FlagCount() != 3 {
		t.Errorf("FlagCount = %d, want 3", set.FlagCount())
	}

	if _, ok := set.Texture("ce_solid.dds"); !ok {
		t.Error("a texture from the game folder is missing")
	}

	if _, ok := set.Palette()["red"]; !ok {
		t.Error("the merged palette lost the game colours")
	}
}

func findTexture(textures []Texture, name string) (Texture, bool) {
	for _, texture := range textures {
		if texture.Name == name {
			return texture, true
		}
	}

	return Texture{}, false
}
