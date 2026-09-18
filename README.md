# pdx-flag-builder (Go)

A rewrite of [pdx-flag-builder](https://github.com/kaiser-chris/pdx-flag-builder) in Go.
The original is written in Odin; this repository ports it while keeping the parts
that were worth keeping and replacing the parts that were not.

> **Status:** early. The application runs on Windows and Linux, reads the coat
> of arms files of a configured game or mod folder, lets you browse the flags and
> textures it found, draws the flag you open, edits it with undo, and saves it
> back to script files or exports it as an image.

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
internal/gui           window bootstrap, theme, fonts, scaling and recorded widgets
internal/uitest        the driver the interface tests click through
internal/render        flag rendering with raylib
internal/texture       reading the games' image files, including Targa and BC7
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

### Drawing a flag

`internal/texture` turns the game's image files into textures. raylib reads PNG
and the older DDS compressions itself and uploads them to the GPU still
compressed. Two formats are decoded in Go instead: Targa, because the raylib
build raylib-go ships leaves that reader out, and BC7, which raylib has no pixel
format for at all. The BC7 partition tables are generated from the `bcdec`
header the Odin version vendored.

`internal/render` composes the flag: the pattern first, then each layer over
it, through a fragment shader that swaps the marker colours the textures are
painted in for the colours the coat of arms asks for, including masks and sub
flags. Rotated emblems reuse the Odin version's approximation of how the games
resize them, which has not been checked against the games yet. Textures are read
on a background goroutine and uploaded a few per frame, so opening a flag never
stalls the window.

The previews in the lists are drawn once each into cells of one large render
target rather than every frame, from a texture cache of their own that only
keeps the most recently used textures. Scrolling through all 1700 flags of a
game therefore never keeps every texture they use on the GPU.

The shader picks the marker colour a pixel is *closest* to, where the Odin
version took the first one within tolerance. That difference removes a line of
raw marker colour that used to show along seams in the pattern.

Settings live in the user's configuration directory
(`%AppData%\pdx-flag-builder` on Windows, `~/.config/pdx-flag-builder` on Linux)
in the same format the Odin version wrote, so an existing installation keeps its
configured folders. The window layout is stored next to it and can be reset from
**View → Reset Layout**.

### Scaling

On a high resolution display the whole interface is drawn larger: text, padding,
windows and the flag preview alike. **Settings → Interface Scale** follows the
monitor's scale by default (**Automatic**, which tracks the window from one
monitor to another) or can be fixed anywhere from 100% to 300%. A change shows
straight away and is kept once the settings are saved.

Text stays sharp at any scale, because Dear ImGui 1.92 rasterises glyphs at the
size they are drawn at. Sizes are always worked out from the unscaled style, so
switching back and forth does not make the layout drift.

Two lists Dear ImGui keeps, its textures and its windows, are read through a
small cgo file (`internal/gui/cvector.go`) rather than cimgui-go's generated
accessors. Those wrap only the first element of a list of pointers, and the
raylib backend crashes on them the moment the font atlas grows, which is what
changing the scale does. The window services the textures itself instead.

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
**Databases → Texture Database**, each with a small preview: the rendered flag,
or the texture as the file has it, along with its size. Clicking a column
header sorts the list, and the part of each name a search matched is marked.
Picking a flag opens it and shows its layers.

Layers are added, reordered and removed in the **Layers** panel, and the
selected one is edited in the panel next to it: its texture, colours, mask and
placements. **File → New Flag** starts from an empty one, and a texture can be
used straight from the texture database as the pattern or as a new layer.
Wherever a texture or a coat of arms is chosen, the list says which folder each
one comes from, so a texture a mod replaces shows up once for the game and once
for the mod.
A placement is changed by dragging its fields, or by typing into them after a
double click. Clicking the flag hands it the arrow keys, as in the Odin
version: they move the selected placement, with Ctrl they scale it and with
Alt, left and right turn it; Shift makes each step ten times bigger. The
placements of a layer can be put in a different order, which decides which is
drawn on top where they overlap.

**Ctrl+Z** and **Ctrl+Y** undo and redo, one step per edit: a whole drag of a
slider is one step, not one per frame. Holding an arrow key down is one step too.

### Saving and exporting

- **File → Save** (`Ctrl+S`) writes the flag back into the file it was read
  from. Only its own definition is replaced, where it stands: comments,
  variables, the other flags, line breaks and the byte order mark are left as
  they were. A renamed flag replaces its old definition.
- **File → Save To File...** (`Ctrl+Shift+S`) puts the flag into a file of
  your choosing: in place of a definition of the same name if the file has
  one, at its end otherwise, and a new file is created with a byte order mark,
  as the games expect. The flag belongs to that file from then on. Saving a
  flag that never had a file does the same.
- **File → Copy Script** puts the flag's script on the clipboard.
- **File → Export Image...** writes the flag as a 768 by 512 PNG with a
  transparent background.

The script is laid out the way the games' files are, one placement per line.
Values the file spelled with `@variables` or `@[expressions]` are written as the
numbers they came to, since that is all that is left of them once read.

Opening another flag, **File → Exit** and closing the window all ask first
when there are unsaved changes, and offer to save them when the flag already
has a file. Files and folders are
picked in the system's own dialogs through
[zenity](https://github.com/ncruces/zenity), which needs no C libraries: on
Linux it runs `zenity` or `kdialog`, whichever the desktop has.

## Testing

```bash
go test ./...
```

Five of the tests read a real installation instead of a fixture, because the
only way to find out what the files really contain is to read the real ones:
they parse, decode, write back and merge every coat of arms, and decode every
coat of arms texture. They are skipped unless you point them at a game or mod
folder of either game:

```bash
PDX_GAME_DIR="/path/to/Victoria 3/game" go test ./... -v
```

Setting `PDX_DUMP_DIR` as well makes the texture test write every Targa and BC7
file it decodes out as a PNG, which is the only way to see whether the decoders
are right.

### Interface tests

```bash
make uitest
```

runs the real application in a hidden window and drives it the way a user
would: it opens menus, clicks buttons, types, drags sliders and presses
shortcuts, then checks the application's state and the pixels of the rendered
flag. The same tag covers the renderer's tests that need an OpenGL context,
such as the thumbnail atlas and the texture cache. They sit behind the
`uitest` build tag (`go test -tags uitest ./...`) because they need a display,
which a headless CI runner does not have; on Linux, `xvfb-run` provides one.

The widgets the application uses come from `internal/gui`, which reports each
one it lays out (its label, window, rectangle and visible area) to the
driver in `internal/uitest`. That is how a test finds "Save" in the settings
window without knowing where it is, and how it notices when a widget has been
scrolled out of view or pushed off the edge of the window. Like a user, the
driver scrolls a widget into view before it clicks it.

## What still has to be ported

- A Windows resource `.syso` so the executable carries `icon.ico`, replacing the
  `resources.rc` the Odin build used

## Credit

The application shows these in **Help → About** as well.

- The application icon is waving flag by Suncheli Project from
  [Noun Project](https://thenounproject.com/browse/icons/term/waving-flag/)
  (CC BY 3.0).
- The interface is set in [Roboto](https://github.com/googlefonts/roboto-classic),
  designed by Christian Robertson, copyright 2011 The Roboto Project Authors,
  under the SIL Open Font License 1.1, which is in
  [assets/fonts/roboto/OFL.txt](assets/fonts/roboto/OFL.txt). Roboto is a
  trademark of Google.
- The BC7 decoder follows [bcdec](https://github.com/iOrange/bcdec) by iOrange,
  which the Odin version vendored.
