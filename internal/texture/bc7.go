package texture

import (
	"encoding/binary"
	"fmt"
)

// bc7BlockSize is the size of one compressed four by four block.
const bc7BlockSize = 16

// How many bits each endpoint component is stored with, per mode.
var (
	bc7ColorBits = [8]uint{4, 6, 5, 7, 5, 7, 7, 5}
	bc7AlphaBits = [8]uint{0, 0, 0, 0, 6, 8, 7, 5}
)

// bc7ModesWithPBits marks the modes that store an extra low bit per endpoint,
// which buys a little precision: modes 0, 1, 3, 6 and 7.
const bc7ModesWithPBits = 0b11001011

// How far along the line between two endpoints each index value sits, out of 64.
var (
	bc7Weights2 = [...]int32{0, 21, 43, 64}
	bc7Weights3 = [...]int32{0, 9, 18, 27, 37, 46, 55, 64}
	bc7Weights4 = [...]int32{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}
)

// DecodeBC7 decodes BC7 compressed blocks into pixels.
//
// BC7 stores a four by four block in sixteen bytes. Which of its eight modes a
// block uses is written in unary at the front, and the mode decides everything
// else: how many subsets the block is cut into, how many bits the endpoints and
// the indices get, and whether alpha is stored separately.
//
// This is the one compression the games use that raylib can neither read nor
// upload, so the decoding has to happen here and the result is handed over as
// plain pixels.
func DecodeBC7(blocks []byte, width, height int) (*Pixels, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("image has no size")
	}

	blocksAcross := (width + 3) / 4
	blocksDown := (height + 3) / 4

	if needed := blocksAcross * blocksDown * bc7BlockSize; len(blocks) < needed {
		return nil, fmt.Errorf("BC7 data is %d bytes, %d are needed for %dx%d", len(blocks), needed, width, height)
	}

	pixels := NewPixels(width, height)

	var block [16][4]byte

	for blockY := range blocksDown {
		for blockX := range blocksAcross {
			offset := (blockY*blocksAcross + blockX) * bc7BlockSize
			decodeBC7Block(blocks[offset:offset+bc7BlockSize], &block)

			for row := range 4 {
				y := blockY*4 + row
				if y >= height {
					break
				}

				for column := range 4 {
					x := blockX*4 + column
					if x >= width {
						break
					}

					pixel := block[row*4+column]
					pixels.Set(x, y, pixel[0], pixel[1], pixel[2], pixel[3])
				}
			}
		}
	}

	return pixels, nil
}

// bc7Stream reads a block a few bits at a time, starting at the lowest bit of
// the first byte.
type bc7Stream struct {
	low  uint64
	high uint64
}

func (s *bc7Stream) read(count uint) uint32 {
	if count == 0 {
		return 0
	}

	mask := uint64(1)<<count - 1
	bits := s.low & mask

	s.low >>= count
	s.low |= (s.high & mask) << (64 - count)
	s.high >>= count

	return uint32(bits)
}

