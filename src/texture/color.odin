package texture

import rl "vendor:raylib"
import "core:c"
import "core:strings"

MAX_RECOLORS :: 8

ColorRecolor :: struct {
    source:      rl.Color,
    replacement: rl.Color,
}

RecolorShader :: struct {
    Shader: rl.Shader,
    RecolorCountLocation: c.int,
    SourceColorsLocation: c.int,
    ReplacementColorsLocation: c.int,
    ToleranceLocation: c.int,
    PreserveShadingLocation: c.int,
}

LoadRecolorShader :: proc(path: string) -> RecolorShader {
    cPath := strings.clone_to_cstring(path)
    defer delete(cPath)
    shader := rl.LoadShader(nil, cPath)

    return RecolorShader{
        Shader = shader,
        RecolorCountLocation = rl.GetShaderLocation(shader, "recolorCount"),
        SourceColorsLocation = rl.GetShaderLocation(shader, "sourceColors[0]"),
        ReplacementColorsLocation = rl.GetShaderLocation(shader, "replacementColors[0]"),
        ToleranceLocation = rl.GetShaderLocation(shader, "tolerance"),
        PreserveShadingLocation = rl.GetShaderLocation(shader, "preserveShading"),
    }
}

@private
colorToVec3 :: proc(color: rl.Color) -> [3]f32 {
    return {
        f32(color.r) / 255.0,
        f32(color.g) / 255.0,
        f32(color.b) / 255.0,
    }
}

@private
configureRecolorTextureShader :: proc(
    recolor: RecolorShader,
    mappings: []ColorRecolor,
    tolerance: f32 = 0.8,
    preserveShading: f32 = 0,
) {
    count := min(len(mappings), MAX_RECOLORS)

    if count <= 0 {
        zero: c.int = 0
        rl.SetShaderValue(
            recolor.Shader,
            recolor.RecolorCountLocation,
            &zero,
            .INT,
        )
        return
    }

    // Contiguous [3]f32 data matches GLSL vec3 array memory layout
    // expected by SetShaderValueV(..., .VEC3, count).
    sourceColors: [MAX_RECOLORS][3]f32
    replacementColors: [MAX_RECOLORS][3]f32

    for i in 0..<count {
        sourceColors[i] = colorToVec3(mappings[i].source)
        replacementColors[i] = colorToVec3(mappings[i].replacement)
    }

    shaderCount := c.int(count)
    toleranceValue := tolerance
    preserveShadingValue := preserveShading

    rl.SetShaderValue(
        recolor.Shader,
        recolor.RecolorCountLocation,
        &shaderCount,
        .INT,
    )

    rl.SetShaderValueV(
        recolor.Shader,
        recolor.SourceColorsLocation,
        raw_data(sourceColors[:]),
        .VEC3,
        c.int(count),
    )

    rl.SetShaderValueV(
        recolor.Shader,
        recolor.ReplacementColorsLocation,
        raw_data(replacementColors[:]),
        .VEC3,
        c.int(count),
    )

    rl.SetShaderValue(
        recolor.Shader,
        recolor.ToleranceLocation,
        &toleranceValue,
        .FLOAT,
    )

    rl.SetShaderValue(
        recolor.Shader,
        recolor.PreserveShadingLocation,
        &preserveShadingValue,
        .FLOAT,
    )
}

DrawRecoloredTexture :: proc(
    texture: rl.Texture2D,
    source, destination: rl.Rectangle,
    colorMappings: []ColorRecolor,
    shader: RecolorShader,
    origin: rl.Vector2 = { 0, 0 },
    rotation: f32 = 0,
    tint: rl.Color = rl.WHITE
) {
    configureRecolorTextureShader(
        shader,
        colorMappings,
    )
    rl.BeginShaderMode(shader.Shader)
    rl.DrawTexturePro(texture, source, destination, origin, rotation, tint)
    rl.EndShaderMode()
}