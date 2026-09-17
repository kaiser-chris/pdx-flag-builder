// Package config loads and stores the user's settings.
//
// The on-disk format is deliberately the same one the Odin version wrote, so an
// existing installation keeps its configured game and mod folders after moving
// to this build.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	// folderName is created inside the user's configuration directory:
	// %AppData%\pdx-flag-builder on Windows, ~/.config/pdx-flag-builder on Linux.
	folderName = "pdx-flag-builder"

	settingsFileName = "settings.json"
	layoutFileName   = "layout.ini"
)

// Color is an 8 bit per channel colour. The lowercase JSON names match the
// settings file written by the Odin version.
type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// Database is a game or mod folder that flags and textures are read from.
type Database struct {
	Name string
	Path string
}

// Settings is the persisted application configuration.
type Settings struct {
	BackgroundColor Color
	Databases       []Database
}

// Default returns the settings used when no settings file exists yet.
func Default() *Settings {
	return &Settings{
		BackgroundColor: Color{R: 90, G: 95, B: 100, A: 255},
		Databases:       []Database{},
	}
}

// Dir returns the directory holding this application's configuration, creating
// it when it does not exist yet.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine user config directory: %w", err)
	}

	dir := filepath.Join(base, folderName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config directory %q: %w", dir, err)
	}

	return dir, nil
}

// SettingsPath returns the full path of the settings file.
func SettingsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, settingsFileName), nil
}

// LayoutPath returns the file Dear ImGui persists the window layout in. It
// lives next to the settings so that removing the config directory resets the
// application completely.
func LayoutPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, layoutFileName), nil
}

// Load reads the settings file. A missing file is not an error: it yields the
// defaults, which is what a first start looks like.
func Load() (*Settings, error) {
	path, err := SettingsPath()
	if err != nil {
		return Default(), err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read settings %q: %w", path, err)
	}

	settings := Default()
	if err := json.Unmarshal(data, settings); err != nil {
		return Default(), fmt.Errorf("parse settings %q: %w", path, err)
	}

	return settings, nil
}

// Save writes the settings back to disk.
func (s *Settings) Save() error {
	path, err := SettingsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings %q: %w", path, err)
	}

	return nil
}
