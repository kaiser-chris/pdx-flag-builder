package pdx_flag_builder

import "core:text/regex"
import "core:fmt"
import "core:os"
import "core:strings"
import "core:strconv"
import "pdx"

DestroyDatabase :: proc(database: ^DatabaseState) {
    destroyNamedColors(database)
    destroyLoadedTextures(database)
}

loadNamedColors :: proc(database: ^DatabaseState) {
    destroyNamedColors(database)

    database.NamedColors = make([dynamic]pdx.FlagColorVariant)
    path, err := os.join_path([]string{ database.Settings.Path, pdx.FOLDER_COMMON, pdx.FOLDER_NAMED_COLORS }, context.allocator)
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
            capture, ok := regex.match(pdx.regexNamedColor, line)
            defer regex.destroy(capture)
            if !ok {
                continue
            }

            name := capture.groups[1]
            type := strings.to_lower(capture.groups[2])
            defer delete(type)

            if type == pdx.COLOR_TYPE_RGB || type == pdx.COLOR_TYPE_IMPLICIT {
                red, redOk := strconv.parse_f64(capture.groups[3])
                green, greenOk := strconv.parse_f64(capture.groups[4])
                blue, blueOk := strconv.parse_f64(capture.groups[5])

                if !redOk || !greenOk || !blueOk {
                    fmt.eprintfln("invalid named rgb color: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
                    continue
                }
                color := new_clone(pdx.FlagColorRgb{
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

            if type == pdx.COLOR_TYPE_HSV {
                hue, hueOk := strconv.parse_f32(capture.groups[3])
                saturation, saturationOk := strconv.parse_f32(capture.groups[4])
                value, valueOk := strconv.parse_f32(capture.groups[5])

                if !hueOk || !saturationOk || !valueOk {
                    fmt.eprintfln("invalid named hsv color: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
                    continue
                }

                color := new_clone(pdx.FlagColorHsv{
                    Name = strings.clone(name),
                    H = hue,
                    S = saturation,
                    V = value,
                })
                append(&database.NamedColors, color)
                continue
            }

            if type == pdx.COLOR_TYPE_HSV360 {
                hue, hueOk := strconv.parse_u64(capture.groups[3])
                saturation, saturationOk := strconv.parse_u64(capture.groups[4])
                value, valueOk := strconv.parse_u64(capture.groups[5])

                if !hueOk || !saturationOk || !valueOk {
                    fmt.eprintfln("invalid named hsv360 color: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
                    continue
                }

                color := new_clone(pdx.FlagColorHsv{
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
destroyNamedColors :: proc(database: ^DatabaseState) {
    if database.NamedColors == nil {
        return
    }
    for _, index in database.NamedColors {
        pdx.DestroyColor(database.NamedColors[index])
    }
    delete(database.NamedColors)
}

loadDatabaseTextures :: proc(database: ^DatabaseState) {
    destroyLoadedTextures(database)

    database.Patterns = make([dynamic]pdx.FlagTexture)
    database.ColoredEmblems = make([dynamic]pdx.FlagTexture)
    database.TexturedEmblems = make([dynamic]pdx.FlagTexture)
    {
        path, err := os.join_path([]string{ database.Settings.Path, pdx.FOLDER_GFX, pdx.FOLDER_COA, pdx.FOLDER_PATTERNS }, context.allocator)
        defer delete(path)
        if err != nil {
            fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_PATTERNS, err)
            return
        }
        loadDatabaseFolderTextures(database, path, pdx.FOLDER_PATTERNS)
    }
    {
        path, err := os.join_path([]string{ database.Settings.Path, pdx.FOLDER_GFX, pdx.FOLDER_COA, pdx.FOLDER_COLORED_EMBLEMS }, context.allocator)
        defer delete(path)
        if err != nil {
            fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_COLORED_EMBLEMS, err)
            return
        }
        loadDatabaseFolderTextures(database, path, pdx.FOLDER_COLORED_EMBLEMS)
    }
    {
        path, err := os.join_path([]string{ database.Settings.Path, pdx.FOLDER_GFX, pdx.FOLDER_COA, pdx.FOLDER_TEXTURED_EMBLEMS }, context.allocator)
        defer delete(path)
        if err != nil {
            fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_TEXTURED_EMBLEMS, err)
            return
        }
        loadDatabaseFolderTextures(database, path, pdx.FOLDER_TEXTURED_EMBLEMS)
    }
}
destroyLoadedTextures :: proc(database: ^DatabaseState) {
    if database.Patterns != nil {
        for _, index in database.Patterns {
            pdx.DestroyFlagTexture(database.Patterns[index])
        }
        delete(database.Patterns)
    }
    if database.ColoredEmblems != nil {
        for _, index in database.ColoredEmblems {
            pdx.DestroyFlagTexture(database.ColoredEmblems[index])
        }
        delete(database.ColoredEmblems)
    }
    if database.TexturedEmblems != nil {
        for _, index in database.TexturedEmblems {
            pdx.DestroyFlagTexture(database.TexturedEmblems[index])
        }
        delete(database.TexturedEmblems)
    }
}

loadDatabaseFolderTextures :: proc(database: ^DatabaseState, path, type: string) {
    walker := os.walker_create_path(path)
    defer os.walker_destroy(&walker)

    for info in os.walker_walk(&walker) {
        _ = os.walker_error(&walker) or_break

        if path, err := os.walker_error(&walker); err != nil {
            fmt.eprintfln("failed walking %s: %s", path, err)
            continue
        }

        if !strings.has_suffix(info.fullpath, ".dds") && !strings.has_suffix(info.fullpath, ".tga") {
            continue
        }

        if info.type != os.File_Type.Regular {
            continue
        }

        texture := pdx.CreateFlagTexture(info.name, info.fullpath)
        switch type {
        case pdx.FOLDER_PATTERNS:
            append(&database.Patterns, texture)
        case pdx.FOLDER_COLORED_EMBLEMS:
            append(&database.ColoredEmblems, texture)
        case pdx.FOLDER_TEXTURED_EMBLEMS:
            append(&database.TexturedEmblems, texture)
        }
    }
}