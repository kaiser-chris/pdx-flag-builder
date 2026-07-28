package pdx_flag_builder

import "core:text/regex"
import "core:fmt"
import "core:os"
import "core:strings"
import "core:strconv"
import "core:time"
import "pdx"
import rl "vendor:raylib"
import mu "vendor:microui"
import "core:sync/chan"
import "texture"
import "core:thread"
import "settings"
import "ui"

State :: struct {
    Settings: ^settings.Settings,
    Databases: [dynamic]Database,
    Context: mu.Context,
    SettingsWindowOpen: bool,
    FlagDatabaseWindowOpen: bool,
    TextureDatabaseWindowOpen: bool,
    DatabaseSearch: string,
    SidebarOpen: bool,
    SidebarWidth: i32,
    AtlasTexture: rl.Texture2D,
    TextureLoadingThread: ^thread.Thread,
    RenderTextureCache: map[int]rl.Texture2D,
    RenderTextureMap: map[string]rl.Texture2D,
    GuiTextureCache: map[string]rl.Texture2D,
    TextureMap: map[string]int,
    ImageLoadChannel: chan.Chan(ImageRequest),
    TextureLoadChannel: chan.Chan(TextureRequest),
    FlagLoadChannel: chan.Chan(FlagRequest),
    TransparencyTexture: rl.Texture2D,
    InvalidTexture: rl.Texture2D,
    Flag: pdx.Flag,
    ButtonIdentifier: i32,
    NextTextureIdentifier: int,
    ColorPickerColor: pdx.FlagColorVariant,
    InstanceEditorInstance: ^pdx.LayerInstance,
    SelectedFlagElement: SelectedFlagElement,
    Done: bool,
    RecolorShader: texture.RecolorShader,
    NamedColors: map[string]pdx.FlagColorVariant,
    Flags: map[string]pdx.Flag,
    GuiFlagMap: map[string]int,
    RenderFlagMap: map[int]pdx.Flag,
    NextFlagIdentifier: int,
    FlagExportChannel: chan.Chan(FlagExportRequest),
    Dropdown: ui.Dropdown,
}

Database :: struct {
    Settings: settings.Database,
    BufferName: [128]byte,
    BufferNameLength: int,
    BufferPath: [512]byte,
    BufferPathLength: int,
    NamedColors: [dynamic]pdx.FlagColorVariant,
    Patterns: [dynamic]pdx.FlagTexture,
    ColoredEmblems: [dynamic]pdx.FlagTexture,
    TexturedEmblems: [dynamic]pdx.FlagTexture,
    Flags: [dynamic]pdx.Flag,
}

FlagRequest :: struct {
    Identifier: int,
    Name: string,
}

ImageRequest :: struct {
    Identifier: int,
    Path: string,
}

TextureRequest :: struct {
    Identifier: int,
    Path: string,
    Image: rl.Image,
}

ExportRequestType :: enum {
    Script,
    Image,
}

FlagExportRequest :: struct {
    Flag: pdx.Flag,
    Type: ExportRequestType,
    Size: [2]i32,
}

state := State{}

