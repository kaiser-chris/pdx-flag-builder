// Command pdx-flag-builder is a flag editor for Victoria 3 and Europa
// Universalis 5.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/app"
)

func init() {
	// raylib creates the window and the OpenGL context on the calling thread
	// and both have to stay there, so the main goroutine is pinned before
	// anything else runs.
	runtime.LockOSThread()
}

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "pdx-flag-builder: %v\n", err)
		os.Exit(1)
	}
}
