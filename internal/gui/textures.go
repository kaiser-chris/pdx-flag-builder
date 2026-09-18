package gui

// Dear ImGui 1.92 keeps its own textures, the font atlas among them, in a list
// the backend is meant to service each frame. The raylib backend walks that
// list through cimgui-go's broken accessor (see cvector.go) and crashes once
// it holds two textures. Two textures is not rare: it is what the font atlas
// looks like for a frame whenever it grows, which changing the interface scale
// does. So the window services the list itself and hides it from the backend
// while the backend renders.

import (
	"github.com/AllenDang/cimgui-go/imgui"
)

// serviceTextures creates, updates and destroys the textures Dear ImGui asks
// for, then empties the list for the backend. The returned function puts the
// list back and has to be called once the backend has rendered.
func (w *Window) serviceTextures(drawData *imgui.DrawData) (restore func()) {
	textures, hide := drawDataTextures(drawData)

	for _, texture := range textures {
		w.serviceTexture(texture)
	}

	return hide()
}

func (w *Window) serviceTexture(texture *imgui.TextureData) {
	switch texture.Status() {
	case imgui.TextureStatusWantCreate:
		w.uploadTexture(texture)

	case imgui.TextureStatusWantUpdates:
		// Uploading the whole texture again is simpler than patching the
		// changed rectangles, and the atlas changes rarely enough for it.
		w.releaseTexture(texture)
		w.uploadTexture(texture)

	case imgui.TextureStatusWantDestroy:
		// Dear ImGui asks for a texture to go while it may still be drawn this
		// frame, and says so by counting the frames it has gone unused.
		if texture.UnusedFrames() > 0 {
			w.releaseTexture(texture)
			texture.SetStatus(imgui.TextureStatusDestroyed)
		}
	}
}

func (w *Window) uploadTexture(texture *imgui.TextureData) {
	// The backend asks Dear ImGui for four bytes per pixel, which is what
	// CreateTexture takes.
	reference := w.backend.CreateTexture(
		cPointer(texture.Pixels()),
		int(texture.Width()),
		int(texture.Height()),
	)

	texture.SetTexID(reference.TexID())
	texture.SetStatus(imgui.TextureStatusOK)
}

func (w *Window) releaseTexture(texture *imgui.TextureData) {
	if id := texture.TexID(); id != 0 {
		w.backend.DeleteTexture(*imgui.NewTextureRefTextureID(id))
		texture.SetTexID(0)
	}
}
