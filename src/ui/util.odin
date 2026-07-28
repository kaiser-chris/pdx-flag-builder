package ui

import "core:fmt"
import mu "vendor:microui"

SetButtonIdentifier :: proc(ctx: ^mu.Context, counter: ^i32) {
    mu.push_id(ctx, fmt.tprintf("%i", counter^))
    counter^ += 1
}