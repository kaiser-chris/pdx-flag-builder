package pdx_flag_builder

import "core:fmt"
import "core:os"
import "core:strings"
import "core:time"
import "pdx"
import rl "vendor:raylib"
import mu "vendor:microui"
import "core:sync/chan"
import "texture"
import "core:thread"
import "settings"
import "ui"

FLAG_ACTIVE: string: "init"

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
    ColorPickerColor: ^pdx.FlagColor,
    InstanceEditorInstance: pdx.LayerInstanceVariant,
    SelectedFlagElement: SelectedFlagElement,
    Done: bool,
    RecolorShader: texture.RecolorShader,
    Flags: map[string]pdx.Flag,
    GuiFlagMap: map[string]int,
    RenderFlagMap: map[int]pdx.Flag,
    NextFlagIdentifier: int,
    FlagExportChannel: chan.Chan(FlagExportRequest),
    Dropdown: ui.Dropdown,
    Toast: ui.ToastContainer,
    Icons: map[ui.IconType]rl.Texture2D,
    SearchCache: SearchCache,
}

SearchCache :: struct {
    FlagDatabaseSearch: string,
    TextureDatabaseSearch: string,
    NamedColorSearch: string,
}

DatabaseType :: enum {
    Vic3,
    Eu5,
    Unknown,
}

Database :: struct {
    Type: DatabaseType,
    Settings: settings.Database,
    Name: string,
    Path: string,
    IsDeleted: bool,
    IsNew: bool,
    BufferName: [128]byte,
    BufferNameLength: int,
    BufferPath: [512]byte,
    BufferPathLength: int,
    Patterns: [dynamic]pdx.FlagTexture,
    ColoredEmblems: [dynamic]pdx.FlagTexture,
    TexturedEmblems: [dynamic]pdx.FlagTexture,
    Flags: [dynamic]pdx.Flag,
    NamedColors: [dynamic]pdx.FlagColor,
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
    Size: [2]i32,
}

state := State{}

CreateState :: proc() {
    state.Dropdown = ui.CreateDropdown()
    state.Toast = ui.CreateToastContainer()
    state.SearchCache = CreateSearchCache()
    state.Databases = make([dynamic]Database)
    state.RenderTextureCache = make(map[int]rl.Texture2D)
    state.RenderTextureMap = make(map[string]rl.Texture2D)
    state.GuiTextureCache = make(map[string]rl.Texture2D)
    state.Icons = make(map[ui.IconType]rl.Texture2D)
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
    state.Flag = pdx.CreateFlag()
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
    delete(state.Flags)
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
    delete(state.GuiFlagMap)
    delete(state.RenderFlagMap)
    delete(state.TextureMap)
    delete(state.RenderTextureMap)
    delete(state.Databases)
    delete(state.DatabaseSearch)
    delete(state.Icons)
    chan.destroy(state.ImageLoadChannel)
    chan.destroy(state.TextureLoadChannel)
    chan.destroy(state.FlagLoadChannel)
    chan.destroy(state.FlagExportChannel)
    thread.destroy(state.TextureLoadingThread)
    settings.DestroySettings(state.Settings)
    pdx.DestroyFlag(&state.Flag)
    DestroySearchCache(&state.SearchCache)
    ui.DestroyDropdown(&state.Dropdown)
    ui.DestroyToastContainer(&state.Toast)
}

NewDatabase :: proc(name, path: string) -> Database {
    return Database{
        Type = .Unknown,
        Name = strings.clone(name),
        Path = strings.clone(path),
        Settings = settings.CreateDatabase(name, path),
        NamedColors = make([dynamic]pdx.FlagColor),
        Patterns = make([dynamic]pdx.FlagTexture),
        TexturedEmblems = make([dynamic]pdx.FlagTexture),
        ColoredEmblems = make([dynamic]pdx.FlagTexture),
        Flags = make([dynamic]pdx.Flag),
    }
}
CreateDatabase :: proc(name, path: string) -> Database {
    database := NewDatabase(name, path)
    database.BufferNameLength = len(name)
    copy(database.BufferName[:], name)
    database.BufferPathLength = len(path)
    copy(database.BufferPath[:], path)
    determineDatabaseType(&database)
    loadNamedColors(&database)
    loadDatabaseTextures(&database)
    loadExistingFlags(&database)
    return database
}
DestroyDatabase :: proc(database: ^Database) {
    settings.DestroyDatabase(&database.Settings)
    delete(database.Name)
    delete(database.Path)
    destroyNamedColors(database)
    destroyDatabaseFlags(database)
    destroyLoadedTextures(database)
}

