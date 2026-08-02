package pdx_flag_builder

import "core:fmt"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "texture"
import "ui"
import "pdx"
import os "core:os"

WINDOM_PREVIEW: string: "Preview"
WINDOM_TOOLBAR: string: "Toolbar"
WINDOM_SETTINGS: string: "Settings"
WINDOM_FLAG_DATABASE: string: "Flag Database"
WINDOM_TEXTURE_DATABASE: string: "Texture Database"
WINDOM_FLAG: string: "Layers"
WINDOM_SELECTED: string: "Selected Layer"
WINDOM_SIDEBAR: string: "Sidebar"
WINDOM_COLOR_PICKER: string: "Color Picker"
WINDOM_INSTANCE_EDITOR: string: "Instance Editor"

TEXTURE_ICON_SUB: string: "assets/textures/icons/sub.dds"
TEXTURE_ICON_EDIT: string: "assets/textures/icons/edit.dds"
TEXTURE_ICON_DELETE: string: "assets/textures/icons/delete.dds"
TEXTURE_ICON_ARROW_UP: string: "assets/textures/icons/arrow_up.dds"
TEXTURE_ICON_ARROW_DOWN: string: "assets/textures/icons/arrow_down.dds"
TEXTURE_ICON_UNKNOWN: string: "assets/textures/icons/unknown.dds"
TEXTURE_TRANSPARENCY: string: "assets/textures/transparency.dds"
TEXTURE_INVALID: string: "assets/textures/invalid.dds"

POPUP_MESSAGE_CLEAR_FLAG: string: "Are you sure you want to clear this flag? It will delete your current changes in the editor."
POPUP_MESSAGE_DELETE_LAYER: string: "Are you sure you want to delete this layer?"
POPUP_MESSAGE_DELETE_INSTANCE: string: "Are you sure you want to delete this layer instance?"
POPUP_MESSAGE_OPEN_FLAG: string: "Are you sure you want to open this flag? It will overwrite your current flag."
POPUP_MESSAGE_DELETE_COLOR: string: "Are you sure you want to delete this color?"

TEXTURE_ICON_GAME_VIC3: string: "assets/textures/icons/game_vic3.dds"
TEXTURE_ICON_GAME_EU5: string: "assets/textures/icons/game_eu5.dds"

SHADER_RECOLOR: string: "assets/shaders/recolor.fs"

CLOSABLE_WINDOWS: []string: {
    WINDOM_SETTINGS,
    WINDOM_FLAG_DATABASE,
    WINDOM_TEXTURE_DATABASE,
    WINDOM_COLOR_PICKER,
    WINDOM_INSTANCE_EDITOR,
}

TOOLBAR_HEIGHT: i32: 35
FLAG_HEIGHT: i32: 512
FLAG_WIDTH: i32: 768

SelectedFlagElement :: union {
    ^pdx.Flag,
    ^pdx.FlagLayerColoredEmblem,
    ^pdx.FlagLayerTexturedEmblem,
    ^pdx.FlagLayerSub,
    ^pdx.LayerInstance,
    ^pdx.LayerInstanceSub,
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

    if state.TextureDatabaseWindowOpen {
        window := mu.get_container(ctx, WINDOM_TEXTURE_DATABASE)
        if window != nil {
            window.open = true
        }
        renderTextureDatabase(ctx)
    }

    if state.FlagDatabaseWindowOpen {
        window := mu.get_container(ctx, WINDOM_FLAG_DATABASE)
        if window != nil {
            window.open = true
        }
        renderFlagDatabase(ctx)
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
        switch color in state.ColorPickerColor.Variant {
        case pdx.FlagColorRgb:
            renderAbsoluteColorPicker(ctx)
        case pdx.FlagColorHsv:
            renderAbsoluteColorPicker(ctx)
        case pdx.FlagColorNamed:
            renderNamedColorPicker(ctx)
        case pdx.FlagColorReference:
            renderReferenceColorPicker(ctx)
        }
    }

    if state.InstanceEditorInstance != nil {
        window := mu.get_container(ctx, WINDOM_INSTANCE_EDITOR)
        if window != nil {
            window.open = true
        }
        renderInstanceEditor(ctx)
    }

    ui.InitDropdown(ctx, &state.Dropdown, &state.ButtonIdentifier)
    ui.InitToast(ctx, &state.Toast, { 10, TOOLBAR_HEIGHT + 10, 200, screenHeight() - (TOOLBAR_HEIGHT + 20) }, &state.ButtonIdentifier)

    scale := getUiScale()
    rl.SetWindowMinSize(i32(f32(FLAG_WIDTH + state.SidebarWidth) * scale), i32(f32(FLAG_HEIGHT + TOOLBAR_HEIGHT) * scale))

    handleStackedWindowClosing(ctx)

    @static alreadySaved: bool
    if (rl.IsKeyDown(rl.KeyboardKey.LEFT_CONTROL) || rl.IsKeyDown(rl.KeyboardKey.RIGHT_CONTROL)) && rl.IsKeyDown(rl.KeyboardKey.S) && !alreadySaved {
        exportFlagAsIs(ctx, &state.Flag)
        alreadySaved = true
    } else if (!rl.IsKeyDown(rl.KeyboardKey.LEFT_CONTROL) && !rl.IsKeyDown(rl.KeyboardKey.RIGHT_CONTROL)) || !rl.IsKeyDown(rl.KeyboardKey.S) {
        alreadySaved = false
    }
}

