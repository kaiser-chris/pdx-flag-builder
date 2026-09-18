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
