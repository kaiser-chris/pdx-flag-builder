package render

import "strings"

// Game is how one of the games draws a country's flag in its own interface:
// the sizes it draws it at, the picture it lays over every one of them, and
// the cloth it hangs one on.
//
// The numbers are the games' own, from the files their interfaces are built
// from: gui/shared/flags.gui in Victoria 3 and gui/shared/coat_of_arms.gui in
// Europa Universalis 5.
type Game struct {
	// Name is the game, as a menu names it.
	Name string

	// Overlay is the picture the game multiplies over every flag it draws.
	Overlay string

	// Sizes are the flat sizes it draws a flag at, largest first.
	Sizes []IconSize

	// Cloths are the waving flags it hangs one on, largest first. A game may
	// draw the same cloth at more than one size.
	Cloths []Cloth
}

// IconSize is one of the sizes a game draws a flag at.
type IconSize struct {
	// Name is what the game's own files call the size.
	Name string

	// Width and Height are the whole picture: the flag and whatever the game
	// draws around it.
	Width, Height int32

	// FlagWidth and FlagHeight are the flag itself, which sits in the middle
	// of that.
	FlagWidth, FlagHeight int32

	// Border is the picture the game frames the flag with, over the whole of
	// it. An empty name leaves the flag unframed.
	Border string

	// FrameWidth and FrameHeight are one frame of a border that holds several
	// side by side, one per rank, as Victoria 3's do. A border of one frame
	// leaves these zero and is stretched over the picture.
	FrameWidth, FrameHeight int32

	// Mask keeps the flag to the part of a picture that is not see through,
	// which is how Europa Universalis 5 draws its round flag.
	Mask string
}

// Cloth is a waving flag of a game, at one of the sizes it draws one.
type Cloth struct {
	// Name is what to call this one, which is only worth more than "Fancy"
	// for a game that draws it at more than one size.
	Name string

	// Width and Height are the size the game shows the cloth at: its own
	// files give these as the render size times the scale they draw it down
	// to.
	Width, Height int32

	// RenderWidth and RenderHeight are the size it draws it at first, which
	// is larger: a cloth with a curved edge reads badly at the size it is
	// shown at, and the wave needs the room the shape of the picture gives it.
	RenderWidth, RenderHeight int32
}

// Games are the games a flag can be previewed in.
var Games = []*Game{&Victoria3, &EuropaUniversalis5}

// Victoria 3 frames a flag from outside: the border of the country's rank is
// larger than the flag and centred on it, and holds one frame per rank.
var Victoria3 = Game{
	Name:    "Victoria 3",
	Overlay: "flag_overlay.dds",

	Sizes: []IconSize{
		{
			Name: "Large", Width: 114, Height: 82,
			FlagWidth: 96, FlagHeight: 64,
			Border:     "flag_power_large.dds",
			FrameWidth: 228, FrameHeight: 164,
		},
		{
			Name: "Normal", Width: 80, Height: 58,
			FlagWidth: 66, FlagHeight: 44,
			Border:     "flag_power_normal.dds",
			FrameWidth: 160, FrameHeight: 116,
		},
		{
			Name: "Small", Width: 62, Height: 46,
			FlagWidth: 48, FlagHeight: 32,
			Border:     "flag_power_small.dds",
			FrameWidth: 124, FrameHeight: 92,
		},
		{
			Name: "Tiny", Width: 33, Height: 24,
			FlagWidth: 27, FlagHeight: 18,
			Border:     "flag_power_tiny.dds",
			FrameWidth: 66, FrameHeight: 48,
		},
	},

	// The game draws the cloth at several sizes, each as a render size and
	// the fraction of it that is shown: 550 by 309 at 0.45 on the end screen,
	// and 320 by 180 at 0.5 beside a country.
	Cloths: []Cloth{
		{Name: "Fancy Large", Width: 248, Height: 139, RenderWidth: 550, RenderHeight: 309},
		{Name: "Fancy Normal", Width: 160, Height: 90, RenderWidth: 320, RenderHeight: 180},
	},
}