handleStackedWindowClosing :: proc(ctx: ^mu.Context) {
    if !rl.IsKeyPressed(rl.KeyboardKey.ESCAPE) {
        return
    }

    highestZIndex: i32
    for closable in CLOSABLE_WINDOWS {
        window := mu.get_container(ctx, closable)
        if window == nil {
            continue
        }
        isOpen: b32
        switch closable {
        case WINDOM_SETTINGS:
            isOpen = window.open && state.SettingsWindowOpen
        case WINDOM_FLAG_DATABASE:
            isOpen = window.open && state.FlagDatabaseWindowOpen
        case WINDOM_TEXTURE_DATABASE:
            isOpen = window.open && state.TextureDatabaseWindowOpen
        case WINDOM_COLOR_PICKER:
            isOpen = window.open && state.ColorPickerColor != nil
        case WINDOM_INSTANCE_EDITOR:
            isOpen = window.open && state.InstanceEditorInstance != nil
        }
        if !isOpen {
            continue
        }
        if window.zindex > highestZIndex {
            highestZIndex = window.zindex
        }
    }

    for closable in CLOSABLE_WINDOWS {
        closableWindow := mu.get_container(ctx, closable)

        if closableWindow.zindex != highestZIndex {
            continue
        }

        switch closable {
        case WINDOM_SETTINGS:
            state.SettingsWindowOpen = false
        case WINDOM_FLAG_DATABASE:
            state.FlagDatabaseWindowOpen = false
        case WINDOM_TEXTURE_DATABASE:
            state.TextureDatabaseWindowOpen = false
        case WINDOM_COLOR_PICKER:
            state.ColorPickerColor = nil
        case WINDOM_INSTANCE_EDITOR:
            state.InstanceEditorInstance = nil
        }
        break
    }
}

renderAbsoluteColorPicker :: proc(ctx: ^mu.Context) {
    @static red: u8
    @static redBuf: [16]byte
    @static redBufLen: int
    @static green: u8
    @static greenBuf: [16]byte
    @static greenBufLen: int
    @static blue: u8
    @static blueBuf: [16]byte
    @static blueBufLen: int

    color := mu.Color{ red, green, blue, 255 }
    width: i32 = 400
    height: i32 = 110
    x: i32 = (screenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (screenHeight() - height) / 2

    if mu.window(ctx, WINDOM_COLOR_PICKER, { x, y, width, height }, { .NO_RESIZE }) {
        guranteeBounds(ctx)
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height

        mu.layout_row(ctx, {-78, -1}, 68)
        mu.layout_begin_column(ctx)
        {
            mu.layout_row(ctx, {80, -68, 60}, 0)
            switch &color in state.ColorPickerColor.Variant {
            case pdx.FlagColorRgb:
                mu.label(ctx, "Red:")
                ui.Slider(ctx, &color.R, 0, 255)
                ui.NumberTextbox(ctx, &color.R, redBuf[:], &redBufLen, 255, 0)

                mu.label(ctx, "Green:")
                ui.Slider(ctx, &color.G, 0, 255)
                ui.NumberTextbox(ctx, &color.G, greenBuf[:], &greenBufLen, 255, 0)

                mu.label(ctx, "Blue:")
                ui.Slider(ctx, &color.B, 0, 255)
                ui.NumberTextbox(ctx, &color.B, blueBuf[:], &blueBufLen, 255, 0)
                red = color.R
                green = color.G
                blue = color.B
            case pdx.FlagColorHsv:
                mu.label(ctx, "Hue:")
                ui.Slider(ctx, &color.H, 0, 360, step = 1)
                ui.NumberTextbox(ctx, &color.H, redBuf[:], &redBufLen, 360, 0)

                mu.label(ctx, "Saturation:")
                ui.Slider(ctx, &color.S, 0, 100, step = 1)
                ui.NumberTextbox(ctx, &color.S, greenBuf[:], &greenBufLen, 100, 0)

                mu.label(ctx, "Value:")
                ui.Slider(ctx, &color.V, 0, 100, step = 1)
                ui.NumberTextbox(ctx, &color.V, blueBuf[:], &blueBufLen, 100, 0)

                rgb := pdx.ToRenderColor(color)
                red = rgb[0]
                green = rgb[1]
                blue = rgb[2]
            case pdx.FlagColorNamed:
            case pdx.FlagColorReference:
            }
        }
        mu.layout_end_column(ctx)

        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, color)
        mu.draw_box(ctx, mu.expand_rect(r, 1), ctx.style.colors[.BORDER])
        mu.draw_control_text(ctx, fmt.tprintf("#%02x%02x%02x", red, green, blue), r, .TEXT, {.ALIGN_CENTER})
    }

    cnt := mu.get_container(ctx, WINDOM_COLOR_PICKER)
    is_open := cnt != nil && cnt.open != false

    if state.ColorPickerColor != nil && !is_open {
        state.ColorPickerColor = nil
    }
}

