package pdx_flag_builder

import mu "vendor:microui"
import "core:fmt"
import "core:encoding/json"
import "core:os"

FILE_NAME_SETTINGS: string: "settings.json"

Settings :: struct {
    BackgroundColor: mu.Color,
    Databases: [dynamic]FlagDatabase,
}

FlagDatabase :: struct {
    Name: string,
    Path: string,
}

loadSettings :: proc() {
    if !os.exists(FILE_NAME_SETTINGS) {
        state.Settings = Settings{
            BackgroundColor = { 90, 95, 100, 255 },
            Databases = make([dynamic]FlagDatabase),
        }
        return
    }
    jsonBytes, err := os.read_entire_file(FILE_NAME_SETTINGS, context.allocator)
    if err != nil {
        fmt.eprintf("Failed to read %s: %v\n", FILE_NAME_SETTINGS, err)
        return
    }

    json_err := json.unmarshal(jsonBytes, &state.Settings)
    if json_err != nil {
        fmt.eprintf("Failed to parse %s: %v\n", FILE_NAME_SETTINGS, err)
        return
    }

    if state.Settings.Databases != nil {
        state.Databases = make([dynamic]DatabaseState, len(state.Settings.Databases))
        for _, index in state.Settings.Databases {
            state.Databases[index] = DatabaseState{
                Settings = state.Settings.Databases[index],
            }
            state.Databases[index].BufferNameLength = len(state.Databases[index].Settings.Name)
            copy(state.Databases[index].BufferName[:], state.Databases[index].Settings.Name)
            state.Databases[index].BufferPathLength = len(state.Databases[index].Settings.Path)
            copy(state.Databases[index].BufferPath[:], state.Databases[index].Settings.Path)
            loadNamedColors(&state.Databases[index])
        }
    }

    delete(jsonBytes)
}

freeSettings :: proc() {
    for _, index in state.Settings.Databases {
        delete(state.Settings.Databases[index].Name)
        delete(state.Settings.Databases[index].Path)
    }
    delete(state.Settings.Databases)
}

saveSettings :: proc() {
    state.Settings.Databases = make([dynamic]FlagDatabase, len(state.Databases))
    for _, index in state.Databases {
        state.Settings.Databases[index] = state.Databases[index].Settings
        destroyNamedColors(&state.Databases[index])
        loadNamedColors(&state.Databases[index])
    }

    json_bytes, err := json.marshal(state.Settings)
    if err != nil {
        fmt.eprintf("Failed to parse settings into JSON: %v\n", err)
        return
    }
    defer delete(json_bytes)

    file_err := os.write_entire_file(FILE_NAME_SETTINGS, json_bytes)
    if file_err != nil {
        fmt.eprintf("Failed to write %s: %v\n", FILE_NAME_SETTINGS, file_err)
        return
    }

    fmt.println(string(json_bytes))
}
