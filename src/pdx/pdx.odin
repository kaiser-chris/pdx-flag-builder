package pdx

import "core:text/regex"
import "core:fmt"
import "core:os"

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

Flag :: struct {
    Pattern: FlagTexture,
    Colors: [dynamic]FlagColorVariant,
    Layers: [dynamic]FlagLayerVariant,
}

FlagTexture :: struct {
    Name: string,
    Path: string,
}

FlagColorVariant :: union {
    ^FlagColorNamed,
    ^FlagColorHsv,
    ^FlagColorRgb,
}

FlagColor :: struct {
    Variant: FlagColorVariant,
    Name: string,
}

FlagColorNamed :: struct {
    using color: FlagColor,
    NamedColor: string,
}

FlagColorHsv :: struct {
    using color: FlagColor,
    H: f32,
    S: f32,
    V: f32,
}

FlagColorRgb :: struct {
    using color: FlagColor,
    R: u8,
    G: u8,
    B: u8,
}

FlagColorReference :: struct {
    using Color: FlagColor,
    Reference: FlagColor,
}

FlagLayerVariant :: union {
    ^FlagLayerColoredEmblem,
    ^FlagLayerTexturedEmblem,
}

FlagLayer :: struct {
    Texture: FlagTexture,
    Instances: [dynamic]LayerInstance,
}

FlagLayerColoredEmblem :: struct {
    using Layer: FlagLayer,
    Colors: [dynamic]FlagColorVariant,
}

FlagLayerTexturedEmblem :: struct {
    using Layer: FlagLayer,
}

LayerVector :: struct {
    X: f64,
    Y: f64,
}

LayerInstance :: struct {
    Rotation: i32,
    Scale: LayerVector,
    Position: LayerVector,
}

createFlag :: proc() -> Flag {
    return Flag{
        Colors = make([dynamic]FlagColorVariant),
        Layers = make([dynamic]FlagLayerVariant),
    }
}
destroyFlag :: proc(flag: Flag) {
    destroyTexture(flag.Pattern)
    for _, index in flag.Colors {
        destroyColor(flag.Colors[index])
    }
    delete(flag.Colors)
    for _, index in flag.Layers {
        destroyLayer(flag.Layers[index])
    }
    delete(flag.Layers)
}

destroyColor :: proc(variant: FlagColorVariant) {
    switch color in variant {
    case ^FlagColorRgb:
        delete(color.Name)
        free(color)
    case ^FlagColorHsv:
        delete(color.Name)
        free(color)
    case ^FlagColorNamed:
        delete(color.Name)
        delete(color.NamedColor)
        free(color)
    }
}

destroyLayer :: proc(variant: FlagLayerVariant) {
    switch layer in variant {
    case ^FlagLayerColoredEmblem:
        destroyTexture(layer.Texture)
        for _, index in layer.Colors {
            destroyColor(layer.Colors[index])
        }
        free(layer)
    case ^FlagLayerTexturedEmblem:
        destroyTexture(layer.Texture)
        free(layer)
    }
}

destroyTexture :: proc(texture: FlagTexture) {
    delete(texture.Name)
    delete(texture.Path)
}