renderNamedColorPicker :: proc(ctx: ^mu.Context) {
    width: i32 = 300
    height: i32 = 500
    x: i32 = (screenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (screenHeight() - height) / 2

    if mu.window(ctx, WINDOM_COLOR_PICKER, { x, y, width, height }, { .NO_RESIZE }) {
        guranteeBounds(ctx)
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height

        mu.layout_row(ctx, { -1 }, 15)
        mu.text(ctx, "Search:")
        @static searchBuffer: [128]byte
        @static searchBufferLength: int
        if .CHANGE in mu.textbox(ctx, searchBuffer[:], &searchBufferLength) {
            delete(state.SearchCache.NamedColorSearch)
            state.SearchCache.NamedColorSearch = strings.to_lower(string(searchBuffer[:searchBufferLength]))
        }
        mu.layout_row(ctx, { -1 }, 5)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])

        options: mu.Options
        if state.SearchCache.NamedColorSearch != "" {
            options = { .EXPANDED }
        }

        for database in state.Databases {
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .ACTIVE in mu.header(ctx, database.Settings.Name, options) {
                for variant in database.NamedColors {
                    searchName := strings.to_lower(variant.Name)
                    defer delete(searchName)

                    if !strings.contains(searchName, state.SearchCache.NamedColorSearch) {
                        continue
                    }

                    name := variant.Name
                    rectColor: mu.Color
                    switch color in variant.Variant {
                    case pdx.FlagColorRgb:
                        rectColor = mu.Color{ color.R, color.G, color.B, 255 }
                    case pdx.FlagColorHsv:
                        tmp := pdx.ToRenderColor(color)
                        rectColor = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
                    case pdx.FlagColorNamed:
                        fmt.eprintfln("Named color is referencing another named color")
                    case pdx.FlagColorReference:
                        fmt.eprintfln("Named color is referencing a reference color")
                    }

                    mu.layout_row(ctx, { -1, -1 }, 20)
                    row := mu.layout_next(ctx)
                    row.x = row.x + ctx.style.indent
                    row.w = row.w - ctx.style.indent

                    colorRect := mu.Rect{
                        x = row.x,
                        y = row.y,
                        w = row.h,
                        h = row.h,
                    }
                    mu.draw_rect(ctx, colorRect, rectColor)

                    buttonWidth: i32 = 60
                    buttonRect := mu.Rect{
                        x = row.x + row.w - buttonWidth,
                        y = row.y,
                        w = buttonWidth,
                        h = row.h,
                    }

                    mu.layout_set_next(ctx, buttonRect, false)
                    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                    if .SUBMIT in mu.button(ctx, "Select") {
                        switch &color in state.ColorPickerColor.Variant {
                        case pdx.FlagColorRgb:
                            fmt.eprintfln("RGB color does not support named color")
                        case pdx.FlagColorHsv:
                            fmt.eprintfln("HSV color does not support named color")
                        case pdx.FlagColorReference:
                            fmt.eprintfln("reference color does not support named color")
                        case pdx.FlagColorNamed:
                            delete(color.NamedColor)
                            color.NamedColor = strings.clone(name)
                        }
                    }
                    mu.pop_id(ctx)

                    textRect := mu.Rect{
                        x = row.x + colorRect.w + ctx.style.spacing,
                        y = row.y,
                        w = row.w - colorRect.w - ctx.style.spacing,
                        h = row.h,
                    }
                    mu.draw_control_text(ctx, name, textRect, .TEXT)
                }
            }
            mu.pop_id(ctx)
        }
    }

    cnt := mu.get_container(ctx, WINDOM_COLOR_PICKER)
    is_open := cnt != nil && cnt.open != false

    if state.ColorPickerColor != nil && !is_open {
        state.ColorPickerColor = nil
    }
}

renderReferenceColorPicker :: proc(ctx: ^mu.Context) {
    width: i32 = 300
    height: i32 = 300
    x: i32 = (screenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (screenHeight() - height) / 2

    if mu.window(ctx, WINDOM_COLOR_PICKER, { x, y, width, height }, { .NO_RESIZE }) {
        guranteeBounds(ctx)
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height

        for variant in state.Flag.Colors {
            name := variant.Name
            rectColor: mu.Color
            resolved: pdx.FlagColor

            #partial switch color in variant.Variant {
            case pdx.FlagColorNamed:
                namedColor, found := getNamedColor(color.NamedColor)
                if !found {
                    continue
                }
                resolved = namedColor
            case:
                resolved = variant
            }

            #partial switch color in resolved.Variant {
            case pdx.FlagColorRgb:
                rectColor = mu.Color{ color.R, color.G, color.B, 255 }
            case pdx.FlagColorHsv:
                tmp := pdx.ToRenderColor(color)
                rectColor = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
            }

            mu.layout_row(ctx, { -1, -1 }, 20)
            row := mu.layout_next(ctx)

            colorRect := mu.Rect{
                x = row.x,
                y = row.y,
                w = row.h,
                h = row.h,
            }
            mu.draw_rect(ctx, colorRect, rectColor)

            buttonWidth: i32 = 60
            buttonRect := mu.Rect{
                x = row.x + row.w - buttonWidth,
                y = row.y,
                w = buttonWidth,
                h = row.h,
            }

            mu.layout_set_next(ctx, buttonRect, false)
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .SUBMIT in mu.button(ctx, "Select") {
                switch &color in state.ColorPickerColor.Variant {
                case pdx.FlagColorRgb:
                    fmt.eprintfln("RGB color does not support reference color")
                case pdx.FlagColorHsv:
                    fmt.eprintfln("HSV color does not support reference color")
                case pdx.FlagColorNamed:
                    fmt.eprintfln("named color does not support reference color")
                case pdx.FlagColorReference:
                    delete(color.Reference)
                    color.Reference = strings.clone(name)
                }
            }
            mu.pop_id(ctx)

            textRect := mu.Rect{
                x = row.x + colorRect.w + ctx.style.spacing,
                y = row.y,
                w = row.w - colorRect.w - ctx.style.spacing,
                h = row.h,
            }
            mu.draw_control_text(ctx, name, textRect, .TEXT)
        }
    }

    cnt := mu.get_container(ctx, WINDOM_COLOR_PICKER)
    is_open := cnt != nil && cnt.open != false

    if state.ColorPickerColor != nil && !is_open {
        state.ColorPickerColor = nil
    }
}