CreateState :: proc() {
    state.Dropdown = ui.CreateDropdown()
    state.Databases = make([dynamic]Database)
    state.RenderTextureCache = make(map[int]rl.Texture2D)
    state.RenderTextureMap = make(map[string]rl.Texture2D)
    state.GuiTextureCache = make(map[string]rl.Texture2D)
    state.NamedColors = make(map[string]pdx.FlagColorVariant)
    state.TextureMap = make(map[string]int)
    state.GuiFlagMap = make(map[string]int)
    state.RenderFlagMap = make(map[int]pdx.Flag)
    imageChannel, imErr := chan.create(chan.Chan(ImageRequest), 2048, context.allocator)
    if imErr != nil {
        fmt.eprintfln("Could not create ImageLoadChannel: %v", imErr)
    }
    state.ImageLoadChannel = imageChannel
    textureChannel, txErr := chan.create(chan.Chan(TextureRequest), 32, context.allocator)
    if txErr != nil {
        fmt.eprintfln("Could not create TextureLoadChannel: %v", txErr)
    }
    state.TextureLoadChannel = textureChannel
    flagChannel, flgErr := chan.create(chan.Chan(FlagRequest), 2048, context.allocator)
    if flgErr != nil {
        fmt.eprintfln("Could not create FlagLoadChannel: %v", txErr)
    }
    state.FlagLoadChannel = flagChannel
    exportChannel, expErr := chan.create(chan.Chan(FlagExportRequest), 32, context.allocator)
    if expErr != nil {
        fmt.eprintfln("Could not create FlagExportChannel: %v", txErr)
    }
    state.FlagExportChannel = exportChannel
    state.SidebarOpen = true
    state.SidebarWidth = 350
    flag := pdx.CreateFlag()
    state.Flags["init"] = flag
    state.Flag = flag
    state.NextTextureIdentifier = 1
    state.NextFlagIdentifier = 1
    state.TextureLoadingThread = createTextureLoadingThread()
    state.SelectedFlagElement = &state.Flag
    state.Settings = settings.LoadSettings(settings.FILE_NAME_SETTINGS)
    for database in state.Settings.Databases {
        append(&state.Databases, CreateDatabase(database.Name, database.Path))
    }
    for _, i in state.Databases {
        for _, j in state.Databases[i].Flags {
            EnrichLoadedFlag(&state.Databases[i].Flags[j], &state.Databases)
            state.Flags[state.Databases[i].Flags[j].Name] = state.Databases[i].Flags[j]
        }
    }
}
DestroyState :: proc() {
    state.Done = true
    for !thread.is_done(state.TextureLoadingThread) {
        time.sleep(10)
    }
    for _, index in state.Databases {
        DestroyDatabase(&state.Databases[index])
    }
    for key in state.RenderTextureCache {
        rl.UnloadTexture(state.RenderTextureCache[key])
    }
    delete(state.RenderTextureCache)
    for key in state.GuiTextureCache {
        delete(key)
    }
    delete(state.GuiTextureCache)
    for key in state.TextureMap {
        delete(key)
    }
    for key in state.GuiFlagMap {
        delete(key)
    }
    flag, exists := state.Flags["init"]
    if exists {
        pdx.DestroyFlag(flag)
    }
    delete(state.GuiFlagMap)
    delete(state.RenderFlagMap)
    delete(state.Flags)
    delete(state.TextureMap)
    delete(state.RenderTextureMap)
    delete(state.Databases)
    delete(state.DatabaseSearch)
    chan.destroy(state.ImageLoadChannel)
    chan.destroy(state.TextureLoadChannel)
    chan.destroy(state.FlagLoadChannel)
    chan.destroy(state.FlagExportChannel)
    thread.destroy(state.TextureLoadingThread)
    delete(state.NamedColors)
    settings.DestroySettings(state.Settings)
}

CreateDatabase :: proc(name, path: string) -> Database {
    database := Database{
        Settings = settings.CreateDatabase(name, path),
        NamedColors = make([dynamic]pdx.FlagColorVariant),
        Patterns = make([dynamic]pdx.FlagTexture),
        TexturedEmblems = make([dynamic]pdx.FlagTexture),
        ColoredEmblems = make([dynamic]pdx.FlagTexture),
        Flags = make([dynamic]pdx.Flag),
    }
    if name != "" {
        database.BufferNameLength = len(name)
        copy(database.BufferName[:], name)
    }
    if path != "" {
        database.BufferPathLength = len(path)
        copy(database.BufferPath[:], path)
    }
    loadNamedColors(&database)
    loadDatabaseTextures(&database)
    loadExistingFlags(&database)
    return database
}
DestroyDatabase :: proc(database: ^Database) {
    settings.DestroyDatabase(&database.Settings)
    destroyNamedColors(database)
    destroyDatabaseFlags(database)
    destroyLoadedTextures(database)
}

