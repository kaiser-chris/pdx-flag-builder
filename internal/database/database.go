// Package database reads the coats of arms, colours and textures of a
// configured game or mod folder.
//
// The two supported games keep the same files in the same places; Europa
// Universalis 5 just splits them across three roots instead of one. Everything
// here is plain file reading with no graphics involved, so it can run off the
// interface goroutine while the application stays responsive.
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx/script"
)

// Game is the folder layout a database follows.
type Game uint8

const (
	GameUnknown Game = iota
	GameVictoria3
	GameEuropaUniversalis5
)

func (g Game) String() string {
	switch g {
	case GameVictoria3:
		return "Victoria 3"
	case GameEuropaUniversalis5:
		return "Europa Universalis 5"
	}

	return "Unknown"
}

// Folder names shared by both games.
const (
	folderCommon          = "common"
	folderGfx             = "gfx"
	folderCoatOfArms      = "coat_of_arms"
	folderNamedColors     = "named_colors"
	folderPatterns        = "patterns"
	folderColoredEmblems  = "colored_emblems"
	folderTexturedEmblems = "textured_emblems"
)

// europaRoots are the three folders Europa Universalis 5 splits its data over.
var europaRoots = []string{"main_menu", "loading_screen", "in_game"}

// textureExtensions are the image formats the games load coat of arms art from.
var textureExtensions = []string{".dds", ".tga", ".png"}

// TextureKind is which part of a coat of arms a texture can be used for.
type TextureKind uint8

const (
	PatternTexture TextureKind = iota
	ColoredEmblemTexture
	TexturedEmblemTexture
)

func (k TextureKind) String() string {
	switch k {
	case PatternTexture:
		return "Pattern"
	case ColoredEmblemTexture:
		return "Colored Emblem"
	case TexturedEmblemTexture:
		return "Textured Emblem"
	}

	return "Unknown"
}

// Texture is one image a coat of arms can refer to.
type Texture struct {
	// Name is the file name, which is how script refers to it.
	Name string

	// Path is where it is on disk.
	Path string

	Kind TextureKind

	// Database is the name of the configured folder it came from.
	Database string
}

// Problem is a file that could not be read, or could only be read in part.
// Problems never stop a load; they are collected so the interface can show a
// mod author what is wrong with their files.
type Problem struct {
	Path    string
	Message string
}

func (p Problem) String() string {
	return fmt.Sprintf("%s: %s", filepath.Base(p.Path), p.Message)
}

// Folder is a configured game or mod folder to read.
type Folder struct {
	Name string
	Path string
}

// Database is everything read from one configured folder.
type Database struct {
	Name string
	Path string
	Game Game

	Flags    []pdx.Flag
	Palette  pdx.Palette
	Textures []Texture

	Problems []Problem
}

// Detect works out which game a folder belongs to from the way it is laid out.
func Detect(path string) Game {
	for _, root := range europaRoots {
		if isDirectory(filepath.Join(path, root)) {
			return GameEuropaUniversalis5
		}
	}

	if isDirectory(filepath.Join(path, folderCommon)) || isDirectory(filepath.Join(path, folderGfx)) {
		return GameVictoria3
	}

	return GameUnknown
}

