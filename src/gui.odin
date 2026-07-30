package pdx_flag_builder

import "core:fmt"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "texture"
import "ui"
import "pdx"
import nfd "nativefiledialog"
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
TEXTURE_TRANSPARENCY: string: "assets/textures/transparency.dds"
TEXTURE_INVALID: string: "assets/textures/invalid.dds"

SHADER_RECOLOR: string: "assets/shaders/recolor.fs"

SIDEBAR_HANDLE_WIDTH:i32: 10
SIDEBAR_MIN_WIDTH: i32: 350
SIDEBAR_MAX_WIDTH: i32: 600
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
    ui.InitToast(ctx, &state.Toast, { 10, TOOLBAR_HEIGHT + 10, 200, rl.GetScreenHeight() - (TOOLBAR_HEIGHT + 20) }, &state.ButtonIdentifier)

    rl.SetWindowMinSize(FLAG_WIDTH + state.SidebarWidth, FLAG_HEIGHT + TOOLBAR_HEIGHT)
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
    x: i32 = (rl.GetScreenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (rl.GetScreenHeight() - height) / 2

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
    x: i32 = (rl.GetScreenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (rl.GetScreenHeight() - height) / 2

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
    x: i32 = (rl.GetScreenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (rl.GetScreenHeight() - height) / 2

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
    x: i32 = (rl.GetScreenWidth() - width - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH) / 2
    y: i32 = (rl.GetScreenHeight() - height) / 2

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
        }
    }

    cnt := mu.get_container(ctx, WINDOM_INSTANCE_EDITOR)
    is_open := cnt != nil && cnt.open != false

    if state.InstanceEditorInstance != nil && !is_open {
        state.InstanceEditorInstance = nil
    }
}

