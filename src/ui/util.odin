package ui

import "core:fmt"
import mu "vendor:microui"

SetButtonIdentifier :: proc(ctx: ^mu.Context, counter: ^i32) {
    mu.push_id(ctx, fmt.tprintf("%i", counter^))
    counter^ += 1
}

ColorSlider :: proc{
    colorSliderRgb,
    colorSliderHsv,
}

colorSliderRgb :: proc(ctx: ^mu.Context, val: ^u8, lo, hi: u8) -> (res: mu.Result_Set) {
    mu.push_id(ctx, uintptr(val))

    @static tmp: mu.Real
    tmp = mu.Real(val^)
    res = mu.slider(ctx, &tmp, mu.Real(lo), mu.Real(hi), 0, "%.0f", {.ALIGN_CENTER})
    val^ = u8(tmp)
    mu.pop_id(ctx)
    return
}

colorSliderHsv :: proc(ctx: ^mu.Context, val: ^f32, lo, hi: u8) -> (res: mu.Result_Set) {
    mu.push_id(ctx, uintptr(val))

    @static tmp: mu.Real
    tmp = mu.Real(val^)
    res = mu.slider(ctx, &tmp, mu.Real(lo), mu.Real(hi), 0, "%.2f", {.ALIGN_CENTER})
    val^ = f32(tmp)
    mu.pop_id(ctx)
    return
}