package texture

import (
	"encoding/binary"
	"fmt"
)

// Targa image types. The games only use the true colour ones, but grayscale
// costs almost nothing to support and is the same code path.
const (
	tgaTrueColor         = 2
	tgaGrayscale         = 3
	tgaRunLengthColor    = 10
	tgaRunLengthGray     = 11
	tgaHeaderSize        = 18
	tgaTopToBottomBit    = 0x20
	tgaRunLengthPacketOn = 0x80
)

// DecodeTGA reads a Targa image.
//
// raylib is built without its Targa reader in the raylib-go distribution, and
// the flag patterns are Targa files, so this is not an optional corner: the
// most used pattern in both games, pattern_solid.tga, comes through here.
//
// Supported are the uncompressed and run length encoded true colour and
// grayscale images, at 8, 16, 24 and 32 bits per pixel, which is everything the
// games ship.
func DecodeTGA(data []byte) (*Pixels, error) {
	if len(data) < tgaHeaderSize {
		return nil, fmt.Errorf("targa file is too short")
	}

	var (
		identifierLength = int(data[0])
		colorMapType     = data[1]
		imageType        = data[2]
		width            = int(binary.LittleEndian.Uint16(data[12:14]))
		height           = int(binary.LittleEndian.Uint16(data[14:16]))
		depth            = int(data[16])
		descriptor       = data[17]
	)

	if colorMapType != 0 {
		return nil, fmt.Errorf("targa colour maps are not supported")
	}

	switch imageType {
	case tgaTrueColor, tgaGrayscale, tgaRunLengthColor, tgaRunLengthGray:
	default:
		return nil, fmt.Errorf("unsupported targa image type %d", imageType)
	}

	bytesPerPixel := depth / 8
	if depth%8 != 0 || bytesPerPixel < 1 || bytesPerPixel > 4 {
		return nil, fmt.Errorf("unsupported targa depth of %d bits", depth)
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("targa image has no size")
	}

	offset := tgaHeaderSize + identifierLength
	if offset > len(data) {
		return nil, fmt.Errorf("targa file ends inside its header")
	}

	body := data[offset:]

	var (
		pixels = NewPixels(width, height)
		err    error
	)

	if imageType == tgaRunLengthColor || imageType == tgaRunLengthGray {
		err = readRunLength(body, pixels, bytesPerPixel)
	} else {
		err = readUncompressed(body, pixels, bytesPerPixel)
	}
	if err != nil {
		return nil, err
	}

	// Targa stores rows from the bottom up unless the descriptor says
	// otherwise, while everything downstream expects the first row to be the
	// top one.
	if descriptor&tgaTopToBottomBit == 0 {
		flipVertically(pixels)
	}

	return pixels, nil
}

func readUncompressed(body []byte, pixels *Pixels, bytesPerPixel int) error {
	count := pixels.Width * pixels.Height

	if len(body) < count*bytesPerPixel {
		return fmt.Errorf("targa file ends after %d of %d pixels", len(body)/bytesPerPixel, count)
	}

	for index := range count {
		writePixel(pixels, index, body[index*bytesPerPixel:], bytesPerPixel)
	}

	return nil
}

// readRunLength reads the run length encoding, which alternates between runs of
// one repeated pixel and runs of literal ones.
func readRunLength(body []byte, pixels *Pixels, bytesPerPixel int) error {
	count := pixels.Width * pixels.Height

	var (
		written = 0
		offset  = 0
	)

	for written < count {
		if offset >= len(body) {
			return fmt.Errorf("targa file ends after %d of %d pixels", written, count)
		}

		packet := body[offset]
		offset++

		length := int(packet&0x7F) + 1
		if written+length > count {
			return fmt.Errorf("targa run reaches past the end of the image")
		}

		if packet&tgaRunLengthPacketOn != 0 {
			// One pixel, repeated.
			if offset+bytesPerPixel > len(body) {
				return fmt.Errorf("targa file ends inside a run")
			}

			for range length {
				writePixel(pixels, written, body[offset:], bytesPerPixel)
				written++
			}

			offset += bytesPerPixel

			continue
		}

		// A run of pixels, each written out.
		if offset+length*bytesPerPixel > len(body) {
			return fmt.Errorf("targa file ends inside a run")
		}

		for range length {
			writePixel(pixels, written, body[offset:], bytesPerPixel)
			written++
			offset += bytesPerPixel
		}
	}

	return nil
}

// writePixel converts one stored pixel into red, green, blue, alpha. Targa
// stores its channels in blue, green, red order.
func writePixel(pixels *Pixels, index int, source []byte, bytesPerPixel int) {
	target := index * 4

	switch bytesPerPixel {
	case 1:
		gray := source[0]
		pixels.Data[target] = gray
		pixels.Data[target+1] = gray
		pixels.Data[target+2] = gray
		pixels.Data[target+3] = 255

	case 2:
		// Five bits per channel packed into two bytes, with the top bit acting
		// as an on or off alpha.
		packed := uint16(source[0]) | uint16(source[1])<<8
		pixels.Data[target] = expandFiveBits(byte(packed >> 10 & 0x1F))
		pixels.Data[target+1] = expandFiveBits(byte(packed >> 5 & 0x1F))
		pixels.Data[target+2] = expandFiveBits(byte(packed & 0x1F))
		pixels.Data[target+3] = 255
		if packed&0x8000 == 0 {
			pixels.Data[target+3] = 0
		}

	case 3:
		pixels.Data[target] = source[2]
		pixels.Data[target+1] = source[1]
		pixels.Data[target+2] = source[0]
		pixels.Data[target+3] = 255

	case 4:
		pixels.Data[target] = source[2]
		pixels.Data[target+1] = source[1]
		pixels.Data[target+2] = source[0]
		pixels.Data[target+3] = source[3]
	}
}

// expandFiveBits stretches a five bit channel across the full eight bit range,
// so that the largest value comes out as white rather than nearly white.
func expandFiveBits(value byte) byte {
	return value<<3 | value>>2
}

func flipVertically(pixels *Pixels) {
	stride := pixels.Width * 4
	row := make([]byte, stride)

	for top, bottom := 0, pixels.Height-1; top < bottom; top, bottom = top+1, bottom-1 {
		topRow := pixels.Data[top*stride : top*stride+stride]
		bottomRow := pixels.Data[bottom*stride : bottom*stride+stride]

		copy(row, topRow)
		copy(topRow, bottomRow)
		copy(bottomRow, row)
	}
}
