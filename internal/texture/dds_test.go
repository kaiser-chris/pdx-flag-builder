package texture

import (
	"encoding/binary"
	"testing"
)

// fakeDDS lays out a DXT compressed DDS file: the header, then the blocks of
// every mipmap level the way a real file stores them, whole blocks each.
func fakeDDS(fourCC string, flags uint32, width, height, levels, blockSize int) []byte {
	header := make([]byte, ddsHeaderSize)
	copy(header[0:4], "DDS ")
	binary.LittleEndian.PutUint32(header[4:], 124)
	binary.LittleEndian.PutUint32(header[ddsHeightOffset:], uint32(height))
	binary.LittleEndian.PutUint32(header[ddsWidthOffset:], uint32(width))
	binary.LittleEndian.PutUint32(header[28:], uint32(levels))
	binary.LittleEndian.PutUint32(header[76:], 32)
	binary.LittleEndian.PutUint32(header[ddsPixelFlagsOffset:], flags)
	copy(header[ddsFourCCOffset:], fourCC)

	data := header

	for level := 0; level < levels; level++ {
		blocks := (max(width>>level, 1) + 3) / 4 * ((max(height>>level, 1) + 3) / 4)
		for index := range blocks * blockSize {
			// Each level marked by its number, to tell the first from the rest.
			data = append(data, byte(level+1+index%2))
		}
	}

	return data
}

// The texture that crashed the Store version: 768 by 512, DXT5, with every
// mipmap level down to one pixel.
func TestReadDDSDXTTakesExactlyTheLargestLevel(t *testing.T) {
	data := fakeDDS("DXT5", 0x4, 768, 512, 10, 16)

	image, ok, err := ReadDDSDXT(data)
	if err != nil || !ok {
		t.Fatalf("ReadDDSDXT = %v, %v, want a DXT5 image", ok, err)
	}

	if image.Width != 768 || image.Height != 512 || image.Format != DXT5 {
		t.Errorf("image = %dx%d %v, want 768x512 DXT5", image.Width, image.Height, image.Format)
	}

	// 192 by 128 blocks of 16 bytes, all of the first level and nothing else.
	if len(image.Blocks) != 192*128*16 {
		t.Errorf("blocks = %d bytes, want %d", len(image.Blocks), 192*128*16)
	}

	for _, value := range image.Blocks {
		if value != 1 && value != 2 {
			t.Fatalf("the blocks include bytes of a smaller level")
		}
	}
}

func TestReadDDSDXTFormats(t *testing.T) {
	tests := []struct {
		fourCC    string
		flags     uint32
		blockSize int
		want      DXTFormat
	}{
		{"DXT1", 0x4, 8, DXT1},
		{"DXT1", 0x5, 8, DXT1Alpha},
		{"DXT3", 0x4, 16, DXT3},
		{"DXT5", 0x4, 16, DXT5},
	}

	for _, test := range tests {
		image, ok, err := ReadDDSDXT(fakeDDS(test.fourCC, test.flags, 64, 32, 1, test.blockSize))
		if err != nil || !ok || image.Format != test.want || len(image.Blocks) != 16*8*test.blockSize {
			t.Errorf("%s with flags %#x: %+v, %v, %v", test.fourCC, test.flags, image, ok, err)
		}
	}
}

// An edge that is not a multiple of four still takes whole blocks.
func TestReadDDSDXTRoundsToWholeBlocks(t *testing.T) {
	image, _, err := ReadDDSDXT(fakeDDS("DXT5", 0x4, 30, 10, 1, 16))
	if err != nil {
		t.Fatal(err)
	}

	if want := 8 * 3 * 16; len(image.Blocks) != want {
		t.Errorf("blocks = %d bytes, want %d for 8 by 3 blocks", len(image.Blocks), want)
	}
}

func TestReadDDSDXTRefusesAFileCutShort(t *testing.T) {
	data := fakeDDS("DXT5", 0x4, 64, 64, 1, 16)

	if _, ok, err := ReadDDSDXT(data[:len(data)-1]); !ok || err == nil {
		t.Errorf("a file one byte short gave %v, %v, want an error", ok, err)
	}
}

func TestReadDDSDXTLeavesOtherFilesAlone(t *testing.T) {
	for name, data := range map[string][]byte{
		"not a DDS file":   []byte("PNG and then some bytes that are long enough to be read as a header, and some more ....................................................."),
		"a DX10 file":      fakeDDS("DX10", 0x4, 64, 64, 1, 16),
		"uncompressed DDS": fakeDDS("\x00\x00\x00\x00", 0x41, 64, 64, 1, 64),
	} {
		if _, ok, err := ReadDDSDXT(data); ok || err != nil {
			t.Errorf("%s: ok %v, error %v, want it left for another reader", name, ok, err)
		}
	}
}
