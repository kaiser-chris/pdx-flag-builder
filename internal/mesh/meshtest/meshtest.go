// Package meshtest writes mesh files for tests.
//
// The meshes the application draws belong to the games, so a test that needs
// one cannot use theirs. Quad writes the smallest file the reader accepts,
// which is enough to stand in for one.
package meshtest

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Quad returns a mesh file holding one square of the given size, facing the
// camera, with its texture coordinates running from corner to corner.
func Quad(width, height float32) []byte {
	half := [2]float32{width / 2, height / 2}

	positions := []float32{
		-half[0], -half[1], 0,
		half[0], -half[1], 0,
		-half[0], half[1], 0,
		half[0], half[1], 0,
	}

	normals := []float32{
		0, 0, -1,
		0, 0, -1,
		0, 0, -1,
		0, 0, -1,
	}

	tangents := []float32{
		1, 0, 0, 1,
		1, 0, 0, 1,
		1, 0, 0, 1,
		1, 0, 0, 1,
	}

	// The first row of the picture is the top of the square.
	coordinates := []float32{
		0, 1,
		1, 1,
		0, 0,
		1, 0,
	}

	triangles := []int32{0, 1, 2, 2, 1, 3}

	file := &bytes.Buffer{}
	file.WriteString("@@b@")

	numbers(file, "pdxasset", []int32{1, 0})
	object(file, 1, "object")
	object(file, 2, "shape")
	object(file, 3, "mesh")

	values(file, "p", positions)
	values(file, "n", normals)
	values(file, "ta", tangents)
	values(file, "u0", coordinates)
	values(file, "u1", coordinates)
	numbers(file, "tri", triangles)

	return file.Bytes()
}

// object writes the start of an object: one bracket per level, then its name.
func object(file *bytes.Buffer, depth int, name string) {
	for range depth {
		file.WriteByte('[')
	}

	file.WriteString(name)
	file.WriteByte(0)
}

// values writes a property holding numbers with a fraction.
func values(file *bytes.Buffer, name string, numbers []float32) {
	property(file, name, 'f', len(numbers))

	for _, value := range numbers {
		binary.Write(file, binary.LittleEndian, math.Float32bits(value))
	}
}

// numbers writes a property holding whole numbers.
func numbers(file *bytes.Buffer, name string, values []int32) {
	property(file, name, 'i', len(values))

	for _, value := range values {
		binary.Write(file, binary.LittleEndian, value)
	}
}

func property(file *bytes.Buffer, name string, kind byte, count int) {
	file.WriteByte('!')
	file.WriteByte(byte(len(name)))
	file.WriteString(name)
	file.WriteByte(kind)
	binary.Write(file, binary.LittleEndian, uint32(count))
}
