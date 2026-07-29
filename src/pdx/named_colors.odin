package pdx

import "parser"
import os "core:os"
import fmt "core:fmt"

ATTRIBUTE_NAMED_COLORS: string: "colors"

LoadNamedColorsFile :: proc(path: string) -> []FlagColor {
    data, err := os.read_entire_file(path, context.allocator)
    if err != nil {
        fmt.eprintfln("could not read named colors file %s: %v", path, err)
        return []FlagColor{}
    }

    content := string(data)
    defer delete(data)
    result := parser.parse_document(content, context.temp_allocator)
    defer free_all(context.temp_allocator)
    if !result.ok {
        when !RELEASE {
            fmt.printfln("could not parse named colors file %s: %s at %i:%i", path, result, result.err.message, result.err.line, result.err.col)
        }
        return []FlagColor{}
    }

    if result.root.kind != .Object {
        return []FlagColor{}
    }

    colorGroups, ok := result.root.data.([]parser.Pair)
    if !ok {
        fmt.eprintln(ERROR_OBJECT_CAST)
        return []FlagColor{}
    }

    resultList := make([dynamic]FlagColor)
    for colorGroup in colorGroups {
        if colorGroup.key != ATTRIBUTE_NAMED_COLORS || colorGroup.value.kind != .Object {
            continue
        }
        colors, valid := colorGroup.value.data.([]parser.Pair)
        if !valid {
            fmt.eprintln(ERROR_OBJECT_CAST)
            return resultList[:]
        }
        for attribute in colors {
            if attribute.value.kind == .Typed {
                // typed color
                type := attribute.value.data.(parser.TypedValue)
                if type.value.kind != .List {
                    fmt.eprintfln("invalid color values for type %s", type.tag)
                    continue
                }
                values := type.value.data.([]parser.Value)
                color, success := loadTypedColor(attribute.key, type.tag, values)
                if success {
                    append(&resultList, color)
                }
            }
            if attribute.value.kind == .List {
                // untyped color
                values := attribute.value.data.([]parser.Value)
                color, success := loadTypedColor(attribute.key, COLOR_TYPE_IMPLICIT, values)
                if success {
                    append(&resultList, color)
                }
            }
        }
    }

    return resultList[:]

}