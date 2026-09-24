package pdx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kaiser-chris/pdx-parser-go/script"
)

// ErrNameTaken is returned by Merge when the coat of arms was renamed to a
// name the file already defines.
var ErrNameTaken = errors.New("the file already defines a coat of arms of that name")

// byteOrderMark starts the files the game tools write. A file created from
// scratch gets one too, which is what the games expect of their script files.
const byteOrderMark = "\xef\xbb\xbf"

// Merge puts a coat of arms into the contents of a coat of arms file and
// returns the new contents.
//
// The definition read under key is replaced where it stands, so that
// everything around it, comments and variables included, stays as it was, and
// the flag keeps its place in the file. A file without such a definition gets
// the flag added at its end. key is the name the flag had when it was read,
// which lets a renamed flag replace its old definition; for a flag new to the
// file it is simply its name.
//
// The file's own line breaks and byte order mark are kept.
func Merge(file string, flag Flag, key string) (merged string, replaced bool, err error) {
	if strings.TrimSpace(strings.TrimPrefix(file, byteOrderMark)) == "" {
		return byteOrderMark + Script(flag, "\n") + "\n", false, nil
	}

	// Parsing never fails: what the parser cannot make sense of it reads past
	// the way the games do, and says so. Writing into such a file is refused
	// all the same, because a definition left open by a missing brace runs to
	// the end of the file, and replacing it would take everything after it
	// along.
	document := script.Parse(file)

	if len(document.Warnings) > 0 {
		return "", false, fmt.Errorf("the file could not be read as it is, so it is left alone: %s", document.Warnings[0])
	}

	if flag.Name != key && findDefinition(document, flag.Name) >= 0 {
		return "", false, fmt.Errorf("%w: %s", ErrNameTaken, flag.Name)
	}

	lineBreak := "\n"
	if strings.Contains(file, "\r\n") {
		lineBreak = "\r\n"
	}

	written := Script(flag, lineBreak)

	if index := findDefinition(document, key); index >= 0 {
		field := document.Fields[index]

		return file[:field.Start] + written + file[field.End:], true, nil
	}

	// Added at the end, a blank line apart from whatever comes before.
	trimmed := strings.TrimRight(file, " \t\r\n")

	return trimmed + lineBreak + lineBreak + written + lineBreak, false, nil
}

// findDefinition returns the index of the top level field defining a coat of
// arms under a name, or minus one. The last one wins, as it does when the
// games read the file.
func findDefinition(document *script.Document, key string) int {
	found := -1

	for index, field := range document.Fields {
		if field.Key == key && field.Value.IsBlock() {
			found = index
		}
	}

	return found
}
