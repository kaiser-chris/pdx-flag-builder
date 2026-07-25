package pdx_flag_builder

import "core:fmt"
import "core:os"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"

WINDOM_PREVIEW: string: "Preview"
WINDOM_TOOLBAR: string: "Toolbar"
WINDOM_SETTINGS: string: "Settings"
WINDOM_DATABASE: string: "Database"
WINDOM_FLAG: string: "Flag"
WINDOM_SELECTED: string: "Selected"
WINDOM_SIDEBAR: string: "Side"

SIDEBAR_HANDLE_WIDTH:i32: 10
SIDEBAR_MIN_WIDTH: i32: 200
SIDEBAR_MAX_WIDTH: i32: 600
TOOLBAR_HEIGHT: i32: 35
FLAG_HEIGHT: i32: 512
FLAG_WIDTH: i32: 768

SelectedFlagElement :: union {
    ^Flag,
    ^FlagLayerColoredEmblem,
    ^FlagLayerTexturedEmblem,
    ^FlagColorNamed,
    ^FlagColorHsv,
    ^FlagColorRgb,
}

renderGui :: proc(ctx: ^mu.Context) {
    renderFlagPreview(ctx)

    renderToolbar(ctx)

    if state.SettingsWindowOpen {
        window := mu.get_container(ctx, WINDOM_SETTINGS)
        if window != nil {
            window.open = true
        }
        renderSettings(ctx)
    }

    if state.DatabaseWindowOpen {
        window := mu.get_container(ctx, WINDOM_DATABASE)
        if window != nil {
            window.open = true
        }
        renderDatabase(ctx)
    }

    if state.SidebarOpen {
        window := mu.get_container(ctx, WINDOM_FLAG)
        if window != nil {
            window.open = true
        }
        renderSidebar(ctx)
    }

    rl.SetWindowMinSize(FLAG_WIDTH + state.SidebarWidth, FLAG_HEIGHT + TOOLBAR_HEIGHT)
}

renderToolbar :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_TOOLBAR, { 0, 0, rl.GetScreenWidth(), TOOLBAR_HEIGHT }, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL}) {
        mu.get_current_container(ctx).rect.w = rl.GetScreenWidth()
        mu.layout_row_items(ctx, 4, TOOLBAR_HEIGHT - 8)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Sidebar") {
            if state.SidebarOpen {
                state.SidebarOpen = false
            } else {
                state.SidebarOpen = true
            }
        }
        mu.layout_end_column(ctx)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "CoA Database") {
            if state.DatabaseWindowOpen {
                state.DatabaseWindowOpen = false
            } else {
                state.DatabaseWindowOpen = true
            }
        }
        mu.layout_end_column(ctx)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Settings") {
            if state.SettingsWindowOpen {
                state.SettingsWindowOpen = false
            } else {
                state.SettingsWindowOpen = true
            }
        }
        mu.layout_end_column(ctx)
    }
}

renderFlagPreview :: proc(ctx: ^mu.Context) {
    destination := mu.Rect{
        x = (rl.GetScreenWidth() - state.TransparencyTexture.width - state.SidebarWidth) / 2,
        y = (rl.GetScreenHeight() - state.TransparencyTexture.height + TOOLBAR_HEIGHT) / 2,
        w = FLAG_WIDTH,
        h = FLAG_HEIGHT,
    }

    if mu.window(ctx, WINDOM_PREVIEW, destination, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL, .NO_FRAME}) {
        window := mu.get_current_container(ctx)
        window.rect = destination
        target := mu.Rect{x = 0, y = 0, w = FLAG_WIDTH, h = FLAG_HEIGHT}
        mu.layout_set_next(ctx, target, true)
        drawTransparentTexture(ctx, "textures/transparency.dds")
        if state.Flag.Pattern.Name != "" {
            mu.layout_set_next(ctx, target, true)
            drawTransparentTexture(ctx, state.Flag.Pattern.Path)
        }
        for variant in state.Flag.Layers {
            switch layer in variant {
            case ^FlagLayerColoredEmblem:
                mu.layout_set_next(ctx, target, true)
                drawTransparentTexture(ctx, layer.Texture.Path)
            case ^FlagLayerTexturedEmblem:
                mu.layout_set_next(ctx, target, true)
                drawTransparentTexture(ctx, layer.Texture.Path)
            }
        }
    }
}