CreateSearchCache :: proc() -> SearchCache {
    return SearchCache{
        FlagDatabaseSearch = strings.clone(""),
        TextureDatabaseSearch = strings.clone(""),
        NamedColorSearch = strings.clone(""),
    }
}
DestroySearchCache :: proc(cache: ^SearchCache) {
    delete(cache.FlagDatabaseSearch)
    delete(cache.TextureDatabaseSearch)
    delete(cache.NamedColorSearch)
}

determineDatabaseType :: proc(database: ^Database) {
    defer free_all(context.temp_allocator)

    folderMainMenu, mmErr := os.join_path({ database.Settings.Path, pdx.FOLDER_EU5_MAIN_MENU }, context.temp_allocator)
    if mmErr != nil {
        fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_EU5_MAIN_MENU, mmErr)
        return
    }
    folderLoadingScreen, lsErr := os.join_path({ database.Settings.Path, pdx.FOLDER_EU5_LOADING_SCREEN }, context.temp_allocator)
    if lsErr != nil {
        fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_EU5_LOADING_SCREEN, lsErr)
        return
    }
    folderInGame, igErr := os.join_path({ database.Settings.Path, pdx.FOLDER_EU5_IN_GAME }, context.temp_allocator)
    if igErr != nil {
        fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_EU5_IN_GAME, igErr)
        return
    }
    if os.exists(folderMainMenu) || os.exists(folderLoadingScreen) || os.exists(folderInGame) {
        database.Type = .Eu5
        return
    }

    folderCommon, comErr := os.join_path({ database.Settings.Path, pdx.FOLDER_COMMON }, context.temp_allocator)
    if comErr != nil {
        fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_COMMON, comErr)
        return
    }
    folderGfx, gfxErr := os.join_path({ database.Settings.Path, pdx.FOLDER_GFX }, context.temp_allocator)
    if gfxErr != nil {
        fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_GFX, gfxErr)
        return
    }
    if os.exists(folderCommon) || os.exists(folderGfx) {
        database.Type = .Vic3
        return
    }

    database.Type = .Unknown
}
determineRootFolder :: proc(database: ^Database) -> []string {
    rootFolders := make([dynamic]string)
    switch database.Type {
    case .Vic3:
        append(&rootFolders, "")
        return rootFolders[:]
    case .Eu5:
        append(&rootFolders, pdx.FOLDER_EU5_MAIN_MENU)
        append(&rootFolders, pdx.FOLDER_EU5_LOADING_SCREEN)
        append(&rootFolders, pdx.FOLDER_EU5_IN_GAME)
        return rootFolders[:]
    case .Unknown:
        return {}
    }
    return {}
}

loadNamedColors :: proc(database: ^Database) {
    destroyNamedColors(database)

    database.NamedColors = make([dynamic]pdx.FlagColor)

    rootFolders := determineRootFolder(database)
    defer delete(rootFolders)
    for root in rootFolders {
        path, err := os.join_path([]string{ database.Settings.Path, root, pdx.FOLDER_COMMON, pdx.FOLDER_NAMED_COLORS }, context.allocator)
        if err != nil {
            fmt.eprintfln("could not build named colors path for database %s: %v", database.Settings.Name, err)
            continue
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

            colors := pdx.LoadNamedColorsFile(info.fullpath)
            for _, index in colors {
                append(&database.NamedColors, colors[index])
            }
            delete(colors)

        }
    }

    fmt.printfln("Loaded %i named colors from database: %s", len(database.NamedColors), database.Settings.Name)
}
destroyNamedColors :: proc(database: ^Database) {
    if database.NamedColors == nil {
        return
    }
    for _, index in database.NamedColors {
        pdx.DestroyFlagColor(&database.NamedColors[index])
    }
    delete(database.NamedColors)
}

