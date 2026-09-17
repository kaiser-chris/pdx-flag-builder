# pdx-flag-builder (Go)

A rewrite of [pdx-flag-builder](https://github.com/kaiser-chris/pdx-flag-builder) in Go.
The original is written in Odin; this repository ports it while keeping the parts
that were worth keeping and replacing the parts that were not.

> **Status:** early. The application shell runs on Windows and Linux, reads the
> coat of arms files of a configured game or mod folder, and lets you browse the
> flags and textures it found. Drawing a flag, editing one and exporting it are
> not ported yet.

## How it is put together

Two decisions shape the whole code base.

**raylib keeps drawing the flag.** Flags are composed on the GPU from pattern and
emblem textures with a recolouring fragment shader (`assets/shaders/recolor.fs`).
That pipeline is the valuable part of the original and it ports across almost
unchanged, so the Go version keeps using raylib through
[raylib-go](https://github.com/gen2brain/raylib-go).

**The interface was rewritten rather than ported.** The Odin version used
microui, which meant hand writing dropdowns, tables, toasts and colour pickers.
The Go version uses [Dear ImGui](https://github.com/ocornut/imgui) through
[cimgui-go](https://github.com/AllenDang/cimgui-go), which brings docking,
sortable tables, modal dialogs and a real colour picker along for free.

Dear ImGui and raylib share one window: cimgui-go ships a raylib backend, so
raylib owns the window, the OpenGL context and the input queue, and Dear ImGui
draws through raylib's immediate mode layer. The flag is rendered into an
offscreen target each frame and the preview panel samples it as an image.

```
cmd/pdx-flag-builder   entry point
assets                 files bundled into the binary with go:embed
internal/app           the shell: panels, menus, windows and the state they share
internal/gui           window bootstrap, theme and fonts
internal/render        flag rendering with raylib
internal/config        settings and layout persistence
internal/database      reading a game or mod folder
internal/pdx           the coat of arms model
internal/pdx/script    the parser for Paradox script files
```

### Reading the game files

`internal/pdx/script` parses the script language the games store their data in:
blocks, lists, repeated keys, tagged values such as `hsv360 { 0 0 5 }`, and the
variables and arithmetic that coat of arms files use to keep proportions
readable (`@third = @[1/3]`, `scale = { @third 0.5 }`).

Two things about it are worth knowing. It **evaluates** variables and
expressions rather than treating them as zero, which the flags with
variable-driven scales depend on. And it is **lenient about junk**: the shipped
game files contain the odd typo, such as a stray bracket in the middle of a
position, and the games read those files anyway, so an unreadable character is
skipped and reported instead of costing every flag in the file.

Reading a folder happens on its own goroutine and the result is handed to the
interface through a channel, so a full game folder — around 1700 flags and 1000
textures — loads without the window ever stalling.

Settings live in the user's configuration directory
(`%AppData%\pdx-flag-builder` on Windows, `~/.config/pdx-flag-builder` on Linux)
in the same format the Odin version wrote, so an existing installation keeps its
configured folders. The window layout is stored next to it and can be reset from
**View → Reset Layout**.

## Building

Both dependencies use cgo, so a C and C++ compiler is required.

### Windows

Install [Go](https://go.dev/dl/) and a MinGW-w64 toolchain (for example
[w64devkit](https://github.com/skeeto/w64devkit) or MSYS2), make sure `gcc` is on
`PATH`, then run:

```bat
build.bat
```

### Linux

Install Go, a compiler and the X11 and OpenGL development headers:

```bash
sudo apt-get install build-essential xorg-dev libgl1-mesa-dev libasound2-dev
./build.sh
```

The binary, with all assets bundled in, lands in `bin/`. A `Makefile` wraps the
usual tasks (`make build`, `make run`, `make release`, `make vet`).

> The first build compiles raylib and Dear ImGui from source and takes a few
> minutes. Later builds use the Go build cache and take seconds.

## Using it

1. Open **Settings** (`Ctrl+,`)
2. Add a folder and point it at a game or mod folder
3. Save

The flags and textures it found are then in **Databases → Flag Database** and
**Databases → Texture Database**. Picking a flag opens it and shows its layers.

## Testing

```bash
go test ./...
```

Two of the tests read a real installation instead of a fixture, because the only
way to find out what the files really contain is to read the real ones. They are
skipped unless you point them at a folder:

```bash
PDX_GAME_DIR="/path/to/Victoria 3/game" go test ./... -v
```

## What still has to be ported

- DDS and BC7 texture loading, including the `bcdec` decoder
- The recolouring shader wrapper and the layer compositing, so the preview draws
  the flag that is open instead of a placeholder
- Editing: adding, reordering and removing layers, and the colour pickers
- The exporters: script, image and clipboard
- A cross platform replacement for `nativefiledialog` so folders can be picked
  instead of typed
- A Windows resource `.syso` so the executable carries `icon.ico`, replacing the
  `resources.rc` the Odin build used

## Credit

This repository inherits the original's credits; see the
[upstream README](https://github.com/kaiser-chris/pdx-flag-builder#credit) for the
icon and library attributions.
