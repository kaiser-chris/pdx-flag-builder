package pdx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx/script"
)

func TestMergeReplacesTheDefinitionInPlace(t *testing.T) {
	file := byteOrderMark + "@third = 0.333\r\n" +
		"# the first flag\r\n" +
		"AAA = {\r\n\tpattern = \"old.dds\"\r\n\tcolor1 = \"red\"\r\n}\r\n" +
		"\r\n" +
		"BBB = { pattern = \"keep.dds\" } # stays untouched\r\n"

	flag := Flag{Name: "AAA", Pattern: "new.dds"}

	merged, replaced, err := Merge(file, flag, "AAA")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	want := byteOrderMark + "@third = 0.333\r\n" +
		"# the first flag\r\n" +
		"AAA = {\r\n\tpattern = \"new.dds\"\r\n}\r\n" +
		"\r\n" +
		"BBB = { pattern = \"keep.dds\" } # stays untouched\r\n"

	if !replaced || merged != want {
		t.Errorf("Merge = %q (replaced %v), want %q", merged, replaced, want)
	}
}

func TestMergeRenamesInPlace(t *testing.T) {
	file := "AAA = { pattern = \"a.dds\" }\nBBB = { pattern = \"b.dds\" }\n"

	merged, replaced, err := Merge(file, Flag{Name: "CCC", Pattern: "c.dds"}, "AAA")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if !replaced || !strings.HasPrefix(merged, "CCC = {") || strings.Contains(merged, "AAA") {
		t.Errorf("Merge = %q, want AAA replaced by CCC where it stood", merged)
	}
}

func TestMergeRefusesATakenName(t *testing.T) {
	file := "AAA = { pattern = \"a.dds\" }\nBBB = { pattern = \"b.dds\" }\n"

	if _, _, err := Merge(file, Flag{Name: "BBB"}, "AAA"); !errors.Is(err, ErrNameTaken) {
		t.Errorf("renaming AAA to BBB gave %v, want ErrNameTaken", err)
	}
}

func TestMergeAppendsANewFlag(t *testing.T) {
	file := "AAA = { pattern = \"a.dds\" }\n\n\n"

	merged, replaced, err := Merge(file, Flag{Name: "NEW"}, "NEW")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	want := "AAA = { pattern = \"a.dds\" }\n\nNEW = {\n}\n"
	if replaced || merged != want {
		t.Errorf("Merge = %q (replaced %v), want %q", merged, replaced, want)
	}
}

func TestMergeStartsAnEmptyFile(t *testing.T) {
	merged, _, err := Merge("", Flag{Name: "NEW", Pattern: "p.dds"}, "NEW")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	want := byteOrderMark + "NEW = {\n\tpattern = \"p.dds\"\n}\n"
	if merged != want {
		t.Errorf("Merge = %q, want %q", merged, want)
	}
}

func TestMergeLeavesAnUnreadableFileAlone(t *testing.T) {
	if _, _, err := Merge("AAA = { pattern = ", Flag{Name: "AAA"}, "AAA"); err == nil {
		t.Error("Merge into a broken file succeeded, want an error")
	}
}

// A coat of arms with nothing in it is written as an empty pair of braces,
// which has to be found again the next time it is saved.
func TestMergeReplacesAnEmptyFlag(t *testing.T) {
	file, _, err := Merge("", Flag{Name: "NEW"}, "NEW")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	again, replaced, err := Merge(file, Flag{Name: "NEW", Pattern: "p.dds"}, "NEW")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if !replaced || strings.Count(again, "NEW = {") != 1 {
		t.Errorf("saving an empty flag again gave %q, want it replaced", again)
	}
}

// TestMergeIntoInstalledGameFiles saves flags back into the files of a real
// installation, in memory, and checks that every other flag in the file comes
// out untouched. It is skipped unless PDX_GAME_DIR points at a game folder.
func TestMergeIntoInstalledGameFiles(t *testing.T) {
	root := os.Getenv("PDX_GAME_DIR")
	if root == "" {
		t.Skip("set PDX_GAME_DIR to a game or mod folder to run this test")
	}

	files, _ := filepath.Glob(filepath.Join(root, "common", "coat_of_arms", "coat_of_arms", "*.txt"))
	merged := 0

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		content := string(data)
		before := scriptsOf(t, content)

		for index, flag := range decodeAll(t, content) {
			// Every tenth is plenty, and keeps the test quick.
			if index%10 != 0 {
				continue
			}

			result, replaced, err := Merge(content, flag, flag.Origin.Key)
			if err != nil || !replaced {
				t.Errorf("%s in %s: replaced %v, %v", flag.Name, filepath.Base(path), replaced, err)

				continue
			}

			after := scriptsOf(t, result)

			if len(after) != len(before) {
				t.Errorf("%s in %s: the file went from %d flags to %d", flag.Name, filepath.Base(path), len(before), len(after))

				continue
			}

			for name, script := range before {
				if name != flag.Name && after[name] != script {
					t.Errorf("saving %s changed %s in %s", flag.Name, name, filepath.Base(path))
				}
			}

			merged++
		}
	}

	t.Logf("merged %d flags back into their files", merged)
}

func decodeAll(t *testing.T, content string) []Flag {
	t.Helper()

	document, err := script.Parse(content)
	if err != nil {
		t.Fatal(err)
	}

	flags, _ := DecodeFlags(document, Origin{})

	return flags
}

// scriptsOf maps every flag of a file to its script, which is an easy way to
// compare two versions of a file flag by flag.
func scriptsOf(t *testing.T, content string) map[string]string {
	t.Helper()

	scripts := map[string]string{}
	for _, flag := range decodeAll(t, content) {
		scripts[flag.Name] = Script(flag, "\n")
	}

	return scripts
}
