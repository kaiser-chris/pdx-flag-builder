package mesh_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/mesh"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/mesh/meshtest"
)

// A mesh holds its geometry as one array per attribute, which is what a
// reader has to hand back.
func TestReadAMesh(t *testing.T) {
	read, err := mesh.Read(meshtest.Quad(9, 6))
	if err != nil {
		t.Fatalf("read the mesh: %v", err)
	}

	if read.Name != "shape" {
		t.Errorf("shape = %q, want the one the file names", read.Name)
	}

	if read.Vertices() != 4 || read.Triangles() != 2 {
		t.Errorf("%d vertices and %d triangles, want 4 and 2", read.Vertices(), read.Triangles())
	}

	// Every attribute the shader draws with is there, at the right count.
	if len(read.Normals) != read.Vertices()*3 || len(read.Tangents) != read.Vertices()*4 {
		t.Errorf("%d normal and %d tangent values for %d vertices", len(read.Normals), len(read.Tangents), read.Vertices())
	}

	if len(read.UV0) != read.Vertices()*2 || len(read.UV1) != read.Vertices()*2 {
		t.Errorf("%d and %d texture coordinates for %d vertices", len(read.UV0), len(read.UV1), read.Vertices())
	}

	low, high := read.Bounds()
	if high[0]-low[0] != 9 || high[1]-low[1] != 6 {
		t.Errorf("the mesh is %v to %v, want nine by six", low, high)
	}
}

// TestReadTheInstalledFlagCloth reads the cloth of a real installation, which
// is the file this reader is written for. It is skipped unless PDX_GAME_DIR
// points at a game folder.
func TestReadTheInstalledFlagCloth(t *testing.T) {
	game := os.Getenv("PDX_GAME_DIR")
	if game == "" {
		t.Skip("set PDX_GAME_DIR to the game folder to run this test")
	}

	data, err := os.ReadFile(filepath.Join(game, "gfx", "models", "ui", "flags", "ui_flag_01.mesh"))
	if err != nil {
		t.Skipf("the game has no flag cloth to read: %v", err)
	}

	cloth, err := mesh.Read(data)
	if err != nil {
		t.Fatalf("read the cloth: %v", err)
	}

	if cloth.Name != "flag_01Shape" || cloth.Vertices() != 651 || cloth.Triangles() != 1200 {
		t.Errorf("the cloth is %s with %d vertices and %d triangles, want flag_01Shape with 651 and 1200",
			cloth.Name, cloth.Vertices(), cloth.Triangles())
	}

	// The game's shader works the wave out from the position divided by nine
	// across and six down, so the cloth is about that big.
	low, high := cloth.Bounds()

	if high[0]-low[0] < 8 || high[0]-low[0] > 9 || high[1]-low[1] < 6 || high[1]-low[1] > 7 {
		t.Errorf("the cloth is %v to %v, want about nine by six", low, high)
	}
}

func TestReadRefusesWhatIsNotAMesh(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":        {},
		"another file": []byte("PNG\r\n"),
		"cut short":    append([]byte("@@b@"), '!', 1, 'p', 'f', 4, 0, 0, 0),
	} {
		if _, err := mesh.Read(data); err == nil {
			t.Errorf("reading %s succeeded", name)
		}
	}
}
