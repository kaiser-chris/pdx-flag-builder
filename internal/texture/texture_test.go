package texture

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// targaHeader builds an 18 byte Targa header.
func targaHeader(imageType byte, width, height int, depth byte, descriptor byte) []byte {
	header := make([]byte, tgaHeaderSize)
	header[2] = imageType
	binary.LittleEndian.PutUint16(header[12:14], uint16(width))
	binary.LittleEndian.PutUint16(header[14:16], uint16(height))
	header[16] = depth
	header[17] = descriptor

	return header
}

func TestDecodeTGAUncompressed(t *testing.T) {
	// Two pixels side by side, written top to bottom. Targa stores its
	// channels the other way round, so these are red then green.
	file := append(targaHeader(tgaTrueColor, 2, 1, 24, tgaTopToBottomBit),
		0x00, 0x00, 0xFF, // blue, green, red
		0x00, 0xFF, 0x00,
	)

	pixels, err := DecodeTGA(file)
	if err != nil {
		t.Fatalf("DecodeTGA: %v", err)
	}

	if pixels.Width != 2 || pixels.Height != 1 {
		t.Fatalf("size = %dx%d, want 2x1", pixels.Width, pixels.Height)
	}

	if got := pixels.Data[0:4]; got[0] != 255 || got[1] != 0 || got[2] != 0 || got[3] != 255 {
		t.Errorf("first pixel = %v, want opaque red", got)
	}

	if got := pixels.Data[4:8]; got[0] != 0 || got[1] != 255 || got[2] != 0 {
		t.Errorf("second pixel = %v, want green", got)
	}
}

func TestDecodeTGAWithAlpha(t *testing.T) {
	file := append(targaHeader(tgaTrueColor, 1, 1, 32, tgaTopToBottomBit),
		0x10, 0x20, 0x30, 0x40,
	)

	pixels, err := DecodeTGA(file)
	if err != nil {
		t.Fatalf("DecodeTGA: %v", err)
	}

	want := []byte{0x30, 0x20, 0x10, 0x40}
	for index, value := range want {
		if pixels.Data[index] != value {
			t.Fatalf("pixel = %v, want %v", pixels.Data[0:4], want)
		}
	}
}

func TestDecodeTGABottomUp(t *testing.T) {
	// Without the top to bottom flag the first row stored is the bottom one,
	// which is how every Targa the games ship is written.
	file := append(targaHeader(tgaTrueColor, 1, 2, 24, 0),
		0x00, 0x00, 0xFF, // bottom row: red
		0x00, 0xFF, 0x00, // top row: green
	)

	pixels, err := DecodeTGA(file)
	if err != nil {
		t.Fatalf("DecodeTGA: %v", err)
	}

	if pixels.Data[1] != 255 {
		t.Errorf("top row = %v, want green", pixels.Data[0:4])
	}

	if pixels.Data[4] != 255 {
		t.Errorf("bottom row = %v, want red", pixels.Data[4:8])
	}
}

func TestDecodeTGARunLength(t *testing.T) {
	file := append(targaHeader(tgaRunLengthColor, 4, 1, 24, tgaTopToBottomBit),
		// A run of three repeats of one red pixel.
		0x82, 0x00, 0x00, 0xFF,
		// Then one literal green pixel.
		0x00, 0x00, 0xFF, 0x00,
	)

	pixels, err := DecodeTGA(file)
	if err != nil {
		t.Fatalf("DecodeTGA: %v", err)
	}

	for pixel := range 3 {
		if pixels.Data[pixel*4] != 255 {
			t.Errorf("pixel %d = %v, want red", pixel, pixels.Data[pixel*4:pixel*4+4])
		}
	}

	if pixels.Data[3*4+1] != 255 {
		t.Errorf("last pixel = %v, want green", pixels.Data[12:16])
	}
}

func TestDecodeTGARejectsRubbish(t *testing.T) {
	cases := map[string][]byte{
		"too short":       {0, 0, 2},
		"colour mapped":   append(targaHeader(1, 1, 1, 8, 0), 0),
		"odd depth":       append(targaHeader(tgaTrueColor, 1, 1, 12, 0), 0),
		"truncated":       append(targaHeader(tgaTrueColor, 4, 4, 24, 0), 1, 2, 3),
		"no size":         targaHeader(tgaTrueColor, 0, 0, 24, 0),
		"run past<br>end": append(targaHeader(tgaRunLengthColor, 1, 1, 24, 0), 0x83, 0, 0, 0),
	}

	for name, file := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeTGA(file); err == nil {
				t.Error("decoding succeeded, want an error")
			}
		})
	}
}

// blockWriter fills a BC7 block a few bits at a time, lowest bit of the first
// byte first, which is the order the format is read in.
type blockWriter struct {
	data     [bc7BlockSize]byte
	position uint
}

func (w *blockWriter) write(value uint32, count uint) {
	for bit := range count {
		if value>>bit&1 != 0 {
			w.data[w.position/8] |= 1 << (w.position % 8)
		}

		w.position++
	}
}

// mode6Block builds the simplest BC7 block there is: one subset, seven bit
// endpoints with an extra low bit each, and four bit indices.
func mode6Block(indices [16]uint32) []byte {
	writer := &blockWriter{}

	// Six zeroes and a one select mode 6.
	writer.write(1<<6, 7)

	// Endpoints, channel by channel: red runs from nothing to full, the other
	// two channels stay put.
	writer.write(0, 7)   // red, first endpoint
	writer.write(127, 7) // red, second endpoint
	writer.write(0, 7)   // green
	writer.write(0, 7)
	writer.write(0, 7) // blue
	writer.write(0, 7)
	writer.write(127, 7) // alpha, both endpoints opaque
	writer.write(127, 7)

	// The extra low bit of each endpoint.
	writer.write(1, 1)
	writer.write(1, 1)

	// The first pixel is the anchor and stores one bit fewer.
	writer.write(indices[0], 3)
	for pixel := 1; pixel < 16; pixel++ {
		writer.write(indices[pixel], 4)
	}

	return writer.data[:]
}

