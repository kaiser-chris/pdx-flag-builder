package pdx_flag_builder

import "core:fmt"
import "core:os"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "texture"
import "ui"
import "pdx"

WINDOM_PREVIEW: string: "Preview"
WINDOM_TOOLBAR: string: "Toolbar"
WINDOM_SETTINGS: string: "Settings"
WINDOM_DATABASE: string: "Database"
WINDOM_FLAG: string: "Layers"
WINDOM_SELECTED: string: "Selected Element"
WINDOM_SIDEBAR: string: "Sidebar"
WINDOM_COLOR_PICKER: string: "Color Picker"

SIDEBAR_HANDLE_WIDTH:i32: 10
SIDEBAR_MIN_WIDTH: i32: 200
SIDEBAR_MAX_WIDTH: i32: 600
TOOLBAR_HEIGHT: i32: 35
FLAG_HEIGHT: i32: 512
FLAG_WIDTH: i32: 768

SelectedFlagElement :: union {
    ^pdx.Flag,
    ^pdx.FlagLayerColoredEmblem,
    ^pdx.FlagLayerTexturedEmblem,
    ^pdx.FlagColorNamed,
    ^pdx.FlagColorHsv,
    ^pdx.FlagColorRgb,
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

    if state.ColorPickerColor != nil {
        window := mu.get_container(ctx, WINDOM_COLOR_PICKER)
        if window != nil {
            window.open = true
        }
        renderColorPicker(ctx)
    }

    rl.SetWindowMinSize(FLAG_WIDTH + state.SidebarWidth, FLAG_HEIGHT + TOOLBAR_HEIGHT)
}

renderColorPicker :: proc(ctx: ^mu.Context) {
    @static red: u8
    @static green: u8
    @static blue: u8

    color := mu.Color{ red, green, blue, 255 }
    width: i32 = 400
    height: i32 = 200
    x: i32 = (rl.GetScreenWidth() - width) / 2
    y: i32 = (rl.GetScreenHeight() - height) / 2

    if mu.window(ctx, WINDOM_COLOR_PICKER, { x, y, width, height }) {
        guranteeBounds(ctx)
        mu.layout_row(ctx, {-78, -1}, 68)
        mu.layout_begin_column(ctx)
        {
            mu.layout_row(ctx, {80, -1}, 0)
            switch color in state.ColorPickerColor {
            case ^pdx.FlagColorRgb:
                mu.label(ctx, "Red:");   ui.ColorSlider(ctx, &color.R, 0, 255)
                mu.label(ctx, "Green:"); ui.ColorSlider(ctx, &color.G, 0, 255)
                mu.label(ctx, "Blue:");  ui.ColorSlider(ctx, &color.B, 0, 255)
                red = color.R
                green = color.G
                blue = color.B
            case ^pdx.FlagColorHsv:
                mu.label(ctx, "Hue:");   ui.ColorSlider(ctx, &color.H, 0, 1)
                mu.label(ctx, "Saturation:"); ui.ColorSlider(ctx, &color.S, 0, 1)
                mu.label(ctx, "Value:");  ui.ColorSlider(ctx, &color.V, 0, 1)
                rgb := pdx.ToRenderColor(color)
                red = rgb[0]
                green = rgb[1]
                blue = rgb[2]
            case ^pdx.FlagColorNamed:
            }
        }
        mu.layout_end_column(ctx)

        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, color)
        mu.draw_box(ctx, mu.expand_rect(r, 1), ctx.style.colors[.BORDER])
        mu.draw_control_text(ctx, fmt.tprintf("#%02x%02x%02x", red, green, blue), r, .TEXT, {.ALIGN_CENTER})

        mu.layout_row(ctx, {-1, -1}, 20)
        if .SUBMIT in mu.button(ctx, "Select Color") {

        }
    }

    cnt := mu.get_container(ctx, WINDOM_COLOR_PICKER)
    is_open := cnt != nil && cnt.open != false

    if state.ColorPickerColor != nil && !is_open {
        state.ColorPickerColor = nil
    }
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
        x = (rl.GetScreenWidth() - FLAG_WIDTH - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH - 10) / 2,
        y = TOOLBAR_HEIGHT + ((rl.GetScreenHeight() - FLAG_HEIGHT - TOOLBAR_HEIGHT - 6) / 2),
        w = FLAG_WIDTH,
        h = FLAG_HEIGHT,
    }

    if mu.window(ctx, WINDOM_PREVIEW, destination, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL, .NO_FRAME}) {
        window := mu.get_current_container(ctx)
        window.rect = destination
        target := mu.Rect{x = 0, y = 0, w = FLAG_WIDTH, h = FLAG_HEIGHT}
        mu.layout_set_next(ctx, target, true)
        drawFlagTexture(ctx)
    }
}

