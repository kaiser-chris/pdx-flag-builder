package pdx

import strings "core:strings"

DEFAULT_SCALE: LayerVector: { 1, 1 }
DEFAULT_POSITION: LayerVector: { 0.5, 0.5 }

MIN_SCALE: f32: 0
MAX_SCALE: f32: 10

MIN_POSITION: f32: -4
MAX_POSITION: f32: 5

Flag :: struct {
    Name: string,
    Pattern: FlagTexture,
    Colors: [dynamic]FlagColorVariant,
    Layers: [dynamic]FlagLayerVariant,
}

CreateFlag :: proc(name: string = "") -> Flag {
    flag := Flag{
        Name = strings.clone(name),
        Colors = make([dynamic]FlagColorVariant),
        Layers = make([dynamic]FlagLayerVariant),
    }

    return flag
}
DestroyFlag :: proc(flag: Flag) {
    DestroyFlagTexture(flag.Pattern)
    for _, index in flag.Colors {
        DestroyColor(flag.Colors[index])
    }
    delete(flag.Colors)
    for _, index in flag.Layers {
        DestroyLayer(flag.Layers[index])
    }
    delete(flag.Layers)
    delete(flag.Name)
}

FlagTexture :: struct {
    Name: string,
    Path: string,
}

CreateFlagTexture :: proc(name, path: string) -> FlagTexture {
    return FlagTexture{
        Name = strings.clone(name),
        Path = strings.clone(path),
    }
}
DestroyFlagTexture :: proc(texture: FlagTexture) {
    delete(texture.Name)
    delete(texture.Path)
}

FlagLayerVariant :: union {
    ^FlagLayerColoredEmblem,
    ^FlagLayerTexturedEmblem,
}
FlagLayer :: struct {
    Texture: FlagTexture,
    Instances: [dynamic]^LayerInstance,
}
FlagLayerColoredEmblem :: struct {
    using Layer: FlagLayer,
    Colors: [dynamic]FlagColorVariant,
}
FlagLayerTexturedEmblem :: struct {
    using Layer: FlagLayer,
}

CreateLayerTexturedEmblem :: proc(name, path: string) -> ^FlagLayerTexturedEmblem {
    return new_clone(FlagLayerTexturedEmblem{
        Texture = CreateFlagTexture(name, path),
        Instances = make([dynamic]^LayerInstance),
    })
}
CreateLayerColoredEmblem :: proc(name, path: string) -> ^FlagLayerColoredEmblem {
    return new_clone(FlagLayerColoredEmblem{
        Texture = CreateFlagTexture(name, path),
        Instances = make([dynamic]^LayerInstance),
        Colors = make([dynamic]FlagColorVariant),
    })
}
DestroyLayer :: proc(variant: FlagLayerVariant) {
    switch layer in variant {
    case ^FlagLayerColoredEmblem:
        DestroyFlagTexture(layer.Texture)
        for _, index in layer.Colors {
            DestroyColor(layer.Colors[index])
        }
        delete(layer.Colors)
        for _, index in layer.Instances {
            DestroyLayerInstance(layer.Instances[index])
        }
        delete(layer.Instances)
        free(layer)
    case ^FlagLayerTexturedEmblem:
        DestroyFlagTexture(layer.Texture)
        for _, index in layer.Instances {
            DestroyLayerInstance(layer.Instances[index])
        }
        delete(layer.Instances)
        free(layer)
    }
}

LayerVector :: struct {
    X: f32,
    Y: f32,
}
LayerInstance :: struct {
    Rotation: i32,
    Scale: LayerVector,
    Position: LayerVector,
}
CreateLayerInstance :: proc(rotation: i32 = 0, scale: LayerVector = { 1, 1 }, position: LayerVector = { 0.5, 0.5 }) -> ^LayerInstance {
    return new_clone(LayerInstance{
        Rotation = rotation,
        Scale = scale,
        Position = position,
    })
}
DestroyLayerInstance :: proc(instance: ^LayerInstance) {
    free(instance)
}

FlagColorVariant :: union {
    ^FlagColorNamed,
    ^FlagColorHsv,
    ^FlagColorRgb,
    ^FlagColorReference,
}
FlagColor :: struct {
    Variant: FlagColorVariant,
    Name: string,
}
FlagColorNamed :: struct {
    using color: FlagColor,
    NamedColor: string,
}
FlagColorHsv :: struct {
    using color: FlagColor,
    H: f32,
    S: f32,
    V: f32,
}
FlagColorRgb :: struct {
    using color: FlagColor,
    R: u8,
    G: u8,
    B: u8,
}
FlagColorReference :: struct {
    using Color: FlagColor,
    Reference: string,
}

CreateColorRgb :: proc(name: string, r, g, b: u8) -> ^FlagColorRgb {
    return new_clone(FlagColorRgb{
        Name = strings.clone(name),
        R = r,
        G = g,
        B = b,
    })
}
CreateColorHsv :: proc(name: string, h, s, v: f32) -> ^FlagColorHsv {
    return new_clone(FlagColorHsv{
        Name = strings.clone(name),
        H = h,
        S = s,
        V = v,
    })
}
CreateColorNamed :: proc(name, namedColor: string) -> ^FlagColorNamed {
    return new_clone(FlagColorNamed{
        Name = strings.clone(name),
        NamedColor = strings.clone(namedColor),
    })
}
CreateColorReference :: proc(name, reference: string) -> ^FlagColorReference {
    return new_clone(FlagColorReference{
        Name = strings.clone(name),
        Reference = strings.clone(reference),
    })
}
DestroyColor :: proc(variant: FlagColorVariant) {
    switch color in variant {
    case ^FlagColorRgb:
        delete(color.Name)
        free(color)
    case ^FlagColorHsv:
        delete(color.Name)
        free(color)
    case ^FlagColorNamed:
        delete(color.Name)
        if color.NamedColor != "" {
            delete(color.NamedColor)
        }
        free(color)
    case ^FlagColorReference:
        delete(color.Name)
        delete(color.Reference)
        free(color)
    }
}