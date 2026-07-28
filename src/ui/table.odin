package ui

import "core:strconv"
import mu "vendor:microui"
import pdx "../pdx"
import fmt "core:fmt"

ButtonProc :: #type proc(index: int)

DrawAttributeRow :: proc{
    drawTextAttributeRow,
    drawColorAttributeRow,
    drawNamedColorAttributeRow,
    drawReferenceColorAttributeRow,
}

setupAttributeRow :: proc(ctx: ^mu.Context, label: string, labelSize: f32) -> mu.Rect {
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
        x = rowRect.x + i32(f32(rowRect.w) * labelSize),
        y = rowRect.y,
        w = 1,
        h = rowRect.h,
    }
    mu.draw_rect(ctx, dividerRect, ctx.style.colors[.BORDER])

    labelRect := mu.Rect{
        x = rowRect.x + ctx.style.padding,
        y = rowRect.y + ctx.style.padding,
        w = i32(f32(rowRect.w) * labelSize) - (ctx.style.padding * 2),
        h = rowRect.h - (ctx.style.padding * 2),
    }

    valueRect := mu.Rect{
        x = rowRect.x + labelRect.w + ctx.style.padding + ctx.style.padding + ctx.style.padding,
        y = rowRect.y + ctx.style.padding,
        w = rowRect.w - labelRect.w - (ctx.style.padding * 2),
        h = rowRect.h - (ctx.style.padding * 2),
    }

    mu.draw_control_text(ctx, label, labelRect, .TEXT, {})

    return valueRect
}

drawTextAttributeRow :: proc(ctx: ^mu.Context, label, value: string, labelSize: f32 = 0.3) {
    valueRect := setupAttributeRow(ctx, label, labelSize)

    mu.draw_control_text(ctx, value, valueRect, .TEXT, {})
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing
}

drawColorAttributeRow :: proc(
    ctx: ^mu.Context,
    label: string,
    variant: pdx.FlagColorVariant,
    labelSize: f32 = 0.3
) -> (mu.Rect, mu.Rect) {
    valueRect := setupAttributeRow(ctx, label, labelSize)

    rgbColor: mu.Color
    colorText: string
    switch color in variant {
    case ^pdx.FlagColorRgb:
        colorText = fmt.tprintf("RGB ( %i %i %i )", color.R, color.G, color.B)
        rgbColor = mu.Color{ color.R, color.G, color.B, 255 }
    case ^pdx.FlagColorHsv:
        colorText = fmt.tprintf("HSV ( %.0f %.0f %.0f )", color.H, color.S, color.V)
        tmp := pdx.ToRenderColor(color)
        rgbColor = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
    case ^pdx.FlagColorNamed:
        fmt.eprintfln("named color is used in absolute color attribute row")
    case ^pdx.FlagColorReference:
        fmt.eprintfln("reference color is used in absolute color attribute row")
    }

    colorPreviewRect := mu.Rect{
        x = valueRect.x + ctx.style.spacing,
        y = valueRect.y + ctx.style.spacing,
        h = valueRect.h - (2 * ctx.style.spacing),
        w = valueRect.h - (2 * ctx.style.spacing),
    }

    buttonEditRect := mu.Rect{
        x = valueRect.x + valueRect.w - 25 - ctx.style.padding - ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = 25,
    }
    buttonDeleteRect := mu.Rect{
        x = buttonEditRect.x - 25 - ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = 25,
    }

    colorTextRect := mu.Rect{
        x = colorPreviewRect.x + colorPreviewRect.w + ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = buttonDeleteRect.x - ctx.style.spacing - (colorPreviewRect.x + colorPreviewRect.w + ctx.style.spacing),
    }
    mu.draw_control_text(ctx, colorText, colorTextRect, .TEXT, {.ALIGN_CENTER})
    mu.draw_rect(ctx, colorPreviewRect, rgbColor)
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing

    return buttonEditRect, buttonDeleteRect
}

