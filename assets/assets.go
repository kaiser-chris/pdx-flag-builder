// Package assets bundles every file the application needs at runtime into the
// executable, replacing the hand-maintained lookup table the Odin version kept
// in src/assets/assets.odin.
//
// Paths are relative to this directory, so "textures/logo.dds" refers to
// assets/textures/logo.dds.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed icon.png shaders textures
var files embed.FS

// Well known assets, referenced by name instead of by a raw path so that a
// typo is a compile error rather than a missing texture at runtime.
const (
	Icon = "icon.png"

	SplashLogo    = "textures/logo.dds"
	SplashSpinner = "textures/spinner.dds"
	Transparency  = "textures/transparency.dds"
	Invalid       = "textures/invalid.dds"

	IconSub       = "textures/icons/sub.dds"
	IconEdit      = "textures/icons/edit.dds"
	IconDelete    = "textures/icons/delete.dds"
	IconArrowUp   = "textures/icons/arrow_up.dds"
	IconArrowDown = "textures/icons/arrow_down.dds"
	IconUnknown   = "textures/icons/unknown.dds"
	IconGameVic3  = "textures/icons/game_vic3.dds"
	IconGameEu5   = "textures/icons/game_eu5.dds"

	ShaderRecolor = "shaders/recolor.fs"
)

// FS exposes the bundled files for callers that want to walk the tree, such as
// the texture database listing every icon it knows about.
func FS() fs.FS {
	return files
}

// Read returns the contents of a bundled asset.
func Read(path string) ([]byte, error) {
	data, err := files.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bundled asset %q: %w", path, err)
	}
	return data, nil
}

// MustRead returns the contents of a bundled asset and panics if it is
// missing. Every path handed to it is a compile time constant from this
// package, so a failure means the binary itself was built wrong.
func MustRead(path string) []byte {
	data, err := Read(path)
	if err != nil {
		panic(err)
	}
	return data
}

// The interface fonts are embedded as strings rather than through the file
// system above on purpose. Dear ImGui keeps the pointer to the font data for as
// long as its atlas lives, because it rasterises glyphs on demand. A string
// declared with go:embed is backed by memory in the binary itself, which is
// never moved and never freed; FS.ReadFile would hand out a garbage collected
// copy instead, and the atlas would end up reading freed memory.
//
// Only the two weights the interface uses are embedded. The rest of the family
// is kept in the repository for later use.
var (
	//go:embed fonts/roboto/Roboto-Regular.ttf
	FontRegular string

	//go:embed fonts/roboto/Roboto-Medium.ttf
	FontMedium string
)
