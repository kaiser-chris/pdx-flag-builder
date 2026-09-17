package gui

import "github.com/AllenDang/cimgui-go/imgui"

// interfaceFontSize is the size the interface font is rasterised at. Dear
// ImGui's stock font is 13 pixels tall, which is cramped on a modern desktop
// display.
const interfaceFontSize = 16

// configureFonts replaces the default bitmap font with the scalable version
// Dear ImGui ships, which stays crisp at any size.
func configureFonts() {
	atlas := imgui.CurrentIO().Fonts()
	if atlas == nil {
		return
	}

	config := imgui.NewFontConfig()
	defer config.Destroy()

	config.SetSizePixels(interfaceFontSize)

	// The atlas copies the configuration, so releasing it afterwards is safe.
	atlas.AddFontDefaultVectorV(config)
}
