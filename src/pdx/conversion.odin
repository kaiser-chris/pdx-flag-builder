package pdx

import "core:math"
import rl "vendor:raylib"

clamp_f32 :: proc(x, low, high: f32) -> f32 {
    return max(low, min(x, high))
}

hsvToRgb :: proc(
    hue_degrees: f32,
    saturation: f32,
    value: f32,
    alpha: u8 = 255,
) -> rl.Color {
    // Wrap hue into the range [0, 360).
    h := math.mod_f32(hue_degrees, 360.0)
    if h < 0.0 {
        h += 360.0
    }

    s := clamp_f32(saturation, 0.0, 1.0)
    v := clamp_f32(value, 0.0, 1.0)

    chroma := v * s
    hue_sector := h / 60.0
    x := chroma * (1.0 - abs(math.mod_f32(hue_sector, 2.0) - 1.0))
    m := v - chroma

    r1, g1, b1: f32

    switch int(hue_sector) {
    case 0:
        r1, g1, b1 = chroma, x, 0.0
    case 1:
        r1, g1, b1 = x, chroma, 0.0
    case 2:
        r1, g1, b1 = 0.0, chroma, x
    case 3:
        r1, g1, b1 = 0.0, x, chroma
    case 4:
        r1, g1, b1 = x, 0.0, chroma
    case:
    // Sector 5, including a wrapped 360-degree hue.
        r1, g1, b1 = chroma, 0.0, x
    }

    return rl.Color{
        u8(clamp_f32((r1 + m) * 255.0, 0.0, 255.0)),
        u8(clamp_f32((g1 + m) * 255.0, 0.0, 255.0)),
        u8(clamp_f32((b1 + m) * 255.0, 0.0, 255.0)),
        alpha,
    }
}

ToRenderColor :: proc{
    RgbToRenderColor,
    HsvToRenderColor,
    VariantToRenderColor,
}

HsvToRenderColor :: proc(color: ^FlagColorHsv) -> rl.Color {
    return hsvToRgb(
        color.H,
        color.S / 100,
        color.V / 100,
    )
}

RgbToRenderColor :: proc(color: ^FlagColorRgb) -> rl.Color {
    return rl.Color{
        color.R,
        color.G,
        color.B,
        255,
    }
}

VariantToRenderColor :: proc(variant: FlagColorVariant) -> rl.Color {
    switch color in variant {
    case ^FlagColorRgb:
        return RgbToRenderColor(color)
    case ^FlagColorHsv:
        return HsvToRenderColor(color)
    case ^FlagColorNamed:
        return rl.Color{}
    }
    return rl.Color{}
}