drawNamedColorAttributeRow :: proc(
    ctx: ^mu.Context,
    label: string,
    color: ^pdx.FlagColorNamed,
    namedColors: map[string]pdx.FlagColorVariant,
    labelSize: f32 = 0.3
) -> (mu.Rect, mu.Rect) {
    valueRect := setupAttributeRow(ctx, label, labelSize)

    rgbColor: mu.Color
    colorText: string
    if color.NamedColor == "" {
        colorText = "Named ( Empty )"
    } else {
        colorText = fmt.tprintf("Named ( %s )", color.NamedColor)
        reference, ok := namedColors[color.NamedColor]
        switch referenceColor in reference {
        case ^pdx.FlagColorRgb:
            rgbColor = mu.Color{ referenceColor.R, referenceColor.G, referenceColor.B, 255 }
        case ^pdx.FlagColorHsv:
            tmp := pdx.ToRenderColor(referenceColor)
            rgbColor = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
        case ^pdx.FlagColorNamed:
            fmt.eprintfln("named color is referencing another named color")
        case ^pdx.FlagColorReference:
            fmt.eprintfln("named color is referencing a reference color")
        }
    }

    colorPreviewRect := mu.Rect{
        x = valueRect.x + ctx.style.spacing,
        y = valueRect.y + ctx.style.spacing,
        h = valueRect.h - (2 * ctx.style.spacing),
        w = valueRect.h - (2 * ctx.style.spacing),
    }

    buttonEditRect := mu.Rect{
        x = valueRect.x + valueRect.w - 25 - ctx.style.padding - ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = 25,
    }
    buttonDeleteRect := mu.Rect{
        x = buttonEditRect.x - 25 - ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = 25,
    }

    colorTextRect := mu.Rect{
        x = colorPreviewRect.x + colorPreviewRect.w + ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = buttonDeleteRect.x - ctx.style.spacing - (colorPreviewRect.x + colorPreviewRect.w + ctx.style.spacing),
    }
    mu.draw_control_text(ctx, colorText, colorTextRect, .TEXT, {.ALIGN_CENTER})
    mu.draw_rect(ctx, colorPreviewRect, rgbColor)
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing

    return buttonEditRect, buttonDeleteRect
}

drawReferenceColorAttributeRow :: proc(
    ctx: ^mu.Context,
    label: string,
    color: ^pdx.FlagColorReference,
    baseColors: []pdx.FlagColorVariant,
    namedColors: map[string]pdx.FlagColorVariant,
    labelSize: f32 = 0.3
) -> (mu.Rect, mu.Rect) {
    valueRect := setupAttributeRow(ctx, label, labelSize)

    colorValue: mu.Color
    for baseColor in baseColors {
        baseColorValue: mu.Color
        baseColorName: string
        switch referenceColor in baseColor {
        case ^pdx.FlagColorRgb:
            baseColorName = referenceColor.Name
            baseColorValue = mu.Color{ referenceColor.R, referenceColor.G, referenceColor.B, 255 }
        case ^pdx.FlagColorHsv:
            baseColorName = referenceColor.Name
            tmp := pdx.ToRenderColor(referenceColor)
            baseColorValue = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
        case ^pdx.FlagColorNamed:
            baseColorName = referenceColor.Name
            reference, ok := namedColors[referenceColor.NamedColor]
            switch referenceColor in reference {
            case ^pdx.FlagColorRgb:
                baseColorValue = mu.Color{ referenceColor.R, referenceColor.G, referenceColor.B, 255 }
            case ^pdx.FlagColorHsv:
                tmp := pdx.ToRenderColor(referenceColor)
                baseColorValue = mu.Color{ tmp[0], tmp[1], tmp[2], 255 }
            case ^pdx.FlagColorNamed:
                fmt.eprintfln("named color is referencing another named color")
            case ^pdx.FlagColorReference:
                fmt.eprintfln("named color is referencing a reference color")
            }
        case ^pdx.FlagColorReference:
            fmt.eprintfln("reference color is referencing a reference color")
        }
        if baseColorName == color.Reference {
            colorValue = baseColorValue
        }
    }

    colorText: string
    if color.Reference == "" {
        colorText = "Reference ( Empty )"
    } else {
        colorText = fmt.tprintf("Reference ( %s )", color.Reference)
    }

    colorPreviewRect := mu.Rect{
        x = valueRect.x + ctx.style.spacing,
        y = valueRect.y + ctx.style.spacing,
        h = valueRect.h - (2 * ctx.style.spacing),
        w = valueRect.h - (2 * ctx.style.spacing),
    }

    buttonEditRect := mu.Rect{
        x = valueRect.x + valueRect.w - 25 - ctx.style.padding - ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = 25,
    }
    buttonDeleteRect := mu.Rect{
        x = buttonEditRect.x - 25 - ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = 25,
    }

    colorTextRect := mu.Rect{
        x = colorPreviewRect.x + colorPreviewRect.w + ctx.style.spacing,
        y = valueRect.y,
        h = valueRect.h,
        w = buttonDeleteRect.x - ctx.style.spacing - (colorPreviewRect.x + colorPreviewRect.w + ctx.style.spacing),
    }
    mu.draw_control_text(ctx, colorText, colorTextRect, .TEXT, {.ALIGN_CENTER})
    mu.draw_rect(ctx, colorPreviewRect, colorValue)
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing

    return buttonEditRect, buttonDeleteRect
}