loadNamedColors :: proc(database: ^Database) {
    destroyNamedColors(database)

    database.NamedColors = make([dynamic]pdx.FlagColorVariant)
    path, err := os.join_path([]string{ database.Settings.Path, pdx.FOLDER_COMMON, pdx.FOLDER_NAMED_COLORS }, context.allocator)
    if err != nil {
        fmt.eprintfln("could not build named colors path for database %s: %v", database.Settings.Name, err)
        return
    }
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
                persistantName := strings.clone(name)
                color := new_clone(pdx.FlagColorRgb{
                    Name = persistantName,
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
                state.NamedColors[persistantName] = color
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

                persistantName := strings.clone(name)
                color := new_clone(pdx.FlagColorHsv{
                    Name = persistantName,
                    H = hue * 360,
                    S = saturation * 100,
                    V = value * 100,
                })
                append(&database.NamedColors, color)
                state.NamedColors[persistantName] = color
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

                persistantName := strings.clone(name)
                color := new_clone(pdx.FlagColorHsv{
                    Name = persistantName,
                    H = f32(hue),
                    S = f32(saturation),
                    V = f32(value),
                })
                append(&database.NamedColors, color)
                state.NamedColors[persistantName] = color
                continue
            }

            fmt.eprintfln("unknown named color type: %s = ( %s %s %s )", name, capture.groups[3], capture.groups[4], capture.groups[5])
        }
    }

    fmt.printfln("Loaded %i named colors from database: %s", len(database.NamedColors), database.Settings.Name)
}
destroyNamedColors :: proc(database: ^Database) {
    if database.NamedColors == nil {
        return
    }
    for color, index in database.NamedColors {
        switch type in color {
        case ^pdx.FlagColorHsv:
            delete_key(&state.NamedColors, type.Name)
        case ^pdx.FlagColorRgb:
            delete_key(&state.NamedColors, type.Name)
        case ^pdx.FlagColorNamed:
            delete_key(&state.NamedColors, type.Name)
        case ^pdx.FlagColorReference:
            delete_key(&state.NamedColors, type.Name)
        }
        pdx.DestroyFlagColor(database.NamedColors[index])
    }
    delete(database.NamedColors)
}

loadExistingFlags :: proc(database: ^Database) {
    destroyDatabaseFlags(database)

    database.Flags = make([dynamic]pdx.Flag)
    path, err := os.join_path([]string{ database.Settings.Path, pdx.FOLDER_COMMON, pdx.FOLDER_COA, pdx.FOLDER_COA }, context.allocator)
    if err != nil {
        fmt.eprintfln("could not build coat of arms path for database %s: %v", database.Settings.Name, err)
        return
    }
    defer delete(path)

    if !os.exists(path) {
        return
    }

    walker := os.walker_create_path(path)
    defer os.walker_destroy(&walker)

    for info in os.walker_walk(&walker) {
        _ = os.walker_error(&walker) or_break

        if path, err := os.walker_error(&walker); err != nil {
            fmt.eprintfln("failed walking %s: %s", path, err)
            continue
        }

        if info.type == .Directory && path != info.fullpath {
            walker.skip_dir = true
            continue
        }

        if !strings.has_suffix(info.fullpath, ".txt") {
            continue
        }

        if info.type != .Regular {
            continue
        }

        flags := pdx.LoadCoaFile(info.fullpath)
        for flag, index in flags {
            append(&database.Flags, flags[index])
        }
        delete(flags)
    }

    fmt.printfln("Loaded %i coat of arms from database: %s", len(database.Flags), database.Settings.Name)
}
destroyDatabaseFlags :: proc(database: ^Database) {
    if database.Flags == nil {
        return
    }
    for _, index in database.Flags {
        pdx.DestroyFlag(database.Flags[index])
    }
    delete(database.Flags)
}

