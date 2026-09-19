//go:build windows

package main

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNeedsOnlyDLLsOfWindows builds the executable and checks every DLL it
// loads at start ships with Windows. A MinGW DLL among them makes it fail to
// start on any machine without MinGW, with nothing but a missing DLL error.
func TestNeedsOnlyDLLsOfWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the executable")
	}

	executable := filepath.Join(t.TempDir(), "pdx-flag-builder.exe")

	build := exec.Command("go", "build", "-o", executable, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}

	file, err := pe.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// ImportedLibraries is not implemented for PE files and returns nothing,
	// so the libraries are read off the imported symbols, which come as
	// "function:library".
	symbols, err := file.ImportedSymbols()
	if err != nil {
		t.Fatal(err)
	}

	libraries := map[string]bool{}

	for _, symbol := range symbols {
		if _, library, found := strings.Cut(symbol, ":"); found {
			libraries[library] = true
		}
	}

	if len(libraries) == 0 {
		t.Fatal("found no imported libraries at all, which cannot be right")
	}

	system := filepath.Join(os.Getenv("SystemRoot"), "System32")

	for library := range libraries {
		// The C runtime's API sets are resolved by Windows itself, and are
		// not files of their own everywhere.
		if strings.HasPrefix(strings.ToLower(library), "api-ms-win-") {
			continue
		}

		if _, err := os.Stat(filepath.Join(system, library)); err != nil {
			t.Errorf("the executable loads %s, which does not come with Windows", library)
		}
	}
}

// TestRunsOnTheStoreHeap builds the executable and checks its manifest asks
// for the segment heap, the one Store installs get, so that a build started
// any other way fails the same way; see internal/gui/resources.go.
func TestRunsOnTheStoreHeap(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the executable")
	}

	executable := filepath.Join(t.TempDir(), "pdx-flag-builder.exe")

	build := exec.Command("go", "build", "-o", executable, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}

	file, err := pe.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	resources := file.Section(".rsrc")
	if resources == nil {
		t.Fatal("the executable carries no resources, so no manifest")
	}

	data, err := resources.Data()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "<heapType") || !strings.Contains(string(data), "SegmentHeap") {
		t.Error("the executable's manifest does not ask for the segment heap")
	}
}
