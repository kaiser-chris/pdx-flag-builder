package pdx

import "coa"
import "core:slice"
import os "core:os"
import fmt "core:fmt"

RELEASE :: #config(RELEASE, false)

ERROR_OBJECT_CAST: string: "Could not cast Object: %v"
ATTRIBUTE_PATTERN: string: "pattern"
ATTRIBUTE_COLORED_EMBLEM: string: "colored_emblem"
ATTRIBUTE_TEXTURED_EMBLEM: string: "textured_emblem"
ATTRIBUTE_SUB: string: "sub"
ATTRIBUTE_TEXTURE: string: "texture"
ATTRIBUTE_PARENT: string: "parent"
ATTRIBUTE_INSTANCE: string: "instance"
ATTRIBUTE_SCALE: string: "scale"
ATTRIBUTE_POSITION: string: "position"
ATTRIBUTE_ROTATION: string: "rotation"
ATTRIBUTE_OFFSET: string: "offset"

TYPE_TEMPLATE: string: "template"

LoadCoaFile :: proc(path: string) -> []Flag {
    data, err := os.read_entire_file(path, context.allocator)
    if err != nil {
        fmt.eprintfln("could not read coa file %s: %v", path, err)
        return []Flag{}
    }

    content := string(data)
    defer delete(data)
    result := coa.parse_document(content, context.temp_allocator)
    defer free_all(context.temp_allocator)
    if !result.ok {
        when !RELEASE {
            fmt.printfln("could not parse coa file %s: %s at %i:%i", path, result, result.err.message, result.err.line, result.err.col)
        }
        return []Flag{}
    }

    if result.root.kind != .Object {
        return []Flag{}
    }

    coaList, ok := result.root.data.([]coa.Pair)
    if !ok {
        fmt.eprintln(ERROR_OBJECT_CAST)
        return []Flag{}
    }

    flags := make([dynamic]Flag)
    for coa in coaList {
        flag, ok := LoadCoa(coa)
        if !ok {
            continue
        }
        append(&flags, flag)
    }

    return flags[:]
}

LoadCoa :: proc(root: coa.Pair) -> (Flag, bool) {

    if root.value.kind != .Object || root.key == TYPE_TEMPLATE {
        return Flag{}, false
    }

    attributes := root.value.data.([]coa.Pair)

    flag := CreateFlag(root.key)

    for attribute in attributes {
        if attribute.key == ATTRIBUTE_PATTERN && attribute.value.kind == .String {
            // the pattern image
            pattern := attribute.value.data.(string)
            flag.Pattern = CreateFlagTexture(pattern, "")
        }
        if (attribute.key == ATTRIBUTE_COLORED_EMBLEM || attribute.key == ATTRIBUTE_TEXTURED_EMBLEM) && attribute.value.kind == .Object {
            // layer
            layerAttributes := attribute.value.data.([]coa.Pair)
            layer, success := loadLayer(attribute.key, layerAttributes)
            if success {
                append(&flag.Layers, layer)
            }
        }
        if (attribute.key == ATTRIBUTE_COLORED_EMBLEM || attribute.key == ATTRIBUTE_TEXTURED_EMBLEM || attribute.key == ATTRIBUTE_SUB) && attribute.value.kind == .Object {
            // layer
            layerAttributes := attribute.value.data.([]coa.Pair)
            layer, success := loadLayer(attribute.key, layerAttributes)
            if success {
                append(&flag.Layers, layer)
            }
        }
    }
    colors := loadColors(attributes)
    for _, index in colors {
        append(&flag.Colors, colors[index])
    }
    delete(colors)


    return flag, true
}

