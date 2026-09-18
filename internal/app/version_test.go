package app

import (
	"os"
	"regexp"
	"testing"
)

// The release workflow rewrites the version by matching its line in
// version.go, so the line has to keep the shape the script looks for.
func TestVersionLineIsWhereTheReleaseLooksForIt(t *testing.T) {
	source, err := os.ReadFile("version.go")
	if err != nil {
		t.Fatal(err)
	}

	line := regexp.MustCompile(`(?m)^const applicationVersion = "([^"]*)"\r?$`)

	found := line.FindAllSubmatch(source, -1)
	if len(found) != 1 {
		t.Fatalf("found the version line %d times in version.go, want exactly once", len(found))
	}

	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-dev)?$`).Match(found[0][1]) {
		t.Errorf("version = %q, want a version such as 1.2.3", found[0][1])
	}
}