renderSettings :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_SETTINGS, {10, 10 + TOOLBAR_HEIGHT, 500, 400}, {}) {
        guranteeBounds(ctx)
        mu.layout_row(ctx, {-1, -1}, 0)
        if .SUBMIT in mu.button(ctx, "Save Settings") {
            saveSettings()
        }

        if .ACTIVE in mu.header(ctx, "Paths", {.EXPANDED}) {
            mu.text(ctx, "These are paths to the root of your mod or the \"game\" folder for the base game.")
            mu.layout_row(ctx, {100, -1}, 0)
            for _, index in state.Databases {
                if .CHANGE in mu.textbox(ctx, state.Databases[index].BufferName[:], &state.Databases[index].BufferNameLength) {
                    state.Databases[index].Settings.Name = string(state.Databases[index].BufferName[:state.Databases[index].BufferNameLength])
                }
                if .CHANGE in mu.textbox(ctx, state.Databases[index].BufferPath[:], &state.Databases[index].BufferPathLength) {
                    state.Databases[index].Settings.Path = string(state.Databases[index].BufferPath[:state.Databases[index].BufferPathLength])
                }
            }
            mu.layout_row(ctx, {-1, -1}, 0)
            if .SUBMIT in mu.button(ctx, "Add new path") {
                element := DatabaseState{
                    Settings = FlagDatabase{"", ""},
                    NamedColors = make([dynamic]FlagColorVariant)
                }
                append(&state.Databases, element)
            }
        }

        if .ACTIVE in mu.header(ctx, "Background Color", {.EXPANDED}) {
            mu.layout_row(ctx, {-78, -1}, 68)
            mu.layout_begin_column(ctx)
            {
                mu.layout_row(ctx, {46, -1}, 0)
                mu.label(ctx, "Red:");   colorSlider(ctx, &state.Settings.BackgroundColor.r, 0, 255)
                mu.label(ctx, "Green:"); colorSlider(ctx, &state.Settings.BackgroundColor.g, 0, 255)
                mu.label(ctx, "Blue:");  colorSlider(ctx, &state.Settings.BackgroundColor.b, 0, 255)
            }
            mu.layout_end_column(ctx)

            r := mu.layout_next(ctx)
            mu.draw_rect(ctx, r, state.Settings.BackgroundColor)
            mu.draw_box(ctx, mu.expand_rect(r, 1), ctx.style.colors[.BORDER])
            mu.draw_control_text(ctx, fmt.tprintf("#%02x%02x%02x", state.Settings.BackgroundColor.r, state.Settings.BackgroundColor.g, state.Settings.BackgroundColor.b), r, .TEXT, {.ALIGN_CENTER})
        }
    }

    cnt := mu.get_container(ctx, WINDOM_SETTINGS)
    is_open := cnt != nil && cnt.open != false

    if state.SettingsWindowOpen && !is_open {
        state.SettingsWindowOpen = false
    }
}

renderDatabase :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_DATABASE, {10, 10 + TOOLBAR_HEIGHT, 500, rl.GetScreenHeight() - 20 - TOOLBAR_HEIGHT}, {}) {
        guranteeBounds(ctx)

        mu.layout_row(ctx, {-1, -1}, 15)
        mu.text(ctx, "Search:")
        @static searchBuffer: [128]byte
        @static searchBufferLength: int
        mu.layout_row(ctx, {-1, -1}, 20)
        if .CHANGE in mu.textbox(ctx, searchBuffer[:], &searchBufferLength) {
            delete(state.DatabaseSearch)
            state.DatabaseSearch = strings.to_lower(string(searchBuffer[:searchBufferLength]))
        }
        mu.layout_row(ctx, {-1, -1}, 5)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])

        options: mu.Options
        if state.DatabaseSearch != "" {
            options = { .EXPANDED }
        }

        for database in state.Databases {
            setButtonIdentifier(ctx)
            if .ACTIVE in mu.header(ctx, database.Settings.Name, {.EXPANDED}) {
                setButtonIdentifier(ctx)
                if .ACTIVE in mu.treenode(ctx, "Patterns", options) {
                    path, err := os.join_path([]string{ database.Settings.Path, FOLDER_GFX, FOLDER_COA, FOLDER_PATTERNS }, context.allocator)
                    defer delete(path)
                    if err != nil {
                        fmt.eprintfln("Could not build %s path: %v", FOLDER_PATTERNS, err)
                    }
                    renderDatabaseFolder(ctx, path, FOLDER_PATTERNS)
                }
                mu.pop_id(ctx)
                setButtonIdentifier(ctx)
                if .ACTIVE in mu.treenode(ctx, "Colored Emblems", options) {
                    path, err := os.join_path([]string{ database.Settings.Path, FOLDER_GFX, FOLDER_COA, FOLDER_COLORED_EMBLEMS }, context.allocator)
                    defer delete(path)
                    if err != nil {
                        fmt.eprintfln("Could not build %s path: %v", FOLDER_COLORED_EMBLEMS, err)
                    }
                    renderDatabaseFolder(ctx, path, FOLDER_COLORED_EMBLEMS)
                }
                mu.pop_id(ctx)
                setButtonIdentifier(ctx)
                if .ACTIVE in mu.treenode(ctx, "Textured Emblems", options) {
                    path, err := os.join_path([]string{ database.Settings.Path, FOLDER_GFX, FOLDER_COA, FOLDER_TEXTURED_EMBLEMS }, context.allocator)
                    defer delete(path)
                    if err != nil {
                        fmt.eprintfln("Could not build %s path: %v", FOLDER_TEXTURED_EMBLEMS, err)
                    }
                    renderDatabaseFolder(ctx, path, FOLDER_TEXTURED_EMBLEMS)
                }
                mu.pop_id(ctx)
            }
            mu.pop_id(ctx)
        }
    }

    cnt := mu.get_container(ctx, WINDOM_DATABASE)
    is_open := cnt != nil && cnt.open != false

    if state.DatabaseWindowOpen && !is_open {
        state.DatabaseWindowOpen = false
    }
}