loadLayer :: proc(type: string, attributes: []coa.Pair) -> (FlagLayerVariant, bool) {
    textureOrParent: string

    for attribute in attributes {
        if (attribute.key == ATTRIBUTE_TEXTURE || attribute.key == ATTRIBUTE_PARENT) && attribute.value.kind == .String {
            textureOrParent = attribute.value.data.(string)
        }
    }

    if type == ATTRIBUTE_COLORED_EMBLEM {
        instances := loadInstances(attributes)
        defer delete(instances)
        layer := CreateLayerColoredEmblem(textureOrParent, "")
        colors := loadColors(attributes)
        for color in colors {
            append(&layer.Colors, color)
        }
        delete(colors)
        for instance in instances {
            append(&layer.Instances, instance)
        }
        return layer, true
    } else if type == ATTRIBUTE_TEXTURED_EMBLEM {
        instances := loadInstances(attributes)
        defer delete(instances)
        layer := CreateLayerTexturedEmblem(textureOrParent, "")
        for instance in instances {
            append(&layer.Instances, instance)
        }
        return layer, true
    } else if type == ATTRIBUTE_SUB {
        instances := loadSubInstances(attributes)
        defer delete(instances)
        layer := CreateLayerSub(textureOrParent)
        for instance in instances {
            append(&layer.Instances, instance)
        }
        return layer, true
    }

    return nil, false
}

loadInstances :: proc(attributeInstances: []coa.Pair) -> []^LayerInstance {
    instances := make([dynamic]^LayerInstance)
    for attributeInstance in attributeInstances {
        if attributeInstance.key != ATTRIBUTE_INSTANCE || attributeInstance.value.kind != .Object {
            continue
        }
        instance := CreateLayerInstance()
        attributes := attributeInstance.value.data.([]coa.Pair)
        for attribute in attributes {
            if attribute.key == ATTRIBUTE_ROTATION && attribute.value.kind == .Number {
                rotation := attribute.value.data.(f64)
                instance.Rotation = i32(rotation)
            }
            if attribute.key == ATTRIBUTE_SCALE && attribute.value.kind == .List {
                values := attribute.value.data.([]coa.Value)
                scale := LayerVector{}
                if len(values) != 2 {
                    fmt.eprintfln("invalid number of values for instance scale: %v", values)
                    continue
                }
                for value, index in values {
                    if value.kind != .Number {
                        fmt.eprintfln("non numeric value for instance scale: %v", value.data)
                        continue
                    }
                    number := value.data.(f64)
                    if index == 0 {
                        scale.X = f32(number)
                    }
                    if index == 1 {
                        scale.Y = f32(number)
                    }
                }
                instance.Scale = scale
            }
            if attribute.key == ATTRIBUTE_POSITION && attribute.value.kind == .List {
                values := attribute.value.data.([]coa.Value)
                position := LayerVector{}
                if len(values) != 2 {
                    fmt.eprintfln("invalid number of values for instance position: %v", values)
                    continue
                }
                for value, index in values {
                    if value.kind != .Number {
                        fmt.eprintfln("non numeric value for instance position: %v", value.data)
                        continue
                    }
                    number := value.data.(f64)
                    if index == 0 {
                        position.X = f32(number)
                    }
                    if index == 1 {
                        position.Y = f32(number)
                    }
                }
                instance.Position = position
            }
        }
        append(&instances, instance)
    }
    return instances[:]
}

loadSubInstances :: proc(attributeInstances: []coa.Pair) -> []^LayerInstanceSub {
    instances := make([dynamic]^LayerInstanceSub)
    for attributeInstance in attributeInstances {
        if attributeInstance.key != ATTRIBUTE_INSTANCE || attributeInstance.value.kind != .Object {
            continue
        }
        instance := CreateLayerInstanceSub()
        attributes := attributeInstance.value.data.([]coa.Pair)
        for attribute in attributes {
            if attribute.key == ATTRIBUTE_SCALE && attribute.value.kind == .List {
                values := attribute.value.data.([]coa.Value)
                scale := LayerVector{}
                if len(values) != 2 {
                    fmt.eprintfln("invalid number of values for instance scale: %v", values)
                    continue
                }
                for value, index in values {
                    if value.kind != .Number {
                        fmt.eprintfln("non numeric value for instance scale: %v", value.data)
                        continue
                    }
                    number := value.data.(f64)
                    if index == 0 {
                        scale.X = f32(number)
                    }
                    if index == 1 {
                        scale.Y = f32(number)
                    }
                }
                instance.Scale = scale
            }
            if attribute.key == ATTRIBUTE_OFFSET && attribute.value.kind == .List {
                values := attribute.value.data.([]coa.Value)
                offset := LayerVector{}
                if len(values) != 2 {
                    fmt.eprintfln("invalid number of values for instance offset: %v", values)
                    continue
                }
                for value, index in values {
                    if value.kind != .Number {
                        fmt.eprintfln("non numeric value for instance offset: %v", value.data)
                        continue
                    }
                    number := value.data.(f64)
                    if index == 0 {
                        offset.X = f32(number)
                    }
                    if index == 1 {
                        offset.Y = f32(number)
                    }
                }
                instance.Offset = offset
            }
        }
        append(&instances, instance)
    }
    return instances[:]
}

