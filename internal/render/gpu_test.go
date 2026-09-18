//go:build uitest

package render

import (
	"runtime"
	"slices"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// withOpenGL opens a hidden window for a test that needs an OpenGL context.
// The context belongs to the thread that created it, and a test runs on a
// goroutine of its own, so each test opens one and stays on its thread.
func withOpenGL(t *testing.T) {
	t.Helper()

	runtime.LockOSThread()

	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowHidden)
	rl.InitWindow(64, 64, "render tests")

	t.Cleanup(func() {
		rl.CloseWindow()
		runtime.UnlockOSThread()
	})
}

// put stands in for a texture that the reader has delivered.
func put(textures *Textures, name string) {
	image := rl.GenImageColor(4, 4, rl.Red)
	defer rl.UnloadImage(image)

	textures.loaded[name] = rl.LoadTextureFromImage(image)
	textures.used[name] = textures.generation
}

func loadedNames(textures *Textures) []string {
	var names []string
	for name := range textures.loaded {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

func TestTrimUnloadsWhatWentUnusedLongest(t *testing.T) {
	withOpenGL(t)

	textures := NewTextures(func(string) (string, bool) { return "", false })
	t.Cleanup(textures.Unload)

	// Three textures, each wanted last in a different round.
	put(textures, "oldest")
	textures.Trim(10)
	put(textures, "middle")
	textures.Trim(10)
	put(textures, "newest")
	textures.Trim(10)

	textures.Trim(2)

	if got := loadedNames(textures); !slices.Equal(got, []string{"middle", "newest"}) {
		t.Errorf("after trimming to two: %v, want the two used most recently", got)
	}
}

func TestTrimKeepsWhatIsInUse(t *testing.T) {
	withOpenGL(t)

	textures := NewTextures(func(string) (string, bool) { return "", false })
	t.Cleanup(textures.Unload)

	put(textures, "first")
	put(textures, "second")
	textures.Trim(10)

	// Asked for since the last trim, which a thumbnail waiting for it does.
	textures.Get("first")
	textures.Get("second")

	textures.Trim(0)

	if got := loadedNames(textures); len(got) != 2 {
		t.Errorf("after trimming to nothing: %v, want both kept while they are wanted", got)
	}

	textures.Trim(0)

	if got := loadedNames(textures); len(got) != 0 {
		t.Errorf("after a round without them: %v, want both unloaded", got)
	}
}

func newTestThumbnails(t *testing.T) *Thumbnails {
	t.Helper()

	withOpenGL(t)

	shader, err := LoadRecolor()
	if err != nil {
		t.Fatalf("LoadRecolor: %v", err)
	}
	t.Cleanup(shader.Unload)

	thumbnails := NewThumbnails(shader, func(string) (string, bool) { return "", false })
	t.Cleanup(thumbnails.Unload)

	return thumbnails
}

func TestThumbnailsReuseTheCellShownLeastRecently(t *testing.T) {
	thumbnails := newTestThumbnails(t)

	// A full atlas.
	thumbnails.free = nil
	thumbnails.cells = map[string]*thumbnailCell{
		"long ago":   {index: 3, shown: 2},
		"a while":    {index: 5, shown: 6},
		"just shown": {index: 7, shown: 9},
	}
	thumbnails.frame = 10

	index, ok := thumbnails.allocate()
	if !ok || index != 3 {
		t.Fatalf("allocate = %d, %v, want the cell shown longest ago, 3", index, ok)
	}

	if _, kept := thumbnails.cells["long ago"]; kept {
		t.Error("the reused cell still belongs to its old thumbnail")
	}

	// What was on screen in the last frame is never taken.
	thumbnails.cells = map[string]*thumbnailCell{"just shown": {index: 7, shown: 9}}

	if _, ok := thumbnails.allocate(); ok {
		t.Error("allocate took a cell that is on screen")
	}
}

func TestForgetFreesEveryCell(t *testing.T) {
	thumbnails := newTestThumbnails(t)
	capacity := len(thumbnails.free)

	for range 3 {
		index, _ := thumbnails.allocate()
		thumbnails.cells[string(rune('a'+index))] = &thumbnailCell{index: index}
	}

	thumbnails.Forget()

	if len(thumbnails.free) != capacity || len(thumbnails.cells) != 0 {
		t.Errorf("after Forget: %d free cells and %d in use, want all %d free", len(thumbnails.free), len(thumbnails.cells), capacity)
	}
}

func TestThumbnailCellsDoNotOverlap(t *testing.T) {
	thumbnails := newTestThumbnails(t)
	capacity := len(thumbnails.free)

	if capacity < 300 {
		t.Errorf("the atlas holds %d thumbnails, want room for a screenful of rows several times over", capacity)
	}

	previous := thumbnails.cellRect(0)

	for index := 1; index < capacity; index++ {
		cell := thumbnails.cellRect(index)

		if cell.X+cell.Width > atlasSize || cell.Y+cell.Height > atlasSize {
			t.Fatalf("cell %d at %+v reaches past the atlas", index, cell)
		}

		// Cells are laid out in rows; one on the same row starts past the last.
		if cell.Y == previous.Y && cell.X < previous.X+previous.Width+thumbnailGutter {
			t.Fatalf("cell %d at %+v overlaps or touches cell %d at %+v", index, cell, index-1, previous)
		}

		previous = cell
	}
}
