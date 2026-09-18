# PDX Flag Builder

A tool for building flags for Victoria 3 and Europa Universalis 5. It reads the
coats of arms, patterns and emblems of the game and of your mods, draws a flag
exactly as its script describes it, and lets you edit it and save it back to the
game's script files.

It runs on Windows and Linux.

## Usage

### Installation

Download the zip for your system from the
[releases page](https://github.com/kaiser-chris/pdx-flag-builder/releases),
extract it anywhere and run `pdx-flag-builder`.

On Linux the file dialogs need `zenity` or `kdialog`, one of which most desktops
already have.

### Setup

The tool reads its flags and textures from the game and from your mods:

1. Open **Settings** (`Ctrl+,`).
2. Click **Add Folder** and then **Browse...**, and pick the game's `game` folder
   or the root folder of a mod.
3. Add as many folders as you like. A folder further down the list overrides
   the ones above it, the way a mod overrides the game.
4. Click **Save**.

### Finding a flag

**Databases → Flag Database** lists every coat of arms that was found, with a
preview. **Databases → Texture Database** lists the patterns and emblems. Type
into the search box to narrow a list down, and click a column header to sort it.

Click a flag to open it. **File → New Flag** starts from an empty one.

### Editing

The **Layers** panel lists the coat of arms and its layers. Click one to edit it
in the **Selected Layer** panel:

- The coat of arms has its name, its pattern and its colours.
- An emblem has its texture, its colours, a mask that limits it to one colour of
  the pattern, and its placements: position, scale and rotation.
- A sub flag draws another coat of arms, with an offset and a scale.

Add layers with **Add Layer**, or straight from the texture database with
**Add as Layer** and **Set as Pattern**, or from the flag database with
**Add as Sub Flag**. The arrows next to a layer move it up and down; the layers
further down are drawn on top.

Number fields are changed by dragging across them. Double-click one to type a
number instead. Hold **Shift** while dragging for bigger steps, **Alt** for
finer ones.

To move a placement with the keyboard, click the flag and use the arrow keys:

| Keys                       | Effect                      |
|----------------------------|-----------------------------|
| Arrow keys                 | Move the selected placement |
| **Ctrl** + arrow keys      | Scale it                    |
| **Alt** + left and right   | Turn it                     |
| **Shift** with any of them | Ten times bigger steps      |

**Ctrl+Z** undoes and **Ctrl+Y** redoes.

### Saving and exporting

| Menu entry                                   | Effect |
|----------------------------------------------|--------|
| **File → Save** (`Ctrl+S`)                   | Writes the flag back into the file it came from. Only its own definition changes; the rest of the file stays as it is. |
| **File → Save To File...** (`Ctrl+Shift+S`)  | Puts the flag into a file of your choosing, replacing a flag of the same name in it or adding it at the end. |
| **File → Copy Script**                       | Copies the flag's script to the clipboard. |
| **File → Export Image...**                   | Saves the flag as a 768 × 512 PNG image. |

Values a file wrote with `@variables` or `@[expressions]` are saved as the
numbers they came to.

The tool asks before it throws away unsaved changes.

### Settings

**Settings → Interface Scale** makes the whole interface larger for high
resolution displays. **View → Reset Layout** puts the panels back where they
started.

Settings are stored in `%AppData%\pdx-flag-builder` on Windows and in
`~/.config/pdx-flag-builder` on Linux.

## Building

The tool is written in Go and draws with [raylib](https://www.raylib.com/) and
[Dear ImGui](https://github.com/ocornut/imgui). Both are C and C++ libraries that
are compiled along with it, so a C and C++ compiler is needed as well as Go.

The first build compiles raylib and Dear ImGui and takes a few minutes. Later
builds use Go's build cache and take seconds.

### Windows

1. Install [Go](https://go.dev/dl/), in the version `go.mod` asks for or newer.
2. Install a MinGW-w64 toolchain, for example
   [w64devkit](https://github.com/skeeto/w64devkit) or the one that comes with
   [MSYS2](https://www.msys2.org/), and put its `bin` folder on `PATH` so that
   `gcc` can be found.
3. Build:

   ```bat
   build.bat
   ```

   The executable lands in `bin\windows`.

### Linux

1. Install [Go](https://go.dev/dl/), in the version `go.mod` asks for or newer.
2. Install a compiler and the OpenGL, X11 and Wayland development files. On
   Debian and Ubuntu:

   ```bash
   sudo apt-get install build-essential libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev
   ```

3. Build:

   ```bash
   ./build.sh
   ```

   The executable lands in `bin/linux`.

### Makefile

With `make` installed, the `Makefile` covers the everyday tasks:

| Command        | Effect |
|----------------|--------|
| `make build`   | Builds a development executable into `bin/` |
| `make run`     | Builds it and starts it |
| `make release` | Builds an optimised executable, without a console window on Windows |
| `make test`    | Runs the tests |
| `make uitest`  | Runs the interface tests as well, which open a hidden window and so need a display |
| `make vet`     | Checks the code with `go vet` |

The tests that read a real installation run when `PDX_GAME_DIR` points at a game
or mod folder:

```bash
PDX_GAME_DIR="/path/to/Victoria 3/game" go test ./...
```