loadColors :: proc(attributes: []coa.Pair) -> []FlagColorVariant {
    colors := make([dynamic]FlagColorVariant)
    for attribute in attributes {
        if slice.contains(COLOR_NAMES, attribute.key) && attribute.value.kind == .String {
            colorName := attribute.value.data.(string)

            color: FlagColorVariant
            if slice.contains(COLOR_NAMES, colorName) {
                color = CreateColorReference(attribute.key, colorName)
            } else {
                color = CreateColorNamed(attribute.key, colorName)
            }

            append(&colors, color)
        }
        if slice.contains(COLOR_NAMES, attribute.key) && attribute.value.kind == .Typed {
            // typed color
            type := attribute.value.data.(coa.TypedValue)
            if type.value.kind != .List {
                fmt.eprintfln("invalid color values for type %s", type.tag)
                return colors[:]
            }
            values := type.value.data.([]coa.Value)
            color, success := loadTypedColor(attribute.key, type.tag, values)
            if success {
                append(&colors, color)
            }
        }
        if slice.contains(COLOR_NAMES, attribute.key) && attribute.value.kind == .List {
            // untyped color
            values := attribute.value.data.([]coa.Value)
            color, success := loadTypedColor(attribute.key, COLOR_TYPE_IMPLICIT, values)
            if success {
                append(&colors, color)
            }
        }
    }
    return colors[:]
}

loadTypedColor :: proc(name: string, type: string, numbers: []coa.Value) -> (FlagColorVariant, bool) {
    firstValue: f64 = 0
    secondValue: f64 = 0
    thirdValue: f64 = 0

    if len(numbers) != 3 {
        fmt.eprintfln("invalid number of color values for type %s", type)
        return nil, false
    }
    for value, index in numbers {
        if value.kind != .Number {
            fmt.eprintfln("non numeric color value for type %s", type)
            return nil, false
        }
        number := value.data.(f64)
        if number < 0 {
            fmt.eprintfln("negative color value for type %s", type)
            return nil, false
        }
        if index == 0 {
            firstValue = number
        }
        if index == 1 {
            secondValue = number
        }
        if index == 2 {
            thirdValue = number
        }
    }

    color: FlagColorVariant
    switch type {
    case COLOR_TYPE_IMPLICIT:
        if firstValue != 0 && firstValue <= 1 && secondValue != 0 && secondValue <= 1 && thirdValue != 0 && thirdValue <= 1 {
            color = CreateColorRgb(name, u8(firstValue * 255), u8(secondValue * 255), u8(thirdValue * 255))
        } else {
            color = CreateColorRgb(name, u8(firstValue), u8(secondValue), u8(thirdValue))
        }
    case COLOR_TYPE_RGB:
        color = CreateColorRgb(name, u8(firstValue), u8(secondValue), u8(thirdValue))
    case COLOR_TYPE_HSV360:
        color = CreateColorHsv(name, f32(firstValue), f32(secondValue), f32(thirdValue))
    case COLOR_TYPE_HSV:
        color = CreateColorHsv(name, f32(firstValue) * 360, f32(secondValue) * 100, f32(thirdValue) * 100)
    case:
        fmt.eprintfln("unknown color type %s", type)
        return nil, false
    }

    return color, true
}