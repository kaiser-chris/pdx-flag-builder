// Package mesh reads the binary mesh files of the Paradox games.
//
// A .mesh file is a tree of named objects holding named properties, each
// property an array of numbers or strings. The geometry of a model sits in a
// mesh object below its shape:
//
//	[object]
//	  [flag_01Shape]
//	    [mesh]
//	      p    the positions, three numbers per vertex
//	      n    the normals, three
//	      ta   the tangents, four, the last being which way the bitangent goes
//	      u0   the texture coordinates, two
//	      u1   a second set, which the games use for a flag's own picture
//	      tri  the triangles, three vertex numbers each
//
// Only the geometry is read here. What else a file holds, its bounding boxes
// and the material a model is drawn with, is skipped, but skipped by reading
// it, so that a file that holds more than this expects is still read to the
// end.
package mesh

import (
	"encoding/binary"
	"fmt"
	"math"
)

// magic starts every binary mesh file.
const magic = "@@b@"

// The properties the geometry is read from.
const (
	keyPositions = "p"
	keyNormals   = "n"
	keyTangents  = "ta"
	keyUV0       = "u0"
	keyUV1       = "u1"
	keyTriangles = "tri"
)

// Mesh is the geometry of one model: its vertices and the triangles between
// them.
//
// The values are laid out the way a graphics library wants them, one flat
// array per attribute, rather than one struct per vertex.
type Mesh struct {
	// Name is the shape the geometry belongs to.
	Name string

	// Positions holds three numbers per vertex, Normals three, Tangents four
	// and UV0 and UV1 two. A file that leaves an attribute out leaves it
	// empty here.
	Positions []float32
	Normals   []float32
	Tangents  []float32
	UV0       []float32
	UV1       []float32

	// Indices holds three vertex numbers per triangle.
	Indices []uint16
}

// Vertices is how many vertices the mesh has.
func (m *Mesh) Vertices() int {
	return len(m.Positions) / 3
}

// Triangles is how many triangles the mesh has.
func (m *Mesh) Triangles() int {
	return len(m.Indices) / 3
}