renderInstanceEditor :: proc(ctx: ^mu.Context) {
    width: i32 = 400
    height: i32 = 160
    x: i32 = (screenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (screenHeight() - height) / 2

    @static rotationBuf: [16]byte
    @static rotationBufLen: int
    @static scaleXBuf: [16]byte
    @static scaleXBufLen: int
    @static scaleYBuf: [16]byte
    @static scaleYBufLen: int
    @static positionXBuf: [16]byte
    @static positionXBufLen: int
    @static positionYBuf: [16]byte
    @static positionYBufLen: int

    if mu.window(ctx, WINDOM_INSTANCE_EDITOR, { x, y, width, height }, { .NO_RESIZE }) {
        guranteeBounds(ctx)
        window := mu.get_current_container(ctx)
        window.rect.w = width
        window.rect.h = height

        mu.layout_row(ctx, {80, -90, 80}, 0)
        switch type in state.InstanceEditorInstance {
        case ^pdx.LayerInstance:
            mu.label(ctx, "Rotation:")
            ui.Slider(ctx, &type.Rotation, 0, 360, step = 1)
            ui.NumberTextbox(ctx, &type.Rotation, rotationBuf[:], &rotationBufLen, 360, 0)
            mu.label(ctx, "Scale - Width:")
            ui.Slider(ctx, &type.Scale.X, 0, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Scale.X, scaleXBuf[:], &scaleXBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
            mu.label(ctx, "Scale - Height:")
            ui.Slider(ctx, &type.Scale.Y, 0, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Scale.Y, scaleYBuf[:], &scaleYBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
            mu.label(ctx, "Position - X:")
            ui.Slider(ctx, &type.Position.X, -2, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Position.X, positionXBuf[:], &positionXBufLen, pdx.MAX_POSITION, pdx.MIN_POSITION, "%.3f")
            mu.label(ctx, "Position - Y:")
            ui.Slider(ctx, &type.Position.Y, -2, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Position.Y, positionYBuf[:], &positionYBufLen, pdx.MAX_POSITION, pdx.MIN_POSITION, "%.3f")

            if rl.IsKeyDown(.DOWN) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Position.Y += 0.001
            }
            if rl.IsKeyDown(.UP) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Position.Y -= 0.001
            }
            if rl.IsKeyDown(.RIGHT) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Position.X += 0.001
            }
            if rl.IsKeyDown(.LEFT) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Position.X -= 0.001
            }
            if rl.IsKeyDown(.DOWN) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.Y += 0.001
            }
            if rl.IsKeyDown(.UP) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.Y -= 0.001
            }
            if rl.IsKeyDown(.RIGHT) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.X += 0.001
            }
            if rl.IsKeyDown(.LEFT) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.X -= 0.001
            }
            if rl.IsKeyDown(.RIGHT) && (rl.IsKeyDown(.LEFT_ALT) || rl.IsKeyDown(.RIGHT_ALT)) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) {
                type.Rotation += 1
            }
            if rl.IsKeyDown(.LEFT) && (rl.IsKeyDown(.LEFT_ALT) || rl.IsKeyDown(.RIGHT_ALT)) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) {
                type.Rotation -= 1
            }
        case ^pdx.LayerInstanceSub:
            mu.label(ctx, "Scale - Width:")
            ui.Slider(ctx, &type.Scale.X, 0, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Scale.X, scaleXBuf[:], &scaleXBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
            mu.label(ctx, "Scale - Height:")
            ui.Slider(ctx, &type.Scale.Y, 0, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Scale.Y, scaleYBuf[:], &scaleYBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
            mu.label(ctx, "Offset - X:")
            ui.Slider(ctx, &type.Offset.X, -2, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Offset.X, positionXBuf[:], &positionXBufLen, pdx.MAX_OFFSET, pdx.MIN_OFFSET, "%.3f")
            mu.label(ctx, "Offset - Y:")
            ui.Slider(ctx, &type.Offset.Y, -2, 3, "%.2f", step = 0.05)
            ui.NumberTextbox(ctx, &type.Offset.Y, positionYBuf[:], &positionYBufLen, pdx.MAX_OFFSET, pdx.MIN_OFFSET, "%.3f")

            if rl.IsKeyDown(.DOWN) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Offset.Y += 0.001
            }
            if rl.IsKeyDown(.UP) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Offset.Y -= 0.001
            }
            if rl.IsKeyDown(.RIGHT) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Offset.X += 0.001
            }
            if rl.IsKeyDown(.LEFT) && !rl.IsKeyDown(.LEFT_CONTROL) && !rl.IsKeyDown(.RIGHT_CONTROL) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Offset.X -= 0.001
            }
            if rl.IsKeyDown(.DOWN) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.Y += 0.001
            }
            if rl.IsKeyDown(.UP) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.Y -= 0.001
            }
            if rl.IsKeyDown(.RIGHT) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.X += 0.001
            }
            if rl.IsKeyDown(.LEFT) && (rl.IsKeyDown(.LEFT_CONTROL) || rl.IsKeyDown(.RIGHT_CONTROL)) && !rl.IsKeyDown(.LEFT_ALT) && !rl.IsKeyDown(.RIGHT_ALT) {
                type.Scale.X -= 0.001
            }
        }
    }

    cnt := mu.get_container(ctx, WINDOM_INSTANCE_EDITOR)
    is_open := cnt != nil && cnt.open != false

    if state.InstanceEditorInstance != nil && !is_open {
        state.InstanceEditorInstance = nil
    }
}

