package gui

import (
	"image/color"

	"github.com/AllenDang/cimgui-go/imgui"
)

// The palette. Dear ImGui's stock dark theme is a neutral grey that makes every
// panel look the same; these values give the application a flatter, slightly
// cooler look with one accent colour used for anything interactive.
var (
	colorText        = rgb(0xDFE3EA)
	colorTextDimmed  = rgb(0x7D8694)
	colorWindowBg    = rgb(0x1E2127)
	colorChildBg     = rgb(0x24282F)
	colorPopupBg     = rgb(0x1A1D22)
	colorBorder      = rgb(0x333A45)
	colorFrame       = rgb(0x2C313A)
	colorFrameHover  = rgb(0x363D48)
	colorFrameActive = rgb(0x3F4753)
	colorAccent      = rgb(0x3D6FB5)
	colorAccentHover = rgb(0x4A82CF)
	colorAccentDown  = rgb(0x345F9C)
	colorTitleBg     = rgb(0x181B20)
	colorTabDimmed   = rgb(0x21252B)
)

// ApplyTheme installs the application's colours and metrics on the current
// Dear ImGui style.
func ApplyTheme() {
	// Start from the stock dark theme so that any colour this function does not
	// set explicitly is still a sane dark value.
	imgui.StyleColorsDark()

	style := imgui.CurrentStyle()

	style.SetWindowPadding(imgui.Vec2{X: 10, Y: 10})
	style.SetFramePadding(imgui.Vec2{X: 8, Y: 4})
	style.SetItemSpacing(imgui.Vec2{X: 8, Y: 6})
	style.SetWindowTitleAlign(imgui.Vec2{X: 0, Y: 0.5})
	style.SetWindowMenuButtonPosition(imgui.DirNone)

	style.SetWindowRounding(6)
	style.SetChildRounding(4)
	style.SetPopupRounding(4)
	style.SetFrameRounding(4)
	style.SetGrabRounding(4)
	style.SetTabRounding(4)
	style.SetScrollbarSize(12)

	style.SetWindowBorderSize(1)
	style.SetFrameBorderSize(0)

	colors := style.Colors()

	colors[imgui.ColText] = colorText
	colors[imgui.ColTextDisabled] = colorTextDimmed
	colors[imgui.ColWindowBg] = colorWindowBg
	colors[imgui.ColChildBg] = colorChildBg
	colors[imgui.ColPopupBg] = colorPopupBg
	colors[imgui.ColBorder] = colorBorder
	colors[imgui.ColBorderShadow] = transparent()

	colors[imgui.ColFrameBg] = colorFrame
	colors[imgui.ColFrameBgHovered] = colorFrameHover
	colors[imgui.ColFrameBgActive] = colorFrameActive

	colors[imgui.ColTitleBg] = colorTitleBg
	colors[imgui.ColTitleBgActive] = colorTitleBg
	colors[imgui.ColTitleBgCollapsed] = colorTitleBg
	colors[imgui.ColMenuBarBg] = colorTitleBg

	colors[imgui.ColScrollbarBg] = transparent()
	colors[imgui.ColScrollbarGrab] = colorFrameHover
	colors[imgui.ColScrollbarGrabHovered] = colorFrameActive
	colors[imgui.ColScrollbarGrabActive] = colorAccent

	colors[imgui.ColCheckMark] = colorAccentHover
	colors[imgui.ColSliderGrab] = colorAccent
	colors[imgui.ColSliderGrabActive] = colorAccentHover

	colors[imgui.ColButton] = colorFrame
	colors[imgui.ColButtonHovered] = colorFrameHover
	colors[imgui.ColButtonActive] = colorAccentDown

	colors[imgui.ColHeader] = colorFrame
	colors[imgui.ColHeaderHovered] = colorAccentDown
	colors[imgui.ColHeaderActive] = colorAccent

	colors[imgui.ColSeparator] = colorBorder
	colors[imgui.ColSeparatorHovered] = colorAccent
	colors[imgui.ColSeparatorActive] = colorAccentHover

	colors[imgui.ColResizeGrip] = transparent()
	colors[imgui.ColResizeGripHovered] = colorFrameHover
	colors[imgui.ColResizeGripActive] = colorAccent

	colors[imgui.ColTab] = colorTabDimmed
	colors[imgui.ColTabHovered] = colorFrameActive
	colors[imgui.ColTabSelected] = colorChildBg
	colors[imgui.ColTabDimmed] = colorTabDimmed
	colors[imgui.ColTabDimmedSelected] = colorChildBg

	colors[imgui.ColDockingPreview] = withAlpha(colorAccent, 0.45)
	colors[imgui.ColDockingEmptyBg] = colorWindowBg

	colors[imgui.ColTableHeaderBg] = colorFrame
	colors[imgui.ColTableBorderStrong] = colorBorder
	colors[imgui.ColTableBorderLight] = withAlpha(colorBorder, 0.5)
	colors[imgui.ColTableRowBg] = transparent()
	colors[imgui.ColTableRowBgAlt] = withAlpha(colorChildBg, 0.4)

	colors[imgui.ColTextSelectedBg] = withAlpha(colorAccent, 0.5)

	style.SetColors(&colors)
}

// rgb turns a 0xRRGGBB literal into an opaque Dear ImGui colour.
func rgb(hex uint32) imgui.Vec4 {
	return imgui.Vec4{
		X: float32((hex>>16)&0xFF) / 255,
		Y: float32((hex>>8)&0xFF) / 255,
		Z: float32(hex&0xFF) / 255,
		W: 1,
	}
}

func withAlpha(color imgui.Vec4, alpha float32) imgui.Vec4 {
	color.W = alpha
	return color
}

func transparent() imgui.Vec4 {
	return imgui.Vec4{}
}

// clearColor is the colour the window is cleared to every frame, the one the
// panels have, so that the interface has no seams between them.
func clearColor() color.RGBA {
	channel := func(value float32) uint8 { return uint8(value*255 + 0.5) }

	return color.RGBA{R: channel(colorWindowBg.X), G: channel(colorWindowBg.Y), B: channel(colorWindowBg.Z), A: 255}
}
