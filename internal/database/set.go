package database

import (
	"sort"
	"strings"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// Set is every configured folder that has been read, in the order they were
// configured.
//
// Lookups run from the back, so a mod listed after the base game overrides it.
// That is the same precedence the games apply when they load a mod.
type Set []*Database

// Flag finds a coat of arms by name.
func (s Set) Flag(name string) (*pdx.Flag, bool) {
	for index := len(s) - 1; index >= 0; index-- {
		database := s[index]

		for flag := range database.Flags {
			if database.Flags[flag].Name == name {
				return &database.Flags[flag], true
			}
		}
	}

	return nil, false
}

// Texture finds a texture by file name.
func (s Set) Texture(name string) (Texture, bool) {
	for index := len(s) - 1; index >= 0; index-- {
		database := s[index]

		for _, texture := range database.Textures {
			if strings.EqualFold(texture.Name, name) {
				return texture, true
			}
		}
	}

	return Texture{}, false
}

// Palette merges the named colours of every folder.
func (s Set) Palette() pdx.Palette {
	merged := pdx.Palette{}

	for _, database := range s {
		for name, color := range database.Palette {
			merged[name] = color
		}
	}

	return merged
}

// Flags returns every coat of arms, sorted by name.
func (s Set) Flags() []pdx.Flag {
	var flags []pdx.Flag

	for _, database := range s {
		flags = append(flags, database.Flags...)
	}

	sort.Slice(flags, func(first, second int) bool {
		if flags[first].Name == flags[second].Name {
			return flags[first].Origin.Database < flags[second].Origin.Database
		}

		return flags[first].Name < flags[second].Name
	})

	return flags
}

// Textures returns every texture, sorted by name.
func (s Set) Textures() []Texture {
	var textures []Texture

	for _, database := range s {
		textures = append(textures, database.Textures...)
	}

	sort.Slice(textures, func(first, second int) bool {
		if textures[first].Name == textures[second].Name {
			return textures[first].Database < textures[second].Database
		}

		return textures[first].Name < textures[second].Name
	})

	return textures
}

// FlagCount is how many coats of arms were read in total.
func (s Set) FlagCount() int {
	total := 0
	for _, database := range s {
		total += len(database.Flags)
	}

	return total
}

// TextureCount is how many textures were found in total.
func (s Set) TextureCount() int {
	total := 0
	for _, database := range s {
		total += len(database.Textures)
	}

	return total
}
