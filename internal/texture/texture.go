// Package texture turns the image files the games ship into textures raylib
// can draw.
//
// raylib reads PNG and uncompressed DDS by itself. The rest is read here:
//
//   - Targa, because the raylib build used by raylib-go leaves that reader out.
//   - BC7, the DDS compression behind a DX10 header, which raylib has no pixel
//     format for at all and so could not upload even if it could read it.
//   - DXT1, DXT3 and DXT5, which the GPU takes as they are, but which raylib's
//     own reader sizes wrongly (see ReadDDSDXT).
//
// The decoders in this package are plain Go and know nothing about raylib, so
// they can be tested on their own.
package texture

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Pixels is a decoded image, four bytes per pixel in red, green, blue, alpha
// order, with the first row at the top.
type Pixels struct {
	Width  int
	Height int
	Data   []byte
}

// NewPixels allocates an image of the given size.
func NewPixels(width, height int) *Pixels {
	return &Pixels{Width: width, Height: height, Data: make([]byte, width*height*4)}
}

// At returns the offset of a pixel in Data.
func (p *Pixels) At(x, y int) int {
	return (y*p.Width + x) * 4
}

// Set writes one pixel.
func (p *Pixels) Set(x, y int, r, g, b, a byte) {
	offset := p.At(x, y)
	p.Data[offset] = r
	p.Data[offset+1] = g
	p.Data[offset+2] = b
	p.Data[offset+3] = a
}

// Load decodes an image file into a raylib image.
//
// name is only used for its extension, so the bytes can come from anywhere. The
// returned image has to be unloaded by the caller once it has been uploaded.
func Load(name string, data []byte) (*rl.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%s is empty", name)
	}

	extension := strings.ToLower(filepath.Ext(name))

	switch extension {
	case ".tga":
		pixels, err := DecodeTGA(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}

		return imageFromPixels(pixels)

	case ".dds":
		// BC7 needs decoding here. DXT files the GPU reads as they are, but
		// raylib's own reader for them overruns its buffer; see ReadDDSDXT.
		// What is left, uncompressed DDS, raylib reads safely.
		if IsBC7(data) {
			pixels, err := DecodeDDSBC7(data)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}

			return imageFromPixels(pixels)
		}

		compressed, ok, err := ReadDDSDXT(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}

		if ok {
			return imageFromBlocks(compressed)
		}
	}

	image := rl.LoadImageFromMemory(extension, data, int32(len(data)))
	if image == nil || image.Width == 0 || image.Height == 0 {
		if image != nil {
			rl.UnloadImage(image)
		}

		return nil, fmt.Errorf("%s: raylib could not read this %s file", name, strings.TrimPrefix(extension, "."))
	}

	return image, nil
}

// imageFromPixels copies decoded pixels into an image raylib owns.
//
// The copy is deliberate. A raylib image holding a Go slice would be a Go
// pointer inside a struct handed to C, which the cgo rules forbid, and the
// garbage collector would be free to move it out from under the upload.
func imageFromPixels(pixels *Pixels) (*rl.Image, error) {
	if pixels.Width <= 0 || pixels.Height <= 0 {
		return nil, fmt.Errorf("image has no size")
	}

	image := rl.GenImageColor(pixels.Width, pixels.Height, rl.Blank)
	if image == nil || image.Data == nil {
		return nil, fmt.Errorf("could not allocate a %dx%d image", pixels.Width, pixels.Height)
	}

	destination := unsafe.Slice((*byte)(image.Data), len(pixels.Data))
	copy(destination, pixels.Data)

	return image, nil
}

// dxtFormats maps the block compressions to raylib's pixel formats.
//
// The numbers are raylib's own, from its PixelFormat enum in raylib.h. The
// constants raylib-go declares for them cannot be used: its list leaves out
// the three half float formats that come before them, so rl.CompressedDxt5Rgba
// is 14, which raylib reads as DXT1, and every compressed format is off by
// three.
var dxtFormats = map[DXTFormat]rl.PixelFormat{
	DXT1:      14, // PIXELFORMAT_COMPRESSED_DXT1_RGB
	DXT1Alpha: 15, // PIXELFORMAT_COMPRESSED_DXT1_RGBA
	DXT3:      16, // PIXELFORMAT_COMPRESSED_DXT3_RGBA
	DXT5:      17, // PIXELFORMAT_COMPRESSED_DXT5_RGBA
}

// imageFromBlocks copies compressed blocks into an image raylib owns, with a
// single mipmap level.
//
// raylib-go offers no allocator of raylib's own, so the memory comes from an
// uncompressed image of the same size, which is always at least as large as
// the blocks and is freed by raylib like any other image.
func imageFromBlocks(compressed *DXTImage) (*rl.Image, error) {
	// Whole blocks, so the buffer holds every block even for an edge that is
	// not a multiple of four.
	width := (compressed.Width + 3) / 4 * 4
	height := (compressed.Height + 3) / 4 * 4

	image := rl.GenImageColor(width, height, rl.Blank)
	if image == nil || image.Data == nil {
		return nil, fmt.Errorf("could not allocate a %dx%d image", compressed.Width, compressed.Height)
	}

	destination := unsafe.Slice((*byte)(image.Data), width*height*4)
	copy(destination, compressed.Blocks)

	image.Width = int32(compressed.Width)
	image.Height = int32(compressed.Height)
	image.Mipmaps = 1
	image.Format = dxtFormats[compressed.Format]

	return image, nil
}
