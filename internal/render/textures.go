package render

import (
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/texture"
)

// uploadsPerFrame caps how many textures are handed to the GPU in one frame, so
// that opening a flag that needs a dozen of them does not cost a visible stall.
const uploadsPerFrame = 4

// Textures keeps the artwork a flag needs on the GPU.
//
// Reading a file and decoding it is slow enough to be worth keeping off the
// interface goroutine, but uploading has to happen where the OpenGL context is.
// So a request goes out to a reader goroutine, comes back as decoded pixels,
// and is uploaded from Upload, which the frame loop calls once a frame.
//
// A texture that has not arrived yet is simply not drawn. Flags come up a
// fraction of a second later rather than the window stopping to wait.
type Textures struct {
	// resolve turns the file name a coat of arms refers to into a path on disk.
	resolve func(name string) (string, bool)

	loaded  map[string]rl.Texture2D
	failed  map[string]error
	pending map[string]struct{}

	requests chan textureRequest
	results  chan textureResult
}

type textureRequest struct {
	name string
	path string
}

type textureResult struct {
	name  string
	image *rl.Image
	err   error
}

// NewTextures starts the reader. Stop has to be called to shut it down again.
func NewTextures(resolve func(name string) (string, bool)) *Textures {
	textures := &Textures{
		resolve:  resolve,
		loaded:   map[string]rl.Texture2D{},
		failed:   map[string]error{},
		pending:  map[string]struct{}{},
		requests: make(chan textureRequest, 256),
		results:  make(chan textureResult, 256),
	}

	go textures.read()

	return textures
}

// Get returns a texture, asking for it to be read if this is the first time it
// has been wanted. It reports false while the texture is still on its way, and
// for one that could not be read at all.
func (t *Textures) Get(name string) (rl.Texture2D, bool) {
	if name == "" {
		return rl.Texture2D{}, false
	}

	if loaded, ok := t.loaded[name]; ok {
		return loaded, true
	}

	if _, failed := t.failed[name]; failed {
		return rl.Texture2D{}, false
	}

	if _, waiting := t.pending[name]; waiting {
		return rl.Texture2D{}, false
	}

	path, ok := t.resolve(name)
	if !ok {
		t.failed[name] = errNotFound{name}

		return rl.Texture2D{}, false
	}

	t.pending[name] = struct{}{}

	select {
	case t.requests <- textureRequest{name: name, path: path}:
	default:
		// The queue is full, so let this one be asked for again next frame
		// rather than blocking the interface.
		delete(t.pending, name)
	}

	return rl.Texture2D{}, false
}

// Upload hands decoded textures to the GPU. It must run on the goroutine that
// owns the OpenGL context, and returns how many were uploaded.
func (t *Textures) Upload() int {
	uploaded := 0

	for uploaded < uploadsPerFrame {
		select {
		case result := <-t.results:
			delete(t.pending, result.name)

			if result.err != nil {
				t.failed[result.name] = result.err

				continue
			}

			gpu := rl.LoadTextureFromImage(result.image)
			rl.UnloadImage(result.image)

			// Emblems are drawn at all sorts of sizes, so they want smooth
			// scaling, and clamping keeps a rotated quad from wrapping the
			// opposite edge of the texture into view.
			rl.SetTextureFilter(gpu, rl.FilterBilinear)
			rl.SetTextureWrap(gpu, rl.WrapClamp)

			t.loaded[result.name] = gpu
			uploaded++

		default:
			return uploaded
		}
	}

	return uploaded
}

// Counts reports how many textures are on the GPU, on their way, and beyond
// help.
func (t *Textures) Counts() (loaded, pending, failed int) {
	return len(t.loaded), len(t.pending), len(t.failed)
}

// Failed returns why a texture could not be loaded.
func (t *Textures) Failed(name string) (error, bool) {
	err, ok := t.failed[name]

	return err, ok
}

// Forget drops everything, so that the next flag drawn reloads what it needs.
// It is how a change to the configured folders takes effect.
func (t *Textures) Forget() {
	for name, loaded := range t.loaded {
		rl.UnloadTexture(loaded)
		delete(t.loaded, name)
	}

	clear(t.failed)
}

// Unload releases every texture. The OpenGL context has to still be alive.
func (t *Textures) Unload() {
	close(t.requests)
	t.Forget()
}

// read is the reader goroutine: it reads files and decodes them, and never
// touches OpenGL.
func (t *Textures) read() {
	for request := range t.requests {
		data, err := os.ReadFile(request.path)
		if err != nil {
			t.results <- textureResult{name: request.name, err: err}

			continue
		}

		image, err := texture.Load(request.name, data)
		t.results <- textureResult{name: request.name, image: image, err: err}
	}
}

type errNotFound struct {
	name string
}

func (e errNotFound) Error() string {
	return e.name + " was not found in the configured folders"
}