renderToolbar :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_TOOLBAR, { 0, 0, screenWidth(), TOOLBAR_HEIGHT }, { .NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL }) {
        mu.get_current_container(ctx).rect.w = screenWidth()
        mu.layout_row_items(ctx, 6, TOOLBAR_HEIGHT - 8)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {200, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "File") {
            exportAsIs :: proc(ctx: ^mu.Context) {
                exportFlagAsIs(ctx, &state.Flag)
            }
            exportToImage :: proc(ctx: ^mu.Context) {
                exportFlagImage(ctx, state.Flag)
            }
            exportToClipboard :: proc(ctx: ^mu.Context) {
                exportFlagToClipboard(ctx, state.Flag)
            }
            exportToFile :: proc(ctx: ^mu.Context) {
                exportFlagAsFile(ctx, state.Flag)
            }
            ui.OpenDropdown(ctx, &state.Dropdown, {
                ui.CreateDropdownElement("Save Changes (CTRL + S)", exportAsIs),
                ui.CreateDropdownElement("Export to Image", exportToImage),
                ui.CreateDropdownElement("Export to Clipboard", exportToClipboard),
                ui.CreateDropdownElement("Export as new Script File", exportToFile),
            })
        }
        mu.layout_end_column(ctx)

        // TODO: move to icon button in the sidebar
//        mu.layout_begin_column(ctx)
//        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
//        if .SUBMIT in mu.button(ctx, "Toggle Sidebar") {
//            if state.SidebarOpen {
//                state.SidebarOpen = false
//            } else {
//                state.SidebarOpen = true
//            }
//        }
//        mu.layout_end_column(ctx)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {200, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Texture Database") {
            if state.TextureDatabaseWindowOpen {
                state.TextureDatabaseWindowOpen = false
            } else {
                state.TextureDatabaseWindowOpen = true
            }
        }
        mu.layout_end_column(ctx)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {200, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Flag Database") {
            if state.FlagDatabaseWindowOpen {
                state.FlagDatabaseWindowOpen = false
            } else {
                state.FlagDatabaseWindowOpen = true
            }
        }
        mu.layout_end_column(ctx)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {200, -1}, TOOLBAR_HEIGHT - 8)
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
    destination: mu.Rect
    if state.SidebarOpen {
        destination = mu.Rect{
            x = (screenWidth() - FLAG_WIDTH - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH - 10) / 2,
            y = TOOLBAR_HEIGHT + ((screenHeight() - FLAG_HEIGHT - TOOLBAR_HEIGHT - 6) / 2),
            w = FLAG_WIDTH,
            h = FLAG_HEIGHT,
        }
    } else {
        destination = mu.Rect{
            x = (screenWidth() - FLAG_WIDTH) / 2,
            y = TOOLBAR_HEIGHT + ((screenHeight() - FLAG_HEIGHT - TOOLBAR_HEIGHT - 6) / 2),
            w = FLAG_WIDTH,
            h = FLAG_HEIGHT,
        }
    }

    if mu.window(ctx, WINDOM_PREVIEW, destination, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL, .NO_FRAME, .NO_INTERACT}) {
        window := mu.get_current_container(ctx)
        window.rect = destination
        target := mu.Rect{x = 0, y = 0, w = FLAG_WIDTH, h = FLAG_HEIGHT}
        mu.draw_rect(ctx, destination, mu.Color{200, 200, 200, 255})
        mu.draw_box(ctx, destination, ctx.style.colors[.BORDER])
        mu.layout_set_next(ctx, target, true)
        drawFlagPreview(ctx)
    }
}

