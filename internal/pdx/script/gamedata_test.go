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

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	t.Logf("parsed %d files, %d failed", parsed, failed)
}
