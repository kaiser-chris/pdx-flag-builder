package pdx_flag_builder

import "core:fmt"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "texture"
import "ui"
import "pdx"

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

TEXTURE_SUB: string: "textures/sub.dds"

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
    ^pdx.LayerInstance,
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
        switch color in state.ColorPickerColor {
        case ^pdx.FlagColorRgb:
            renderAbsoluteColorPicker(ctx)
        case ^pdx.FlagColorHsv:
            renderAbsoluteColorPicker(ctx)
        case ^pdx.FlagColorNamed:
            renderNamedColorPicker(ctx)
        case ^pdx.FlagColorReference:
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
            switch color in state.ColorPickerColor {
            case ^pdx.FlagColorRgb:
                mu.label(ctx, "Red:")
                ui.Slider(ctx, &color.R, 0, 255)
                ui.NumberTextbox(ctx, &color.R, &redBuf, &redBufLen, 255, 0)

                mu.label(ctx, "Green:")
                ui.Slider(ctx, &color.G, 0, 255)
                ui.NumberTextbox(ctx, &color.G, &greenBuf, &greenBufLen, 255, 0)

                mu.label(ctx, "Blue:")
                ui.Slider(ctx, &color.B, 0, 255)
                ui.NumberTextbox(ctx, &color.B, &blueBuf, &blueBufLen, 255, 0)
                red = color.R
                green = color.G
                blue = color.B
            case ^pdx.FlagColorHsv:
                mu.label(ctx, "Hue:")
                ui.Slider(ctx, &color.H, 0, 360, step = 1)
                ui.NumberTextbox(ctx, &color.H, &redBuf, &redBufLen, 360, 0)

                mu.label(ctx, "Saturation:")
                ui.Slider(ctx, &color.S, 0, 100, step = 1)
                ui.NumberTextbox(ctx, &color.S, &greenBuf, &greenBufLen, 100, 0)

                mu.label(ctx, "Value:")
                ui.Slider(ctx, &color.V, 0, 100, step = 1)
                ui.NumberTextbox(ctx, &color.V, &blueBuf, &blueBufLen, 100, 0)

                rgb := pdx.ToRenderColor(color)
                red = rgb[0]
                green = rgb[1]
                blue = rgb[2]
            case ^pdx.FlagColorNamed:
            case ^pdx.FlagColorReference:
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

        for database in state.Databases {
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .ACTIVE in mu.header(ctx, database.Settings.Name) {
                for variant in database.NamedColors {
                    name: string
                    rectColor: mu.Color
                    switch color in variant {
                    case ^pdx.FlagColorRgb:
                        name = color.Name
                        rectColor = mu.Color{ color.R, color.G, color.B, 255 }
                    case ^pdx.FlagColorHsv:
                        name = color.Name
                        tmp := pdx.ToRenderColor(color)
                        rectColor = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
                    case ^pdx.FlagColorNamed:
                        fmt.eprintfln("Named color is referencing another named color")
                    case ^pdx.FlagColorReference:
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
                        switch color in state.ColorPickerColor {
                        case ^pdx.FlagColorRgb:
                            fmt.eprintfln("RGB color does not support named color")
                        case ^pdx.FlagColorHsv:
                            fmt.eprintfln("HSV color does not support named color")
                        case ^pdx.FlagColorReference:
                            fmt.eprintfln("reference color does not support named color")
                        case ^pdx.FlagColorNamed:
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
            name: string
            rectColor: mu.Color

            switch color in variant {
            case ^pdx.FlagColorRgb:
                name = color.Name
                rectColor = mu.Color{ color.R, color.G, color.B, 255 }
            case ^pdx.FlagColorHsv:
                name = color.Name
                tmp := pdx.ToRenderColor(color)
                rectColor = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
            case ^pdx.FlagColorNamed:
                renderColor, _ := resolveNamedColor(color.NamedColor)
                name = color.Name
                rectColor = mu.Color{ renderColor[0], renderColor[1], renderColor[2], 255 }
            case ^pdx.FlagColorReference:
                fmt.eprintfln("Named color is referencing a reference color")
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
                switch color in state.ColorPickerColor {
                case ^pdx.FlagColorRgb:
                    fmt.eprintfln("RGB color does not support reference color")
                case ^pdx.FlagColorHsv:
                    fmt.eprintfln("HSV color does not support reference color")
                case ^pdx.FlagColorNamed:
                    fmt.eprintfln("named color does not support reference color")
                case ^pdx.FlagColorReference:
                    color.Reference = name
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
        mu.label(ctx, "Rotation:")
        ui.Slider(ctx, &state.InstanceEditorInstance.Rotation, 0, 360, step = 1)
        ui.NumberTextbox(ctx, &state.InstanceEditorInstance.Rotation, &rotationBuf, &rotationBufLen, 360, 0)

        mu.label(ctx, "Scale - Width:")
        ui.Slider(ctx, &state.InstanceEditorInstance.Scale.X, 0, 3, "%.2f", step = 0.05)
        ui.NumberTextbox(ctx, &state.InstanceEditorInstance.Scale.X, &scaleXBuf, &scaleXBufLen, 3, 0, "%.3f")

        mu.label(ctx, "Scale - Height:")
        ui.Slider(ctx, &state.InstanceEditorInstance.Scale.Y, 0, 3, "%.2f", step = 0.05)
        ui.NumberTextbox(ctx, &state.InstanceEditorInstance.Scale.Y, &scaleYBuf, &scaleYBufLen, 3, 0, "%.3f")

        mu.label(ctx, "Position - X:")
        ui.Slider(ctx, &state.InstanceEditorInstance.Position.X, -2, 3, "%.2f", step = 0.05)
        ui.NumberTextbox(ctx, &state.InstanceEditorInstance.Position.X, &positionXBuf, &positionXBufLen, 3, -2, "%.3f")

        mu.label(ctx, "Position - Y:")
        ui.Slider(ctx, &state.InstanceEditorInstance.Position.Y, -2, 3, "%.2f", step = 0.05)
        ui.NumberTextbox(ctx, &state.InstanceEditorInstance.Position.Y, &positionYBuf, &positionYBufLen, 3, -2, "%.3f")
    }

    cnt := mu.get_container(ctx, WINDOM_INSTANCE_EDITOR)
    is_open := cnt != nil && cnt.open != false

    if state.InstanceEditorInstance != nil && !is_open {
        state.InstanceEditorInstance = nil
    }
}

renderToolbar :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_TOOLBAR, { 0, 0, rl.GetScreenWidth(), TOOLBAR_HEIGHT }, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL}) {
        mu.get_current_container(ctx).rect.w = rl.GetScreenWidth()
        mu.layout_row_items(ctx, 4, TOOLBAR_HEIGHT - 8)

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

        mu.layout_begin_column(ctx)
        mu.layout_row(ctx, {150, -1}, TOOLBAR_HEIGHT - 8)
        if .SUBMIT in mu.button(ctx, "Toggle Sidebar") {
            if state.SidebarOpen {
                state.SidebarOpen = false
            } else {
                state.SidebarOpen = true
            }
        }
        mu.layout_end_column(ctx)

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

    if mu.window(ctx, WINDOM_PREVIEW, destination, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .NO_SCROLL, .NO_FRAME}) {
        window := mu.get_current_container(ctx)
        window.rect = destination
        target := mu.Rect{x = 0, y = 0, w = FLAG_WIDTH, h = FLAG_HEIGHT}
        mu.draw_rect(ctx, destination, mu.Color{100, 100, 100, 255})
        mu.draw_box(ctx, destination, ctx.style.colors[.BORDER])
        mu.layout_set_next(ctx, target, true)
        drawFlagTexture(ctx)
    }
}

renderFlagPreviewTexture :: proc(cmd: ^mu.Command_Rect) {
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}
    rl.BeginScissorMode(cmd.rect.x, cmd.rect.y, cmd.rect.w, cmd.rect.h)

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

    for variant in state.Flag.Layers {
        switch layer in variant {
        case ^pdx.FlagLayerColoredEmblem:
            renderColoredEmblemInstances(cmd, layer)
        case ^pdx.FlagLayerTexturedEmblem:
            renderTexturedEmblemInstances(cmd, layer)
        }
    }
    rl.EndScissorMode()
}

renderColoredEmblemInstances :: proc(cmd: ^mu.Command_Rect, layer: ^pdx.FlagLayerColoredEmblem) {
    emblem, ok := state.RenderTextureMap[layer.Texture.Path]
    if !ok {
        return
    }
    source := rl.Rectangle{0, 0, f32(emblem.width), f32(emblem.height)}

    colorMappings := make([dynamic]texture.ColorRecolor)
    defer delete(colorMappings)
    for color in layer.Colors {
        fillColorMapping(&colorMappings, color, pdx.COLORED_EMBLEM_REPLACE_COLORS)
    }

    for instance in layer.Instances {
        texture.DrawRecoloredTexture(
            emblem,
            source,
            calculateInstanceDestination(emblem, instance, cmd.rect),
            colorMappings[:],
            state.RecolorShader,
            rotation = f32(instance.Rotation)
        )
    }

    if len(layer.Instances) == 0 {
        instance := pdx.LayerInstance{
            Rotation = 0,
            Position = pdx.DEFAULT_POSITION,
            Scale = pdx.DEFAULT_SCALE,
        }
        texture.DrawRecoloredTexture(
            emblem,
            source,
            calculateInstanceDestination(emblem, &instance, cmd.rect),
            colorMappings[:],
            state.RecolorShader,
            rotation = f32(instance.Rotation)
        )
    }
}

renderTexturedEmblemInstances :: proc(cmd: ^mu.Command_Rect, layer: ^pdx.FlagLayerTexturedEmblem) {
    emblem, ok := state.RenderTextureMap[layer.Texture.Path]
    if !ok {
        return
    }
    source := rl.Rectangle{0, 0, f32(emblem.width), f32(emblem.height)}

    for instance in layer.Instances {
        rl.DrawTexturePro(
            emblem,
            source,
            calculateInstanceDestination(emblem, instance, cmd.rect),
            { 0, 0 },
            f32(instance.Rotation),
            rl.WHITE
        )
    }

    if len(layer.Instances) == 0 {
        instance := pdx.LayerInstance{
            Rotation = 0,
            Position = pdx.DEFAULT_POSITION,
            Scale = pdx.DEFAULT_SCALE,
        }
        rl.DrawTexturePro(
            emblem,
            source,
            calculateInstanceDestination(emblem, &instance, cmd.rect),
            { 0, 0 },
            f32(instance.Rotation),
            rl.WHITE
        )
    }
}

calculateInstanceDestination :: proc(texture: rl.Texture2D, instance: ^pdx.LayerInstance, target: mu.Rect) -> rl.Rectangle {
    width: f32 = f32(target.w) * instance.Scale.X
    height: f32 = f32(target.h) * instance.Scale.Y

    // Default position 0.5 is centered
    // The position then moves the instance by the flag width/height
    x := f32(target.x) + ((f32(target.w) - width) / 2) + (f32(target.w) * (instance.Position.X - 0.5))
    y := f32(target.y) + ((f32(target.h) - height) / 2) + (f32(target.h) * (instance.Position.Y - 0.5))

    return rl.Rectangle{ x, y, width, height }
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
        retrieve, exists := state.NamedColors[color.NamedColor]
        if !exists {
            return
        }
        targetColor := pdx.ToRenderColor(retrieve)
        mapping, ok := checkColorMapping(color.Name, targetColor, sourceColors)
        if ok {
            append(mappings, mapping)
        }
    case ^pdx.FlagColorReference:
        for baseColorVariant in state.Flag.Colors {
            baseColorName: string
            switch baseColor in baseColorVariant {
            case ^pdx.FlagColorRgb:
                if baseColor.Name != color.Reference {
                    continue
                }
                targetColor := pdx.ToRenderColor(baseColor)
                mapping, ok := checkColorMapping(color.Name, targetColor, sourceColors)
                if ok {
                    append(mappings, mapping)
                }
            case ^pdx.FlagColorHsv:
                if baseColor.Name != color.Reference {
                    continue
                }
                targetColor := pdx.ToRenderColor(baseColor)
                mapping, ok := checkColorMapping(color.Name, targetColor, sourceColors)
                if ok {
                    append(mappings, mapping)
                }
            case ^pdx.FlagColorNamed:
                if baseColor.Name != color.Reference {
                    continue
                }
                retrieve, exists := state.NamedColors[baseColor.NamedColor]
                if !exists {
                    return
                }
                targetColor := pdx.ToRenderColor(retrieve)
                mapping, ok := checkColorMapping(color.Name, targetColor, sourceColors)
                if ok {
                    append(mappings, mapping)
                }
            case ^pdx.FlagColorReference:
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
                mu.label(ctx, "Red:");   ui.Slider(ctx, &state.Settings.BackgroundColor.r, 0, 255)
                mu.label(ctx, "Green:"); ui.Slider(ctx, &state.Settings.BackgroundColor.g, 0, 255)
                mu.label(ctx, "Blue:");  ui.Slider(ctx, &state.Settings.BackgroundColor.b, 0, 255)
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

renderFlagDatabase :: proc(ctx: ^mu.Context) {
    if mu.window(ctx, WINDOM_FLAG_DATABASE, {10, 10 + TOOLBAR_HEIGHT, 500, rl.GetScreenHeight() - 20 - TOOLBAR_HEIGHT}, {}) {
        guranteeBounds(ctx)

        mu.layout_row(ctx, {-1, -1}, 15)
        mu.text(ctx, "Search:")
        @static searchBuffer: [128]byte
        @static searchBufferLength: int
        @static search: string
        mu.layout_row(ctx, {-1, -1}, 20)
        if .CHANGE in mu.textbox(ctx, searchBuffer[:], &searchBufferLength) {
            delete(search)
            search = strings.to_lower(string(searchBuffer[:searchBufferLength]))
        }
        mu.layout_row(ctx, {-1, -1}, 5)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])

        options: mu.Options
        if search != "" {
            options = { .EXPANDED }
        }

        for database in state.Databases {
            ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
            if .ACTIVE in mu.header(ctx, database.Settings.Name, options) {

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
        @static search: string
        mu.layout_row(ctx, {-1, -1}, 20)
        if .CHANGE in mu.textbox(ctx, searchBuffer[:], &searchBufferLength) {
            delete(search)
            search = strings.to_lower(string(searchBuffer[:searchBufferLength]))
        }
        mu.layout_row(ctx, {-1, -1}, 5)
        r := mu.layout_next(ctx)
        mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])

        options: mu.Options
        if search != "" {
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
                        if search == "" || strings.contains(name, search) {
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
                        if search == "" || strings.contains(name, search) {
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
                        if search == "" || strings.contains(name, search) {
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
                renderLayerInstances(ctx, layer.Instances)
            case ^pdx.FlagLayerTexturedEmblem:
                mu.layout_row(ctx, {-1, -1}, 20)
                ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
                if .SUBMIT in mu.button(ctx, fmt.tprintf("Textured Emblem: %s", layer.Texture.Name)) {
                    state.SelectedFlagElement = layer
                }
                mu.pop_id(ctx)
                renderLayerInstances(ctx, layer.Instances)
            }
        }
        mu.layout_row(ctx, { -1, -1 }, 20)
    }
}

renderLayerInstances :: proc(ctx: ^mu.Context, instances: [dynamic]^pdx.LayerInstance) {
    for instance, index in instances {
        mu.layout_row(ctx, {-1, -1}, 20)
        buttonRect := mu.layout_next(ctx)

        subRect := mu.Rect{
            buttonRect.x, buttonRect.y, 20, 20
        }

        buttonRect.w = buttonRect.w - ctx.style.indent
        buttonRect.x = buttonRect.x + ctx.style.indent

        mu.layout_set_next(ctx, subRect, false)
        drawTransparentTexture(ctx, TEXTURE_SUB, 20, 20)

        mu.layout_set_next(ctx, buttonRect, false)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in mu.button(ctx, fmt.tprintf("Instance: %i", index + 1)) {
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
            case ^pdx.LayerInstance:
                renderSelectedLayerInstance(ctx, element)
            }
        }
    }
}

renderSelectedFlag :: proc(ctx: ^mu.Context, flag: ^pdx.Flag) {
    ui.DrawAttributeRow(ctx, "Layer", "Pattern & Colors")
    if flag.Pattern.Name != "" {
        ui.DrawAttributeRow(ctx, "Pattern", flag.Pattern.Name)
    } else {
        ui.DrawAttributeRow(ctx, "Pattern", "None")
    }
    for variant, index in flag.Colors {
        switch color in variant {
        case ^pdx.FlagColorNamed:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color, state.NamedColors)
            renderColorButtons(ctx, variant, &state.Flag.Colors, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorRgb:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color)
            renderColorButtons(ctx, variant, &state.Flag.Colors, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorHsv:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color)
            renderColorButtons(ctx, variant, &state.Flag.Colors, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorReference:
            // Should not happen
        }
    }
    renderAddColorButtons(ctx, &flag.Colors)
}

renderSelectedFlagLayerColoredEmblem :: proc(ctx: ^mu.Context, layer: ^pdx.FlagLayerColoredEmblem) {
    ui.DrawAttributeRow(ctx, "Type", "Colored Emblem")
    ui.DrawAttributeRow(ctx, "Texture", layer.Texture.Name)
    ui.DrawAttributeRow(ctx, "Instances", fmt.tprintf("%i", len(layer.Instances)))
    for variant, index in layer.Colors {
        switch color in variant {
        case ^pdx.FlagColorNamed:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color, state.NamedColors)
            renderColorButtons(ctx, variant, &layer.Colors, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorRgb:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color)
            renderColorButtons(ctx, variant, &layer.Colors, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorHsv:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color)
            renderColorButtons(ctx, variant, &layer.Colors, index, buttonEditRect, buttonDelRect)
        case ^pdx.FlagColorReference:
            buttonEditRect, buttonDelRect := ui.DrawAttributeRow(ctx, color.Name, color, state.Flag.Colors[:], state.NamedColors)
            renderColorButtons(ctx, variant, &layer.Colors, index, buttonEditRect, buttonDelRect)
        }
    }
    renderAddColorButtons(ctx, &layer.Colors)
    mu.layout_row(ctx, {-1,-1}, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add reference color") {
        colorName := pdx.GetNextFreeColor(layer.Colors[:])
        if colorName != "" {
            color := pdx.CreateColorReference(colorName, pdx.COLOR_NAMES[0])
            append(&layer.Colors, color)
        }
    }
    mu.pop_id(ctx)

    mu.layout_row(ctx, { -1, -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add new insance") {
        instance := pdx.CreateLayerInstance()
        append(&layer.Instances, instance)
    }
    mu.pop_id(ctx)
}

renderSelectedFlagLayerTexturedEmblem :: proc(ctx: ^mu.Context, layer: ^pdx.FlagLayerTexturedEmblem) {
    ui.DrawAttributeRow(ctx, "Type", "Textured Emblem")
    ui.DrawAttributeRow(ctx, "Texture", layer.Texture.Name)
    ui.DrawAttributeRow(ctx, "Instances", fmt.tprintf("%i", len(layer.Instances)))
    mu.layout_row(ctx, { -1, -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add new insance") {
        instance := pdx.CreateLayerInstance()
        append(&layer.Instances, instance)
    }
    mu.pop_id(ctx)
}

renderSelectedLayerInstance :: proc(ctx: ^mu.Context, instance: ^pdx.LayerInstance) {
    ui.DrawAttributeRow(ctx, "Type", "Layer Instance")
    ui.DrawAttributeRow(ctx, "Rotation", fmt.tprintf("%i", instance.Rotation))
    ui.DrawAttributeRow(ctx, "Scale Width", fmt.tprintf("%.2f", instance.Scale.X))
    ui.DrawAttributeRow(ctx, "Scale Height", fmt.tprintf("%.2f", instance.Scale.Y))
    ui.DrawAttributeRow(ctx, "Position X", fmt.tprintf("%.2f", instance.Position.X))
    ui.DrawAttributeRow(ctx, "Position Y", fmt.tprintf("%.2f", instance.Position.Y))
    mu.layout_row(ctx, { -1, -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Edit insance") {
        state.InstanceEditorInstance = instance
    }
    mu.pop_id(ctx)
}

renderColorButtons :: proc(ctx: ^mu.Context, color: pdx.FlagColorVariant, list: ^[dynamic]pdx.FlagColorVariant, index: int, edit, delete: mu.Rect) {
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
        ordered_remove(list, index)
        pdx.DestroyColor(color)
    }
    mu.pop_id(ctx)
}

renderAddColorButtons :: proc(ctx: ^mu.Context, list: ^[dynamic]pdx.FlagColorVariant) {
    mu.layout_row(ctx, {-1,-1}, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add RGB color") {
        colorName := pdx.GetNextFreeColor(list[:])
        if colorName != "" {
            color := pdx.CreateColorRgb(colorName, 0, 0, 0)
            append(list, color)
        }
    }
    mu.pop_id(ctx)
    mu.layout_row(ctx, {-1,-1}, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add HSV color") {
        colorName := pdx.GetNextFreeColor(list[:])
        if colorName != "" {
            color := pdx.CreateColorHsv(colorName, 0, 0, 0)
            append(list, color)
        }
    }
    mu.pop_id(ctx)
    mu.layout_row(ctx, {-1,-1}, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Add Named color") {
        colorName := pdx.GetNextFreeColor(list[:])
        if colorName != "" {
            color := pdx.CreateColorNamed(colorName, "")
            append(list, color)
        }
    }
    mu.pop_id(ctx)
}

removeLayer :: proc(index: int){
    variant := state.Flag.Layers[index]
    ordered_remove(&state.Flag.Layers, index)
    pdx.DestroyLayer(variant)
}