package app

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/render"
	"github.com/kaiser-chris/pdx-parser-go/script"
)

// Labels of the File menu's save and export actions.
const (
	labelSave        = "Save"
	labelSaveToFile  = "Save To File..."
	labelCopyScript  = "Copy Script"
	labelExportImage = "Export Image..."
)

// The folder coats of arms live in, inside a game or mod folder.
var coatOfArmsFolder = filepath.Join("common", "coat_of_arms", "coat_of_arms")

// imageExport is an image waiting for its flag's textures before it can be
// drawn and written.
type imageExport struct {
	flag pdx.Flag
	path string
}

// save writes the open flag back into the file it came from. A flag that has
// no file yet is saved the way Save To File does it.
func (a *App) save() {
	flag := a.state.flag
	if flag == nil {
		return
	}

	if flag.Origin.Path == "" {
		a.saveToFile()

		return
	}

	a.writeFlag(flag.Origin.Path, flag.Origin.Key)
}

// saveToFile asks for a coat of arms file and puts the open flag into it: in
// place of a definition of the same name if the file has one, at its end
// otherwise, and a file that does not exist yet is created. The flag belongs
// to that file from then on.
func (a *App) saveToFile() {
	flag := a.state.flag
	if flag == nil || !a.checkName(flag.Name) {
		return
	}

	request := fileRequest{
		title:  "Save " + flag.Name + " To File",
		start:  a.scriptStart(flag),
		filter: scriptFiles,
	}

	a.ask(request, func(path string) {
		current := a.state.flag
		if current == nil {
			return
		}

		path = withExtension(path, ".txt")

		// Back into its own file, the flag replaces the definition it was read
		// from, whatever it is called now. Anywhere else it goes by its name.
		key := current.Name
		if samePath(path, current.Origin.Path) {
			key = current.Origin.Key
		}

		a.writeFlag(path, key)
	})
}

// writeFlag puts the open flag into a file, replacing the definition read
// under key.
func (a *App) writeFlag(path, key string) {
	flag := a.state.flag
	if flag == nil || !a.checkName(flag.Name) {
		return
	}

	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		a.saveFailed(flag.Name, err)

		return
	}

	merged, replaced, err := pdx.Merge(string(existing), *flag, key)
	if err != nil {
		a.saveFailed(flag.Name, err)

		return
	}

	if err := writeFileAtomically(path, []byte(merged)); err != nil {
		a.saveFailed(flag.Name, err)

		return
	}

	flag.Origin = pdx.Origin{
		Database: a.databaseOf(path),
		File:     filepath.Base(path),
		Path:     path,
		Key:      flag.Name,
	}
	a.state.modified = false

	verb := "Added"
	if replaced {
		verb = "Saved"
	}

	// The folders are read again so that the lists, and any flag using this
	// one as a sub flag, show what was just saved. Reading them reports
	// itself in the status bar, which would hide that the save worked.
	a.state.statusAfterLoad = fmt.Sprintf("%s %s to %s", verb, flag.Name, filepath.Base(path))
	a.setStatus("%s", a.state.statusAfterLoad)
	a.state.library.reload(a.settings.Databases)
}

func (a *App) saveFailed(name string, err error) {
	warn(err)
	a.setStatus("%s could not be saved: %v", name, err)
}

// checkName refuses to save a flag under a name the games could not read back
// as the key of a definition.
func (a *App) checkName(name string) bool {
	if script.IsKey(name) {
		return true
	}

	a.setStatus("%q cannot be saved: a name starts with a letter or an underscore and has no spaces", name)

	return false
}

// copyScript puts the open flag's script on the clipboard, for pasting into a
// file by hand.
func (a *App) copyScript() {
	flag := a.state.flag
	if flag == nil {
		return
	}

	rl.SetClipboardText(pdx.Script(*flag, "\n"))
	a.setStatus("Copied the script of %s", flag.Name)
}

