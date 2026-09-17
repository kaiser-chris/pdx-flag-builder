package database

import (
	"os"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// TestLoadInstalledGame reads a real installation.
//
// It is skipped unless PDX_GAME_DIR points at a game or mod folder. What it
// checks is deliberately loose: the point is to see that a real folder loads at
// all and to report what came out of it, not to pin down numbers that change
// with every patch.
func TestLoadInstalledGame(t *testing.T) {
	path := os.Getenv("PDX_GAME_DIR")
	if path == "" {
		t.Skip("set PDX_GAME_DIR to a game or mod folder to run this test")
	}

	database, err := Load(Folder{Name: "game", Path: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	t.Logf("game %v: %d flags, %d colours, %d textures, %d problems",
		database.Game, len(database.Flags), len(database.Palette),
		len(database.Textures), len(database.Problems))

	for index, problem := range database.Problems {
		if index >= 10 {
			t.Logf("... and %d more problems", len(database.Problems)-index)

			break
		}

		t.Logf("problem: %s", problem)
	}

	if len(database.Flags) == 0 {
		t.Error("no flags were read")
	}

	// Every layer should have come out as one of the three known kinds, and
	// every emblem should name a texture.
	missingTexture := 0

	for _, flag := range database.Flags {
		for _, layer := range flag.Layers {
			if texture, hasTexture := textureOf(layer); hasTexture && texture == "" {
				missingTexture++
			}
		}
	}

	if missingTexture > 0 {
		t.Errorf("%d emblems came out without a texture", missingTexture)
	}
}

// textureOf is pdx.Texture, restated here so the test does not depend on the
// order of the type switch in the package under test.
func textureOf(layer pdx.Layer) (string, bool) {
	return pdx.Texture(layer)
}
