//go:build uitest

package app

import (
	"path/filepath"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/uitest"
)

func TestOpenFlagFromDatabase(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_split")

	if application.state.flag == nil || application.state.flag.Name != "TST_split" {
		t.Fatalf("open flag = %v, want TST_split", application.state.flag)
	}

	waitForArtwork(t, application, driver, 1)

	// The pattern's two marker colours come out as the flag's two colours.
	if got := pixel(application, 384, 100); !near(got, fixtureBlue) {
		t.Errorf("top half = %v, want the flag's first colour %v", got, fixtureBlue)
	}

	if got := pixel(application, 384, 400); !near(got, fixtureWhite) {
		t.Errorf("bottom half = %v, want the flag's second colour %v", got, fixtureWhite)
	}
}

func TestColoredEmblemIsRecoloured(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)
	driver.Click("", "TST_emblem")

	waitForArtwork(t, application, driver, 2)

	// A half scale emblem in the middle, in the emblem's own colour, over a
	// pattern in the flag's.
	if got := pixel(application, 384, 256); !near(got, fixtureGreen) {
		t.Errorf("centre = %v, want the emblem colour %v", got, fixtureGreen)
	}

	if got := pixel(application, 20, 20); !near(got, fixtureBlue) {
		t.Errorf("corner = %v, want the pattern colour %v", got, fixtureBlue)
	}
}

func TestSearchFiltersTheFlagDatabase(t *testing.T) {
	_, driver := startApp(t)

	driver.Menu("Databases", windowFlagDatabase)

	if !driver.Exists("", "TST_split") || !driver.Exists("", "TST_emblem") {
		t.Fatal("the fixture flags are not listed")
	}

	driver.Fill(driver.Find(windowFlagDatabase, "##flag-search"), "emblem")

	if driver.Exists("", "TST_split") {
		t.Error("TST_split is still listed after searching for emblem")
	}

	if !driver.Exists("", "TST_emblem") {
		t.Error("TST_emblem disappeared although it matches the search")
	}
}

func TestSettingsAddFolder(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")
	driver.Click(windowSettings, "Add Folder")

	names := driver.FindAll(windowSettings, "##name")
	paths := driver.FindAll(windowSettings, "##path")

	if len(names) != 2 || len(paths) != 2 {
		t.Fatalf("got %d name and %d path fields, want a second row of each", len(names), len(paths))
	}

	modFolder := filepath.Dir(newFixtureGame(t))

	driver.Fill(names[1], "mod")
	driver.Fill(paths[1], modFolder)
	driver.Click(windowSettings, "Save")

	saved, err := application.store.Load()
	if err != nil {
		t.Fatalf("read the saved settings: %v", err)
	}

	if len(saved.Databases) != 2 || saved.Databases[1].Name != "mod" || saved.Databases[1].Path != modFolder {
		t.Fatalf("saved folders = %+v, want the new mod folder second", saved.Databases)
	}

	// Saving rereads the folders.
	driver.WaitFor("the new folder to be read", func() bool {
		return !application.state.library.loading && len(application.state.library.set) == 2
	})
}

func TestEscapeClosesTheFocusedWindow(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")

	if !application.state.showSettings {
		t.Fatal("the settings window did not open")
	}

	// Clicking into the window gives it focus, which is what Escape acts on.
	driver.Click(windowSettings, "Add Folder")
	driver.Press(imgui.KeyEscape)

	if application.state.showSettings {
		t.Error("Escape left the settings window open")
	}
}

// The harness itself: an application that reported nothing would make every
// other test here pass or fail for the wrong reason.
func TestHarnessSeesTheMenuBar(t *testing.T) {
	_, driver := startApp(t)

	for _, menu := range []string{"File", "Databases", "View", "Settings", "Help"} {
		if !driver.Exists(uitest.MainMenuBar, menu) {
			t.Errorf("menu %q is not in the main menu bar", menu)
		}
	}
}

func TestInterfaceScaleSetting(t *testing.T) {
	application, driver := startApp(t)

	driver.Menu("Settings", "Open Settings")
	before := driver.Find(windowSettings, "Save")

	driver.Click(windowSettings, "Interface Scale")
	driver.Click("", "200%")
	driver.Frames(2)

	if got := application.window.Scale(); got != 2 {
		t.Fatalf("scale = %v after choosing 200%%, want it applied straight away", got)
	}

	// Everything is drawn twice the size: text and padding alike.
	after := driver.Find(windowSettings, "Save")
	ratio := (after.Max.Y - after.Min.Y) / (before.Max.Y - before.Min.Y)

	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("a button grew %.2f times, want about twice", ratio)
	}

	// The settings window grows with its contents, so Save is still in view.
	driver.Click(windowSettings, "Save")

	saved, err := application.store.Load()
	if err != nil {
		t.Fatalf("read the saved settings: %v", err)
	}

	if saved.InterfaceScale != 2 {
		t.Errorf("saved scale = %v, want 2", saved.InterfaceScale)
	}

	// Back to following the monitor, and back to the original size, with no
	// drift from having been scaled up and down.
	driver.Click(windowSettings, "Interface Scale")
	driver.Click("", automaticScaleLabel())
	driver.Frames(2)

	if got, want := application.window.Scale(), gui.MonitorScale(); got != want {
		t.Errorf("scale = %v on automatic, want the monitor's %v", got, want)
	}

	if want := gui.MonitorScale(); want == 1 {
		restored := driver.Find(windowSettings, "Save")

		if restored.Max.Y-restored.Min.Y != before.Max.Y-before.Min.Y {
			t.Errorf("a button is %v tall after scaling back, want the original %v",
				restored.Max.Y-restored.Min.Y, before.Max.Y-before.Min.Y)
		}
	}
}