// exportImage asks for a file and writes the open flag to it as a PNG image
// at the canvas's own size.
func (a *App) exportImage() {
	flag := a.state.flag
	if flag == nil {
		return
	}

	request := fileRequest{
		title:            "Export " + flag.Name + " As Image",
		start:            a.imageStart(flag),
		filter:           imageFiles,
		confirmOverwrite: true,
	}

	a.ask(request, func(path string) {
		if a.state.flag == nil {
			return
		}

		// The flag as it is when the file was chosen, whatever happens to it
		// while its textures load.
		a.state.export = &imageExport{flag: a.state.flag.Clone(), path: withExtension(path, ".png")}
		a.setStatus("Exporting %s", a.state.flag.Name)
	})
}

// finishExport draws and writes a waiting image export once every texture
// its flag needs has arrived. It runs with raylib drawing active.
func (a *App) finishExport() {
	export := a.state.export
	if export == nil || !a.painter.Ready(export.flag) {
		return
	}

	a.state.export = nil

	picture := render.Export(a.painter, export.flag, render.FlagWidth, render.FlagHeight)

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		a.exportFailed(export.flag.Name, err)

		return
	}

	if err := writeFileAtomically(export.path, encoded.Bytes()); err != nil {
		a.exportFailed(export.flag.Name, err)

		return
	}

	a.setStatus("Exported %s to %s", export.flag.Name, filepath.Base(export.path))
}

func (a *App) exportFailed(name string, err error) {
	warn(err)
	a.setStatus("%s could not be exported: %v", name, err)
}

// scriptStart is where the save dialog opens: at the flag's own file, or else
// in the coat of arms folder of the last configured folder, which is the one
// that overrides the others and so usually the mod being worked on.
func (a *App) scriptStart(flag *pdx.Flag) string {
	if flag.Origin.Path != "" {
		return flag.Origin.Path
	}

	folder := a.workingFolder()
	if folder == "" {
		return flag.Name + ".txt"
	}

	return filepath.Join(folder, flag.Name+".txt")
}

// imageStart is where the export dialog opens: next to the flag's file, or in
// the last configured folder.
func (a *App) imageStart(flag *pdx.Flag) string {
	folder := filepath.Dir(flag.Origin.Path)
	if flag.Origin.Path == "" {
		folder = a.workingFolder()
	}

	return filepath.Join(folder, flag.Name+".png")
}

// workingFolder is the coat of arms folder of the last configured folder, or
// that folder itself when it has none yet.
func (a *App) workingFolder() string {
	databases := a.settings.Databases
	if len(databases) == 0 {
		return ""
	}

	root := databases[len(databases)-1].Path

	if info, err := os.Stat(filepath.Join(root, coatOfArmsFolder)); err == nil && info.IsDir() {
		return filepath.Join(root, coatOfArmsFolder)
	}

	return root
}

// databaseOf names the configured folder a file is in. A file in several, a
// mod kept inside the game folder say, belongs to the last, as it would when
// the folders are read.
func (a *App) databaseOf(path string) string {
	found := ""

	for _, database := range a.settings.Databases {
		relative, err := filepath.Rel(database.Path, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			found = database.Name
		}
	}

	return found
}

// samePath reports whether two paths name the same file.
func samePath(first, second string) bool {
	if first == "" || second == "" {
		return false
	}

	relative, err := filepath.Rel(first, second)

	return err == nil && relative == "."
}

// withExtension adds an extension to a file name typed without one.
func withExtension(path, extension string) string {
	if filepath.Ext(path) == "" {
		return path + extension
	}

	return path
}

// writeFileAtomically writes a file so that it is never left half written:
// the contents go to a file next to it, which then takes its place.
func writeFileAtomically(path string, contents []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}

	// Removing it after a successful rename fails harmlessly.
	defer os.Remove(temporary.Name())

	// The file keeps the permissions it had; a new one gets the usual ones
	// rather than the private ones a temporary file is created with.
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()

		return err
	}

	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()

		return err
	}

	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(temporary.Name(), path)
}
