package gui

import (
	"unsafe"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/assets"
)

// interfaceFontSize is the size the interface is drawn at. Dear ImGui's stock
// font is 13 pixels tall, which is cramped on a modern desktop display.
const interfaceFontSize = 16

// The icons of Material Symbols, by the code point the font maps them to.
// Each is a string rather than a rune, since that is what a label is.
const (
	IconCopy    = "\ue14d"
	IconDelete  = "\ue872"
	IconDown    = "\ue5db"
	IconLink    = "\ue157"
	IconLinkOff = "\ue16f"
	IconUp      = "\ue5d8"
)

// iconFontSize is the size the icons are baked at when they stand among text.
// An icon drawn on its own is drawn at the size of whatever it is drawn in,
// which is what drawIcon does.
const iconFontSize = interfaceFontSize

// strongFont is the heavier weight used for headings. It is kept here rather
// than passed around because there is exactly one interface.
var strongFont *imgui.Font

// configureFonts replaces Dear ImGui's built in font with Roboto.
func configureFonts() {
	atlas := imgui.CurrentIO().Fonts()
	if atlas == nil {
		return
	}

	addFont(atlas, assets.FontRegular)
	addIcons(atlas)

	strongFont = addFont(atlas, assets.FontMedium)
}

// addIcons folds the icon font into the one added before it, so that an icon
// can be drawn wherever text can, as the code point it is mapped to.
func addIcons(atlas *imgui.FontAtlas) {
	config := imgui.NewFontConfig()
	defer config.Destroy()

	config.SetFontDataOwnedByAtlas(false)
	config.SetMergeMode(true)

	atlas.AddFontFromMemoryTTFV(
		uintptr(unsafe.Pointer(unsafe.StringData(assets.FontIcons))),
		int32(len(assets.FontIcons)),
		iconFontSize,
		config,
		nil,
	)
}

func addFont(atlas *imgui.FontAtlas, ttf string) *imgui.Font {
	config := imgui.NewFontConfig()
	defer config.Destroy()

	// The data lives in the binary and outlives the atlas, so Dear ImGui must
	// not try to free it.
	config.SetFontDataOwnedByAtlas(false)

	return atlas.AddFontFromMemoryTTFV(
		uintptr(unsafe.Pointer(unsafe.StringData(ttf))),
		int32(len(ttf)),
		interfaceFontSize,
		config,
		nil,
	)
}

// PushStrongFont switches to the heavier weight until PopFont is called. It is
// meant for headings and other short, emphasised runs of text.
func PushStrongFont() {
	if strongFont == nil {
		return
	}

	// A size of zero keeps whatever size is currently in effect.
	imgui.PushFont(strongFont, 0)
}

// PopFont undoes the most recent PushStrongFont.
func PopFont() {
	if strongFont == nil {
		return
	}

	imgui.PopFont()
}
