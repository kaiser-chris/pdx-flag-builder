package pdx

import "parser"
import "core:slice"
import os "core:os"
import fmt "core:fmt"
import strings "core:strings"

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
        fmt.eprintfln("could not read coat of arms file %s: %v", path, err)
        return []Flag{}
    }

    content := string(data)
    defer delete(data)
    result := parser.parse_document(content, context.temp_allocator)
    defer free_all(context.temp_allocator)
    if !result.ok {
        when !RELEASE {
            fmt.printfln("could not parse coat of arms file %s: %s at %i:%i", path, result, result.err.message, result.err.line, result.err.col)
        }
        return []Flag{}
    }

    if result.root.kind != .Object {
        return []Flag{}
    }

    coaList, ok := result.root.data.([]parser.Pair)
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

LoadCoa :: proc(root: parser.Pair) -> (Flag, bool) {

    if root.value.kind != .Object || root.key == TYPE_TEMPLATE {
        return Flag{}, false
    }

    attributes := root.value.data.([]parser.Pair)

    flag := CreateFlag(root.key)

    for attribute in attributes {
        if attribute.key == ATTRIBUTE_PATTERN && attribute.value.kind == .String {
            // the pattern image
            pattern := attribute.value.data.(string)
            flag.Pattern = CreateFlagTexture(pattern, "")
        }
        if (attribute.key == ATTRIBUTE_COLORED_EMBLEM || attribute.key == ATTRIBUTE_TEXTURED_EMBLEM || attribute.key == ATTRIBUTE_SUB) && attribute.value.kind == .Object {
            // layer
            layerAttributes := attribute.value.data.([]parser.Pair)
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

loadLayer :: proc(type: string, attributes: []parser.Pair) -> (FlagLayerVariant, bool) {
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
        for &color in colors {
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

loadInstances :: proc(attributeInstances: []parser.Pair) -> []^LayerInstance {
    instances := make([dynamic]^LayerInstance)
    for attributeInstance in attributeInstances {
        if attributeInstance.key != ATTRIBUTE_INSTANCE || attributeInstance.value.kind != .Object {
            continue
        }
        instance := CreateLayerInstance()
        attributes := attributeInstance.value.data.([]parser.Pair)
        for attribute in attributes {
            if attribute.key == ATTRIBUTE_ROTATION && attribute.value.kind == .Number {
                rotation := attribute.value.data.(f64)
                instance.Rotation = i32(rotation)
            }
            if attribute.key == ATTRIBUTE_SCALE && attribute.value.kind == .List {
                values := attribute.value.data.([]parser.Value)
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
                values := attribute.value.data.([]parser.Value)
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

loadSubInstances :: proc(attributeInstances: []parser.Pair) -> []^LayerInstanceSub {
    instances := make([dynamic]^LayerInstanceSub)
    for attributeInstance in attributeInstances {
        if attributeInstance.key != ATTRIBUTE_INSTANCE || attributeInstance.value.kind != .Object {
            continue
        }
        instance := CreateLayerInstanceSub()
        attributes := attributeInstance.value.data.([]parser.Pair)
        for attribute in attributes {
            if attribute.key == ATTRIBUTE_SCALE && attribute.value.kind == .List {
                values := attribute.value.data.([]parser.Value)
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
                values := attribute.value.data.([]parser.Value)
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

loadColors :: proc(attributes: []parser.Pair) -> []FlagColor {
    colors := make([dynamic]FlagColor)
    for attribute in attributes {
        if slice.contains(COLOR_NAMES, attribute.key) && attribute.value.kind == .String {
            colorName := attribute.value.data.(string)

            color: FlagColor
            if slice.contains(COLOR_NAMES, colorName) {
                color = CreateColorReference(attribute.key, colorName)
            } else {
                color = CreateColorNamed(attribute.key, colorName)
            }

            append(&colors, color)
        }
        if slice.contains(COLOR_NAMES, attribute.key) && attribute.value.kind == .Typed {
            // typed color
            type := attribute.value.data.(parser.TypedValue)
            if type.value.kind != .List {
                fmt.eprintfln("invalid color values for type %s", type.tag)
                return colors[:]
            }
            values := type.value.data.([]parser.Value)
            color, success := loadTypedColor(attribute.key, type.tag, values)
            if success {
                append(&colors, color)
            }
        }
        if slice.contains(COLOR_NAMES, attribute.key) && attribute.value.kind == .List {
            // untyped color
            values := attribute.value.data.([]parser.Value)
            color, success := loadTypedColor(attribute.key, COLOR_TYPE_IMPLICIT, values)
            if success {
                append(&colors, color)
            }
        }
    }
    return colors[:]
}

loadTypedColor :: proc(name: string, type: string, numbers: []parser.Value) -> (FlagColor, bool) {
    firstValue: f64 = 0
    secondValue: f64 = 0
    thirdValue: f64 = 0

    if len(numbers) != 3 {
        fmt.eprintfln("invalid number of color values for type %s", type)
        return FlagColor{}, false
    }
    for value, index in numbers {
        if value.kind != .Number {
            fmt.eprintfln("non numeric color value for type %s", type)
            return FlagColor{}, false
        }
        number := value.data.(f64)
        if number < 0 {
            fmt.eprintfln("negative color value for type %s", type)
            return FlagColor{}, false
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

    switch type {
    case COLOR_TYPE_IMPLICIT:
        if firstValue <= 1 && secondValue <= 1 && thirdValue <= 1 {
            return CreateColorRgb(name, u8(firstValue * 255), u8(secondValue * 255), u8(thirdValue * 255)), true
        } else {
            return CreateColorRgb(name, u8(firstValue), u8(secondValue), u8(thirdValue)), true
        }
    case COLOR_TYPE_RGB:
        if firstValue <= 1 && secondValue <= 1 && thirdValue <= 1 {
            return CreateColorRgb(name, u8(firstValue * 255), u8(secondValue * 255), u8(thirdValue * 255)), true
        } else {
            return CreateColorRgb(name, u8(firstValue), u8(secondValue), u8(thirdValue)), true
        }
    case COLOR_TYPE_HSV360:
        return CreateColorHsv(name, f32(firstValue), f32(secondValue), f32(thirdValue)), true
    case COLOR_TYPE_HSV:
        return CreateColorHsv(name, f32(firstValue) * 360, f32(secondValue) * 100, f32(thirdValue) * 100), true
    }

    fmt.eprintfln("unknown color type %s", type)
    return FlagColor{}, false
}

WriteFlag :: proc(flag: Flag, lineBreak: string) -> string {
    builder, err := strings.builder_make(context.temp_allocator)
    defer free_all(context.temp_allocator)
    if err != nil {
        fmt.eprintfln("could not init flag script builder: %v", err)
        return ""
    }

    if flag.Name == "" {
        strings.write_string(&builder, "NONE = {")
    } else {
        strings.write_string(&builder, fmt.tprintf("%s = {{", flag.Name))
    }
    strings.write_string(&builder, lineBreak)

    if flag.Pattern.Name != "" {
        strings.write_string(&builder, fmt.tprintf("\t%s = \"%s\"", ATTRIBUTE_PATTERN, flag.Pattern.Name))
        strings.write_string(&builder, lineBreak)
    }

    writeColors(&builder, flag.Colors[:], "\t", lineBreak)

    if len(flag.Layers) > 0 {
        strings.write_string(&builder, lineBreak)
    }

    for variant, index in flag.Layers {
        switch layer in variant {
        case ^FlagLayerColoredEmblem:
            writeColoredEmblemLayer(&builder, layer, lineBreak)
        case ^FlagLayerTexturedEmblem:
            writeTexturedEmblemLayer(&builder, layer, lineBreak)
        case ^FlagLayerSub:
            writeSubFlagLayer(&builder, layer, lineBreak)
        }
        strings.write_string(&builder, lineBreak)
        if index + 1 < len(flag.Layers) {
            strings.write_string(&builder, lineBreak)
        }
    }

    strings.write_string(&builder, "}")

    return strings.clone(strings.to_string(builder))
}

writeColoredEmblemLayer :: proc(builder: ^strings.Builder, layer: ^FlagLayerColoredEmblem, lineBreak: string) {
    strings.write_string(builder, fmt.tprintf("\t%s = {{", ATTRIBUTE_COLORED_EMBLEM))
    strings.write_string(builder, lineBreak)

    strings.write_string(builder, fmt.tprintf("\t\t%s = \"%s\"", ATTRIBUTE_TEXTURE, layer.Texture.Name))
    strings.write_string(builder, lineBreak)

    writeColors(builder, layer.Colors[:], "\t\t", lineBreak)

    if len(layer.Instances) > 0 {
        strings.write_string(builder, lineBreak)
    }

    for instance, index in layer.Instances {
        strings.write_string(builder, fmt.tprintf("\t\t%s = {{", ATTRIBUTE_INSTANCE))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = %i", ATTRIBUTE_ROTATION, instance.Rotation))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = {{ %.3f %.3f }}", ATTRIBUTE_SCALE, instance.Scale.X, instance.Scale.Y))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = {{ %.3f %.3f }}", ATTRIBUTE_POSITION, instance.Position.X, instance.Position.Y))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, "\t\t}")
        strings.write_string(builder, lineBreak)
        if index + 1 < len(layer.Instances) {
            strings.write_string(builder, lineBreak)
        }
    }

    strings.write_string(builder, "\t}")
}

writeTexturedEmblemLayer :: proc(builder: ^strings.Builder, layer: ^FlagLayerTexturedEmblem, lineBreak: string) {
    strings.write_string(builder, fmt.tprintf("\t%s = {{", ATTRIBUTE_TEXTURED_EMBLEM))
    strings.write_string(builder, lineBreak)

    strings.write_string(builder, fmt.tprintf("\t\t%s = \"%s\"", ATTRIBUTE_TEXTURE, layer.Texture.Name))
    strings.write_string(builder, lineBreak)

    if len(layer.Instances) > 0 {
        strings.write_string(builder, lineBreak)
    }

    for instance, index in layer.Instances {
        strings.write_string(builder, fmt.tprintf("\t\t%s = {{", ATTRIBUTE_INSTANCE))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = %i", ATTRIBUTE_ROTATION, instance.Rotation))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = {{ %.3f %.3f }}", ATTRIBUTE_SCALE, instance.Scale.X, instance.Scale.Y))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = {{ %.3f %.3f }}", ATTRIBUTE_POSITION, instance.Position.X, instance.Position.Y))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, "\t\t}")
        strings.write_string(builder, lineBreak)
        if index + 1 < len(layer.Instances) {
            strings.write_string(builder, lineBreak)
        }
    }

    strings.write_string(builder, "\t}")
}

writeSubFlagLayer :: proc(builder: ^strings.Builder, layer: ^FlagLayerSub, lineBreak: string) {
    strings.write_string(builder, fmt.tprintf("\t%s = {{", ATTRIBUTE_SUB))
    strings.write_string(builder, lineBreak)

    strings.write_string(builder, fmt.tprintf("\t\t%s = \"%s\"", ATTRIBUTE_PARENT, layer.Parent))
    strings.write_string(builder, lineBreak)

    if len(layer.Instances) > 0 {
        strings.write_string(builder, lineBreak)
    }

    for instance, index in layer.Instances {
        strings.write_string(builder, fmt.tprintf("\t\t%s = {{", ATTRIBUTE_INSTANCE))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = {{ %.3f %.3f }}", ATTRIBUTE_SCALE, instance.Scale.X, instance.Scale.Y))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, fmt.tprintf("\t\t\t%s = {{ %.3f %.3f }}", ATTRIBUTE_OFFSET, instance.Offset.X, instance.Offset.Y))
        strings.write_string(builder, lineBreak)
        strings.write_string(builder, "\t\t}")
        strings.write_string(builder, lineBreak)
        if index + 1 < len(layer.Instances) {
            strings.write_string(builder, lineBreak)
        }
    }

    strings.write_string(builder, "\t}")
}

writeColors :: proc(builder: ^strings.Builder, colors: []FlagColor, prefix, lineBreak: string) {
    for variant in colors {
        switch color in variant.Variant {
        case FlagColorNamed:
            strings.write_string(builder, fmt.tprintf("%s%s = \"%s\"", prefix, variant.Name, color.NamedColor))
            strings.write_string(builder, lineBreak)
        case FlagColorReference:
            strings.write_string(builder, fmt.tprintf("%s%s = %s", prefix, variant.Name, color.Reference))
            strings.write_string(builder, lineBreak)
        case FlagColorRgb:
            strings.write_string(builder, fmt.tprintf("%s%s = %s{{ %i %i %i }}", prefix, COLOR_TYPE_RGB, variant.Name, color.R, color.G, color.B))
            strings.write_string(builder, lineBreak)
        case FlagColorHsv:
            strings.write_string(builder, fmt.tprintf("%s%s = %s{{ %.0f %.0f %.0f }}", prefix, COLOR_TYPE_HSV360, variant.Name, color.H, color.S, color.V))
            strings.write_string(builder, lineBreak)
        }
    }
}