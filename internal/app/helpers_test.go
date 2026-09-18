package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// These cover the parts of the application that need no window. The interface
// itself is tested by the uitest tagged tests next to them.

func TestFilteredRowsFilterOnlyWhenSomethingChanged(t *testing.T) {
	names := []string{"GBR", "FRA", "gbr_subject", "PRU"}
	asked := 0

	matches := func(index int, query string) bool {
		asked++

		return containsFold(names[index], query)
	}

	var rows filteredRows

	if got := rows.get("  GbR ", 1, len(names), matches); !slices.Equal(got, []int{0, 2}) {
		t.Fatalf("rows for GbR = %v, want both spellings of gbr", got)
	}

	before := asked
	rows.get("  GbR ", 1, len(names), matches)

	if asked != before {
		t.Error("the same query over the same data was filtered again")
	}

	// New data behind the same query is filtered again.
	rows.get("  GbR ", 2, len(names), matches)

	if asked == before {
		t.Error("a new version of the data was not filtered again")
	}

	// An empty query still goes through the filter, which a list narrowed down
	// by something other than the search relies on.
	asked = 0
	if got := rows.get("", 2, len(names), matches); len(got) != len(names) || asked != len(names) {
		t.Errorf("rows for no query = %v after %d checks, want every row, each checked", got, asked)
	}
}

func TestFilteredRowsInOrder(t *testing.T) {
	names := []string{"b", "A", "c", "a"}
	byName := func(first, second, _ int) int { return compareFold(names[first], names[second]) }

	var rows filteredRows
	rows.get("", 1, len(names), func(int, string) bool { return true })

	// Rows that compare equal keep the order they were found in.
	if got := rows.inOrder(tableOrder{column: 0}, byName); !slices.Equal(got, []int{1, 3, 0, 2}) {
		t.Errorf("ascending = %v, want A and a in the order found, then b and c", got)
	}

	if got := rows.inOrder(tableOrder{column: 0, descending: true}, byName); !slices.Equal(got, []int{2, 0, 1, 3}) {
		t.Errorf("descending = %v, want c and b, then A and a still in the order found", got)
	}

	if got := rows.inOrder(unsorted, byName); !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Errorf("unsorted = %v, want the order the rows came in", got)
	}
}

func TestWithExtension(t *testing.T) {
	tests := map[string]string{
		"flags":           "flags.txt",
		"flags.txt":       "flags.txt",
		"my.flags.txt":    "my.flags.txt",
		"folder.d/flags":  "folder.d/flags.txt",
		"already.TXT":     "already.TXT",
		"other_extension": "other_extension.txt",
	}

	for path, want := range tests {
		if got := withExtension(path, ".txt"); got != want {
			t.Errorf("withExtension(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestSamePath(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "flags.txt")

	if !samePath(file, filepath.Join(folder, ".", "sub", "..", "flags.txt")) {
		t.Error("two spellings of one path are not the same path")
	}

	if samePath(file, filepath.Join(folder, "other.txt")) {
		t.Error("two files are the same path")
	}

	// A flag without a file is never the file just chosen.
	if samePath(file, "") || samePath("", "") {
		t.Error("an empty path matched")
	}
}

func TestDatabaseOf(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "game")
	mod := filepath.Join(game, "mod")

	application := &App{settings: &config.Settings{Databases: []config.Database{
		{Name: "game", Path: game},
		{Name: "mod", Path: mod},
		{Name: "gamedata", Path: filepath.Join(root, "gamedata")},
	}}}

	tests := map[string]string{
		filepath.Join(game, "common", "flags.txt"): "game",
		filepath.Join(mod, "common", "flags.txt"):  "mod",
		filepath.Join(root, "gamedata", "x.txt"):   "gamedata",
		filepath.Join(root, "elsewhere.txt"):       "",
		// A folder whose name starts like a configured one is not inside it.
		filepath.Join(root, "game2", "x.txt"): "",
	}

	for path, want := range tests {
		if got := application.databaseOf(path); got != want {
			t.Errorf("databaseOf(%s) = %q, want %q", path, got, want)
		}
	}
}

func TestScriptStart(t *testing.T) {
	mod := t.TempDir()
	folder := filepath.Join(mod, coatOfArmsFolder)

	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}

	application := &App{settings: &config.Settings{Databases: []config.Database{
		{Name: "game", Path: t.TempDir()},
		{Name: "mod", Path: mod},
	}}}

	// A flag with a file opens the dialog at it.
	saved := &pdx.Flag{Name: "AAA", Origin: pdx.Origin{Path: filepath.Join(mod, "x.txt")}}
	if got := application.scriptStart(saved); got != saved.Origin.Path {
		t.Errorf("start for a saved flag = %s, want its file", got)
	}

	// A new one in the coat of arms folder of the last configured folder.
	if got, want := application.scriptStart(&pdx.Flag{Name: "NEW"}), filepath.Join(folder, "NEW.txt"); got != want {
		t.Errorf("start for a new flag = %s, want %s", got, want)
	}
}

func TestWriteFileAtomically(t *testing.T) {
	folder := t.TempDir()
	path := filepath.Join(folder, "flags.txt")

	if err := writeFileAtomically(path, []byte("first")); err != nil {
		t.Fatalf("write a new file: %v", err)
	}

	if err := writeFileAtomically(path, []byte("second")); err != nil {
		t.Fatalf("replace the file: %v", err)
	}

	if got := readFile(t, path); got != "second" {
		t.Errorf("file = %q, want the second contents", got)
	}

	// Nothing is left behind next to it.
	entries, _ := os.ReadDir(folder)
	if len(entries) != 1 {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}

		t.Errorf("the folder holds %s, want only flags.txt", strings.Join(names, ", "))
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestWordingHelpers(t *testing.T) {
	checks := []struct{ got, want string }{
		{plural(1, "flag", "flags"), "1 flag"},
		{plural(0, "flag", "flags"), "0 flags"},
		{plural(2, "match", "matches"), "2 matches"},
		{scaleLabel(1.25), "125%"},
		{maskLabel(0), "None"},
		{maskLabel(2), "Pattern colour 2"},
		{instanceHeading(0, true), "Placement (default)"},
		{instanceHeading(1, false), "1 placement"},
		{instanceHeading(3, false), "3 placements"},
		{describeLayer(&pdx.ColoredEmblem{Texture: "ce_star.dds"}), "Colored Emblem  ce_star.dds"},
		{describeLayer(&pdx.SubFlag{}), "Sub Flag  no parent"},
		{textureAction(database.PatternTexture), labelSetPattern},
		{textureAction(database.ColoredEmblemTexture), labelAddAsLayer},
	}

	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("got %q, want %q", check.got, check.want)
		}
	}
}
