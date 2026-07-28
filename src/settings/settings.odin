package settings

import "core:fmt"
import "core:encoding/json"
import "core:os"
import "core:strings"
import mu "vendor:microui"

FILE_NAME_SETTINGS: string: "settings.json"

Settings :: struct {
    BackgroundColor: mu.Color,
    Databases: [dynamic]Database,
}

Database :: struct {
    Name: string,
    Path: string,
}

CreateSettings :: proc() -> ^Settings {
    return new_clone(Settings{
        BackgroundColor = { 90, 95, 100, 255 },
        Databases = make([dynamic]Database),
    })
}
DestroySettings :: proc(settings: ^Settings) {
    for _, index in settings.Databases {
        DestroyDatabase(&settings.Databases[index])
    }
    delete(settings.Databases)
    free(settings)
}

CreateDatabase :: proc(name, path: string) -> Database {
    return Database{
        Name = strings.clone(name),
        Path = strings.clone(path),
    }
}
DestroyDatabase :: proc(database: ^Database) {
    delete(database.Name)
    delete(database.Path)
}

LoadSettings :: proc(path: string) -> ^Settings {
    settings := CreateSettings()
    if !os.exists(path) {
        return settings
    }
    jsonBytes, err := os.read_entire_file(path, context.allocator)
    defer delete(jsonBytes)
    if err != nil {
        fmt.eprintf("Failed to read %s: %v\n", path, err)
        return settings
    }
    json_err := json.unmarshal(jsonBytes, settings)
    if json_err != nil {
        fmt.eprintf("Failed to parse %s: %v\n", path, err)
        return settings
    }
    return settings
}

SaveSettings :: proc(path: string, settings: ^Settings) {
    json_bytes, err := json.marshal(settings)
    if err != nil {
        fmt.eprintf("Failed to parse settings into JSON: %v\n", err)
        return
    }
    defer delete(json_bytes)
    file_err := os.write_entire_file(path, json_bytes)
    if file_err != nil {
        fmt.eprintf("Failed to write %s: %v\n", path, file_err)
        return
    }
}