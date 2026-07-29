package ui

import mu "vendor:microui"

IconType :: enum u32 {
    Sub    = 10000,
    Edit   = 10001,
    Delete = 10002,
}

Icon :: proc(ctx: ^mu.Context, icon: IconType, tint := mu.Color{ 255, 255, 255, 255 }) {
    rect := mu.layout_next(ctx)
    mu.draw_icon(ctx, mu.Icon(icon), rect, tint)
}