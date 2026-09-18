package script

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseInstalledGameFiles runs the parser over a real installation.
//
// The unit tests above cover the grammar as it is meant to be; this one covers
// the grammar as the games actually write it, which is the only way to find out
// what the files really contain. It is skipped unless PDX_GAME_DIR points at a
// game or mod folder, so it never runs in a checkout without one.
func TestParseInstalledGameFiles(t *testing.T) {
	root := os.Getenv("PDX_GAME_DIR")
	if root == "" {
		t.Skip("set PDX_GAME_DIR to a game or mod folder to run this test")
	}

	var parsed, failed int

	visit := func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".txt") {
			return nil //nolint:nilerr // an unreadable folder is not this test's problem
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if _, err := Parse(string(data)); err != nil {
			failed++
			t.Errorf("%s: %v", path, err)
		} else {
			parsed++
		}

		return nil
	}

	for _, folder := range readFolders(root) {
		if err := filepath.WalkDir(folder, visit); err != nil {
			t.Fatalf("walk %s: %v", folder, err)
		}
	}

	t.Logf("parsed %d files, %d failed", parsed, failed)
}

// readFolders narrows a game or mod folder down to where the tool reads
// script from, since the rest of a game holds script in dialects of its own
// that this parser never has to read. A folder without those is taken whole,
// so the test can also be pointed at a folder of loose files.
func readFolders(root string) []string {
	var folders []string

	for _, folder := range []string{
		filepath.Join(root, "common", "coat_of_arms"),
		filepath.Join(root, "common", "named_colors"),
	} {
		if info, err := os.Stat(folder); err == nil && info.IsDir() {
			folders = append(folders, folder)
		}
	}

	if len(folders) == 0 {
		return []string{root}
	}

	return folders
}