// Europa Universalis 5 frames a flag from the outside in: the frame is the
// size of the widget and the flag is drawn inside it, at the size of the
// button that holds it.
var EuropaUniversalis5 = Game{
	Name:    "Europa Universalis 5",
	Overlay: "flag_texture.dds",

	Sizes: []IconSize{
		euSize("Country", 180, 120, frameLarge),
		euSize("Big", 96, 64, frameSmall),
		euSize("Mid Plus", 80, 54, frameSmall),
		euSize("Mid", 72, 48, frameSmall),
		euSize("Small Plus", 54, 36, frameLarge),
		euSize("Small", 36, 24, frameLarge),
		euSize("Very Small", 30, 20, frameLarge),
		euRound("Round", 32, 20),
		euSize("Mini", 22, 14, frameLarge),
	},

	// One size, drawn from a picture four times as wide as it is shown at,
	// which is the frame size its own widget asks for.
	Cloths: []Cloth{{Name: "Fancy", Width: 90, Height: 64, RenderWidth: 360, RenderHeight: 256}},
}

// The two frames Europa Universalis 5 draws around a flag. Which one a size
// takes is the game's own choice, not a matter of how large it is.
const (
	frameLarge = "frame_1.dds"
	frameSmall = "frame_2.dds"
)

// euFlagPercent is how much of its widget the flag of Europa Universalis 5
// fills: the button it is drawn in is that much of it, and the frame around
// the whole.
const euFlagPercent = 81

// euSize is one of the sizes of Europa Universalis 5.
func euSize(name string, width, height int32, border string) IconSize {
	size := euRound(name, width, height)
	size.Border = border
	size.Mask = ""

	return size
}

// euRound is its round flag, which has no frame and is cut to a mask instead.
func euRound(name string, width, height int32) IconSize {
	return IconSize{
		Name: name, Width: width, Height: height,
		FlagWidth: width * euFlagPercent / 100, FlagHeight: height * euFlagPercent / 100,
		Mask: "country_flag_mask.dds",
	}
}

// GameFile is one file of a game's own that a preview draws a flag with.
type GameFile struct {
	// Name is what the file is asked for by, which is its own name: the
	// folders are searched by name, the way a coat of arms names a texture.
	Name string

	// Path is where the file sits below a game folder, which is what to tell
	// someone who has to go and find it.
	Path string
}

// Files are every file of the game's own its preview draws a flag with. None
// of them belong to a coat of arms, so a set of folders without the game in
// it has none, and the preview says so rather than showing half of what the
// game would.
func (g *Game) Files() []GameFile {
	var files []GameFile

	seen := map[string]bool{}

	add := func(name string) {
		if name == "" || seen[name] {
			return
		}

		seen[name] = true
		files = append(files, GameFile{Name: name, Path: gameFolder(name) + "/" + name})
	}

	add(g.Overlay)

	for _, name := range []string{ClothMesh, ClothDiffuse, ClothNormal, ClothProperties} {
		add(name)
	}

	for _, size := range g.Sizes {
		add(size.Border)
		add(size.Mask)
	}

	return files
}

// The folders the games keep this artwork in, below a game folder.
const (
	victoriaFlagFolder = "gfx/interface/flag"
	europaFlagFolder   = "gfx/interface/coat_of_arms"
	europaFrameFolder  = "gfx/interface/component_decoration/flag_frames"
	europaMaskFolder   = "gfx/interface/component_masks"

	// ClothFolder is where both games keep the cloth of the waving flag.
	ClothFolder = "gfx/models/ui/flags"
)

// gameFolder is where a file of the games' own belongs.
func gameFolder(name string) string {
	switch {
	case strings.HasPrefix(name, "ui_flag_"):
		return ClothFolder

	case strings.HasPrefix(name, "frame_"):
		return europaFrameFolder

	case strings.HasSuffix(name, "_mask.dds"):
		return europaMaskFolder

	case name == EuropaUniversalis5.Overlay:
		return europaFlagFolder
	}

	return victoriaFlagFolder
}