renderDatabaseFolder :: proc(ctx: ^mu.Context, path: string, type: string) {
    walker := os.walker_create_path(path)
    defer os.walker_destroy(&walker)

    count: int
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

        lower := strings.to_lower(info.name)
        defer delete(lower)
        if state.DatabaseSearch != "" && !strings.contains(lower, state.DatabaseSearch) {
            continue
        }

        texture := state.GuiTextureCache[info.fullpath]
        width := i32((f32(texture.width) / f32(texture.height)) * 128)

        mu.layout_row(ctx, {200, -1}, 128)
        mu.layout_begin_column(ctx)
        {
            mu.layout_row(ctx, {192, -1}, 128)
            drawTexture(ctx, info.fullpath, width, 128)
        }
        mu.layout_end_column(ctx)
        mu.layout_begin_column(ctx)
        {
            mu.layout_row(ctx, {-1, -1}, 20)
            switch type {
            case FOLDER_PATTERNS:
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, "Set as pattern") {
                    state.Flag.Pattern = FlagTexture{
                        Name = strings.clone(info.name),
                        Path = strings.clone(info.fullpath),
                    }
                }
                mu.pop_id(ctx)
            case FOLDER_COLORED_EMBLEMS:
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, "Add as layer") {
                    layer := new_clone(FlagLayerColoredEmblem{
                        Instances = make([dynamic]LayerInstance),
                        Colors = make([dynamic]FlagColorVariant),
                    })
                    layer.Texture = FlagTexture{
                        Name = strings.clone(info.name),
                        Path = strings.clone(info.fullpath),
                    }
                    fmt.printfln("Colored Emblem: %s", layer.Texture.Name)
                    append(&state.Flag.Layers, layer)
                }
                mu.pop_id(ctx)
            case FOLDER_TEXTURED_EMBLEMS:
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, "Add as layer") {
                    layer := new_clone(FlagLayerTexturedEmblem{
                        Instances = make([dynamic]LayerInstance),
                    })
                    layer.Texture = FlagTexture{
                        Name = strings.clone(info.name),
                        Path = strings.clone(info.fullpath),
                    }
                    fmt.printfln("Textured Emblem: %s", layer.Texture.Name)
                    append(&state.Flag.Layers, layer)
                }
                mu.pop_id(ctx)
            }
            mu.layout_row(ctx, {-1, -1}, 20)
            mu.text(ctx, info.name)
            mu.layout_row(ctx, {-1, -1}, 20)
            mu.text(ctx, fmt.tprintf("%ix%i", texture.width, texture.height))
        }
        mu.layout_end_column(ctx)
        count += 1
    }
    if count == 0 {
        mu.layout_row(ctx, {-1, -1}, 50)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
        mu.draw_control_text(ctx, "No Items", r, .TEXT, {.ALIGN_CENTER})
    }
}

renderSidebar :: proc(ctx: ^mu.Context) {
    renderFlagMenu(ctx)
    renderSelectedElementMenu(ctx)
    renderSidebarHandle(ctx)

    {
        cnt := mu.get_container(ctx, WINDOM_FLAG)
        is_open := cnt != nil && cnt.open != false

        if state.SidebarOpen && !is_open {
            state.SidebarOpen = false
        }
    }
    {
        cnt := mu.get_container(ctx, WINDOM_SELECTED)
        is_open := cnt != nil && cnt.open != false

        if state.SidebarOpen && !is_open {
            state.SidebarOpen = false
        }
    }
}

