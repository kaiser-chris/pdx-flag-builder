//go:build uitest

package app

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// fakeDialogs answers file dialogs with paths a test gave it, in order, and
// cancels once it runs out.
type fakeDialogs struct {
	mu       sync.Mutex
	answers  []string
	requests []fileRequest
}

func (f *fakeDialogs) choose(request fileRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, request)

	if len(f.answers) == 0 {
		return "", nil
	}

	answer := f.answers[0]
	f.answers = f.answers[1:]

	return answer, nil
}

func (f *fakeDialogs) asked() []fileRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]fileRequest(nil), f.requests...)
}

// answerDialogs has the next dialogs answered with the given paths.
func answerDialogs(application *App, paths ...string) *fakeDialogs {
	dialogs := &fakeDialogs{answers: paths}
	application.dialogs = dialogs

	return dialogs
}

// fixtureFile is the coat of arms file of the fixture game.
func fixtureFile(application *App) string {
	return filepath.Join(application.settings.Databases[0].Path, coatOfArmsFolder, "00_test.txt")
}

func readText(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(data)
}

func TestSaveWritesBackIntoItsFile(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")

	// Renamed in the editor, the flag still replaces the definition it was
	// read from.
	driver.Fill(driver.Find(panelSelected, "Name"), "TST_renamed")
	driver.Shortcut(imgui.ModCtrl, imgui.KeyS)

	saved := readText(t, fixtureFile(application))

	if !strings.Contains(saved, "TST_renamed = {") || strings.Contains(saved, "TST_split") {
		t.Errorf("the file does not have TST_split replaced by TST_renamed:\n%s", saved)
	}

	if !strings.Contains(saved, "TST_emblem = {") {
		t.Errorf("saving one flag lost another from the file:\n%s", saved)
	}

	if application.state.modified {
		t.Error("the flag still counts as modified after saving")
	}

	// The folders are read again, so the database lists the saved flag.
	driver.WaitFor("the folders to be read again", func() bool { return !application.state.library.loading })
	driver.Frame()

	if !driver.Exists("", "TST_renamed") || driver.Exists("", "TST_split") {
		t.Error("the flag database does not show the flag under its saved name")
	}

	if !strings.HasPrefix(application.state.status, "Saved TST_renamed") {
		t.Errorf("status = %q, want it to report the save", application.state.status)
	}
}

func TestSaveAsksForAFileTheFirstTime(t *testing.T) {
	application, driver := startApp(t)

	target := filepath.Join(t.TempDir(), "mod_flags")
	dialogs := answerDialogs(application, target)

	driver.Menu("File", "New Flag")
	driver.Menu("File", labelSave)

	driver.WaitFor("the new file to be written", func() bool {
		_, err := os.Stat(target + ".txt")

		return err == nil
	})

	written := readText(t, target+".txt")
	if !strings.HasPrefix(written, "\xef\xbb\xbfnew_flag = {") {
		t.Errorf("new file = %q, want a byte order mark and the flag", written)
	}

	// From then on the flag belongs to that file and saves without asking.
	driver.Shortcut(imgui.ModCtrl, imgui.KeyS)

	if asked := len(dialogs.asked()); asked != 1 {
		t.Errorf("the dialog was shown %d times, want only for the first save", asked)
	}

	if again := readText(t, target+".txt"); strings.Count(again, "new_flag = {") != 1 {
		t.Errorf("saving twice wrote the flag %d times", strings.Count(again, "new_flag = {"))
	}
}

func TestSaveRefusesANameTheGamesCannotRead(t *testing.T) {
	application, driver := startApp(t)

	before := readText(t, fixtureFile(application))

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")
	driver.Fill(driver.Find(panelSelected, "Name"), "two words")
	driver.Shortcut(imgui.ModCtrl, imgui.KeyS)

	if after := readText(t, fixtureFile(application)); after != before {
		t.Errorf("the file changed although the name cannot be saved:\n%s", after)
	}

	if !strings.Contains(application.state.status, "cannot be saved") {
		t.Errorf("status = %q, want it to say why nothing was saved", application.state.status)
	}
}

