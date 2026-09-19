package texture

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDecodeInstalledTextures runs the decoders over a real installation.
//
// Skipped unless PDX_GAME_DIR points at a game or mod folder. Set PDX_DUMP_DIR
// as well to write what was decoded out as PNG files, which is the only way to
// actually see whether a BC7 block decoder is right.
func TestDecodeInstalledTextures(t *testing.T) {
	root := os.Getenv("PDX_GAME_DIR")
	if root == "" {
		t.Skip("set PDX_GAME_DIR to a game or mod folder to run this test")
	}

	dump := os.Getenv("PDX_DUMP_DIR")
	if dump != "" {
		if err := os.MkdirAll(dump, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	var targa, bc7, dxt int

	visit := func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //nolint:nilerr // an unreadable folder is not this test's problem
		}

		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".tga" && extension != ".dds" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var pixels *Pixels

		switch {
		case extension == ".tga":
			pixels, err = DecodeTGA(data)
			targa++

		case IsBC7(data):
			pixels, err = DecodeDDSBC7(data)
			bc7++

		default:
			// DXT stays compressed for the GPU, so the check is that the
			// largest level is all there.
			if _, ok, err := ReadDDSDXT(data); ok {
				dxt++

				if err != nil {
					t.Errorf("%s: %v", filepath.Base(path), err)
				}

				return nil
			}

			// Everything else is raylib's job and is covered by the loading
			// test in the application rather than here.
			return nil
		}

		if err != nil {
			t.Errorf("%s: %v", filepath.Base(path), err)

			return nil
		}

		if pixels.Width <= 0 || pixels.Height <= 0 || len(pixels.Data) != pixels.Width*pixels.Height*4 {
			t.Errorf("%s: decoded to %dx%d with %d bytes", filepath.Base(path), pixels.Width, pixels.Height, len(pixels.Data))

			return nil
		}

		// An emblem is artwork, so a single flat colour means the decoder
		// produced nothing useful. Patterns are exempt, since a solid pattern
		// really is one colour waiting to be recoloured, and so are the solid
		// and empty emblems both games ship.
		if !meantToBeFlat(path) && uniform(pixels) {
			t.Errorf("%s: decoded to a single flat colour", filepath.Base(path))
		}

		if dump != "" {
			writePNG(t, filepath.Join(dump, filepath.Base(path)+".png"), pixels)
		}

		return nil
	}

	for _, folder := range readFolders(root) {
		if err := filepath.WalkDir(folder, visit); err != nil {
			t.Fatalf("walk %s: %v", folder, err)
		}
	}

	t.Logf("decoded %d targa and %d BC7 files, read %d DXT files", targa, bc7, dxt)

	if targa == 0 && bc7 == 0 && dxt == 0 {
		t.Skip("the folder held no files these decoders handle")
	}
}

// uniform reports whether every pixel of an image is the same.
func uniform(pixels *Pixels) bool {
	if len(pixels.Data) < 8 {
		return true
	}

	first := pixels.Data[0:4]

	for offset := 4; offset < len(pixels.Data); offset += 4 {
		if !equalPixel(pixels.Data[offset:offset+4], first) {
			return false
		}
	}

	return true
}

func equalPixel(left, right []byte) bool {
	return left[0] == right[0] && left[1] == right[1] && left[2] == right[2] && left[3] == right[3]
}

func writePNG(t *testing.T, path string, pixels *Pixels) {
	t.Helper()

	picture := image.NewNRGBA(image.Rect(0, 0, pixels.Width, pixels.Height))

	for y := range pixels.Height {
		for x := range pixels.Width {
			offset := pixels.At(x, y)
			picture.SetNRGBA(x, y, color.NRGBA{
				R: pixels.Data[offset],
				G: pixels.Data[offset+1],
				B: pixels.Data[offset+2],
				A: pixels.Data[offset+3],
			})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()

	if err := png.Encode(file, picture); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}

// readFolders narrows a game or mod folder down to where coat of arms
// textures are, since the rest of a game holds textures of other sorts, such
// as flat property maps, that the tool never reads. A folder without that one
// is taken whole, so the test can also be pointed at a folder of loose files.
func readFolders(root string) []string {
	var folders []string

	// Europa Universalis 5 keeps its artwork in three folders of its own.
	for _, base := range []string{root, filepath.Join(root, "main_menu"), filepath.Join(root, "in_game"), filepath.Join(root, "loading_screen")} {
		folder := filepath.Join(base, "gfx", "coat_of_arms")
		if info, err := os.Stat(folder); err == nil && info.IsDir() {
			folders = append(folders, folder)
		}
	}

	if len(folders) == 0 {
		return []string{root}
	}

	return folders
}

// meantToBeFlat reports whether a texture is one colour by design: every
// pattern can be, and emblems named solid or empty are.
func meantToBeFlat(path string) bool {
	name := strings.ToLower(filepath.Base(path))

	return filepath.Base(filepath.Dir(path)) == "patterns" ||
		strings.Contains(name, "solid") || strings.Contains(name, "empty")
}