func decodeBC7Block(block []byte, pixels *[16][4]byte) {
	stream := bc7Stream{
		low:  binary.LittleEndian.Uint64(block[0:8]),
		high: binary.LittleEndian.Uint64(block[8:16]),
	}

	// The mode is written in unary: as many zero bits as the mode number,
	// then a one.
	mode := 0
	for mode < 8 && stream.read(1) == 0 {
		mode++
	}

	if mode >= 8 {
		// Not a mode BC7 defines. The specification calls for transparent
		// black rather than for guessing.
		*pixels = [16][4]byte{}

		return
	}

	var (
		subsets        = 1
		partition      uint32
		rotation       uint32
		indexSelection uint32
	)

	switch mode {
	case 0:
		subsets = 3
		partition = stream.read(4)
	case 2:
		subsets = 3
		partition = stream.read(6)
	case 1, 3, 7:
		subsets = 2
		partition = stream.read(6)
	case 4, 5:
		// The one subset modes with separate alpha can move the alpha channel
		// into one of the colour channels instead.
		rotation = stream.read(2)

		if mode == 4 {
			indexSelection = stream.read(1)
		}
	}

	endpointCount := subsets * 2

	colorBits := bc7ColorBits[mode]
	alphaBits := bc7AlphaBits[mode]

	// Endpoints are stored channel by channel rather than endpoint by endpoint.
	var endpoints [6][4]int32

	for channel := range 3 {
		for endpoint := range endpointCount {
			endpoints[endpoint][channel] = int32(stream.read(colorBits))
		}
	}

	if alphaBits > 0 {
		for endpoint := range endpointCount {
			endpoints[endpoint][3] = int32(stream.read(alphaBits))
		}
	}

	hasPBits := bc7ModesWithPBits>>mode&1 == 1
	if hasPBits {
		readPBits(&stream, &endpoints, endpointCount, mode)
	}

	extra := uint(0)
	if hasPBits {
		extra = 1
	}

	expandEndpoints(&endpoints, endpointCount, colorBits+extra, alphaBits+extra)

	if alphaBits == 0 {
		// Modes without alpha are fully opaque.
		for endpoint := range endpointCount {
			endpoints[endpoint][3] = 255
		}
	}

	indexBits := uint(2)

	switch mode {
	case 0, 1:
		indexBits = 3
	case 6:
		indexBits = 4
	}

	alphaIndexBits := uint(0)

	switch mode {
	case 4:
		alphaIndexBits = 3
	case 5:
		alphaIndexBits = 2
	}

	weights := bc7Weights2[:]

	switch indexBits {
	case 3:
		weights = bc7Weights3[:]
	case 4:
		weights = bc7Weights4[:]
	}

	alphaWeights := bc7Weights3[:]
	if alphaIndexBits == 2 {
		alphaWeights = bc7Weights2[:]
	}

	subsetOf := subsetTable(subsets, partition)

	// Colour indices come first for the whole block, so they have to be read
	// before anything can be interpolated.
	var indices [16]uint32

	for pixel := range 16 {
		bits := indexBits
		if subsetOf[pixel]&anchorFlag != 0 {
			bits--
		}

		indices[pixel] = stream.read(bits)
	}

	for pixel := range 16 {
		subset := int(subsetOf[pixel] & subsetMask)
		low := &endpoints[subset*2]
		high := &endpoints[subset*2+1]

		var red, green, blue, alpha int32

		if alphaIndexBits == 0 {
			index := indices[pixel]

			red = interpolate(low[0], high[0], weights, index)
			green = interpolate(low[1], high[1], weights, index)
			blue = interpolate(low[2], high[2], weights, index)
			alpha = interpolate(low[3], high[3], weights, index)
		} else {
			bits := alphaIndexBits
			if pixel == 0 {
				bits--
			}

			alphaIndex := stream.read(bits)

			// With two sets of indices, the selection bit decides which set
			// drives the colour and which the alpha.
			colorIndex, transparencyIndex := indices[pixel], alphaIndex
			colorWeights, transparencyWeights := weights, alphaWeights

			if indexSelection != 0 {
				colorIndex, transparencyIndex = alphaIndex, indices[pixel]
				colorWeights, transparencyWeights = alphaWeights, weights
			}

			red = interpolate(low[0], high[0], colorWeights, colorIndex)
			green = interpolate(low[1], high[1], colorWeights, colorIndex)
			blue = interpolate(low[2], high[2], colorWeights, colorIndex)
			alpha = interpolate(low[3], high[3], transparencyWeights, transparencyIndex)
		}

		// A rotation means alpha was stored in one of the colour channels.
		switch rotation {
		case 1:
			alpha, red = red, alpha
		case 2:
			alpha, green = green, alpha
		case 3:
			alpha, blue = blue, alpha
		}

		pixels[pixel] = [4]byte{byte(red), byte(green), byte(blue), byte(alpha)}
	}
}

// readPBits applies the extra low bit some modes give their endpoints. Mode 1
// is the odd one out: there both endpoints of a subset share a single bit.
func readPBits(stream *bc7Stream, endpoints *[6][4]int32, endpointCount, mode int) {
	for endpoint := range endpointCount {
		for channel := range 4 {
			endpoints[endpoint][channel] <<= 1
		}
	}

	if mode == 1 {
		first := int32(stream.read(1))
		second := int32(stream.read(1))

		for channel := range 3 {
			endpoints[0][channel] |= first
			endpoints[1][channel] |= first
			endpoints[2][channel] |= second
			endpoints[3][channel] |= second
		}

		return
	}

	for endpoint := range endpointCount {
		bit := int32(stream.read(1))

		for channel := range 4 {
			endpoints[endpoint][channel] |= bit
		}
	}
}

// expandEndpoints stretches endpoints stored with fewer than eight bits across
// the full range, by shifting them up and filling the freed low bits with the
// high ones. A five bit 31 has to come out as 255, not as 248.
func expandEndpoints(endpoints *[6][4]int32, endpointCount int, colorPrecision, alphaPrecision uint) {
	for endpoint := range endpointCount {
		for channel := range 3 {
			value := endpoints[endpoint][channel] << (8 - colorPrecision)
			endpoints[endpoint][channel] = value | value>>colorPrecision
		}

		value := endpoints[endpoint][3] << (8 - alphaPrecision)
		endpoints[endpoint][3] = value | value>>alphaPrecision
	}
}

// subsetTable says which subset each of the sixteen pixels belongs to, and
// which pixels are anchors. A block with one subset has no partition to look
// up: everything is subset zero and the first pixel is its anchor.
func subsetTable(subsets int, partition uint32) [16]uint8 {
	switch subsets {
	case 2:
		return bc7Partitions2[partition]
	case 3:
		return bc7Partitions3[partition]
	}

	table := [16]uint8{}
	table[0] = anchorFlag

	return table
}

func interpolate(low, high int32, weights []int32, index uint32) int32 {
	weight := weights[index]

	return (low*(64-weight) + high*weight + 32) >> 6
}
