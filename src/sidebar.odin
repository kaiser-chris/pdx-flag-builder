package pdx_flag_builder

import "core:fmt"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "ui"
import "pdx"

SIDEBAR_HANDLE_WIDTH:i32: 10
SIDEBAR_MIN_WIDTH: i32: 350
SIDEBAR_MAX_WIDTH: i32: 600

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
                renderLayerButton(ctx, layer.Texture.Name, index, layer, &state.Flag.Layers)
                renderLayerInstances(ctx, &layer.Instances)
            case ^pdx.FlagLayerTexturedEmblem:
                renderLayerButton(ctx, layer.Texture.Name, index, layer, &state.Flag.Layers)
                renderLayerInstances(ctx, &layer.Instances)
            case ^pdx.FlagLayerSub:
                renderLayerButton(ctx, layer.Parent, index, layer, &state.Flag.Layers)
                renderLayerSubInstances(ctx, layer.Instances)
            }
        }
        mu.layout_row(ctx, { -1, -1 }, 20)
    }
}

renderLayerButton :: proc(ctx: ^mu.Context, name: string, index: int, layer: SelectedFlagElement, list: ^$D/[dynamic]$T) {
    buttonText := name
    #partial switch type in layer {
    case ^pdx.FlagLayerColoredEmblem:
        buttonText = fmt.tprintf("Colored Emblem: %s", name)
    case ^pdx.FlagLayerTexturedEmblem:
        buttonText = fmt.tprintf("Textured Emblem: %s", name)
    case ^pdx.FlagLayerSub:
        buttonText = fmt.tprintf("Sub Flag: %s", name)
    case ^pdx.LayerInstance:
        buttonText = fmt.tprintf(
            "Instance: %i, s{{ %.2f %.2f }}, p{{ %.2f %.2f }}",
            type.Rotation,
            type.Scale.X, type.Scale.Y,
            type.Position.X, type.Position.Y
        )
    case ^pdx.LayerInstanceSub:
        buttonText = "Instance"
    }

    mu.layout_row(ctx, { 15, -73, 20, 20, 20 }, 20)
    // Sub arrow icon
    {
        ui.Icon(ctx, .Sub)
    }
    // Main button
    {
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        if .SUBMIT in mu.button(ctx, buttonText) {
            state.SelectedFlagElement = layer
        }
        mu.pop_id(ctx)
    }
    // Arrow Up button
    {
        enabled := len(list) > 1 && index != 0
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        r := mu.layout_next(ctx)
        mu.layout_set_next(ctx, r, false)
        if .SUBMIT in ui.Button(ctx, ui.IconType.ArrowUp, r.h - 4, r.h - 4, enabled = enabled) {
            moveUp(list, index)
        }
        mu.pop_id(ctx)
    }
    // Arrow Down button
    {
        enabled := len(list) > 1 && index + 1 < len(list)
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        r := mu.layout_next(ctx)
        mu.layout_set_next(ctx, r, false)
        if .SUBMIT in ui.Button(ctx, ui.IconType.ArrowDown, r.h - 4, r.h - 4, enabled = enabled) {
            moveDown(list, index)
        }
        mu.pop_id(ctx)
    }
    // Delete button
    {
        ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
        r := mu.layout_next(ctx)
        mu.layout_set_next(ctx, r, false)
        if .SUBMIT in ui.Button(ctx, ui.IconType.Delete, r.h - 4, r.h - 4, ui.ButtonStyle.Danger) {
            #partial switch type in layer {
            case ^pdx.FlagLayerColoredEmblem:
                removeLayer(type)
                state.SelectedFlagElement = nil
            case ^pdx.FlagLayerTexturedEmblem:
                removeLayer(type)
                state.SelectedFlagElement = nil
            case ^pdx.FlagLayerSub:
                removeLayer(type)
                state.SelectedFlagElement = nil
            case ^pdx.LayerInstance:
                removeInstance(type)
                state.SelectedFlagElement = nil
            case ^pdx.LayerInstanceSub:
                removeInstance(type)
                state.SelectedFlagElement = nil
            }
        }
        mu.pop_id(ctx)
    }
}

