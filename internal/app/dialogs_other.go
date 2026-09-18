//go:build !windows

package app

import "github.com/ncruces/zenity"

// dialogParent asks for a modal dialog. Attaching it to the application window
// would need the window's X11 id, and raylib hands out its GLFW handle
// instead, so the dialog cannot name its parent on Linux.
func dialogParent() zenity.Option {
	return zenity.Modal()
}