func TestDecodeBC7Mode6(t *testing.T) {
	var indices [16]uint32
	indices[1] = 15 // the far endpoint
	indices[2] = 8  // halfway along

	pixels, err := DecodeBC7(mode6Block(indices), 4, 4)
	if err != nil {
		t.Fatalf("DecodeBC7: %v", err)
	}

	if pixels.Width != 4 || pixels.Height != 4 {
		t.Fatalf("size = %dx%d, want 4x4", pixels.Width, pixels.Height)
	}

	// With the extra bit set, an endpoint of 0 expands to 1 and one of 127 to
	// 255, so the first pixel sits at the near end and the second at the far.
	cases := []struct {
		pixel int
		red   byte
	}{
		{0, 1},
		{1, 255},
		{2, 136},
		{3, 1},
	}

	for _, test := range cases {
		offset := test.pixel * 4

		if got := pixels.Data[offset]; got != test.red {
			t.Errorf("pixel %d red = %d, want %d", test.pixel, got, test.red)
		}

		if got := pixels.Data[offset+3]; got != 255 {
			t.Errorf("pixel %d alpha = %d, want 255", test.pixel, got)
		}
	}
}

func TestDecodeBC7UnknownModeIsTransparent(t *testing.T) {
	// A block of nothing but zero bits names no mode at all.
	pixels, err := DecodeBC7(make([]byte, bc7BlockSize), 4, 4)
	if err != nil {
		t.Fatalf("DecodeBC7: %v", err)
	}

	for index, value := range pixels.Data {
		if value != 0 {
			t.Fatalf("byte %d = %d, want a fully transparent block", index, value)
		}
	}
}

func TestDecodeBC7ShortData(t *testing.T) {
	if _, err := DecodeBC7(make([]byte, bc7BlockSize), 8, 8); err == nil {
		t.Error("decoding four blocks worth of image from one block succeeded")
	}
}

func TestDecodeBC7PartialBlocks(t *testing.T) {
	// An image that is not a whole number of blocks across keeps its own size.
	pixels, err := DecodeBC7(mode6Block([16]uint32{}), 3, 2)
	if err != nil {
		t.Fatalf("DecodeBC7: %v", err)
	}

	if pixels.Width != 3 || pixels.Height != 2 || len(pixels.Data) != 3*2*4 {
		t.Errorf("size = %dx%d with %d bytes, want 3x2 with 24", pixels.Width, pixels.Height, len(pixels.Data))
	}
}

func TestIsBC7(t *testing.T) {
	file := make([]byte, ddsHeaderSize+ddsExtendedSize)
	copy(file[0:4], ddsMagic[:])
	copy(file[ddsFourCCOffset:], dx10Magic[:])
	binary.LittleEndian.PutUint32(file[ddsDXGIFormatOffs:], dxgiBC7Unorm)

	if !IsBC7(file) {
		t.Error("a BC7 file was not recognised")
	}

	// A file compressed the older way has a plain four character code, which
	// raylib reads by itself.
	copy(file[ddsFourCCOffset:], []byte("DXT5"))

	if IsBC7(file) {
		t.Error("a DXT5 file was taken for BC7")
	}

	if IsBC7([]byte("not a dds file at all")) {
		t.Error("rubbish was taken for BC7")
	}
}

// The DXT pixel formats handed to raylib have to be the numbers raylib's C
// enum gives them, which raylib-go's constants are not. raylib.h is read from
// the raylib-go module, so an update that renumbers them fails here.
func TestDXTFormatsMatchRaylib(t *testing.T) {
	output, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/gen2brain/raylib-go/raylib").Output()
	if err != nil {
		t.Skipf("the raylib-go module is not at hand: %v", err)
	}

	header, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(output)), "raylib.h"))
	if err != nil {
		t.Fatal(err)
	}

	// Read the enum the way the C compiler numbers it.
	enum := regexp.MustCompile(`(?s)typedef enum \{\s*(PIXELFORMAT_UNCOMPRESSED_GRAYSCALE = 1.*?)\} PixelFormat;`).FindSubmatch(header)
	if enum == nil {
		t.Fatal("the PixelFormat enum was not found in raylib.h")
	}

	values := map[string]int{}
	next := 0

	for _, line := range strings.Split(string(enum[1]), "\n") {
		name := regexp.MustCompile(`^\s*(PIXELFORMAT_\w+)\s*(?:=\s*(\d+))?`).FindStringSubmatch(line)
		if name == nil {
			continue
		}

		if name[2] != "" {
			next, _ = strconv.Atoi(name[2])
		}

		values[name[1]] = next
		next++
	}

	for format, name := range map[DXTFormat]string{
		DXT1:      "PIXELFORMAT_COMPRESSED_DXT1_RGB",
		DXT1Alpha: "PIXELFORMAT_COMPRESSED_DXT1_RGBA",
		DXT3:      "PIXELFORMAT_COMPRESSED_DXT3_RGBA",
		DXT5:      "PIXELFORMAT_COMPRESSED_DXT5_RGBA",
	} {
		if got, want := int(dxtFormats[format]), values[name]; got != want {
			t.Errorf("%s is handed to raylib as %d, raylib.h says %d", name, got, want)
		}
	}
}