// Bounds is the box the mesh fits in.
func (m *Mesh) Bounds() (low, high [3]float32) {
	low = [3]float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	high = [3]float32{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	for index, value := range m.Positions {
		axis := index % 3
		low[axis] = min(low[axis], value)
		high[axis] = max(high[axis], value)
	}

	return low, high
}

// Read reads the first mesh of a binary mesh file.
//
// A file holds one mesh per level of detail of each of its shapes; the first
// is the shape's full detail, which is the one a tool that draws the model
// wants.
func Read(data []byte) (*Mesh, error) {
	reader := &reader{data: data}

	if !reader.take(len(magic)) || string(reader.taken) != magic {
		return nil, fmt.Errorf("not a Paradox mesh: it does not start with %q", magic)
	}

	var (
		found *Mesh
		shape string
	)

	for reader.offset < len(reader.data) {
		switch reader.data[reader.offset] {
		case '[':
			depth, name, err := reader.object()
			if err != nil {
				return nil, err
			}

			// The objects are, in order: the file, each shape, and the
			// meshes of a shape, one per level of detail.
			switch {
			case depth == 2:
				shape = name

			case depth == 3 && name == "mesh" && found == nil:
				found = &Mesh{Name: shape}
			}

		case '!':
			name, values, indices, err := reader.property()
			if err != nil {
				return nil, err
			}

			if found == nil || found.filled() {
				continue
			}

			switch name {
			case keyPositions:
				found.Positions = values
			case keyNormals:
				found.Normals = values
			case keyTangents:
				found.Tangents = values
			case keyUV0:
				found.UV0 = values
			case keyUV1:
				found.UV1 = values
			case keyTriangles:
				found.Indices = indices
			}

		default:
			return nil, fmt.Errorf("mesh: unexpected byte %#x at %d", reader.data[reader.offset], reader.offset)
		}
	}

	if found == nil {
		return nil, fmt.Errorf("mesh: the file holds no mesh")
	}

	return found, found.check()
}

// filled reports whether everything the geometry needs has been read, which is
// what keeps the levels of detail after the first out of it.
func (m *Mesh) filled() bool {
	return len(m.Positions) > 0 && len(m.Indices) > 0
}

// check reports a mesh whose attributes do not agree with one another, which
// would draw as nonsense rather than fail.
func (m *Mesh) check() error {
	if len(m.Positions) == 0 || len(m.Positions)%3 != 0 {
		return fmt.Errorf("mesh %s: %d position values, want three per vertex", m.Name, len(m.Positions))
	}

	if len(m.Indices) == 0 || len(m.Indices)%3 != 0 {
		return fmt.Errorf("mesh %s: %d vertex numbers, want three per triangle", m.Name, len(m.Indices))
	}

	vertices := m.Vertices()

	for _, attribute := range []struct {
		name   string
		values []float32
		each   int
	}{
		{keyNormals, m.Normals, 3},
		{keyTangents, m.Tangents, 4},
		{keyUV0, m.UV0, 2},
		{keyUV1, m.UV1, 2},
	} {
		if len(attribute.values) != 0 && len(attribute.values) != vertices*attribute.each {
			return fmt.Errorf("mesh %s: %d values of %s for %d vertices, want %d each",
				m.Name, len(attribute.values), attribute.name, vertices, attribute.each)
		}
	}

	for _, index := range m.Indices {
		if int(index) >= vertices {
			return fmt.Errorf("mesh %s: a triangle uses vertex %d of %d", m.Name, index, vertices)
		}
	}

	return nil
}

// reader walks the file.
type reader struct {
	data   []byte
	offset int

	// taken is what the last take returned.
	taken []byte
}

func (r *reader) take(count int) bool {
	if count < 0 || r.offset+count > len(r.data) {
		return false
	}

	r.taken = r.data[r.offset : r.offset+count]
	r.offset += count

	return true
}

// object reads the start of an object: one bracket per level, then its name.
func (r *reader) object() (depth int, name string, err error) {
	for r.offset < len(r.data) && r.data[r.offset] == '[' {
		depth++
		r.offset++
	}

	name, err = r.text()

	return depth, name, err
}

// text reads a name, which runs to the next zero byte.
func (r *reader) text() (string, error) {
	for index := r.offset; index < len(r.data); index++ {
		if r.data[index] == 0 {
			name := string(r.data[r.offset:index])
			r.offset = index + 1

			return name, nil
		}
	}

	return "", fmt.Errorf("mesh: a name at %d runs to the end of the file", r.offset)
}

// property reads one property: its name, its type and its values. Numbers come
// back as values, whole numbers also as indices, and strings are read but
// handed back as nothing, since nothing here wants them.
func (r *reader) property() (name string, values []float32, indices []uint16, err error) {
	r.offset++ // the exclamation mark

	if !r.take(1) {
		return "", nil, nil, fmt.Errorf("mesh: a property at %d has no name", r.offset)
	}

	if !r.take(int(r.taken[0])) {
		return "", nil, nil, fmt.Errorf("mesh: a property name at %d runs to the end of the file", r.offset)
	}

	name = string(r.taken)

	if !r.take(1) {
		return "", nil, nil, fmt.Errorf("mesh: property %s has no type", name)
	}

	kind := r.taken[0]

	count, err := r.number()
	if err != nil {
		return "", nil, nil, fmt.Errorf("mesh: property %s: %w", name, err)
	}

	switch kind {
	case 'f':
		if !r.take(int(count) * 4) {
			return "", nil, nil, fmt.Errorf("mesh: property %s wants %d numbers the file does not hold", name, count)
		}

		values = make([]float32, count)
		for index := range values {
			values[index] = math.Float32frombits(binary.LittleEndian.Uint32(r.taken[index*4:]))
		}

	case 'i':
		if !r.take(int(count) * 4) {
			return "", nil, nil, fmt.Errorf("mesh: property %s wants %d whole numbers the file does not hold", name, count)
		}

		indices = make([]uint16, count)
		for index := range indices {
			indices[index] = uint16(binary.LittleEndian.Uint32(r.taken[index*4:]))
		}

	case 's':
		// A string property is a count of strings, each a length and that
		// many bytes, the last of them a zero.
		for range count {
			length, err := r.number()
			if err != nil {
				return "", nil, nil, fmt.Errorf("mesh: property %s: %w", name, err)
			}

			if !r.take(int(length)) {
				return "", nil, nil, fmt.Errorf("mesh: property %s wants %d bytes the file does not hold", name, length)
			}
		}

	default:
		return "", nil, nil, fmt.Errorf("mesh: property %s is of unknown type %q", name, kind)
	}

	return name, values, indices, nil
}

// number reads one whole number, which is how the file counts things.
func (r *reader) number() (uint32, error) {
	if !r.take(4) {
		return 0, fmt.Errorf("a count at %d runs to the end of the file", r.offset)
	}

	return binary.LittleEndian.Uint32(r.taken), nil
}