renderFlagPreviewTexture :: proc(cmd: ^mu.Command_Rect) {
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}

    if state.Flag.Pattern.Name != "" {
        pattern, ok := state.RenderTextureMap[state.Flag.Pattern.Path]
        source := rl.Rectangle{0, 0, f32(pattern.width), f32(pattern.height)}

        colorMappings := make([dynamic]texture.ColorRecolor)
        defer delete(colorMappings)
        for color in state.Flag.Colors {
            fillColorMapping(&colorMappings, color, pdx.PATTERN_REPLACE_COLORS)
        }

        texture.DrawRecoloredTexture(
            pattern,
            source,
            destination,
            colorMappings[:],
            state.RecolorShader,
        )
    }
}

fillColorMapping :: proc(mappings: ^[dynamic]texture.ColorRecolor, variant: pdx.FlagColorVariant, sourceColors: []rl.Color) {
    switch color in variant {
    case ^pdx.FlagColorRgb:
        targetColor := pdx.ToRenderColor(color)
        mapping, ok := checkColorMapping(color.Name, targetColor, sourceColors)
        if ok {
            append(mappings, mapping)
        }
    case ^pdx.FlagColorHsv:
        targetColor := pdx.ToRenderColor(color)
        mapping, ok := checkColorMapping(color.Name, targetColor, sourceColors)
        if ok {
            append(mappings, mapping)
        }
    case ^pdx.FlagColorNamed:
        //TODO
    }
}