renderSidebarHandle :: proc(ctx: ^mu.Context) {
    width := SIDEBAR_HANDLE_WIDTH
    height := rl.GetScreenHeight() - (TOOLBAR_HEIGHT + 1)
    x := rl.GetScreenWidth() - width - state.SidebarWidth
    y := TOOLBAR_HEIGHT + 1

    if mu.window(ctx, WINDOM_SIDEBAR, { x, y, width, height }, {.NO_RESIZE, .NO_CLOSE, .NO_TITLE, .NO_SCROLL}) {
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height
        window.rect.x = x
        window.rect.y = y

        mu.layout_set_next(ctx, window.rect, false)
        r := mu.layout_next(ctx)

        id := mu.get_id(ctx, "sidebar-resize")
        mu.update_control(ctx, id, r)

        if ctx.focus_id == id && .LEFT in ctx.mouse_down_bits {
            state.SidebarWidth -= ctx.mouse_delta[0]

            if state.SidebarWidth < SIDEBAR_MIN_WIDTH {
                state.SidebarWidth = SIDEBAR_MIN_WIDTH
            }

            if state.SidebarWidth > SIDEBAR_MAX_WIDTH {
                state.SidebarWidth = SIDEBAR_MAX_WIDTH
            }
        }

        if mu.mouse_over(ctx, r) || ctx.focus_id == id {
            mu.draw_rect(ctx, r, ctx.style.colors[.BUTTON_HOVER])
        } else {
            mu.draw_rect(ctx, r, ctx.style.colors[.BORDER])
        }

        mu.draw_control_text(ctx, "||", r, .TEXT, {.ALIGN_CENTER})
    }
}

renderFlagMenu :: proc(ctx: ^mu.Context) {
    width := state.SidebarWidth
    height: i32

    if state.SidebarOpen {
        height = 300
    } else {
        height = rl.GetScreenHeight() - (TOOLBAR_HEIGHT + 1)
    }
    x := rl.GetScreenWidth() - width
    y := TOOLBAR_HEIGHT + 1

    if mu.window(ctx, WINDOM_FLAG, { x, y, width, height }, {.NO_RESIZE, .NO_CLOSE, .NO_INTERACT}) {
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height
        window.rect.x = x
        window.rect.y = y

        mu.layout_row(ctx, {-1, -1}, 20)
        setButtonIdentifier(ctx)
        if .SUBMIT in mu.button(ctx, "Select Flag") {
            state.SelectedFlagElement = &state.Flag
        }
        mu.pop_id(ctx)

        for variant, index in state.Flag.Layers {
            switch layer in variant {
            case ^FlagLayerColoredEmblem:
                mu.layout_row(ctx, {-1, -1}, 20)
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Colored Emblem: %s", layer.Texture.Name)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
            case ^FlagLayerTexturedEmblem:
                mu.layout_row(ctx, {-1, -1}, 20)
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Textured Emblem: %s", layer.Texture.Name)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
            }
        }
        mu.layout_row(ctx, { -1, -1 }, 20)
    }
}

renderSelectedElementMenu :: proc(ctx: ^mu.Context) {
    width := state.SidebarWidth
    height: i32
    x: i32
    y: i32

    if state.SidebarOpen {
        flagWindow := mu.get_container(ctx, WINDOM_FLAG)
        height = rl.GetScreenHeight() - (flagWindow.rect.y + flagWindow.rect.h + 1)
        x = rl.GetScreenWidth() - width
        y = flagWindow.rect.y + flagWindow.rect.h + 1
    } else {
        height = rl.GetScreenHeight() - (TOOLBAR_HEIGHT + 1)
        x = rl.GetScreenWidth() - width
        y = TOOLBAR_HEIGHT + 1
    }

    if mu.window(ctx, WINDOM_SELECTED, { x, y, width, height }, {.NO_RESIZE, .NO_CLOSE, .NO_INTERACT}) {
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height
        window.rect.x = x
        window.rect.y = y

        if state.SelectedFlagElement == nil {
            mu.layout_row(ctx, {-1, -1}, 50)
            r := mu.layout_next(ctx)
            mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
            mu.draw_control_text(ctx, "Nothing Selected", r, .TEXT, {.ALIGN_CENTER})
            mu.layout_row(ctx, {-1, -1}, 50)
            r = mu.layout_next(ctx)
            mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
            mu.draw_control_text(ctx, "Click an element in the window above.", r, .TEXT, {.ALIGN_CENTER})
        }
    }
}

removeLayer :: proc(index: int){
    variant := state.Flag.Layers[index]
    ordered_remove(&state.Flag.Layers, index)
    destroyLayer(variant)
}