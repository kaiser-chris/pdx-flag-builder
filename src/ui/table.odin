package ui

import mu "vendor:microui"
import fb "../pdx"


drawTextAttributeRow :: proc(ctx: ^mu.Context, name, value: string, nameSize: f32 = 0.3) {
    mu.layout_row(ctx, { -1, -1 }, 20)
    window := mu.get_current_container(ctx)

    originalRect := mu.layout_next(ctx)
    rowRect := mu.Rect{
        x = window.rect.x,
        y = originalRect.y - ctx.style.padding,
        w = window.rect.w,
        h = originalRect.h + (ctx.style.padding * 2),
    }
    mu.layout_set_next(ctx, rowRect, false)
    mu.layout_begin_column(ctx)
    mu.draw_rect(ctx, rowRect, ctx.style.colors[.SELECTION_BG])
    mu.draw_box(ctx, rowRect, ctx.style.colors[.BORDER])

    dividerRect := mu.Rect{
        x = rowRect.x + i32(f32(rowRect.w) * nameSize),
        y = rowRect.y,
        w = 1,
        h = rowRect.h,
    }
    mu.draw_rect(ctx, dividerRect, ctx.style.colors[.BORDER])

    nameRect := mu.Rect{
        x = rowRect.x + ctx.style.padding,
        y = rowRect.y + ctx.style.padding,
        w = i32(f32(rowRect.w) * nameSize) - (ctx.style.padding * 2),
        h = rowRect.h - (ctx.style.padding * 2),
    }

    valueRect := mu.Rect{
        x = rowRect.x + nameRect.w + ctx.style.padding + ctx.style.padding + ctx.style.padding,
        y = rowRect.y + ctx.style.padding,
        w = rowRect.w - nameRect.w - (ctx.style.padding * 2),
        h = rowRect.h - (ctx.style.padding * 2),
    }

    mu.draw_control_text(ctx, name, nameRect, .TEXT, {})

    mu.draw_control_text(ctx, value, valueRect, .TEXT, {})
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing
}

drawNamedColorAttributeRow :: proc(ctx: ^mu.Context, name: string, color: ^fb.FlagColorNamed, nameSize: f32 = 0.3) {
    mu.layout_row(ctx, { -1, -1 }, 20)
    window := mu.get_current_container(ctx)

    originalRect := mu.layout_next(ctx)
    rowRect := mu.Rect{
        x = window.rect.x,
        y = originalRect.y - ctx.style.padding,
        w = window.rect.w,
        h = originalRect.h + (ctx.style.padding * 2),
    }
    mu.layout_set_next(ctx, rowRect, false)
    mu.layout_begin_column(ctx)
    mu.draw_rect(ctx, rowRect, ctx.style.colors[.SELECTION_BG])
    mu.draw_box(ctx, rowRect, ctx.style.colors[.BORDER])

    dividerRect := mu.Rect{
        x = rowRect.x + i32(f32(rowRect.w) * nameSize),
        y = rowRect.y,
        w = 1,
        h = rowRect.h,
    }
    mu.draw_rect(ctx, dividerRect, ctx.style.colors[.BORDER])

    nameRect := mu.Rect{
        x = rowRect.x + ctx.style.padding,
        y = rowRect.y + ctx.style.padding,
        w = i32(f32(rowRect.w) * nameSize) - (ctx.style.padding * 2),
        h = rowRect.h - (ctx.style.padding * 2),
    }

    valueRect := mu.Rect{
        x = rowRect.x + nameRect.w + ctx.style.padding + ctx.style.padding + ctx.style.padding,
        y = rowRect.y + ctx.style.padding,
        w = rowRect.w - nameRect.w - (ctx.style.padding * 2),
        h = rowRect.h - (ctx.style.padding * 2),
    }

    mu.draw_control_text(ctx, name, nameRect, .TEXT, {})

    mu.draw_control_text(ctx, color.NamedColor, valueRect, .TEXT, {})
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing
}