loadExistingFlags :: proc(database: ^Database) {
    destroyDatabaseFlags(database)

    database.Flags = make([dynamic]pdx.Flag)

    rootFolders := determineRootFolder(database)
    defer delete(rootFolders)
    for root in rootFolders {
        path, err := os.join_path([]string{ database.Settings.Path, root, pdx.FOLDER_COMMON, pdx.FOLDER_COA, pdx.FOLDER_COA }, context.allocator)
        if err != nil {
            fmt.eprintfln("could not build coat of arms path for database %s: %v", database.Settings.Name, err)
            continue
        }
        defer delete(path)

        if !os.exists(path) {
            continue
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

            flags := pdx.LoadCoaFile(info.fullpath, database.Settings.Name)
            for flag, index in flags {
                append(&database.Flags, flags[index])
            }
            delete(flags)
        }
    }

    fmt.printfln("Loaded %i coat of arms from database: %s", len(database.Flags), database.Settings.Name)
}
destroyDatabaseFlags :: proc(database: ^Database) {
    if database.Flags == nil {
        return
    }
    for _, index in database.Flags {
        pdx.DestroyFlag(&database.Flags[index])
    }
    delete(database.Flags)
}

loadDatabaseTextures :: proc(database: ^Database) {
    destroyLoadedTextures(database)

    database.Patterns = make([dynamic]pdx.FlagTexture)
    database.ColoredEmblems = make([dynamic]pdx.FlagTexture)
    database.TexturedEmblems = make([dynamic]pdx.FlagTexture)

    rootFolders := determineRootFolder(database)
    defer delete(rootFolders)
    for root in rootFolders {
        {
            path, err := os.join_path({ database.Settings.Path, root, pdx.FOLDER_GFX, pdx.FOLDER_COA, pdx.FOLDER_PATTERNS }, context.allocator)
            defer delete(path)
            if err != nil {
                fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_PATTERNS, err)
                continue
            }
            loadDatabaseFolderTextures(database, path, pdx.FOLDER_PATTERNS)
        }
        {
            path, err := os.join_path({ database.Settings.Path, root, pdx.FOLDER_GFX, pdx.FOLDER_COA, pdx.FOLDER_COLORED_EMBLEMS }, context.allocator)
            defer delete(path)
            if err != nil {
                fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_COLORED_EMBLEMS, err)
                continue
            }
            loadDatabaseFolderTextures(database, path, pdx.FOLDER_COLORED_EMBLEMS)
        }
        {
            path, err := os.join_path({ database.Settings.Path, root, pdx.FOLDER_GFX, pdx.FOLDER_COA, pdx.FOLDER_TEXTURED_EMBLEMS }, context.allocator)
            defer delete(path)
            if err != nil {
                fmt.eprintfln("Could not build %s path: %v", pdx.FOLDER_TEXTURED_EMBLEMS, err)
                continue
            }
            loadDatabaseFolderTextures(database, path, pdx.FOLDER_TEXTURED_EMBLEMS)
        }
    }
}
destroyLoadedTextures :: proc(database: ^Database) {
    if database.Patterns != nil {
        for _, index in database.Patterns {
            pdx.DestroyFlagTexture(&database.Patterns[index])
        }
        delete(database.Patterns)
    }
    if database.ColoredEmblems != nil {
        for _, index in database.ColoredEmblems {
            pdx.DestroyFlagTexture(&database.ColoredEmblems[index])
        }
        delete(database.ColoredEmblems)
    }
    if database.TexturedEmblems != nil {
        for _, index in database.TexturedEmblems {
            pdx.DestroyFlagTexture(&database.TexturedEmblems[index])
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
        pdx.DestroyFlagTexture(&flag.Pattern)
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
                pdx.DestroyFlagTexture(&layer.Texture)
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
                pdx.DestroyFlagTexture(&layer.Texture)
                layer.Texture = pdx.CreateFlagTexture(name, path)
            }
        case ^pdx.FlagLayerSub:
            // Nothing to do
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

getNamedColor :: proc(name: string) -> (pdx.FlagColor, bool) {
    for database in state.Databases {
        for color in database.NamedColors {
            if color.Name == name {
                return color, true
            }
        }
    }
    return pdx.FlagColor{}, false
}

getReferencedColor :: proc(reference: string, flag: pdx.Flag) -> (pdx.FlagColor, bool) {
    referencedColor: pdx.FlagColor
    found: bool
    for color in flag.Colors {
        if color.Name == reference {
            referencedColor = color
            found = true
        }
    }
    #partial switch type in referencedColor.Variant {
    case pdx.FlagColorNamed:
        referencedColor, found = getNamedColor(type.NamedColor)
    }
    if found {
        return referencedColor, true
    }
    return pdx.FlagColor{}, false
}

SaveSettings :: proc() {
//    for key in state.TextureMap {
//        delete(key)
//    }
//    for key in state.GuiFlagMap {
//        delete(key)
//    }
//    clear_map(&state.GuiFlagMap)
//    clear_map(&state.RenderFlagMap)
    state.FlagDatabaseWindowOpen = false
    state.TextureDatabaseWindowOpen = false
    for _, index in state.Settings.Databases {
        settings.DestroyDatabase(&state.Settings.Databases[index])
    }
    clear(&state.Settings.Databases)
    delete(state.Settings.Databases)
    state.Settings.Databases = make([dynamic]settings.Database)
    for database in state.Databases {
        if database.IsDeleted {
            continue
        }
        newDatabase := settings.CreateDatabase(database.Name, database.Path)
        append(&state.Settings.Databases, newDatabase)
    }
    for _, index in state.Databases {
        DestroyDatabase(&state.Databases[index])
    }
    clear(&state.Databases)
    delete(state.Databases)
    state.Databases = make([dynamic]Database)
    delete(state.Flags)
    state.Flags = make(map[string]pdx.Flag)
    for database in state.Settings.Databases {
        append(&state.Databases, CreateDatabase(database.Name, database.Path))
    }
    for _, i in state.Databases {
        for _, j in state.Databases[i].Flags {
            EnrichLoadedFlag(&state.Databases[i].Flags[j], &state.Databases)
            state.Flags[state.Databases[i].Flags[j].Name] = state.Databases[i].Flags[j]
        }
    }
    settings.SaveSettings(settings.FILE_NAME_SETTINGS, state.Settings)
}

loadIcons :: proc() {
    state.Icons[.Sub] = texture.LoadTexture(TEXTURE_ICON_SUB)
    state.Icons[.Edit] = texture.LoadTexture(TEXTURE_ICON_EDIT)
    state.Icons[.Delete] = texture.LoadTexture(TEXTURE_ICON_DELETE)
    state.Icons[.ArrowUp] = texture.LoadTexture(TEXTURE_ICON_ARROW_UP)
    state.Icons[.ArrowDown] = texture.LoadTexture(TEXTURE_ICON_ARROW_DOWN)
    state.Icons[.Unknown] = texture.LoadTexture(TEXTURE_ICON_UNKNOWN)
}
unloadIcons :: proc() {
    for key in state.Icons {
        rl.UnloadTexture(state.Icons[key])
    }
}

removeLayer :: proc(layer: pdx.FlagLayerVariant) {
    switch type in layer {
    case ^pdx.FlagLayerColoredEmblem:
        for _, index in state.Flag.Layers {
            if state.Flag.Layers[index] == layer {
                ordered_remove(&state.Flag.Layers, index)
                pdx.DestroyFlagLayer(layer)
            }
        }
    case ^pdx.FlagLayerTexturedEmblem:
        for _, index in state.Flag.Layers {
            if state.Flag.Layers[index] == layer {
                ordered_remove(&state.Flag.Layers, index)
                pdx.DestroyFlagLayer(layer)
            }
        }
    case ^pdx.FlagLayerSub:
        for _, index in state.Flag.Layers {
            if state.Flag.Layers[index] == layer {
                ordered_remove(&state.Flag.Layers, index)
                pdx.DestroyFlagLayer(layer)
            }
        }
    }
}

removeInstance :: proc(instance: pdx.LayerInstanceVariant) {
    for layer in state.Flag.Layers {
        removeLayerInstance(layer, instance)
    }
    switch type in instance {
    case ^pdx.LayerInstance:
        pdx.DestroyLayerInstance(type)
    case ^pdx.LayerInstanceSub:
        pdx.DestroyLayerInstanceSub(type)
    }
}

removeLayerInstance :: proc(layerVariant: pdx.FlagLayerVariant, instance: pdx.LayerInstanceVariant) {
    switch layer in layerVariant {
    case ^pdx.FlagLayerColoredEmblem:
        for _, index in layer.Instances {
            #partial switch type in instance {
            case ^pdx.LayerInstance:
                if layer.Instances[index] == type {
                    ordered_remove(&layer.Instances, index)
                }
            }
        }
    case ^pdx.FlagLayerTexturedEmblem:
        for _, index in layer.Instances {
            #partial switch type in instance {
            case ^pdx.LayerInstance:
                if layer.Instances[index] == type {
                    ordered_remove(&layer.Instances, index)
                }
            }
        }
    case ^pdx.FlagLayerSub:
        for _, index in layer.Instances {
            #partial switch type in instance {
            case ^pdx.LayerInstanceSub:
                if layer.Instances[index] == type {
                    ordered_remove(&layer.Instances, index)
                }
            }
        }
    }
}