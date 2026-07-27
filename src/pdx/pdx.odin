package pdx

import "core:text/regex"
import "core:fmt"
import "core:os"
import rl "vendor:raylib"

FOLDER_GFX: string: "gfx"
FOLDER_COA: string: "coat_of_arms"
FOLDER_PATTERNS: string: "patterns"
FOLDER_COLORED_EMBLEMS: string: "colored_emblems"
FOLDER_TEXTURED_EMBLEMS: string: "textured_emblems"

FOLDER_COMMON: string: "common"
FOLDER_NAMED_COLORS: string: "named_colors"

PATTERN_NAMED_COLOR: string: `^\s*([a-zA-Z0-9_-]+)\s*=\s*(rgb|hsv360|hsv|)\s*\{\s*([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s*\}\s*$`
COLOR_TYPE_IMPLICIT: string: ""
COLOR_TYPE_RGB: string: "rgb"
COLOR_TYPE_HSV: string: "hsv"
COLOR_TYPE_HSV360: string: "hsv360"

PATTERN_REPLACE_COLORS: []rl.Color: {
    rl.Color{255, 0, 0, 255},
    rl.Color{255, 255, 0, 255},
    rl.Color{255, 255, 255, 255},
}

COLORED_EMBLEM_REPLACE_COLORS: []rl.Color: {
    rl.Color{0, 0, 123, 255},
    rl.Color{0, 255, 123, 255},
    rl.Color{255, 0, 123, 255},
}

COLOR_NAMES: []string: {
    "color1",
    "color2",
    "color3",
    "color4",
    "color5",
    "color6",
    "color7",
    "color8",
    "color9",
}

regexNamedColor: regex.Regular_Expression

setupParsing :: proc() {
    regex, err := regex.create(PATTERN_NAMED_COLOR, {})
    if err != nil {
        fmt.eprintfln("Could not parse named color pattern: %v", err)
        os.exit(1)
    }
    regexNamedColor = regex
}
destroyParsing :: proc() {
    regex.destroy(regexNamedColor)
}

GetNextFreeColor :: proc(colors: []FlagColorVariant) -> string {
    for name in COLOR_NAMES {
        found := false
        for variant in colors {
            switch color in variant {
            case ^FlagColorNamed:
                if color.Name == name {
                    found = true
                    break
                }
            case ^FlagColorRgb:
                if color.Name == name {
                    found = true
                    break
                }
            case ^FlagColorHsv:
                if color.Name == name {
                    found = true
                    break
                }
            case ^FlagColorReference:
                if color.Name == name {
                    found = true
                    break
                }
            }
        }
        if !found {
            return name
        }
    }
    return ""
}