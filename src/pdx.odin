package pdx_flag_builder

import "core:text/regex"
import "core:fmt"
import "core:os"
import "core:strings"
import "core:strconv"

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
freeParsing :: proc() {
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

destroyNamedColors :: proc(database: ^DatabaseState) {
    for _, index in database.NamedColors {
        destroyColor(database.NamedColors[index])
    }
    delete(database.NamedColors)
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

loadNamedColors :: proc(database: ^DatabaseState) {
    database.NamedColors = make([dynamic]FlagColorVariant)
    path, err := os.join_path([]string{ database.Settings.Path, FOLDER_COMMON, FOLDER_NAMED_COLORS }, context.allocator)
    defer delete(path)

    walker := os.walker_create_path(path)
    defer os.walker_destroy(&walker)

    for info in os.walker_walk(&walker) {
        _ = os.walker_error(&walker) or_break

        if path, err := os.walker_error(&walker); err != nil {
            fmt.eprintfln("failed walking %s: %s", path, err)
            continue
        }

        if !strings.has_suffix(info.fullpath, ".txt") {
            continue
        }

        if info.type != os.File_Type.Regular {
            continue
        }

        data, err := os.read_entire_file_from_path(info.fullpath, context.allocator)
        defer delete(data)
        if err != nil {
            fmt.eprintfln("could not load named color file %s: %v", info.fullpath, err)
            continue
        }

        lines := strings.split_lines(string(data))
        defer delete(lines)

        for line in lines {
            capture, ok := regex.match(regexNamedColor, line)
            defer regex.destroy(capture)
            if !ok {
                continue
            }

            name := capture.groups[1]
            type := strings.to_lower(capture.groups[2])
            defer delete(type)

            if type == COLOR_TYPE_RGB || type == COLOR_TYPE_IMPLICIT {
                red, redOk := strconv.parse_f64(capture.groups[3])
                green, greenOk := strconv.parse_f64(capture.groups[4])
                blue, blueOk := strconv.parse_f64(capture.groups[5])

                if !redOk || !greenOk || !blueOk {
                    fmt.eprintfln("invalid named rgb color: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
                    continue
                }
                color := new_clone(FlagColorRgb{
                    Name = strings.clone(name),
                    R = u8(red),
                    G = u8(green),
                    B = u8(blue),
                })

                // Assume its a comma value
                if red <= 1 && green <= 1 && blue <= 1 {
                    color.R = u8(red * 255)
                    color.G = u8(green * 255)
                    color.B = u8(blue * 255)
                }

                append(&database.NamedColors, color)
                continue
            }

            if type == COLOR_TYPE_HSV {
                hue, hueOk := strconv.parse_f32(capture.groups[3])
                saturation, saturationOk := strconv.parse_f32(capture.groups[4])
                value, valueOk := strconv.parse_f32(capture.groups[5])

                if !hueOk || !saturationOk || !valueOk {
                    fmt.eprintfln("invalid named hsv color: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
                    continue
                }

                color := new_clone(FlagColorHsv{
                    Name = strings.clone(name),
                    H = hue,
                    S = saturation,
                    V = value,
                })
                append(&database.NamedColors, color)
                continue
            }

            if type == COLOR_TYPE_HSV360 {
                hue, hueOk := strconv.parse_u64(capture.groups[3])
                saturation, saturationOk := strconv.parse_u64(capture.groups[4])
                value, valueOk := strconv.parse_u64(capture.groups[5])

                if !hueOk || !saturationOk || !valueOk {
                    fmt.eprintfln("invalid named hsv360 color: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
                    continue
                }

                color := new_clone(FlagColorHsv{
                    Name = strings.clone(name),
                    H = f32(hue) / 360,
                    S = f32(saturation) / 360,
                    V = f32(value) / 360,
                })
                append(&database.NamedColors, color)
                continue
            }

            fmt.eprintfln("unknown named color type: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
        }
    }

    fmt.printfln("Loaded %i named colors from database: %s", len(database.NamedColors), database.Settings.Name)
}