fillColorMapping :: proc(colors: []pdx.FlagColor, flag: pdx.Flag, sourceColors: []rl.Color) -> []texture.ColorRecolor {
    colorMappings := make([dynamic]texture.ColorRecolor)
    for variant in colors {
        switch color in variant.Variant {
        case pdx.FlagColorRgb:
            targetColor := pdx.ToRenderColor(color)
            mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
            if ok {
                append(&colorMappings, mapping)
            }
        case pdx.FlagColorHsv:
            targetColor := pdx.ToRenderColor(color)
            mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
            if ok {
                append(&colorMappings, mapping)
            }
        case pdx.FlagColorNamed:
            retrieve, exists := getNamedColor(color.NamedColor)
            if !exists {
                continue
            }
            targetColor := pdx.ToRenderColor(retrieve)
            mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
            if ok {
                append(&colorMappings, mapping)
            }
        case pdx.FlagColorReference:
            for baseColorVariant in flag.Colors {
                baseColorName: string
                switch baseColor in baseColorVariant.Variant {
                case pdx.FlagColorRgb:
                    if baseColorVariant.Name != color.Reference {
                        continue
                    }
                    targetColor := pdx.ToRenderColor(baseColor)
                    mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
                    if ok {
                        append(&colorMappings, mapping)
                    }
                case pdx.FlagColorHsv:
                    if baseColorVariant.Name != color.Reference {
                        continue
                    }
                    targetColor := pdx.ToRenderColor(baseColor)
                    mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
                    if ok {
                        append(&colorMappings, mapping)
                    }
                case pdx.FlagColorNamed:
                    if baseColorVariant.Name != color.Reference {
                        continue
                    }
                    retrieve, exists := getNamedColor(baseColor.NamedColor)
                    if !exists {
                        continue
                    }
                    targetColor := pdx.ToRenderColor(retrieve)
                    mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
                    if ok {
                        append(&colorMappings, mapping)
                    }
                case pdx.FlagColorReference:
                    // Should not happen
                }
            }
        }
    }
    return colorMappings[:]
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
    if mu.window(ctx, WINDOM_SETTINGS, { 10, 10 + TOOLBAR_HEIGHT, 670, 290 }, {}) {
        windows := mu.get_current_container(ctx)
        if windows.rect.w < 670 {
            windows.rect.w = 670
        }
        guranteeBounds(ctx)
        @static hasUnsavedChanges: bool

        mu.layout_row(ctx, { -1 }, 0)
        if .SUBMIT in mu.button(ctx, "Save Settings") {
            hasUnsavedChanges = false
            SaveSettings()
            ui.Toast(&state.Toast, "Settings", "Successfully saved settings.")
        }

        if hasUnsavedChanges {
            warn := mu.layout_next(ctx)
            next := mu.layout_next(ctx)
            mu.draw_rect(ctx, warn, ui.COLOR_BUTTON_WARNING_BACKGROUND_DISABLED)
            mu.draw_control_text(ctx, "You have unsaved changes!", warn, .TEXT, { .ALIGN_CENTER })
            mu.layout_set_next(ctx, next, false)
        }

        hasUnsavedChanges = false
        if .ACTIVE in mu.header(ctx, "Paths", {.EXPANDED}) {
            mu.text(ctx, "These are paths to the root of your mod or the \"game\" folder for the base game.")
            mu.layout_row(ctx, {20, 100, -209, 100, 100}, 20)
            for _, index in state.Databases {
                if state.Databases[index].Name != state.Databases[index].Settings.Name {
                    hasUnsavedChanges = true
                }
                if state.Databases[index].Path != state.Databases[index].Settings.Path {
                    hasUnsavedChanges = true
                }
                if state.Databases[index].IsDeleted {
                    hasUnsavedChanges = true
                }
                if state.Databases[index].IsNew {
                    hasUnsavedChanges = true
                }
                switch state.Databases[index].Type {
                case .Vic3:
                    drawTransparentTexture(ctx, TEXTURE_ICON_GAME_VIC3)
                case .Eu5:
                    drawTransparentTexture(ctx, TEXTURE_ICON_GAME_EU5)
                case .Unknown:
                    ui.Icon(ctx, .Unknown, ui.COLOR_BUTTON_DANGER_BACKGROUND_HOVER)
                }
                if .CHANGE in mu.textbox(ctx, state.Databases[index].BufferName[:], &state.Databases[index].BufferNameLength) {
                    name := strings.clone(string(state.Databases[index].BufferName[:state.Databases[index].BufferNameLength]))
                    delete(state.Databases[index].Name)
                    state.Databases[index].Name = name
                }
                if .CHANGE in mu.textbox(ctx, state.Databases[index].BufferPath[:], &state.Databases[index].BufferPathLength) {
                    path := strings.clone(string(state.Databases[index].BufferPath[:state.Databases[index].BufferPathLength]))
                    delete(state.Databases[index].Path)
                    state.Databases[index].Path = path
                }
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in ui.Button(ctx, "Choose Folder") {
                    path, success := chooseFolder()
                    if success {
                        state.Databases[index].BufferPathLength = len(path)
                        copy(state.Databases[index].BufferPath[:], path)
                        delete(state.Databases[index].Path)
                        state.Databases[index].Path = path

                        _, file := os.split_path(path)
                        state.Databases[index].BufferNameLength = len(file)
                        copy(state.Databases[index].BufferName[:], file)
                        delete(state.Databases[index].Name)
                        state.Databases[index].Name = strings.clone(file)

                        determineDatabaseType(&state.Databases[index])
                    }
                }
                mu.pop_id(ctx)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if state.Databases[index].IsDeleted {
                    if .SUBMIT in mu.button(ctx, "Undo") {
                        state.Databases[index].IsDeleted = false
                    }
                } else if state.Databases[index].IsNew {
                    if .SUBMIT in ui.Button(ctx, "Delete", ui.ButtonStyle.Danger) {
                        DestroyDatabase(&state.Databases[index])
                        ordered_remove(&state.Databases, index)
                    }
                } else {
                    if .SUBMIT in ui.Button(ctx, "Delete", ui.ButtonStyle.Danger) {
                        state.Databases[index].IsDeleted = true
                    }
                }
                mu.pop_id(ctx)
            }
            mu.layout_row(ctx, {-1, -1}, 0)
            if .SUBMIT in ui.Button(ctx, "Add new path") {
                element := NewDatabase("", "")
                element.IsNew = true
                append(&state.Databases, element)
            }
        }

        if .ACTIVE in mu.header(ctx, "Background Color", {.EXPANDED}) {
            mu.layout_row(ctx, {-78, -1}, 68)
            mu.layout_begin_column(ctx)
            {
                mu.layout_row(ctx, {46, -1}, 0)
                mu.label(ctx, "Red:");   ui.Slider(ctx, &state.Settings.BackgroundColor.r, 0, 255)
                mu.label(ctx, "Green:"); ui.Slider(ctx, &state.Settings.BackgroundColor.g, 0, 255)
                mu.label(ctx, "Blue:");  ui.Slider(ctx, &state.Settings.BackgroundColor.b, 0, 255)
            }
            mu.layout_end_column(ctx)

            r := mu.layout_next(ctx)
            mu.draw_rect(ctx, r, state.Settings.BackgroundColor)
            mu.draw_box(ctx, mu.expand_rect(r, 1), ctx.style.colors[.BORDER])
            mu.draw_control_text(ctx, fmt.tprintf(
                "#%02x%02x%02x",
                state.Settings.BackgroundColor.r,
                state.Settings.BackgroundColor.g,
                state.Settings.BackgroundColor.b
            ), r, .TEXT, { .ALIGN_CENTER })
        }
    }

    cnt := mu.get_container(ctx, WINDOM_SETTINGS)
    is_open := cnt != nil && cnt.open != false

    if state.SettingsWindowOpen && !is_open {
        state.SettingsWindowOpen = false
        for _, index in state.Databases {
            state.Databases[index].IsDeleted = false
        }
    }
}