moveUp :: proc(list: ^$D/[dynamic]$T, i: int) {
    if i <= 0 || i >= len(list^) {
        return
    }
    list^[i], list^[i-1] = list^[i-1], list^[i]
}
moveDown :: proc(list: ^$D/[dynamic]$T, i: int) {
    if i < 0 || i >= len(list^) - 1 {
        return
    }
    list^[i], list^[i+1] = list^[i+1], list^[i]
}

renderLayerInstances :: proc(ctx: ^mu.Context, instances: ^[dynamic]^pdx.LayerInstance) {
    for instance, index in instances {
        mu.layout_row(ctx, { 15, -1 }, 20)
        mu.layout_next(ctx)
        mu.layout_begin_column(ctx)
        renderLayerButton(ctx, "", index, instance, instances)
        mu.layout_end_column(ctx)
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
    if .SUBMIT in ButtonTextConfirm(ctx, "Clear flag", "Clear Flag", POPUP_MESSAGE_CLEAR_FLAG, ui.ButtonStyle.Danger) {
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
    if .SUBMIT in ButtonTextConfirm(ctx, "Delete layer", "Delete Layer", POPUP_MESSAGE_DELETE_LAYER, ui.ButtonStyle.Danger) {
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
    if .SUBMIT in ButtonTextConfirm(ctx, "Delete layer", "Delete Layer", POPUP_MESSAGE_DELETE_LAYER, ui.ButtonStyle.Danger) {
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
    if .SUBMIT in ButtonTextConfirm(ctx, "Delete layer", "Delete Layer", POPUP_MESSAGE_DELETE_LAYER, ui.ButtonStyle.Danger) {
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
        ui.DrawAttributeRowNumberTextbox(ctx, "Rotation (ALT + LEFT/RIGHT)", &type.Rotation, rotationBuf[:], &rotationBufLen, 360, 0, "%i", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Width (CTRL + LEFT/RIGHT)", &type.Scale.X, scaleXBuf[:], &scaleXBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Height (CTRL + UP/DOWN)", &type.Scale.Y, scaleYBuf[:], &scaleYBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Position X (LEFT/RIGHT)", &type.Position.X, positionXBuf[:], &positionXBufLen, pdx.MAX_POSITION, pdx.MIN_POSITION, "%.3f", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Position Y (UP/DOWN)", &type.Position.Y, positionYBuf[:], &positionYBufLen, pdx.MAX_POSITION, pdx.MIN_POSITION, "%.3f", 0.7)

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
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Width (CTRL + LEFT/RIGHT)", &type.Scale.X, scaleXBuf[:], &scaleXBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Scale Height (CTRL + UP/DOWN)", &type.Scale.Y, scaleYBuf[:], &scaleYBufLen, pdx.MAX_SCALE, pdx.MIN_SCALE, "%.3f", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Offset X (LEFT/RIGHT)", &type.Offset.X, positionXBuf[:], &positionXBufLen, pdx.MAX_OFFSET, pdx.MIN_OFFSET, "%.3f", 0.7)
        ui.DrawAttributeRowNumberTextbox(ctx, "Offset Y (UP/DOWN)", &type.Offset.Y, positionYBuf[:], &positionYBufLen, pdx.MAX_OFFSET, pdx.MIN_OFFSET, "%.3f", 0.7)

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
    mu.layout_row(ctx, { -1 }, 20)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in mu.button(ctx, "Edit instance") {
        state.InstanceEditorInstance = instance
    }
    mu.pop_id(ctx)
    ui.SetButtonIdentifier(ctx, &state.ButtonIdentifier)
    if .SUBMIT in ButtonTextConfirm(ctx, "Delete instance", "Delete Instance", POPUP_MESSAGE_DELETE_LAYER, ui.ButtonStyle.Danger) {
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
    if .SUBMIT in ButtonIconConfirm(ctx, ui.IconType.Delete, "Delete Color", POPUP_MESSAGE_DELETE_COLOR, delete.h - 4, delete.h - 4, ui.ButtonStyle.Danger) {
        if color == state.ColorPickerColor {
            state.ColorPickerColor = nil
        }
        pdx.DestroyFlagColor(color)
        ordered_remove(list, index)
    }
    mu.pop_id(ctx)
}