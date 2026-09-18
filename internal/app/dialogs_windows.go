package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ncruces/zenity"
)

// dialogParent makes the dialogs children of the application window, so that
// they open over it and block it while they are open, as dialogs do.
func dialogParent() zenity.Option {
	return zenity.Attach(uintptr(rl.GetWindowHandle()))
}
