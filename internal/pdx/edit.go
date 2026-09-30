package pdx

// Edits the editor makes to a coat of arms. They are plain functions over the
// model so that they can be tested without an interface, and the interface
// only decides when to call them.

// Limits the editor keeps values within. They match the ranges the Odin
// version offered, which are wide enough for everything the games ship.
const (
	MinPosition = -2
	MaxPosition = 3
	MinScale    = -3
	MaxScale    = 3
	MaxRotation = 360
	MaxMask     = 3
)

// MoveItem moves the element at index by delta places, clamped to the ends of
// the slice. It returns the element's new index, and whether it moved at all.
func MoveItem[T any](items []T, index, delta int) (int, bool) {
	if index < 0 || index >= len(items) {
		return index, false
	}

	target := min(max(index+delta, 0), len(items)-1)
	if target == index {
		return index, false
	}

	moved := items[index]

	if target < index {
		copy(items[target+1:index+1], items[target:index])
	} else {
		copy(items[index:target], items[index+1:target+1])
	}

	items[target] = moved

	return target, true
}

// RemoveItem removes the element at index, keeping the order of the rest.
func RemoveItem[T any](items []T, index int) []T {
	if index < 0 || index >= len(items) {
		return items
	}

	return append(items[:index], items[index+1:]...)
}

// RemoveColor empties a colour slot. It reports whether the slot was filled.
//
// Colours are the parser library's type, which is why this is a function
// rather than a method on them.
func RemoveColor(colors *Colors, slot string) bool {
	for index, color := range *colors {
		if color.Slot == slot {
			*colors = RemoveItem(*colors, index)

			return true
		}
	}

	return false
}

// NewColoredEmblem is a coloured emblem as the editor adds one.
//
// It has no colours: a layer that was just added has said nothing about its
// colours yet, so it shows the marker colours its texture is painted in until
// a slot is filled in, and nothing is written to the file that the author did
// not ask for.
func NewColoredEmblem(texture string) *ColoredEmblem {
	return &ColoredEmblem{Texture: texture}
}

// NewTexturedEmblem is a textured emblem as the editor adds one.
func NewTexturedEmblem(texture string) *TexturedEmblem {
	return &TexturedEmblem{Texture: texture}
}

// NewSubFlag is a sub flag layer as the editor adds one.
func NewSubFlag(parent string) *SubFlag {
	return &SubFlag{Parent: parent}
}
