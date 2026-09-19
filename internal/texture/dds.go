package texture

import (
	"encoding/binary"
	"fmt"
)

// Offsets into a DDS file. The header is a fixed layout, so the few fields that
// matter can be read straight out of it.
const (
	ddsMagicSize      = 4
	ddsHeaderSize     = 128 // magic plus the 124 byte header
	ddsExtendedSize   = 20  // the DX10 header that may follow
	ddsHeightOffset   = 12
	ddsWidthOffset    = 16
	ddsFourCCOffset   = 84
	ddsDXGIFormatOffs = 128
)

// DXGI formats for BC7. Both spellings appear in the shipped files; they differ
// only in whether the values are meant to be read as sRGB, which the recolour
// shader does not care about.
const (
	dxgiBC7Unorm     = 98
	dxgiBC7UnormSRGB = 99
)

var (
	ddsMagic  = [4]byte{'D', 'D', 'S', ' '}
	dx10Magic = [4]byte{'D', 'X', '1', '0'}
)

// IsBC7 reports whether a DDS file is BC7 compressed, which is the one
// compression raylib can neither read nor upload.
func IsBC7(data []byte) bool {
	format, ok := ddsExtendedFormat(data)

	return ok && (format == dxgiBC7Unorm || format == dxgiBC7UnormSRGB)
}

// ddsExtendedFormat returns the DXGI format of a DDS file that carries a DX10
// header, and whether it had one.
func ddsExtendedFormat(data []byte) (uint32, bool) {
	if len(data) < ddsHeaderSize+ddsExtendedSize {
		return 0, false
	}

	if [4]byte(data[0:4]) != ddsMagic {
		return 0, false
	}

	if [4]byte(data[ddsFourCCOffset:ddsFourCCOffset+4]) != dx10Magic {
		return 0, false
	}

	return binary.LittleEndian.Uint32(data[ddsDXGIFormatOffs : ddsDXGIFormatOffs+4]), true
}

// DecodeDDSBC7 decodes a BC7 compressed DDS file.
//
// Only the largest mipmap level is read, which is the only one the flag
// renderer draws.
func DecodeDDSBC7(data []byte) (*Pixels, error) {
	if _, ok := ddsExtendedFormat(data); !ok {
		return nil, fmt.Errorf("not a DX10 DDS file")
	}

	width := int(binary.LittleEndian.Uint32(data[ddsWidthOffset : ddsWidthOffset+4]))
	height := int(binary.LittleEndian.Uint32(data[ddsHeightOffset : ddsHeightOffset+4]))

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("DDS image has no size")
	}

	blocks := data[ddsHeaderSize+ddsExtendedSize:]

	return DecodeBC7(blocks, width, height)
}

// Where the pixel format of a DDS file starts: its flags, then its FourCC.
const ddsPixelFlagsOffset = 80

// ddsAlphaPixels is the pixel format flag a DXT1 file sets when its blocks
// carry one bit of alpha.
const ddsAlphaPixels = 0x1

// DXTFormat is one of the block compressions the GPU reads as it is.
type DXTFormat int

const (
	DXT1 DXTFormat = iota + 1
	DXT1Alpha
	DXT3
	DXT5
)

// BlockSize is how many bytes one block of four by four pixels takes.
func (f DXTFormat) BlockSize() int {
	if f == DXT1 || f == DXT1Alpha {
		return 8
	}

	return 16
}

// DXTImage is the largest mipmap level of a DXT compressed DDS file: the
// compressed blocks as the file stores them, exactly as many as the image
// needs.
type DXTImage struct {
	Width, Height int
	Format        DXTFormat
	Blocks        []byte
}

// ReadDDSDXT reads the largest mipmap level of a DXT1, DXT3 or DXT5
// compressed DDS file. It reports false for any other file.
//
// raylib reads these files itself, but sizes the chain of mipmap levels by
// rule of thumb, as a third more than the largest level. For a texture with
// many levels that is a few bytes short of what it then uploads, and the
// upload reads past the end of its own buffer. Most allocators leave some
// memory there and nothing happens; the one Windows gives Store applications
// puts the end of the buffer at the end of a page, and the application
// crashes. Only the largest level is ever drawn, so only that is read here,
// sized from its blocks.
func ReadDDSDXT(data []byte) (*DXTImage, bool, error) {
	if len(data) < ddsHeaderSize || [4]byte(data[0:4]) != ddsMagic {
		return nil, false, nil
	}

	var format DXTFormat

	switch string(data[ddsFourCCOffset : ddsFourCCOffset+4]) {
	case "DXT1":
		format = DXT1
		if binary.LittleEndian.Uint32(data[ddsPixelFlagsOffset:ddsPixelFlagsOffset+4])&ddsAlphaPixels != 0 {
			format = DXT1Alpha
		}
	case "DXT3":
		format = DXT3
	case "DXT5":
		format = DXT5
	default:
		return nil, false, nil
	}

	width := int(binary.LittleEndian.Uint32(data[ddsWidthOffset : ddsWidthOffset+4]))
	height := int(binary.LittleEndian.Uint32(data[ddsHeightOffset : ddsHeightOffset+4]))

	if width <= 0 || height <= 0 {
		return nil, true, fmt.Errorf("DDS image has no size")
	}

	// Blocks cover four by four pixels, so an edge that is not a multiple of
	// four still takes a whole block.
	size := (width + 3) / 4 * ((height + 3) / 4) * format.BlockSize()

	if len(data) < ddsHeaderSize+size {
		return nil, true, fmt.Errorf("DDS file is cut short: %d bytes of blocks, want %d", len(data)-ddsHeaderSize, size)
	}

	return &DXTImage{
		Width:  width,
		Height: height,
		Format: format,
		Blocks: data[ddsHeaderSize : ddsHeaderSize+size],
	}, true, nil
}
