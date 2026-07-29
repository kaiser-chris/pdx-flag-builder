package ui

import mu "vendor:microui"

ButtonStyle :: enum {
    Default,
    Danger,
}

COLOR_BUTTON_DANGER_BACKGROUND: mu.Color: { 100, 16, 0, 255 }
COLOR_BUTTON_DANGER_BACKGROUND_HOVER: mu.Color: { 167, 16, 0, 255 }
COLOR_BUTTON_DANGER_BORDER: mu.Color: { 62, 16, 0, 255 }

Button :: proc(ctx: ^mu.Context, label: string, style: ButtonStyle = .Default, options: mu.Options = {.ALIGN_CENTER}) -> (res: mu.Result_Set) {
    id := mu.get_id(ctx, label)
    rect := mu.layout_next(ctx)
    mu.update_control(ctx, id, rect, options)

    if ctx.mouse_pressed_bits == { .LEFT } && ctx.focus_id == id {
        res += { .SUBMIT }
    }

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

    if len(label) > 0 {
        mu.draw_control_text(ctx, label, rect, .TEXT, options)
    }
    return
}