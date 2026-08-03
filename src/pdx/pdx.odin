package pdx

import rl "vendor:raylib"

RELEASE :: #config(RELEASE, false)

FOLDER_EU5_MAIN_MENU: string: "main_menu"
FOLDER_EU5_LOADING_SCREEN: string: "loading_screen"
FOLDER_EU5_IN_GAME: string: "in_game"

FOLDER_GFX: string: "gfx"
FOLDER_COA: string: "coat_of_arms"
FOLDER_PATTERNS: string: "patterns"
FOLDER_COLORED_EMBLEMS: string: "colored_emblems"
FOLDER_TEXTURED_EMBLEMS: string: "textured_emblems"

FOLDER_COMMON: string: "common"
FOLDER_NAMED_COLORS: string: "named_colors"

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
    rl.Color{0, 0, 128, 255},
    rl.Color{0, 255, 128, 255},
    rl.Color{255, 0, 128, 255},
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

GetNextFreeColor :: proc(colors: []FlagColor) -> string {
    for name in COLOR_NAMES {
        found := false
        for color in colors {
            if color.Name == name {
                found = true
                break
            }
        }
        if !found {
            return name
        }
    }
    return ""
}