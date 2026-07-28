package ui

import mu "vendor:microui"

Slider :: proc{
    slider_u8,
    slider_f32,
    slider_i32,
}

slider_u8 :: proc(ctx: ^mu.Context, val: ^u8, lo, hi: u8, fmtString := "%.0f", step: f32 = 0) -> (res: mu.Result_Set) {
    mu.push_id(ctx, uintptr(val))

    @static tmp: mu.Real
    tmp = mu.Real(val^)
    res = mu.slider(ctx, &tmp, mu.Real(lo), mu.Real(hi), step, fmtString, {.ALIGN_CENTER})
    val^ = u8(tmp)
    mu.pop_id(ctx)
    return
}

slider_f32 :: proc(ctx: ^mu.Context, val: ^f32, lo, hi: f32, fmtString := "%.0f", step: f32 = 0) -> (res: mu.Result_Set) {
    mu.push_id(ctx, uintptr(val))

    @static tmp: mu.Real
    tmp = mu.Real(val^)
    res = mu.slider(ctx, &tmp, mu.Real(lo), mu.Real(hi), step, fmtString, {.ALIGN_CENTER})
    val^ = f32(tmp)
    mu.pop_id(ctx)
    return
}

slider_i32 :: proc(ctx: ^mu.Context, val: ^i32, lo, hi: i32, fmtString := "%.0f", step: f32 = 0) -> (res: mu.Result_Set) {
    mu.push_id(ctx, uintptr(val))

    @static tmp: mu.Real
    tmp = mu.Real(val^)
    res = mu.slider(ctx, &tmp, mu.Real(lo), mu.Real(hi), step, fmtString, {.ALIGN_CENTER})
    val^ = i32(tmp)
    mu.pop_id(ctx)
    return
}