func TestCopyScript(t *testing.T) {
	application, driver := startApp(t)

	// The clipboard is the user's; it gets back what it had.
	previous := rl.GetClipboardText()
	t.Cleanup(func() { rl.SetClipboardText(previous) })

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")
	driver.Menu("File", labelCopyScript)

	copied := rl.GetClipboardText()
	if !strings.HasPrefix(copied, "TST_split = {") || !strings.Contains(copied, `pattern = "pattern_split.png"`) {
		t.Errorf("clipboard = %q, want the script of TST_split", copied)
	}

	if application.state.flag.Name != "TST_split" {
		t.Fatalf("open flag = %q", application.state.flag.Name)
	}
}

func TestExportImage(t *testing.T) {
	application, driver := startApp(t)

	target := filepath.Join(t.TempDir(), "split")
	answerDialogs(application, target)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")
	driver.Menu("File", labelExportImage)

	driver.WaitFor("the image to be written", func() bool {
		_, err := os.Stat(target + ".png")

		return err == nil
	})

	file, err := os.Open(target + ".png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	picture, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode the image: %v", err)
	}

	if size := picture.Bounds().Size(); size.X != 768 || size.Y != 512 {
		t.Errorf("image is %v, want the canvas size 768x512", size)
	}

	// The flag alone, the right way up and opaque: no checkerboard, no frame.
	top := toRGBA(picture.At(384, 100))
	bottom := toRGBA(picture.At(384, 400))
	corner := toRGBA(picture.At(0, 0))

	if !near(top, fixtureBlue) || !near(bottom, fixtureWhite) {
		t.Errorf("image halves are %v and %v, want %v over %v", top, bottom, fixtureBlue, fixtureWhite)
	}

	if top.A != 255 || corner.A != 255 || !near(corner, fixtureBlue) {
		t.Errorf("corner = %v, top = %v, want both opaque and the corner blue", corner, top)
	}
}

func TestBrowseForAFolder(t *testing.T) {
	application, driver := startApp(t)

	folder := filepath.Join(t.TempDir(), "my_mod")
	answerDialogs(application, folder)

	driver.Menu("Settings", "Open Settings")
	driver.Click(windowSettings, "Add Folder")

	browse := driver.FindAll(windowSettings, labelBrowse)
	if len(browse) != 2 {
		t.Fatalf("got %d Browse buttons, want one per folder", len(browse))
	}

	driver.ClickItem(browse[1])
	driver.WaitFor("the dialog's answer", func() bool { return application.state.dialog == nil })

	entry := application.state.databases[1]
	if entry.Path != folder || entry.Name != "my_mod" {
		t.Errorf("new row = %+v, want the chosen folder and its name", entry)
	}
}

func toRGBA(value color.Color) color.RGBA {
	r, g, b, a := value.RGBA()

	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func TestQuittingAsksAboutUnsavedChanges(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")
	driver.Fill(driver.Find(panelSelected, "Name"), "TST_changed")

	// The window's close button takes the same way as File > Exit.
	if application.allowQuit() {
		t.Fatal("closing the window with unsaved changes was allowed straight away")
	}

	// The question is asked from the next frame on.
	driver.Frames(2)
	driver.Click("", labelCancel)
	driver.Menu("File", "Exit")

	if application.window.ShouldClose() {
		t.Fatal("File > Exit closed the application over unsaved changes")
	}

	driver.Click("", labelDiscard)

	if !application.window.ShouldClose() {
		t.Error("discarding the changes did not go on to close the application")
	}
}

func TestSaveBeforeOpeningAnotherFlag(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")
	driver.Fill(driver.Find(panelSelected, "Name"), "TST_kept")

	driver.Click("", "TST_emblem")
	driver.Click("", labelSaveFirst)

	if name := application.state.flag.Name; name != "TST_emblem" {
		t.Errorf("open flag = %q after saving, want TST_emblem opened as asked", name)
	}

	if saved := readText(t, fixtureFile(application)); !strings.Contains(saved, "TST_kept = {") {
		t.Errorf("the changes were not saved before opening another flag:\n%s", saved)
	}
}
