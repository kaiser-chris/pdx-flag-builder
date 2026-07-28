package ui

import "core:strconv"
import "core:fmt"
import mu "vendor:microui"

NumberTextbox :: proc{
    numberTextbox_u8,
    numberTextbox_i32,
    numberTextbox_f32,
}

numberTextbox_u8 :: proc(ctx: ^mu.Context, val: ^u8, buf: []byte, bufLen: ^int, hi, lo: u8) {
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

numberTextbox_i32 :: proc(ctx: ^mu.Context, val: ^i32, buf: []byte, bufLen: ^int, hi, lo: i32) {
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

numberTextbox_f32 :: proc(ctx: ^mu.Context, val: ^f32, buf: []byte, bufLen: ^int, hi, lo: f32, fmtString := "%.0f") {
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