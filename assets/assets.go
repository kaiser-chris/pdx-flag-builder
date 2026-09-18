// Package assets bundles every file the application needs at runtime into the
// executable, replacing the hand-maintained lookup table the Odin version kept
// in src/assets/assets.odin.
//
// Paths are relative to this directory, so "shaders/recolor.fs" refers to
// assets/shaders/recolor.fs.
package assets

import (
	"embed"
	"fmt"
)

//go:embed icon.png shaders
var files embed.FS

// Well known assets, referenced by name instead of by a raw path so that a
// typo is a compile error rather than a missing file at runtime.
const (
	Icon          = "icon.png"
	ShaderRecolor = "shaders/recolor.fs"
)

// Read returns the contents of a bundled asset.
func Read(path string) ([]byte, error) {
	data, err := files.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bundled asset %q: %w", path, err)
	}

	return data, nil
}

// The interface fonts are embedded as strings rather than through the file
// system above on purpose. Dear ImGui keeps the pointer to the font data for as
// long as its atlas lives, because it rasterises glyphs on demand. A string
// declared with go:embed is backed by memory in the binary itself, which is
// never moved and never freed; FS.ReadFile would hand out a garbage collected
// copy instead, and the atlas would end up reading freed memory.
var (
	//go:embed fonts/roboto/Roboto-Regular.ttf
	FontRegular string

	//go:embed fonts/roboto/Roboto-Medium.ttf
	FontMedium string
)