renderFlagDatabase :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_FLAG_DATABASE, {10, 10 + TOOLBAR_HEIGHT, 500, screenHeight() - 20 - TOOLBAR_HEIGHT}, {}) {
        guranteeBounds(ctx)

        mu.layout_row(ctx, { -1 }, 15)
        mu.text(ctx, "Search:")
        @static searchBuffer: [128]byte
        @static searchBufferLength: int
        if .CHANGE in mu.textbox(ctx, searchBuffer[:], &searchBufferLength) {
            delete(state.SearchCache.FlagDatabaseSearch)
            state.SearchCache.FlagDatabaseSearch = strings.to_lower(string(searchBuffer[:searchBufferLength]))
        }
        mu.layout_row(ctx, { -1 }, 5)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])

        options: mu.Options
        if state.SearchCache.FlagDatabaseSearch != "" {
            options = { .EXPANDED }
        }

        for database in state.Databases {
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .ACTIVE in mu.header(ctx, database.Settings.Name, options) {
                for flag in database.Flags {
                    searchName := strings.to_lower(flag.Name)
                    defer delete(searchName)
                    if !strings.contains(searchName, state.SearchCache.FlagDatabaseSearch) {
                        continue
                    }
                    mu.layout_row(ctx, {40, -139, 80, 50}, 24)
                    drawFlag(ctx, flag.Name, 36, 24)
                    mu.text(ctx, flag.Name)
                    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                    if .SUBMIT in mu.button(ctx, "Add as sub") {
                        sub := pdx.CreateLayerSub(flag.Name)
                        append(&state.Flag.Layers, sub)
                    }
                    mu.pop_id(ctx)
                    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                    if .SUBMIT in ButtonTextConfirm(ctx, "Open", "Open Flag", POPUP_MESSAGE_OPEN_FLAG, ui.ButtonStyle.Danger) {
                        pdx.DestroyFlag(&state.Flag)
                        clonedFlag := pdx.CloneFlag(flag)
                        state.Flag = clonedFlag
                    }
                    mu.pop_id(ctx)
                }
            }
            mu.pop_id(ctx)
        }
    }

    cnt := mu.get_container(ctx, WINDOM_FLAG_DATABASE)
    is_open := cnt != nil && cnt.open != false

    if state.FlagDatabaseWindowOpen && !is_open {
        state.FlagDatabaseWindowOpen = false
    }
}