loadDatabaseTextures :: proc(database: ^Database) {
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
destroyLoadedTextures :: proc(database: ^Database) {
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

loadDatabaseFolderTextures :: proc(database: ^Database, path, type: string) {
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

EnrichLoadedFlag  :: proc(flag: ^pdx.Flag, databases: ^[dynamic]Database) -> bool {
    if flag.Pattern.Path == "" && flag.Pattern.Name != "" {
        path, found := findPatternTexturePath(flag.Pattern.Name, databases)
        if !found {
            fmt.eprintfln("could not find pattern texture path: %s", flag.Pattern.Name)
            return false
        }
        name := strings.clone(flag.Pattern.Name)
        defer delete(name)
        pdx.DestroyFlagTexture(flag.Pattern)
        flag.Pattern = pdx.CreateFlagTexture(name, path)
    }
    for variant, index in flag.Layers {
        switch layer in variant {
        case ^pdx.FlagLayerColoredEmblem:
            if layer.Texture.Path == "" && layer.Texture.Name != "" {
                path, found := findColoredEmblemTexturePath(layer.Texture.Name, databases)
                if !found {
                    fmt.eprintfln("could not find colored emblem texture path: %s", layer.Texture.Name)
                    return false
                }
                name := strings.clone(layer.Texture.Name)
                defer delete(name)
                pdx.DestroyFlagTexture(layer.Texture)
                layer.Texture = pdx.CreateFlagTexture(name, path)
            }
        case ^pdx.FlagLayerTexturedEmblem:
            if layer.Texture.Path == "" && layer.Texture.Name != "" {
                path, found := findTexturedEmblemTexturePath(layer.Texture.Name, databases)
                if !found {
                    fmt.eprintfln("could not find textured emblem texture path: %s", layer.Texture.Name)
                    return false
                }
                name := strings.clone(layer.Texture.Name)
                defer delete(name)
                pdx.DestroyFlagTexture(layer.Texture)
                layer.Texture = pdx.CreateFlagTexture(name, path)
            }
        }
    }

    return true
}

findPatternTexturePath :: proc(name: string, databases: ^[dynamic]Database) -> (string, bool) {
    for database in databases {
        for texture in database.Patterns {
            if name == texture.Name {
                return texture.Path, true
            }
        }
    }
    return "", false
}
findColoredEmblemTexturePath :: proc(name: string, databases: ^[dynamic]Database) -> (string, bool) {
    for database in databases {
        for texture in database.ColoredEmblems {
            if name == texture.Name {
                return texture.Path, true
            }
        }
    }
    return "", false
}
findTexturedEmblemTexturePath :: proc(name: string, databases: ^[dynamic]Database) -> (string, bool) {
    for database in databases {
        for texture in database.TexturedEmblems {
            if name == texture.Name {
                return texture.Path, true
            }
        }
    }
    return "", false
}
resolveNamedColor :: proc(name: string) -> (rl.Color, pdx.FlagColorVariant) {
    variant, exists := state.NamedColors[name]
    if !exists {
        return rl.Color{}, nil
    }
    switch color in variant {
    case ^pdx.FlagColorRgb:
        return pdx.ToRenderColor(variant), variant
    case ^pdx.FlagColorHsv:
        return pdx.ToRenderColor(variant), variant
    case ^pdx.FlagColorNamed:
    case ^pdx.FlagColorReference:
    }
    return rl.Color{}, nil
}

SaveSettings :: proc() {
    for _, index in state.Settings.Databases {
        settings.DestroyDatabase(&state.Settings.Databases[index])
    }
    delete(state.Settings.Databases)
    state.Settings.Databases = make([dynamic]settings.Database, len(state.Databases))
    for _, index in state.Databases {
        database := state.Databases[index]
        state.Settings.Databases[index] = settings.CreateDatabase(database.Settings.Name, database.Settings.Path)
        loadNamedColors(&database)
        loadDatabaseTextures(&database)
        loadExistingFlags(&database)
    }
    settings.SaveSettings(settings.FILE_NAME_SETTINGS, state.Settings)
}