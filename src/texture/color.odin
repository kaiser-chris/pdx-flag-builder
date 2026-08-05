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
    BlueChannelShadingLocation: c.int,
    UseMaskLocation: c.int,
    MaskTextureLocation: c.int,
    MaskColorLocation: c.int,
    MaskUVOffsetLocation: c.int,
    MaskUVAxisXLocation: c.int,
    MaskUVAxisYLocation: c.int,
}

// Describes how a colored emblem is restricted to the flag pattern pixels
// that match a specific PATTERN_REPLACE_COLORS entry (see coa mask attribute).
MaskParams :: struct {
    Texture: rl.Texture2D,
    Color: rl.Color,
    // Affine mapping from the emblem's own [0,1] texcoord to the pattern
    // texture's [0,1] UV space: maskUV = UVOffset + u*UVAxisX + v*UVAxisY.
    UVOffset: [2]f32,
    UVAxisX: [2]f32,
    UVAxisY: [2]f32,
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
        BlueChannelShadingLocation = rl.GetShaderLocation(shader, "blueChannelShading"),
        UseMaskLocation = rl.GetShaderLocation(shader, "useMask"),
        MaskTextureLocation = rl.GetShaderLocation(shader, "maskTexture"),
        MaskColorLocation = rl.GetShaderLocation(shader, "maskColor"),
        MaskUVOffsetLocation = rl.GetShaderLocation(shader, "maskUVOffset"),
        MaskUVAxisXLocation = rl.GetShaderLocation(shader, "maskUVAxisX"),
        MaskUVAxisYLocation = rl.GetShaderLocation(shader, "maskUVAxisY"),
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
configureMaskShader :: proc(recolor: RecolorShader, mask: Maybe(MaskParams)) {
    params, hasMask := mask.?

    useMaskValue: c.int = hasMask ? 1 : 0
    rl.SetShaderValue(recolor.Shader, recolor.UseMaskLocation, &useMaskValue, .INT)

    if !hasMask {
        return
    }

    rl.SetShaderValueTexture(recolor.Shader, recolor.MaskTextureLocation, params.Texture)

    maskColorValue := colorToVec3(params.Color)
    rl.SetShaderValue(recolor.Shader, recolor.MaskColorLocation, &maskColorValue, .VEC3)

    uvOffsetValue := params.UVOffset
    rl.SetShaderValue(recolor.Shader, recolor.MaskUVOffsetLocation, &uvOffsetValue, .VEC2)

    uvAxisXValue := params.UVAxisX
    rl.SetShaderValue(recolor.Shader, recolor.MaskUVAxisXLocation, &uvAxisXValue, .VEC2)

    uvAxisYValue := params.UVAxisY
    rl.SetShaderValue(recolor.Shader, recolor.MaskUVAxisYLocation, &uvAxisYValue, .VEC2)
}

@private
configureRecolorTextureShader :: proc(
    recolor: RecolorShader,
    mappings: []ColorRecolor,
    tolerance: f32 = 0.8,
    preserveShading: f32 = 0,
    blueChannelShading: bool = false,
    mask: Maybe(MaskParams) = nil,
) {
    configureMaskShader(recolor, mask)

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

    blueChannelShadingValue: c.int = blueChannelShading ? 1 : 0
    rl.SetShaderValue(
        recolor.Shader,
        recolor.BlueChannelShadingLocation,
        &blueChannelShadingValue,
        .INT,
    )
}

DrawRecoloredTexture :: proc(
    texture: rl.Texture2D,
    source, destination: rl.Rectangle,
    colorMappings: []ColorRecolor,
    shader: RecolorShader,
    origin: rl.Vector2 = { 0, 0 },
    rotation: f32 = 0,
    tint: rl.Color = rl.WHITE,
    blueChannelShading: bool = false,
    mask: Maybe(MaskParams) = nil,
) {
    // BeginShaderMode must run before configuring the shader: SetShaderValueTexture
    // (used for the mask sampler) targets whichever shader program is currently
    // bound, unlike SetShaderValue which enables its own shader internally.
    rl.BeginShaderMode(shader.Shader)
    configureRecolorTextureShader(
        shader,
        colorMappings,
        blueChannelShading = blueChannelShading,
        mask = mask,
    )
    rl.DrawTexturePro(texture, source, destination, origin, rotation, tint)
    rl.EndShaderMode()
}