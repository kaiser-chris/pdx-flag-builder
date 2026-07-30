package pdx_flag_builder

import "core:unicode/utf8"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "core:fmt"
import "core:sync/chan"
import "core:os"
import "pdx"
import nfd "nativefiledialog"
import ui "ui"

exportFlagImage :: proc(ctx: ^mu.Context, flag: pdx.Flag, width: i32 = 0, height: i32 = 0) {
    request := FlagExportRequest{
        Flag = flag,
        Size = { FLAG_WIDTH, FLAG_HEIGHT },
    }
    if width != 0 {
        request.Size[0] = width
    }
    if height != 0 {
        request.Size[1] = height
    }
    ok := chan.try_send(state.FlagExportChannel, request)
    if !ok {
        fmt.eprintfln("Could not export flag: %s", flag.Name)
    }
}

exportFlagToClipboard :: proc(ctx: ^mu.Context, flag: pdx.Flag) {
    script := pdx.WriteFlag(flag, "\n")
    defer free_all(context.temp_allocator)
    defer delete(script)
    clipString := strings.clone_to_cstring(script)
    defer delete(clipString)
    rl.SetClipboardText(clipString)
    ui.Toast(&state.Toast, "Export", "Copied to clipboard")
}

exportFlagAsFile :: proc(ctx: ^mu.Context, flag: pdx.Flag) {
    path: cstring
    filters := [1]nfd.Filter_Item { { "Script", "txt" } }
    args := nfd.Save_Dialog_Args {
        filter_list = raw_data(filters[:]),
        filter_count = len(filters),
    }

    result := nfd.SaveDialogU8_With(&path, &args)
    defer nfd.FreePathU8(path)
    if result == .Cancel {
        return
    }
    if result == .Error {
        ui.Toast(&state.Toast, "Export", fmt.tprintf("Export dialog error: %s", nfd.GetError()))
        return
    }

    normalPath := strings.clone_from_cstring(path)
    defer delete(normalPath)

    script := pdx.WriteFlag(flag, "\n")
    defer free_all(context.temp_allocator)
    defer delete(script)

    scriptWithBom := fmt.tprintf("%c%s", utf8.RUNE_BOM, script)

    err := os.write_entire_file(normalPath, scriptWithBom)
    if err != nil {
        ui.Toast(&state.Toast, "Export", fmt.tprintf("Could not write file: %v", err))
    }
    ui.Toast(&state.Toast, "Export", "Successfully saved the flag script.")
}

exportFlagAsIs :: proc(ctx: ^mu.Context, flag: ^pdx.Flag) {
    if flag.Path == "" {
        path: cstring
        filters := [1]nfd.Filter_Item { { "Script", "txt" } }
        args := nfd.Open_Dialog_Args {
            filter_list = raw_data(filters[:]),
            filter_count = len(filters),
        }
        result := nfd.OpenDialogU8_With(&path, &args)
        defer nfd.FreePathU8(path)
        if result == .Cancel {
            return
        }
        if result == .Error {
            ui.Toast(&state.Toast, "Export", fmt.tprintf("Export dialog error: %s", nfd.GetError()))
            return
        }

        flag.Path = strings.clone_from_cstring(path)

        _, file := os.split_path(flag.Path)
        flag.File = strings.clone(file)
    }

    if !os.exists(flag.Path) {
        ui.Toast(&state.Toast, "Export", "Target file does not exist.")
        return
    }

    data, err := os.read_entire_file(flag.Path, context.temp_allocator)
    defer free_all(context.temp_allocator)
    if err != nil {
        ui.Toast(&state.Toast, "Export", fmt.tprintf("Could not open target file: %v", err))
        return
    }
    content := string(data)

    isFirstLine: bool
    lines := strings.split_lines(content, context.temp_allocator)

    lineBreak: string
    if strings.contains(content, "\r\n") {
        lineBreak = "\r\n"
    } else {
        lineBreak = "\n"
    }

    lineStart: int
    lineEnd: int
    rBraceCount: int
    lBraceCount: int
    foundStart: bool
    foundEnd: bool
    for line, index in lines {
        cleanLine := strings.trim(line, " \t")
        if strings.has_prefix(cleanLine, flag.Name) {
        // We have found the start
            foundStart = true
            lineStart = index
        }
        if !foundStart {
        // We have not found it and the line does not contain it
            continue
        }
        // count open and closed braces
        lBraceCount += strings.count(line, "{")
        rBraceCount += strings.count(line, "}")

        // every opened brace is closed so the entry is over
        if lBraceCount == rBraceCount {
            foundEnd = true
            lineEnd = index
            break
        }
    }

    if foundStart && !foundEnd {
        ui.Toast(&state.Toast, "Export", "Could not determine flag location: Found open brace but no end brace", .Danger)
        return
    }

    script := pdx.WriteFlag(flag^, lineBreak)
    defer delete(script)

    mergedScript: string
    if !foundStart && !foundEnd {
        ui.Toast(&state.Toast, "Export", "Flag not found in file, appending at the end", .Warning)
        mergedScript := fmt.tprintf("%s%s%s", content, lineBreak, script)
    } else {
        sb, bErr := strings.builder_make(context.temp_allocator)
        if lineStart == 0 {
        // we removed the first line so we need to fix the bom
            strings.write_rune(&sb, utf8.RUNE_BOM)
        }
        // Write lines before the found flag
        for line, index in lines {
            if index >= lineStart {
                break
            }
            strings.write_string(&sb, line)
            strings.write_string(&sb, lineBreak)
        }
        // Write our modified flag
        strings.write_string(&sb, script)
        if len(lines) > lineEnd + 1 {
            strings.write_string(&sb, lineBreak)
        }
        // Write lines after modified flag
        for line, index in lines {
            if index <= lineEnd {
                continue
            }
            strings.write_string(&sb, line)
            if len(lines) > index + 1 {
                strings.write_string(&sb, lineBreak)
            }
        }
        mergedScript = strings.to_string(sb)
    }

    err = os.write_entire_file(flag.Path, mergedScript)
    if err != nil {
        ui.Toast(&state.Toast, "Export", fmt.tprintf("Could not write file: %v", err))
    }
    ui.Toast(&state.Toast, "Export", "Successfully saved the flag script.")
}