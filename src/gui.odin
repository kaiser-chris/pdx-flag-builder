package pdx_flag_builder

import "core:fmt"
import "core:strings"
import "core:os"
import rl "vendor:raylib"
import mu "vendor:microui"

FOLDER_GFX: string: "gfx"
FOLDER_COA: string: "coat_of_arms"
FOLDER_PATTERNS: string: "patterns"
FOLDER_COLORED_EMBLEMS: string: "colored_emblems"
FOLDER_TEXTURED_EMBLEMS: string: "textured_emblems"

WINDOM_TOOLBAR: string: "Toolbar"
WINDOM_SETTINGS: string: "Settings"
WINDOM_DATABASE: string: "Database"
WINDOM_FLAG: string: "Flag Creator"

Flag :: struct {
    Pattern: FlagTexture,
    Colors: [dynamic]FlagColorVariant,
    Layers: [dynamic]FlagLayer,
}

FlagTexture :: struct {
    Name: string,
    Texture: cstring,
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

FlagLayer :: struct {
    using FlagTexture: FlagTexture,
    Colors: [dynamic]FlagColorVariant,
    Instances: [dynamic]LayerInstance,
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

renderGui :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_TOOLBAR, { 0, 0, rl.GetScreenWidth(), TOOLBAR_HEIGHT }, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL}) {
        mu.get_current_container(ctx).rect.w = rl.GetScreenWidth()
        mu.layout_row_items(ctx, 3, TOOLBAR_HEIGHT - 8)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Flag Creator") {
            if state.FlagWindowOpen {
                state.FlagWindowOpen = false
            } else {
                state.FlagWindowOpen = true
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

    if state.SettingsWindowOpen {
        cnt := mu.get_container(ctx, WINDOM_SETTINGS)
        if cnt != nil {
            cnt.open = true
        }
        renderSettings(ctx)
    }

    if state.DatabaseWindowOpen {
        cnt := mu.get_container(ctx, WINDOM_DATABASE)
        if cnt != nil {
            cnt.open = true
        }
        renderDatabase(ctx)
    }

    if state.FlagWindowOpen {
        cnt := mu.get_container(ctx, WINDOM_FLAG)
        if cnt != nil {
            cnt.open = true
        }
        renderFlagMenu(ctx)
    }

}

renderSettings :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_SETTINGS, {10, 10 + TOOLBAR_HEIGHT, 500, 400}, {}) {
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
                if state.Databases == nil {
                    state.Databases = make([dynamic]DatabaseState)
                }
                element := DatabaseState{
                    Settings = FlagDatabase{"", ""}
                }
                append(&state.Databases, element)
            }
        }

        if .ACTIVE in mu.header(ctx, "Background Color", {.EXPANDED}) {
            mu.layout_row(ctx, {-78, -1}, 68)
            mu.layout_begin_column(ctx)
            {
                mu.layout_row(ctx, {46, -1}, 0)
                mu.label(ctx, "Red:");   u8_slider(ctx, &state.Settings.BackgroundColor.r, 0, 255)
                mu.label(ctx, "Green:"); u8_slider(ctx, &state.Settings.BackgroundColor.g, 0, 255)
                mu.label(ctx, "Blue:");  u8_slider(ctx, &state.Settings.BackgroundColor.b, 0, 255)
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
        mu.get_current_container(ctx).rect.h = rl.GetScreenHeight() - 20 - TOOLBAR_HEIGHT
        for database in state.Databases {
            setButtonIdentifier(ctx)
            if .ACTIVE in mu.header(ctx, database.Settings.Name) {
                setButtonIdentifier(ctx)
                if .ACTIVE in mu.treenode(ctx, "Patterns") {
                    patternsPath, err := os.join_path([]string{ database.Settings.Path, FOLDER_GFX, FOLDER_COA, FOLDER_PATTERNS }, context.allocator)
                    if err != nil {
                        fmt.eprintfln("wtf: %v", err)
                    }
                    renderDatabaseFolder(ctx, patternsPath, FOLDER_PATTERNS)
                }
                mu.pop_id(ctx)
                setButtonIdentifier(ctx)
                if .ACTIVE in mu.treenode(ctx, "Colored Emblems") {
                    patternsPath, err := os.join_path([]string{ database.Settings.Path, FOLDER_GFX, FOLDER_COA, FOLDER_COLORED_EMBLEMS }, context.allocator)
                    if err != nil {
                        fmt.eprintfln("wtf: %v", err)
                    }
                    renderDatabaseFolder(ctx, patternsPath, FOLDER_COLORED_EMBLEMS)
                }
                mu.pop_id(ctx)
                setButtonIdentifier(ctx)
                if .ACTIVE in mu.treenode(ctx, "Textured Emblems") {
                    patternsPath, err := os.join_path([]string{ database.Settings.Path, FOLDER_GFX, FOLDER_COA, FOLDER_TEXTURED_EMBLEMS }, context.allocator)
                    if err != nil {
                        fmt.eprintfln("wtf: %v", err)
                    }
                    renderDatabaseFolder(ctx, patternsPath, FOLDER_TEXTURED_EMBLEMS)
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

        path := strings.clone_to_cstring(info.fullpath)
        defer delete(path)
        texture := state.TextureCache[path]
        width := (f32(texture.width) / f32(texture.height)) * 128

        mu.layout_row(ctx, {200, -1}, 128)
        mu.layout_begin_column(ctx)
        {
            mu.layout_row(ctx, {192, -1}, 128)
            drawTexture(ctx, info.fullpath, i32(width), 128, true)
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
                        Texture = strings.clone_to_cstring(info.fullpath),
                    }
                }
                mu.pop_id(ctx)
            case FOLDER_COLORED_EMBLEMS:
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, "Add as layer") {
                    fmt.printfln("Colored Emblem: %s", info.name)
                }
                mu.pop_id(ctx)
            case FOLDER_TEXTURED_EMBLEMS:
                setButtonIdentifier(ctx)
                if .SUBMIT in mu.button(ctx, "Add as layer") {
                    fmt.printfln("Textured Emblem: %s", info.name)
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

renderFlagMenu :: proc(ctx: ^mu.Context) {
    width := i32(500)
    if mu.window(ctx, WINDOM_FLAG, {rl.GetScreenWidth() - 10 - width, 10 + TOOLBAR_HEIGHT, width, 400}, {}) {
        window := mu.get_current_container(ctx)
        window.rect.h = rl.GetScreenHeight() - 20 - TOOLBAR_HEIGHT
        window.rect.x = rl.GetScreenWidth() - 10 - window.rect.w
        pattern := state.Flag.Pattern
        mu.text(ctx, fmt.tprintf("Pattern: %s", pattern.Name))
    }

    cnt := mu.get_container(ctx, WINDOM_FLAG)
    is_open := cnt != nil && cnt.open != false

    if state.FlagWindowOpen && !is_open {
        state.FlagWindowOpen = false
    }
}