renderTextureDatabase :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_TEXTURE_DATABASE, {10, 10 + TOOLBAR_HEIGHT, 500, screenHeight() - 20 - TOOLBAR_HEIGHT}, {}) {
        guranteeBounds(ctx)

        mu.layout_row(ctx, {-1, -1}, 15)
        mu.text(ctx, "Search:")
        @static searchBuffer: [128]byte
        @static searchBufferLength: int
        mu.layout_row(ctx, {-1, -1}, 20)
        if .CHANGE in mu.textbox(ctx, searchBuffer[:], &searchBufferLength) {
            delete(state.SearchCache.TextureDatabaseSearch)
            state.SearchCache.TextureDatabaseSearch = strings.to_lower(string(searchBuffer[:searchBufferLength]))
        }
        mu.layout_row(ctx, {-1, -1}, 5)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])

        options: mu.Options
        if state.SearchCache.TextureDatabaseSearch != "" {
            options = { .EXPANDED }
        }

        for database in state.Databases {
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .ACTIVE in mu.header(ctx, database.Settings.Name, {.EXPANDED}) {
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .ACTIVE in mu.treenode(ctx, "Patterns", options) {
                    for texture in database.Patterns {
                        name := strings.to_lower(texture.Name)
                        defer delete(name)
                        if state.SearchCache.TextureDatabaseSearch == "" || strings.contains(name, state.SearchCache.TextureDatabaseSearch) {
                            renderTextureDatabaseItem(ctx, texture, pdx.FOLDER_PATTERNS)
                        }
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
                        name := strings.to_lower(texture.Name)
                        defer delete(name)
                        if state.SearchCache.TextureDatabaseSearch == "" || strings.contains(name, state.SearchCache.TextureDatabaseSearch) {
                            renderTextureDatabaseItem(ctx, texture, pdx.FOLDER_COLORED_EMBLEMS)
                        }
                    }
                    if len(database.ColoredEmblems) == 0 {
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
                        name := strings.to_lower(texture.Name)
                        defer delete(name)
                        if state.SearchCache.TextureDatabaseSearch == "" || strings.contains(name, state.SearchCache.TextureDatabaseSearch) {
                            renderTextureDatabaseItem(ctx, texture, pdx.FOLDER_TEXTURED_EMBLEMS)
                        }
                    }
                    if len(database.TexturedEmblems) == 0 {
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

    cnt := mu.get_container(ctx, WINDOM_TEXTURE_DATABASE)
    is_open := cnt != nil && cnt.open != false

    if state.TextureDatabaseWindowOpen && !is_open {
        state.TextureDatabaseWindowOpen = false
    }
}

renderTextureDatabaseItem :: proc(ctx: ^mu.Context, item: pdx.FlagTexture, type: string) {
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

ButtonTextConfirm :: proc(
    ctx: ^mu.Context,
    label: string,
    confirmHeader, confirmText: string,
    style: ui.ButtonStyle = .Default,
    options: mu.Options = { .ALIGN_CENTER },
    enabled := true,
) -> (res: mu.Result_Set) {

    isButtonEnabled := enabled && state.PopupIdentifier == 0

    if .SUBMIT in ui.Button(ctx, label, style, options, isButtonEnabled) {
        // our button did the popup
        state.PopupIdentifier = ctx.last_id
    }

    // our button did the popup?
    if state.PopupIdentifier == ctx.last_id {
        return popupWindow(ctx, confirmHeader, confirmText, style, enabled)
    }

    return
}

ButtonIconConfirm :: proc(
    ctx: ^mu.Context,
    icon: ui.IconType,
    confirmHeader, confirmText: string,
    width: i32 = 0,
    height: i32 = 0,
    style: ui.ButtonStyle = .Default,
    tint := ui.COLOR_TINT_NONE,
    options: mu.Options = { .ALIGN_CENTER },
    enabled := true,
) -> (res: mu.Result_Set) {

    isButtonEnabled := enabled && state.PopupIdentifier == 0

    if .SUBMIT in ui.Button(ctx, icon, width, height, style, tint, options, isButtonEnabled) {
        // our button activates the popup
        state.PopupIdentifier = ctx.last_id
    }

    // our button activated the popup?
    if state.PopupIdentifier == ctx.last_id {
        return popupWindow(ctx, confirmHeader, confirmText, style, enabled)
    }

    return
}

popupWindow :: proc(
    ctx: ^mu.Context,
    confirmHeader, confirmText: string,
    style: ui.ButtonStyle,
    enabled: bool,
) -> (res: mu.Result_Set) {
    center := mu.Rect{
        x = (screenWidth() - 300) / 2,
        y = (screenHeight() - 200) / 2,
        w = 300,
        h = 100,
    }
    if mu.window(ctx, confirmHeader, center, { .NO_RESIZE, .NO_CLOSE, .AUTO_SIZE }) {
        window := mu.get_current_container(ctx)

        // always on top
        window.zindex = ctx.last_zindex + 1

        // always center
        window.rect.x = (screenWidth() - window.rect.w) / 2
        window.rect.y = (screenHeight() - window.rect.h) / 2

        mu.layout_row(ctx, { 300 }, 0)
        mu.text(ctx, confirmText)

        mu.layout_row(ctx, { 300 }, 20)
        row := mu.layout_next(ctx)

        buttonOk := mu.Rect{
            x = row.x + ((row.w - 200) / 2),
            y = row.y,
            w = 90,
            h = row.h,
        }
        mu.layout_set_next(ctx, buttonOk, false)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in ui.Button(ctx, "Ok (Enter)", style, enabled = enabled) {
            res += { .SUBMIT }
            state.PopupIdentifier = 0
        }
        mu.pop_id(ctx)

        buttonCancel := mu.Rect{
            x = row.x + ((row.w - 200) / 2) + 110,
            y = row.y,
            w = 90,
            h = row.h,
        }
        mu.layout_set_next(ctx, buttonCancel, false)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in ui.Button(ctx, "Cancel (Esc)") {
            state.PopupIdentifier = 0
        }
        mu.pop_id(ctx)

        if rl.IsKeyPressed(rl.KeyboardKey.ENTER) && enabled {
            res += { .SUBMIT }
            state.PopupIdentifier = 0
        }

        if rl.IsKeyPressed(rl.KeyboardKey.ESCAPE) {
            state.PopupIdentifier = 0
        }
    }
    return
}