DrawAttributeRowTextbox :: proc(
    ctx: ^mu.Context,
    label: string,
    buf: []byte,
    bufLen: ^int,
    labelSize: f32 = 0.3
) -> mu.Result_Set {
    valueRect := setupAttributeRow(ctx, label, labelSize)
    valueRect.w = valueRect.w - (2 * ctx.style.padding)

    id := mu.get_id(ctx, uintptr(&buf[0]))
    result := mu.textbox_raw(ctx, buf, bufLen, id, valueRect)
    mu.layout_end_column(ctx)

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing

    return result
}

DrawAttributeRowNumberTextbox :: proc{
    drawAttributeRowNumberTextbox_f32,
    drawAttributeRowNumberTextbox_i32,
}

drawAttributeRowNumberTextbox_f32 :: proc(
    ctx: ^mu.Context,
    label: string,
    val: ^f32,
    buf: []byte,
    bufLen: ^int,
    hi, lo: f32,
    fmtString := "%.0f",
    labelSize: f32 = 0.3
) {
    org := fmt.tprintf(fmtString, val^)
    bufLen^ = len(org)
    copy(buf[:], org)

    valueRect := setupAttributeRow(ctx, label, labelSize)
    mu.layout_end_column(ctx)
    valueRect.w = valueRect.w - (2 * ctx.style.padding)

    id := mu.get_id(ctx, uintptr(&buf[0]))
    if .CHANGE in mu.textbox_raw(ctx, buf, bufLen, id, valueRect) {
        text := string(buf[:bufLen^])
        if text == "" {
            val^ = 0
            return
        }
        tmp, ok := strconv.parse_f32(string(buf[:bufLen^]))
        if !ok {
            return
        }
        if tmp > hi {
            val^ = hi
            return
        }
        if tmp < lo {
            val^ = lo
            return
        }
        val^ = tmp
    }

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing
}

drawAttributeRowNumberTextbox_i32 :: proc(
    ctx: ^mu.Context,
    label: string,
    val: ^i32,
    buf: []byte,
    bufLen: ^int,
    hi, lo: i32,
    fmtString := "%i",
    labelSize: f32 = 0.3
) {
    org := fmt.tprintf(fmtString, val^)
    bufLen^ = len(org)
    copy(buf[:], org)

    valueRect := setupAttributeRow(ctx, label, labelSize)
    mu.layout_end_column(ctx)
    valueRect.w = valueRect.w - (2 * ctx.style.padding)

    id := mu.get_id(ctx, uintptr(&buf[0]))
    if .CHANGE in mu.textbox_raw(ctx, buf, bufLen, id, valueRect) {
        text := string(buf[:bufLen^])
        if text == "" {
            val^ = 0
            return
        }
        tmp, ok := strconv.parse_i64(string(buf[:bufLen^]))
        if !ok {
            return
        }
        if i32(tmp) > hi {
            val^ = hi
            return
        }
        if i32(tmp) < lo {
            val^ = lo
            return
        }
        val^ = i32(tmp)
    }

    layout := mu.get_layout(ctx)
    layout.position.y += ctx.style.spacing
    layout.next_row += ctx.style.spacing
}