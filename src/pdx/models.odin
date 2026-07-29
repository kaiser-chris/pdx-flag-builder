package pdx

import strings "core:strings"
import fmt "core:fmt"

DEFAULT_SCALE: LayerVector: { 1, 1 }
DEFAULT_POSITION: LayerVector: { 0.5, 0.5 }
DEFAULT_OFFSET: LayerVector: { 0, 0 }

MIN_SCALE: f32: 0
MAX_SCALE: f32: 10

MIN_POSITION: f32: -4
MAX_POSITION: f32: 5

MIN_OFFSET: f32: -5
MAX_OFFSET: f32: 5

Flag :: struct {
    Name: string,
    Pattern: FlagTexture,
    Colors: [dynamic]FlagColor,
    Layers: [dynamic]FlagLayerVariant,
}

CreateFlag :: proc(name: string = "") -> Flag {
    flag := Flag{
        Name = strings.clone(name),
        Colors = make([dynamic]FlagColor),
        Layers = make([dynamic]FlagLayerVariant),
    }

    return flag
}
DestroyFlag :: proc(flag: ^Flag) {
    DestroyFlagTexture(flag.Pattern)
    for _, index in flag.Colors {
        DestroyFlagColor(&flag.Colors[index])
    }
    delete(flag.Colors)
    for _, index in flag.Layers {
        DestroyFlagLayer(flag.Layers[index])
    }
    delete(flag.Layers)
    delete(flag.Name)
}
CloneFlag :: proc(flag: Flag) -> Flag {
    clonedflag := CreateFlag(flag.Name)
    clonedflag.Pattern = CloneFlagTexture(flag.Pattern)
    for variant in flag.Colors {
        append(&clonedflag.Colors, CloneFlagColor(variant))
    }
    for variant in flag.Layers {
        append(&clonedflag.Layers, CloneFlagLayer(variant))
    }
    return clonedflag
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
CloneFlagTexture :: proc(texture: FlagTexture) -> FlagTexture {
    return FlagTexture{
        Name = strings.clone(texture.Name),
        Path = strings.clone(texture.Path),
    }
}

FlagLayerVariant :: union {
    ^FlagLayerColoredEmblem,
    ^FlagLayerTexturedEmblem,
    ^FlagLayerSub,
}
FlagLayer :: struct {
    Instances: [dynamic]^LayerInstance,
    Texture: FlagTexture,
}
FlagLayerColoredEmblem :: struct {
    using Layer: FlagLayer,
    Colors: [dynamic]FlagColor,
}
FlagLayerTexturedEmblem :: struct {
    using Layer: FlagLayer,
}
FlagLayerSub :: struct {
    Instances: [dynamic]^LayerInstanceSub,
    Parent: string,
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
        Colors = make([dynamic]FlagColor),
    })
}
CreateLayerSub :: proc(parent: string) -> ^FlagLayerSub {
    return new_clone(FlagLayerSub{
        Instances = make([dynamic]^LayerInstanceSub),
        Parent = strings.clone(parent)
    })
}
DestroyFlagLayer :: proc(variant: FlagLayerVariant) {
    switch layer in variant {
    case ^FlagLayerColoredEmblem:
        DestroyFlagTexture(layer.Texture)
        for _, index in layer.Colors {
            DestroyFlagColor(&layer.Colors[index])
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
    case ^FlagLayerSub:
        for _, index in layer.Instances {
            DestroyLayerInstanceSub(layer.Instances[index])
        }
        delete(layer.Instances)
        delete(layer.Parent)
        free(layer)
    }
}
CloneFlagLayer :: proc(variant: FlagLayerVariant) -> FlagLayerVariant {
    switch layer in variant {
    case ^FlagLayerColoredEmblem:
        clonedLayer := CreateLayerColoredEmblem(layer.Texture.Name, layer.Texture.Path)
        for instance in layer.Instances {
            append(&clonedLayer.Instances, CloneLayerInstance(instance))
        }
        for color in layer.Colors {
            append(&clonedLayer.Colors, CloneFlagColor(color))
        }
        return clonedLayer
    case ^FlagLayerTexturedEmblem:
        clonedLayer := CreateLayerTexturedEmblem(layer.Texture.Name, layer.Texture.Path)
        for instance in layer.Instances {
            append(&clonedLayer.Instances, CloneLayerInstance(instance))
        }
        return clonedLayer
    case ^FlagLayerSub:
        clonedLayer := CreateLayerSub(layer.Parent)
        for instance in layer.Instances {
            append(&clonedLayer.Instances, CloneLayerInstanceSub(instance))
        }
        return clonedLayer
    }
    return nil
}

LayerInstanceVariant :: union {
    ^LayerInstance,
    ^LayerInstanceSub,
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
CloneLayerInstance :: proc(instance: ^LayerInstance) -> ^LayerInstance {
    return CreateLayerInstance(instance.Rotation, { instance.Scale.X, instance.Scale.Y }, { instance.Position.X, instance.Position.Y })
}

LayerInstanceSub :: struct {
    Scale: LayerVector,
    Offset: LayerVector,
}
CreateLayerInstanceSub :: proc(scale: LayerVector = { 1, 1 }, offset: LayerVector = { 0.0, 0.0 }) -> ^LayerInstanceSub {
    return new_clone(LayerInstanceSub{
        Scale = scale,
        Offset = offset,
    })
}
DestroyLayerInstanceSub :: proc(instance: ^LayerInstanceSub) {
    free(instance)
}
CloneLayerInstanceSub :: proc(instance: ^LayerInstanceSub) -> ^LayerInstanceSub {
    return CreateLayerInstanceSub({ instance.Scale.X, instance.Scale.Y }, { instance.Offset.X, instance.Offset.Y })
}

FlagColor :: struct {
    Name: string,
    Variant: union {
        FlagColorNamed,
        FlagColorHsv,
        FlagColorRgb,
        FlagColorReference,
    },
}
FlagColorNamed :: struct {
    NamedColor: string,
}
FlagColorHsv :: struct {
    H: f32,
    S: f32,
    V: f32,
}
FlagColorRgb :: struct {
    R: u8,
    G: u8,
    B: u8,
}
FlagColorReference :: struct {
    Reference: string,
}

CreateColor :: proc($T: typeid, name: string) -> FlagColor {
    color := FlagColor{}
    color.Name = strings.clone(name)
    color.Variant = T{}
    return color
}

CreateColorRgb :: proc(name: string, r, g, b: u8) -> FlagColor {
    color := CreateColor(FlagColorRgb, name)
    variant := color.Variant.(FlagColorRgb)
    variant.R = r
    variant.G = g
    variant.B = b
    return color
}
CreateColorHsv :: proc(name: string, h, s, v: f32) -> FlagColor {
    color := CreateColor(FlagColorHsv, name)
    variant := color.Variant.(FlagColorHsv)
    variant.H = h
    variant.S = s
    variant.V = v
    return color
}
CreateColorNamed :: proc(name, namedColor: string) -> FlagColor {
    color := CreateColor(FlagColorNamed, name)
    variant := &color.Variant.(FlagColorNamed)
    variant.NamedColor = strings.clone(namedColor)
    return color
}
CreateColorReference :: proc(name, reference: string) -> FlagColor {
    color := CreateColor(FlagColorReference, name)
    variant := &color.Variant.(FlagColorReference)
    variant.Reference = strings.clone(reference)
    return color
}
DestroyFlagColor :: proc(variant: ^FlagColor) {
    delete(variant.Name)
    switch &color in variant.Variant {
    case FlagColorRgb:
    case FlagColorHsv:
    case FlagColorNamed:
        delete(color.NamedColor)
    case FlagColorReference:
        delete(color.Reference)
    }
}
CloneFlagColor :: proc(variant: FlagColor) -> FlagColor {
    switch color in variant.Variant {
    case FlagColorNamed:
        return CreateColorNamed(variant.Name, color.NamedColor)
    case FlagColorReference:
        return CreateColorReference(variant.Name, color.Reference)
    case FlagColorRgb:
        return CreateColorRgb(variant.Name, color.R, color.G, color.B)
    case FlagColorHsv:
        return CreateColorHsv(variant.Name, color.H, color.S, color.V)
    }
    return FlagColor{}
}