renderToolbar :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_TOOLBAR, { 0, 0, rl.GetScreenWidth(), TOOLBAR_HEIGHT }, { .NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL }) {
        mu.get_current_container(ctx).rect.w = rl.GetScreenWidth()
        mu.layout_row_items(ctx, 6, TOOLBAR_HEIGHT - 8)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "File") {
            exportAsIs :: proc(ctx: ^mu.Context) {
                exportFlagAsIs(ctx, state.Flag)
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
                ui.CreateDropdownElement("Save Changes", exportAsIs),
                ui.CreateDropdownElement("Export to Image", exportToImage),
                ui.CreateDropdownElement("Export to Clipboard", exportToClipboard),
                ui.CreateDropdownElement("Export as script File", exportToFile),
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
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Texture Database") {
            if state.TextureDatabaseWindowOpen {
                state.TextureDatabaseWindowOpen = false
            } else {
                state.TextureDatabaseWindowOpen = true
            }
        }
        mu.layout_end_column(ctx)

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Flag Database") {
            if state.FlagDatabaseWindowOpen {
                state.FlagDatabaseWindowOpen = false
            } else {
                state.FlagDatabaseWindowOpen = true
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
    destination: mu.Rect
    if state.SidebarOpen {
        destination = mu.Rect{
            x = (rl.GetScreenWidth() - FLAG_WIDTH - state.SidebarWidth - SIDEBAR_HANDLE_WIDTH - 10) / 2,
            y = TOOLBAR_HEIGHT + ((rl.GetScreenHeight() - FLAG_HEIGHT - TOOLBAR_HEIGHT - 6) / 2),
            w = FLAG_WIDTH,
            h = FLAG_HEIGHT,
        }
    } else {
        destination = mu.Rect{
            x = (rl.GetScreenWidth() - FLAG_WIDTH) / 2,
            y = TOOLBAR_HEIGHT + ((rl.GetScreenHeight() - FLAG_HEIGHT - TOOLBAR_HEIGHT - 6) / 2),
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

fillColorMapping :: proc(mappings: ^[dynamic]texture.ColorRecolor, variant: pdx.FlagColor, flag: pdx.Flag, sourceColors: []rl.Color) {
    switch color in variant.Variant {
    case pdx.FlagColorRgb:
        targetColor := pdx.ToRenderColor(color)
        mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
        if ok {
            append(mappings, mapping)
        }
    case pdx.FlagColorHsv:
        targetColor := pdx.ToRenderColor(color)
        mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
        if ok {
            append(mappings, mapping)
        }
    case pdx.FlagColorNamed:
        retrieve, exists := getNamedColor(color.NamedColor)
        if !exists {
            return
        }
        targetColor := pdx.ToRenderColor(retrieve)
        mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
        if ok {
            append(mappings, mapping)
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
                    append(mappings, mapping)
                }
            case pdx.FlagColorHsv:
                if baseColorVariant.Name != color.Reference {
                    continue
                }
                targetColor := pdx.ToRenderColor(baseColor)
                mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
                if ok {
                    append(mappings, mapping)
                }
            case pdx.FlagColorNamed:
                if baseColorVariant.Name != color.Reference {
                    continue
                }
                retrieve, exists := getNamedColor(baseColor.NamedColor)
                if !exists {
                    return
                }
                targetColor := pdx.ToRenderColor(retrieve)
                mapping, ok := checkColorMapping(variant.Name, targetColor, sourceColors)
                if ok {
                    append(mappings, mapping)
                }
            case pdx.FlagColorReference:
                // Should not happen
            }
        }
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
    if mu.window(ctx, WINDOM_SETTINGS, { 10, 10 + TOOLBAR_HEIGHT, 670, 290 }, {}) {
        windows := mu.get_current_container(ctx)
        windows.rect.w = 670
        guranteeBounds(ctx)

        mu.layout_row(ctx, { -1 }, 0)
        if .SUBMIT in mu.button(ctx, "Save Settings") {
            SaveSettings()
            ui.Toast(&state.Toast, "Settings", "Successfully saved settings.")
        }

        if .ACTIVE in mu.header(ctx, "Paths", {.EXPANDED}) {
            mu.text(ctx, "These are paths to the root of your mod or the \"game\" folder for the base game.")
            mu.layout_row(ctx, {100, -209, 100, 100}, 0)
            for _, index in state.Databases {
                if .CHANGE in mu.textbox(ctx, state.Databases[index].BufferName[:], &state.Databases[index].BufferNameLength) {
                    name := strings.clone(string(state.Databases[index].BufferName[:state.Databases[index].BufferNameLength]))
                    delete(state.Databases[index].Settings.Name)
                    state.Databases[index].Settings.Name = name
                }
                if .CHANGE in mu.textbox(ctx, state.Databases[index].BufferPath[:], &state.Databases[index].BufferPathLength) {
                    path := strings.clone(string(state.Databases[index].BufferPath[:state.Databases[index].BufferPathLength]))
                    delete(state.Databases[index].Settings.Path)
                    state.Databases[index].Settings.Path = path
                }
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in ui.Button(ctx, "Choose Folder") {
                    path, success := chooseFolder()
                    if success {
                        state.Databases[index].BufferPathLength = len(path)
                        copy(state.Databases[index].BufferPath[:], path)
                        delete(state.Databases[index].Settings.Path)
                        state.Databases[index].Settings.Path = path

                        _, file := os.split_path(path)
                        state.Databases[index].BufferNameLength = len(file)
                        copy(state.Databases[index].BufferName[:], file)
                        delete(state.Databases[index].Settings.Name)
                        state.Databases[index].Settings.Name = strings.clone(file)
                    }
                }
                mu.pop_id(ctx)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if state.Databases[index].IsDeleted {
                    if .SUBMIT in mu.button(ctx, "Undo") {
                        state.Databases[index].IsDeleted = false
                    }
                } else {
                    if .SUBMIT in ui.Button(ctx, "Delete", ui.ButtonStyle.Danger) {
                        state.Databases[index].IsDeleted = true
                    }
                }
                mu.pop_id(ctx)
            }
            mu.layout_row(ctx, {-1, -1}, 0)
            if .SUBMIT in mu.button(ctx, "Add new path") {
                element := NewDatabase("", "")
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
    if mu.window(ctx, WINDOM_FLAG_DATABASE, {10, 10 + TOOLBAR_HEIGHT, 500, rl.GetScreenHeight() - 20 - TOOLBAR_HEIGHT}, {}) {
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
                    mu.layout_row(ctx, {40, -136, 80, 40}, 24)
                    drawFlag(ctx, flag.Name, 36, 24)
                    mu.text(ctx, flag.Name)
                    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                    if .SUBMIT in mu.button(ctx, "Add as sub") {
                        sub := pdx.CreateLayerSub(flag.Name)
                        append(&state.Flag.Layers, sub)
                    }
                    mu.pop_id(ctx)
                    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                    if .SUBMIT in mu.button(ctx, "Open") {
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
    if mu.window(ctx, WINDOM_TEXTURE_DATABASE, {10, 10 + TOOLBAR_HEIGHT, 500, rl.GetScreenHeight() - 20 - TOOLBAR_HEIGHT}, {}) {
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
            rl.SetMouseCursor(.RESIZE_EW)
        } else {
            rl.SetMouseCursor(.DEFAULT)
        }

        mu.draw_rect(ctx, r, ctx.style.colors[.BORDER])
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
        if .SUBMIT in mu.button(ctx, "Coat of Arms") {
            state.SelectedFlagElement = &state.Flag
        }
        mu.pop_id(ctx)

        for variant, index in state.Flag.Layers {
            switch layer in variant {
            case ^pdx.FlagLayerColoredEmblem:
                mu.layout_row(ctx, {15, -1}, 20)
                ui.Icon(ctx, .Sub)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Colored Emblem: %s", layer.Texture.Name)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
                renderLayerInstances(ctx, layer.Instances)
            case ^pdx.FlagLayerTexturedEmblem:
                mu.layout_row(ctx, {15, -1}, 20)
                ui.Icon(ctx, .Sub)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Textured Emblem: %s", layer.Texture.Name)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
                renderLayerInstances(ctx, layer.Instances)
            case ^pdx.FlagLayerSub:
                mu.layout_row(ctx, {15, -1}, 20)
                ui.Icon(ctx, .Sub)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Sub Flag: %s", layer.Parent)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
                renderLayerSubInstances(ctx, layer.Instances)
            }
        }
        mu.layout_row(ctx, { -1, -1 }, 20)
    }
}

renderLayerInstances :: proc(ctx: ^mu.Context, instances: [dynamic]^pdx.LayerInstance) {
    for instance, index in instances {
        mu.layout_row(ctx, {-1, -1}, 20)
        buttonRect := mu.layout_next(ctx)
        buttonRect.w = buttonRect.w - 19
        buttonRect.x = buttonRect.x + 19

        subRect := mu.Rect{
            buttonRect.x, buttonRect.y, 15, 20
        }

        buttonRect.w = buttonRect.w - 19
        buttonRect.x = buttonRect.x + 19

        mu.layout_set_next(ctx, subRect, false)
        ui.Icon(ctx, .Sub)
        mu.layout_set_next(ctx, buttonRect, false)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in mu.button(ctx, "Instance") {
            state.SelectedFlagElement = instance
        }
        mu.pop_id(ctx)
    }
}

renderLayerSubInstances :: proc(ctx: ^mu.Context, instances: [dynamic]^pdx.LayerInstanceSub) {
    for instance, index in instances {
        mu.layout_row(ctx, {-1, -1}, 20)
        buttonRect := mu.layout_next(ctx)
        buttonRect.w = buttonRect.w - 19
        buttonRect.x = buttonRect.x + 19

        subRect := mu.Rect{
            buttonRect.x, buttonRect.y, 15, 20
        }

        buttonRect.w = buttonRect.w - 19
        buttonRect.x = buttonRect.x + 19

        mu.layout_set_next(ctx, subRect, false)
        ui.Icon(ctx, .Sub)

        mu.layout_set_next(ctx, buttonRect, false)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in mu.button(ctx, "Instance") {
            state.SelectedFlagElement = instance
        }
        mu.pop_id(ctx)
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
                renderSelectedFlag(ctx, element)
            case ^pdx.FlagLayerColoredEmblem:
                renderSelectedFlagLayerColoredEmblem(ctx, element)
            case ^pdx.FlagLayerTexturedEmblem:
                renderSelectedFlagLayerTexturedEmblem(ctx, element)
            case ^pdx.FlagLayerSub:
                renderSelectedFlagLayerSub(ctx, element)
            case ^pdx.LayerInstance:
                renderSelectedLayerInstance(ctx, element)
            case ^pdx.LayerInstanceSub:
                renderSelectedLayerInstance(ctx, element)
            }
        }
    }
}

renderSelectedFlag :: proc(ctx: ^mu.Context, flag: ^pdx.Flag) {
    ui.DrawAttributeRow(ctx, "Layer", "Coat of Arms")
    @static buf: [128]byte
    @static bufLen: int
    bufLen = len(flag.Name)
    copy(buf[:], flag.Name)
    if .CHANGE in ui.DrawAttributeRowTextbox(ctx, "Name", buf[:], &bufLen) {
        name := strings.clone(string(buf[:bufLen]))
        delete(flag.Name)
        flag.Name = name
    }
    if flag.Pattern.Name != "" {
        ui.DrawAttributeRow(ctx, "Pattern", flag.Pattern.Name)
    } else {
        ui.DrawAttributeRow(ctx, "Pattern", "None")
    }
    if flag.File != "" {
        ui.DrawAttributeRow(ctx, "File", flag.File)
    } else {
        ui.DrawAttributeRow(ctx, "File", "None")
    }
    if flag.Source != "" {
        ui.DrawAttributeRow(ctx, "Database", flag.Source)
    } else {
        ui.DrawAttributeRow(ctx, "Database", "None")
    }
    for _, index in flag.Colors {
        variant := flag.Colors[index]
        switch color in variant.Variant {
        case pdx.FlagColorNamed:
            targetColor, found := getNamedColor(color.NamedColor)
            targetColorValue := mu.Color{ 0, 0, 0, 255 }
            if found {
                renderColorValue := pdx.ToRenderColor(targetColor)
                targetColorValue = mu.Color{ renderColorValue[0], renderColorValue[1], renderColorValue[2], 255 }
            }
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, color, targetColorValue)
            renderColorButtons(ctx, &flag.Colors[index], &state.Flag.Colors, index, buttonEditRect, buttonDelRect)
        case pdx.FlagColorRgb:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, variant)
            renderColorButtons(ctx, &flag.Colors[index], &state.Flag.Colors, index, buttonEditRect, buttonDelRect)
        case pdx.FlagColorHsv:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, variant)
            renderColorButtons(ctx, &flag.Colors[index], &state.Flag.Colors, index, buttonEditRect, buttonDelRect)
        case pdx.FlagColorReference:
            // Should not happen
        }
    }
    mu.layout_row(ctx, { -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add RGB color") {
        colorName := pdx.GetNextFreeColor(flag.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorRgb(colorName, 0, 0, 0)
            append(&flag.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add HSV color") {
        colorName := pdx.GetNextFreeColor(flag.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorHsv(colorName, 0, 0, 0)
            append(&flag.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add Named color") {
        colorName := pdx.GetNextFreeColor(flag.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorNamed(colorName, "")
            append(&flag.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, "Clear Flag", ui.ButtonStyle.Danger) {
        state.SelectedFlagElement = nil
        pdx.DestroyFlag(&state.Flag)
        flag := pdx.CreateFlag()
        state.Flag = flag
    }
    mu.pop_id(ctx)
}

renderSelectedFlagLayerColoredEmblem :: proc(ctx: ^mu.Context, layer: ^pdx.FlagLayerColoredEmblem) {
    ui.DrawAttributeRow(ctx, "Type", "Colored Emblem")
    ui.DrawAttributeRow(ctx, "Texture", layer.Texture.Name)
    ui.DrawAttributeRow(ctx, "Instances", fmt.tprintf("%i", len(layer.Instances)))
    for _, index in layer.Colors {
        variant := layer.Colors[index]
        switch color in variant.Variant {
        case pdx.FlagColorNamed:
            targetColor, found := getNamedColor(color.NamedColor)
            targetColorValue := mu.Color{ 0, 0, 0, 255 }
            if found {
                renderColorValue := pdx.ToRenderColor(targetColor)
                targetColorValue = mu.Color{ renderColorValue[0], renderColorValue[1], renderColorValue[2], 255 }
            }
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, color, targetColorValue)
            renderColorButtons(ctx, &layer.Colors[index], &layer.Colors, index, buttonEditRect, buttonDelRect)
        case pdx.FlagColorRgb:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, variant)
            renderColorButtons(ctx, &layer.Colors[index], &layer.Colors, index, buttonEditRect, buttonDelRect)
        case pdx.FlagColorHsv:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, variant)
            renderColorButtons(ctx, &layer.Colors[index], &layer.Colors, index, buttonEditRect, buttonDelRect)
        case pdx.FlagColorReference:
            targetColor, found := getReferencedColor(color.Reference, state.Flag)
            targetColorValue := mu.Color{ 0, 0, 0, 255 }
            if found {
                renderColorValue := pdx.ToRenderColor(targetColor)
                targetColorValue = mu.Color{ renderColorValue[0], renderColorValue[1], renderColorValue[2], 255 }
            }
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, variant.Name, color, targetColorValue)
            renderColorButtons(ctx, &layer.Colors[index], &layer.Colors, index, buttonEditRect, buttonDelRect)
        }
    }
    mu.layout_row(ctx, { -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add RGB color") {
        colorName := pdx.GetNextFreeColor(layer.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorRgb(colorName, 0, 0, 0)
            append(&layer.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add HSV color") {
        colorName := pdx.GetNextFreeColor(layer.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorHsv(colorName, 0, 0, 0)
            append(&layer.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add Named color") {
        colorName := pdx.GetNextFreeColor(layer.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorNamed(colorName, "")
            append(&layer.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add reference color") {
        colorName := pdx.GetNextFreeColor(layer.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorReference(colorName, pdx.COLOR_NAMES[0])
            append(&layer.Colors, color)
        }
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add new insance") {
        instance := pdx.CreateLayerInstance()
        append(&layer.Instances, instance)
        state.SelectedFlagElement = instance
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, "Delete Layer", ui.ButtonStyle.Danger) {
        removeLayer(layer)
        state.SelectedFlagElement = nil
    }
    mu.pop_id(ctx)
}

renderSelectedFlagLayerTexturedEmblem :: proc(ctx: ^mu.Context, layer: ^pdx.FlagLayerTexturedEmblem) {
    ui.DrawAttributeRow(ctx, "Type", "Textured Emblem")
    ui.DrawAttributeRow(ctx, "Texture", layer.Texture.Name)
    ui.DrawAttributeRow(ctx, "Instances", fmt.tprintf("%i", len(layer.Instances)))
    mu.layout_row(ctx, { -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add new insance") {
        instance := pdx.CreateLayerInstance()
        append(&layer.Instances, instance)
        state.SelectedFlagElement = instance
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, "Delete Layer", ui.ButtonStyle.Danger) {
        removeLayer(layer)
        state.SelectedFlagElement = nil
    }
    mu.pop_id(ctx)
}

renderSelectedFlagLayerSub :: proc(ctx: ^mu.Context, layer: ^pdx.FlagLayerSub) {
    ui.DrawAttributeRow(ctx, "Type", "Sub Flag")
    ui.DrawAttributeRow(ctx, "Parent", layer.Parent)
    ui.DrawAttributeRow(ctx, "Instances", fmt.tprintf("%i", len(layer.Instances)))
    mu.layout_row(ctx, { -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add new insance") {
        instance := pdx.CreateLayerInstanceSub()
        append(&layer.Instances, instance)
        state.SelectedFlagElement = instance
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, "Delete Layer", ui.ButtonStyle.Danger) {
        removeLayer(layer)
        state.SelectedFlagElement = nil
    }
    mu.pop_id(ctx)
}

renderSelectedLayerInstance :: proc(ctx: ^mu.Context, instance: pdx.LayerInstanceVariant) {
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

    ui.DrawAttributeRow(ctx, "Type", "Layer Instance")
    switch type in instance {
    case ^pdx.LayerInstance:
        ui.DrawAttributeRowNumberTextbox(ctx, "Rotation", &type.Rotation, rotationBuf[:], &rotationBufLen, 360, 0)
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Width", &type.Scale.X, scaleXBuf[:], &scaleXBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Height", &type.Scale.Y, scaleYBuf[:], &scaleYBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
        ui.DrawAttributeRowNumberTextbox(ctx, "Position X", &type.Position.X, positionXBuf[:], &positionXBufLen, pdx.MAX_POSITION, pdx.MIN_POSITION, "%.3f")
        ui.DrawAttributeRowNumberTextbox(ctx, "Position Y", &type.Position.Y, positionYBuf[:], &positionYBufLen, pdx.MAX_POSITION, pdx.MIN_POSITION, "%.3f")
    case ^pdx.LayerInstanceSub:
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Width", &type.Scale.X, scaleXBuf[:], &scaleXBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Height", &type.Scale.Y, scaleYBuf[:], &scaleYBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f")
        ui.DrawAttributeRowNumberTextbox(ctx, "Offset X", &type.Offset.X, positionXBuf[:], &positionXBufLen, pdx.MAX_OFFSET, pdx.MIN_OFFSET, "%.3f")
        ui.DrawAttributeRowNumberTextbox(ctx, "Offset Y", &type.Offset.Y, positionYBuf[:], &positionYBufLen, pdx.MAX_OFFSET, pdx.MIN_OFFSET, "%.3f")
    }
    mu.layout_row(ctx, { -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Edit instance") {
        state.InstanceEditorInstance = instance
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, "Delete instance", ui.ButtonStyle.Danger) {
        removeInstance(instance)
        state.SelectedFlagElement = nil
    }
    mu.pop_id(ctx)
}

renderColorButtons :: proc(ctx: ^mu.Context, color: ^pdx.FlagColor, list: ^[dynamic]pdx.FlagColor, index: int, edit, delete: mu.Rect) {
    mu.layout_set_next(ctx, edit, false)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, ui.IconType.Edit, edit.h - 4, edit.h - 4) {
        state.ColorPickerColor = color
    }
    mu.pop_id(ctx)
    mu.layout_set_next(ctx, delete, false)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ui.Button(ctx, ui.IconType.Delete, delete.h - 4, delete.h - 4, ui.ButtonStyle.Danger) {
        if color == state.ColorPickerColor {
            state.ColorPickerColor = nil
        }
        pdx.DestroyFlagColor(color)
        ordered_remove(list, index)
    }
    mu.pop_id(ctx)
}