checkColorMapping :: proc(name: string, targetColor: rl.Color, sourceColors: []rl.Color) -> (texture.ColorRecolor, bool) {
    if len(sourceColors) >= 1 && name == pdx.COLOR_NAMES[0] {
        return texture.ColorRecolor{
            sourceColors[0],
            targetColor,
        }, true
    }
    if len(sourceColors) >= 2 && name == pdx.COLOR_NAMES[1] {
        return texture.ColorRecolor{
            sourceColors[1],
            targetColor,
        }, true
    }
    if len(sourceColors) >= 3 && name == pdx.COLOR_NAMES[2] {
        return texture.ColorRecolor{
            sourceColors[2],
            targetColor,
        }, true
    }
    return texture.ColorRecolor{}, false
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
                    NamedColors = make([dynamic]pdx.FlagColorVariant)
                }
                append(&state.Databases, element)
            }
        }

        if .ACTIVE in mu.header(ctx, "Background Color", {.EXPANDED}) {
            mu.layout_row(ctx, {-78, -1}, 68)
            mu.layout_begin_column(ctx)
            {
                mu.layout_row(ctx, {46, -1}, 0)
                mu.label(ctx, "Red:");   ui.ColorSlider(ctx, &state.Settings.BackgroundColor.r, 0, 255)
                mu.label(ctx, "Green:"); ui.ColorSlider(ctx, &state.Settings.BackgroundColor.g, 0, 255)
                mu.label(ctx, "Blue:");  ui.ColorSlider(ctx, &state.Settings.BackgroundColor.b, 0, 255)
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
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .ACTIVE in mu.header(ctx, database.Settings.Name, {.EXPANDED}) {
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .ACTIVE in mu.treenode(ctx, "Patterns", options) {
                    for texture in database.Patterns {
                        renderDatabaseItem(ctx, texture, pdx.FOLDER_PATTERNS)
                    }
                    if len(database.Patterns) == 0 {
                        mu.layout_row(ctx, {-1, -1}, 50)
                        r := mu.layout_next(ctx)
                        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
                        mu.draw_control_text(ctx, "No Items", r, .TEXT, {.ALIGN_CENTER})
                    }
                }
                mu.pop_id(ctx)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .ACTIVE in mu.treenode(ctx, "Colored Emblems", options) {
                    for texture in database.ColoredEmblems {
                        renderDatabaseItem(ctx, texture, pdx.FOLDER_COLORED_EMBLEMS)
                    }
                    if len(database.Patterns) == 0 {
                        mu.layout_row(ctx, {-1, -1}, 50)
                        r := mu.layout_next(ctx)
                        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
                        mu.draw_control_text(ctx, "No Items", r, .TEXT, {.ALIGN_CENTER})
                    }
                }
                mu.pop_id(ctx)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .ACTIVE in mu.treenode(ctx, "Textured Emblems", options) {
                    for texture in database.TexturedEmblems {
                        renderDatabaseItem(ctx, texture, pdx.FOLDER_TEXTURED_EMBLEMS)
                    }
                    if len(database.Patterns) == 0 {
                        mu.layout_row(ctx, {-1, -1}, 50)
                        r := mu.layout_next(ctx)
                        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
                        mu.draw_control_text(ctx, "No Items", r, .TEXT, {.ALIGN_CENTER})
                    }
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

renderDatabaseItem :: proc(ctx: ^mu.Context, item: pdx.FlagTexture, type: string) {
    texture := state.GuiTextureCache[item.Path]
    width := i32((f32(texture.width) / f32(texture.height)) * 128)

    mu.layout_row(ctx, {200, -1}, 128)
    mu.layout_begin_column(ctx)
    {
        mu.layout_row(ctx, {192, -1}, 128)
        drawTexture(ctx, item.Path, width, 128)
    }
    mu.layout_end_column(ctx)
    mu.layout_begin_column(ctx)
    {
        mu.layout_row(ctx, {-1, -1}, 20)
        switch type {
        case pdx.FOLDER_PATTERNS:
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .SUBMIT in mu.button(ctx, "Set as pattern") {
                state.Flag.Pattern = pdx.CreateFlagTexture(item.Name, item.Path)
            }
            mu.pop_id(ctx)
        case pdx.FOLDER_COLORED_EMBLEMS:
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .SUBMIT in mu.button(ctx, "Add as layer") {
                layer := pdx.CreateLayerColoredEmblem(item.Name, item.Path)
                append(&state.Flag.Layers, layer)
            }
            mu.pop_id(ctx)
        case pdx.FOLDER_TEXTURED_EMBLEMS:
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .SUBMIT in mu.button(ctx, "Add as layer") {
                layer := pdx.CreateLayerTexturedEmblem(item.Name, item.Path)
                append(&state.Flag.Layers, layer)
            }
            mu.pop_id(ctx)
        }
        mu.layout_row(ctx, {-1, -1}, 20)
        mu.text(ctx, item.Name)
        mu.layout_row(ctx, {-1, -1}, 20)
        mu.text(ctx, fmt.tprintf("%ix%i", texture.width, texture.height))
    }
    mu.layout_end_column(ctx)
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
    height: i32 = 300
    x := rl.GetScreenWidth() - width
    y := TOOLBAR_HEIGHT + 1

    if mu.window(ctx, WINDOM_FLAG, { x, y, width, height }, {.NO_RESIZE, .NO_CLOSE, .NO_INTERACT}) {
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height
        window.rect.x = x
        window.rect.y = y

        mu.layout_row(ctx, {-1, -1}, 20)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in mu.button(ctx, "Pattern & Colors") {
            state.SelectedFlagElement = &state.Flag
        }
        mu.pop_id(ctx)

        for variant, index in state.Flag.Layers {
            switch layer in variant {
            case ^pdx.FlagLayerColoredEmblem:
                mu.layout_row(ctx, {-1, -1}, 20)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Colored Emblem: %s", layer.Texture.Name)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
            case ^pdx.FlagLayerTexturedEmblem:
                mu.layout_row(ctx, {-1, -1}, 20)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
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
    flagWindow := mu.get_container(ctx, WINDOM_FLAG)
    width := state.SidebarWidth
    height := rl.GetScreenHeight() - (flagWindow.rect.y + flagWindow.rect.h + 1)
    x :=  rl.GetScreenWidth() - width
    y := flagWindow.rect.y + flagWindow.rect.h + 1

    if mu.window(ctx, WINDOM_SELECTED, { x, y, width, height }, {.NO_RESIZE, .NO_CLOSE, .NO_INTERACT}) {
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height
        window.rect.x = x
        window.rect.y = y

        if state.SelectedFlagElement == nil {
            mu.layout_row(ctx, {-1, -1}, -1)
            r := mu.layout_next(ctx)
            mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
            mu.draw_control_text(ctx, "Nothing Selected", r, .TEXT, {.ALIGN_CENTER})
        } else {
            switch element in state.SelectedFlagElement {
            case ^pdx.Flag:
                renderSelectedFlag(ctx)
            case ^pdx.FlagLayerColoredEmblem:
                renderSelectedFlagLayerColoredEmblem(ctx)
            case ^pdx.FlagLayerTexturedEmblem:
                renderSelectedFlagLayerTexturedEmblem(ctx)
            case ^pdx.FlagColorNamed:
                renderSelectedFlagColorNamed(ctx)
            case ^pdx.FlagColorHsv:
                renderSelectedFlagColorHsv(ctx)
            case ^pdx.FlagColorRgb:
                renderSelectedFlagColorRgb(ctx)
            }
        }
    }
}

renderSelectedFlag :: proc(ctx: ^mu.Context) {
    ui.DrawAttributeRow(ctx, "Layer", "Pattern & Colors")
    if state.Flag.Pattern.Name != "" {
        ui.DrawAttributeRow(ctx, "Pattern", state.Flag.Pattern.Name)
    } else {
        ui.DrawAttributeRow(ctx, "Pattern", "None")
    }


    for variant, index in state.Flag.Colors {
        switch color in variant {
        case ^pdx.FlagColorNamed:
            ui.DrawAttributeRow(ctx, color.Name, color)
        case ^pdx.FlagColorRgb:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color)
            renderColorButtons(ctx, variant, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorHsv:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color)
            renderColorButtons(ctx, variant, index, buttonEditRect, buttonDelRect)
        }
    }
    mu.layout_row(ctx, { -1, -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add new color") {
        colorName := pdx.GetNextFreeColor(state.Flag.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorRgb(colorName, 0, 0, 0)
            append(&state.Flag.Colors, color)
        }
    }
    mu.pop_id(ctx)
}

renderSelectedFlagLayerColoredEmblem :: proc(ctx: ^mu.Context) {
    ui.DrawAttributeRow(ctx, "Type", "Colored Emblem")
}

renderSelectedFlagLayerTexturedEmblem :: proc(ctx: ^mu.Context) {
    ui.DrawAttributeRow(ctx, "Type", "Textured Emblem")
}

renderSelectedFlagColorNamed :: proc(ctx: ^mu.Context) {
    ui.DrawAttributeRow(ctx, "Type", "Named Color")
}

renderSelectedFlagColorHsv :: proc(ctx: ^mu.Context) {
    ui.DrawAttributeRow(ctx, "Type", "HSV Color")
}

renderSelectedFlagColorRgb :: proc(ctx: ^mu.Context) {
    ui.DrawAttributeRow(ctx, "Type", "RGB Color")
}

renderColorButtons :: proc(ctx: ^mu.Context, color: pdx.FlagColorVariant, index: int, edit, delete: mu.Rect) {
    mu.layout_set_next(ctx, edit, false)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "edit") {
        state.ColorPickerColor = color
    }
    mu.pop_id(ctx)
    mu.layout_set_next(ctx, delete, false)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "del") {
        if color == state.ColorPickerColor {
            state.ColorPickerColor = nil
        }
        ordered_remove(&state.Flag.Colors, index)
        pdx.DestroyColor(color)
    }
    mu.pop_id(ctx)
}

removeLayer :: proc(index: int){
    variant := state.Flag.Layers[index]
    ordered_remove(&state.Flag.Layers, index)
    pdx.DestroyLayer(variant)
}