package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func newTestStore(t *testing.T) Store {
	t.Helper()

	store, err := NewStore(filepath.Join(t.TempDir(), "config"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	return store
}

func TestLoadWithoutAFileGivesTheDefaults(t *testing.T) {
	settings, err := newTestStore(t).Load()
	if err != nil {
		t.Fatalf("Load on a first start: %v", err)
	}

	if !reflect.DeepEqual(settings, Default()) {
		t.Errorf("Load = %+v, want the defaults", settings)
	}
}

func TestSaveAndLoad(t *testing.T) {
	store := newTestStore(t)

	want := &Settings{
		Databases: []Database{
			{Name: "game", Path: `E:\SteamLibrary\steamapps\common\Victoria 3\game`},
			{Name: "mod", Path: "/home/user/mods/gate"},
		},
		InterfaceScale: 1.5,
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %+v, want what was saved, %+v", got, want)
	}
}

// A settings file written by the Odin version has to keep working: its folders
// are read, and the background colour it also stored, which this version no
// longer has, is ignored rather than refused.
func TestLoadASettingsFileOfTheOdinVersion(t *testing.T) {
	store := newTestStore(t)

	odin := `{"BackgroundColor":{"r":90,"g":95,"b":100,"a":255},` +
		`"Databases":[{"Name":"game","Path":"C:\\Games\\Victoria 3\\game"}]}`

	if err := os.WriteFile(store.SettingsPath(), []byte(odin), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []Database{{Name: "game", Path: `C:\Games\Victoria 3\game`}}
	if !reflect.DeepEqual(settings.Databases, want) {
		t.Errorf("folders = %+v, want %+v", settings.Databases, want)
	}

	// Automatic, which is what a file from before the setting existed means.
	if settings.InterfaceScale != 0 {
		t.Errorf("interface scale = %v, want 0 for automatic", settings.InterfaceScale)
	}
}

func TestLoadABrokenFile(t *testing.T) {
	store := newTestStore(t)

	if err := os.WriteFile(store.SettingsPath(), []byte(`{"Databases": [`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := store.Load()
	if err == nil {
		t.Error("Load of a broken file gave no error, want one to report")
	}

	// The application still starts, with the defaults.
	if !reflect.DeepEqual(settings, Default()) {
		t.Errorf("Load = %+v, want the defaults to start with", settings)
	}
}

func TestLayoutLivesNextToTheSettings(t *testing.T) {
	store := newTestStore(t)

	if filepath.Dir(store.LayoutPath()) != filepath.Dir(store.SettingsPath()) {
		t.Errorf("layout at %s and settings at %s, want them in one folder", store.LayoutPath(), store.SettingsPath())
	}
}
