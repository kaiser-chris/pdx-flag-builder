package ui

import mu "vendor:microui"

ButtonStyle :: enum {
    Default,
    Warning,
    Danger,
}

COLOR_BUTTON_DANGER_BACKGROUND: mu.Color: { 100, 16, 0, 255 }
COLOR_BUTTON_DANGER_BACKGROUND_HOVER: mu.Color: { 167, 16, 0, 255 }
COLOR_BUTTON_DANGER_BACKGROUND_DISABLED: mu.Color: { 60, 16, 0, 255 }
COLOR_BUTTON_DANGER_BORDER: mu.Color: { 62, 16, 0, 255 }

COLOR_BUTTON_WARNING_BACKGROUND: mu.Color: { 122, 102, 0, 255 }
COLOR_BUTTON_WARNING_BACKGROUND_HOVER: mu.Color: { 188, 154, 0, 255 }
COLOR_BUTTON_WARNING_BACKGROUND_DISABLED: mu.Color: { 104, 87, 0, 255 }
COLOR_BUTTON_WARNING_BORDER: mu.Color: { 112, 93, 0, 255 }

Button :: proc{
    ButtonText,
    ButtonIcon,
}

ButtonText :: proc(
    ctx: ^mu.Context,
    label: string,
    style: ButtonStyle = .Default,
    options: mu.Options = {.ALIGN_CENTER},
    enabled := true,
) -> (res: mu.Result_Set) {
    id := mu.get_id(ctx, label)
    rect := mu.layout_next(ctx)
    mu.update_control(ctx, id, rect, options)

    if ctx.mouse_pressed_bits == { .LEFT } && ctx.focus_id == id && enabled {
        res += { .SUBMIT }
    }

    drawButtonBox(ctx, id, style, rect, enabled)

    if len(label) > 0 {
        mu.draw_control_text(ctx, label, rect, .TEXT, options)
    }
    return
}

ButtonIcon :: proc(
    ctx: ^mu.Context,
    icon: IconType,
    width: i32 = 0,
    height: i32 = 0,
    style: ButtonStyle = .Default,
    tint := COLOR_TINT_NONE,
    options: mu.Options = {.ALIGN_CENTER},
    enabled := true,
) -> (res: mu.Result_Set) {
    id := mu.get_id(ctx, uintptr(icon))
    buttonRect := mu.layout_next(ctx)
    mu.update_control(ctx, id, buttonRect, options)

    if ctx.mouse_pressed_bits == { .LEFT } && ctx.focus_id == id && enabled {
        res += { .SUBMIT }
    }

    drawButtonBox(ctx, id, style, buttonRect, enabled)

    iconRect := mu.Rect{
        x = buttonRect.x,
        y = buttonRect.y,
        w = buttonRect.w,
        h = buttonRect.h,
    }

    if width != 0 && width < buttonRect.w {
        iconRect.w = width
    }
    if height != 0 && height < buttonRect.h {
        iconRect.h = height
    }

    if width != 0 || height != 0 {
        if .ALIGN_CENTER in options {
            iconRect.x = buttonRect.x + (buttonRect.w - iconRect.w) / 2
        } else if .ALIGN_RIGHT in options {
            iconRect.x = buttonRect.x + buttonRect.w - iconRect.w
        } else {
            iconRect.x = buttonRect.x
        }
    }

    iconRect.y = buttonRect.y + (buttonRect.h - iconRect.h) / 2
    mu.draw_icon(ctx, mu.Icon(icon), iconRect, { 255, 255, 255, 255 })

    return
}

drawButtonBox :: proc(ctx: ^mu.Context, id: mu.Id, style: ButtonStyle, rect: mu.Rect, enabled: bool) {
    if enabled {
        switch style {
        case .Default:
            if ctx.hover_id == id || ctx.focus_id == id {
                mu.draw_rect(ctx, rect, ctx.style.colors[.BUTTON_HOVER])
            } else {
                mu.draw_rect(ctx, rect, ctx.style.colors[.BUTTON])
            }
            if ctx.style.colors[.BORDER].a != 0 { /* draw border */
                mu.draw_box(ctx, mu.expand_rect(rect, 1), ctx.style.colors[.BORDER])
            }
        case .Warning:
            if ctx.hover_id == id || ctx.focus_id == id {
                mu.draw_rect(ctx, rect, COLOR_BUTTON_WARNING_BACKGROUND_HOVER)
            } else {
                mu.draw_rect(ctx, rect, COLOR_BUTTON_WARNING_BACKGROUND)
            }
            if ctx.style.colors[.BORDER].a != 0 { /* draw border */
                mu.draw_box(ctx, mu.expand_rect(rect, 1), COLOR_BUTTON_WARNING_BORDER)
            }
        case .Danger:
            if ctx.hover_id == id || ctx.focus_id == id {
                mu.draw_rect(ctx, rect, COLOR_BUTTON_DANGER_BACKGROUND_HOVER)
            } else {
                mu.draw_rect(ctx, rect, COLOR_BUTTON_DANGER_BACKGROUND)
            }
            if ctx.style.colors[.BORDER].a != 0 { /* draw border */
                mu.draw_box(ctx, mu.expand_rect(rect, 1), COLOR_BUTTON_DANGER_BORDER)
            }
        }
    } else {
        switch style {
        case .Default:
            color := ctx.style.colors[.BUTTON]
            color.r -= 20
            color.g -= 20
            color.b -= 20
            mu.draw_rect(ctx, rect, color)
            if ctx.style.colors[.BORDER].a != 0 { /* draw border */
                mu.draw_box(ctx, mu.expand_rect(rect, 1), ctx.style.colors[.BORDER])
            }
        case .Warning:
            mu.draw_rect(ctx, rect, COLOR_BUTTON_WARNING_BACKGROUND_DISABLED)
            if ctx.style.colors[.BORDER].a != 0 { /* draw border */
                mu.draw_box(ctx, mu.expand_rect(rect, 1), COLOR_BUTTON_WARNING_BORDER)
            }
        case .Danger:
            mu.draw_rect(ctx, rect, COLOR_BUTTON_DANGER_BACKGROUND_DISABLED)
            if ctx.style.colors[.BORDER].a != 0 { /* draw border */
                mu.draw_box(ctx, mu.expand_rect(rect, 1), COLOR_BUTTON_DANGER_BORDER)
            }
        }
    }
}