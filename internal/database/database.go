// Package database reads the coats of arms, colours and textures of a
// configured game or mod folder.
//
// The files themselves are read by the pdx-parser-go library, which knows
// where the games keep them, how a game and its mods are combined, and how to
// read what is in them. What this package adds is the editor's view of it: one
// database per configured folder, so that the interface can show which folder
// every flag and every texture came from, and edit the one the user picked.
//
// A folder is read with the folders configured before it underneath it, the
// way the games load a mod on top of the game, and only what the folder
// itself defines is kept. A mod that changes a flag of the game therefore
// shows the flag as the game will draw it, while the game's own version stays
// listed under the game.
//
// Everything here is plain file reading with no graphics involved, so it can
// run off the interface goroutine while the application stays responsive.
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kaiser-chris/pdx-parser-go/folders"
	"github.com/kaiser-chris/pdx-parser-go/report"
	"github.com/kaiser-chris/pdx-parser-go/victoria3"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
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

// Folders that tell the two games apart.
const (
	folderCommon = "common"
	folderGfx    = "gfx"
)

// europaRoots are the three folders Europa Universalis 5 splits its data over.
var europaRoots = []string{"main_menu", "loading_screen", "in_game"}

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

// textureKinds maps the kinds the library reads to the kinds here, which are
// the same three under the names the interface shows.
var textureKinds = map[victoria3.TextureKind]TextureKind{
	victoria3.PatternTexture:        PatternTexture,
	victoria3.ColoredEmblemTexture:  ColoredEmblemTexture,
	victoria3.TexturedEmblemTexture: TexturedEmblemTexture,
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

// Load reads one configured folder on its own. It fails only when the folder
// itself cannot be used; anything missing inside it is left empty, because a
// mod that only adds emblems is perfectly normal.
func Load(folder Folder) (*Database, error) {
	return load([]Folder{folder}, 0)
}

// LoadAll reads every configured folder, in order, each with the folders
// before it underneath it.
func LoadAll(configured []Folder) (Set, []Problem) {
	var (
		set      Set
		problems []Problem
	)

	for index := range configured {
		database, err := load(configured, index)
		if err != nil {
			problems = append(problems, Problem{Path: configured[index].Path, Message: err.Error()})

			continue
		}

		problems = append(problems, database.Problems...)
		set = append(set, database)
	}

	return set, problems
}

// load reads the folder at index of the configured folders, with the ones
// before it loaded underneath it.
func load(configured []Folder, index int) (*Database, error) {
	folder := configured[index]

	info, err := os.Stat(folder.Path)
	if err != nil {
		return nil, fmt.Errorf("open folder %q: %w", folder.Path, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a folder", folder.Path)
	}

	sources := make([]folders.Source, 0, index+1)

	for _, earlier := range configured[:index+1] {
		if earlier.Path != "" {
			sources = append(sources, folders.Source{Name: earlier.Name, Path: earlier.Path})
		}
	}

	set := folders.Open(sources)
	heraldry := victoria3.LoadHeraldry(set)

	database := &Database{
		Name:    folder.Name,
		Path:    folder.Path,
		Game:    Detect(folder.Path),
		Palette: heraldry.Colors,
	}

	database.readFlags(heraldry)
	database.readTextures(heraldry)
	database.readProblems(set.Diagnostics, heraldry.Diagnostics)

	sort.Slice(database.Flags, func(first, second int) bool {
		return database.Flags[first].Name < database.Flags[second].Name
	})

	sort.Slice(database.Textures, func(first, second int) bool {
		return database.Textures[first].Name < database.Textures[second].Name
	})

	return database, nil
}

// readFlags keeps the coats of arms this folder defines or changes. The rest
// belongs to the folders underneath it, which are listed as databases of
// their own.
func (d *Database) readFlags(heraldry *victoria3.Heraldry) {
	for _, arms := range heraldry.CoatOfArms.All() {
		origin, ok := d.origin(arms)
		if !ok {
			continue
		}

		d.Flags = append(d.Flags, pdx.FromCoatOfArms(arms, origin))
	}
}

// origin is where in this folder a coat of arms was written, which is the
// file the editor writes it back to. A flag the folder does not touch has
// none.
func (d *Database) origin(arms *victoria3.CoatOfArms) (pdx.Origin, bool) {
	origin := pdx.Origin{Database: d.Name, Key: arms.Key}
	found := false

	// The last one wins: a folder that both defines a flag and injects into
	// it is edited where it said the most about it.
	for _, source := range arms.Origins {
		if !d.holds(source.Path) {
			continue
		}

		origin.File = filepath.Base(source.Path)
		origin.Path = source.Path
		origin.Line = source.Line
		found = true
	}

	return origin, found
}

func (d *Database) readTextures(heraldry *victoria3.Heraldry) {
	for _, texture := range heraldry.Textures {
		if !d.holds(texture.Path) {
			continue
		}

		d.Textures = append(d.Textures, Texture{
			Name:     filepath.Base(texture.Path),
			Path:     texture.Path,
			Kind:     textureKinds[texture.Kind],
			Database: d.Name,
		})
	}
}

// readProblems keeps the diagnostics about this folder's own files that a mod
// author can act on. What the library reports at info severity is not a
// problem but a remark, such as one file overriding another, and would only
// bury the rest.
func (d *Database) readProblems(diagnostics ...report.Diagnostics) {
	for _, group := range diagnostics {
		for _, diagnostic := range group.Filter(report.SeverityWarning) {
			if !d.holds(diagnostic.Path) {
				continue
			}

			message := diagnostic.Message
			if diagnostic.Line > 0 {
				message = fmt.Sprintf("line %d: %s", diagnostic.Line, message)
			}

			if diagnostic.Subject != "" {
				message = diagnostic.Subject + ": " + message
			}

			d.Problems = append(d.Problems, Problem{Path: diagnostic.Path, Message: message})
		}
	}
}

// holds reports whether a file lies inside this folder, which is how what the
// folder itself holds is told apart from what it was loaded on top of. A
// folder holds the files of its downloadable content as well, which the
// library reads as folders of their own.
func (d *Database) holds(path string) bool {
	if path == "" {
		return false
	}

	relative, err := filepath.Rel(d.Path, path)

	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}