// Load reads one configured folder. It fails only when the folder itself cannot
// be used; anything missing inside it is left empty, because a mod that only
// adds emblems is perfectly normal.
func Load(folder Folder) (*Database, error) {
	info, err := os.Stat(folder.Path)
	if err != nil {
		return nil, fmt.Errorf("open folder %q: %w", folder.Path, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a folder", folder.Path)
	}

	database := &Database{
		Name:    folder.Name,
		Path:    folder.Path,
		Game:    Detect(folder.Path),
		Palette: pdx.Palette{},
	}

	for _, root := range database.roots() {
		database.readPalette(filepath.Join(root, folderCommon, folderNamedColors))
		database.readFlags(filepath.Join(root, folderCommon, folderCoatOfArms, folderCoatOfArms))

		gfx := filepath.Join(root, folderGfx, folderCoatOfArms)
		database.readTextures(filepath.Join(gfx, folderPatterns), PatternTexture)
		database.readTextures(filepath.Join(gfx, folderColoredEmblems), ColoredEmblemTexture)
		database.readTextures(filepath.Join(gfx, folderTexturedEmblems), TexturedEmblemTexture)
	}

	sort.Slice(database.Flags, func(first, second int) bool {
		return database.Flags[first].Name < database.Flags[second].Name
	})

	sort.Slice(database.Textures, func(first, second int) bool {
		return database.Textures[first].Name < database.Textures[second].Name
	})

	return database, nil
}

// LoadAll reads every configured folder, in order.
func LoadAll(folders []Folder) (Set, []Problem) {
	var (
		set      Set
		problems []Problem
	)

	for _, folder := range folders {
		database, err := Load(folder)
		if err != nil {
			problems = append(problems, Problem{Path: folder.Path, Message: err.Error()})

			continue
		}

		problems = append(problems, database.Problems...)
		set = append(set, database)
	}

	return set, problems
}

// roots returns the folders to read, which for Europa Universalis 5 is one per
// part of the game and for everything else is the folder itself.
func (d *Database) roots() []string {
	if d.Game != GameEuropaUniversalis5 {
		return []string{d.Path}
	}

	roots := make([]string, 0, len(europaRoots))
	for _, root := range europaRoots {
		full := filepath.Join(d.Path, root)
		if isDirectory(full) {
			roots = append(roots, full)
		}
	}

	return roots
}

func (d *Database) readFlags(folder string) {
	for _, path := range scriptFiles(folder) {
		document, err := parseFile(path)
		if err != nil {
			d.Problems = append(d.Problems, Problem{Path: path, Message: err.Error()})

			continue
		}

		for _, warning := range document.Warnings {
			d.Problems = append(d.Problems, Problem{Path: path, Message: warning.String()})
		}

		origin := pdx.Origin{
			Database: d.Name,
			File:     filepath.Base(path),
			Path:     path,
		}

		flags, issues := pdx.DecodeFlags(document, origin)

		for _, issue := range issues {
			d.Problems = append(d.Problems, Problem{Path: path, Message: issue.String()})
		}

		d.Flags = append(d.Flags, flags...)
	}
}

func (d *Database) readPalette(folder string) {
	for _, path := range scriptFiles(folder) {
		document, err := parseFile(path)
		if err != nil {
			d.Problems = append(d.Problems, Problem{Path: path, Message: err.Error()})

			continue
		}

		palette, issues := pdx.DecodePalette(document)

		for _, issue := range issues {
			d.Problems = append(d.Problems, Problem{Path: path, Message: issue.String()})
		}

		// Later files override earlier ones, the way the games read them.
		for name, color := range palette {
			d.Palette[name] = color
		}
	}
}

func (d *Database) readTextures(folder string, kind TextureKind) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		// A folder a mod does not have is not a problem worth reporting.
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !hasTextureExtension(entry.Name()) {
			continue
		}

		d.Textures = append(d.Textures, Texture{
			Name:     entry.Name(),
			Path:     filepath.Join(folder, entry.Name()),
			Kind:     kind,
			Database: d.Name,
		})
	}
}

func parseFile(path string) (*script.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return script.Parse(string(data))
}

// scriptFiles lists the script files of a folder, sorted by name. The games
// read them in that order and let later files override earlier ones, which is
// what the leading numbers in their file names are for.
func scriptFiles(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}

	var paths []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			continue
		}

		paths = append(paths, filepath.Join(folder, entry.Name()))
	}

	sort.Strings(paths)

	return paths
}

func hasTextureExtension(name string) bool {
	extension := strings.ToLower(filepath.Ext(name))

	for _, candidate := range textureExtensions {
		if extension == candidate {
			return true
		}
	}

	return false
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}
