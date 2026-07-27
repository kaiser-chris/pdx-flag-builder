package ui

import "core:strconv"
import "core:fmt"
import mu "vendor:microui"

SetButtonIdentifier :: proc(ctx: ^mu.Context, counter: ^i32) {
    mu.push_id(ctx, fmt.tprintf("%i", counter^))
    counter^ += 1
}

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

NumberTextbox :: proc{
    numberTextbox_u8,
    numberTextbox_i32,
    numberTextbox_f32,
}

numberTextbox_u8 :: proc(ctx: ^mu.Context, val: ^u8, buf: ^[16]byte, bufLen: ^int, hi, lo: u8) {
    org := fmt.tprintf("%i", val^)
    bufLen^ = len(org)
    copy(buf[:], org)
    if .CHANGE in mu.textbox(ctx, buf[:], bufLen) {
        text := string(buf[:bufLen^])
        if text == "" {
            val^ = 0
            return
        }
        tmp, ok := strconv.parse_u64(string(buf[:bufLen^]))
        if !ok {
            return
        }
        if tmp > u64(hi) {
            val^ = hi
            return
        }
        if tmp < u64(lo) {
            val^ = lo
            return
        }
        val^ = u8(tmp)
    }
}

numberTextbox_i32 :: proc(ctx: ^mu.Context, val: ^i32, buf: ^[16]byte, bufLen: ^int, hi, lo: i32) {
    org := fmt.tprintf("%i", val^)
    bufLen^ = len(org)
    copy(buf[:], org)
    if .CHANGE in mu.textbox(ctx, buf[:], bufLen) {
        text := string(buf[:bufLen^])
        if text == "" {
            val^ = 0
            return
        }
        bigInt, ok := strconv.parse_i64(string(buf[:bufLen^]))
        if !ok {
            return
        }
        smallInt := i32(bigInt)
        if smallInt > hi {
            val^ = hi
            return
        }
        if smallInt < lo {
            val^ = lo
            return
        }
        val^ = smallInt
    }
}

numberTextbox_f32 :: proc(ctx: ^mu.Context, val: ^f32, buf: ^[16]byte, bufLen: ^int, hi, lo: f32, fmtString := "%.0f") {
    org := fmt.tprintf(fmtString, val^)
    bufLen^ = len(org)
    copy(buf[:], org)
    if .CHANGE in mu.textbox(ctx, buf[:], bufLen) {
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
}