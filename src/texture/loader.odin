package texture

import "bcdec"
import rl "vendor:raylib"
import "core:strings"
import "core:fmt"
import better "raylib_tga"

LoadImage :: proc(path: string) -> rl.Image {
    // Try Raylib
    {
        cPath := strings.clone_to_cstring(path)
        defer delete(cPath)
        image := rl.LoadImage(cPath)
        if rl.IsImageValid(image) {
            // Raylib can handle it
            return image
        }
        // Unload raylibs attempt
        rl.UnloadImage(image)
    }

    if strings.has_suffix(path, ".dds") {
        image, err := bcdec.LoadBc7Image(path)
        if err != nil {
            fmt.printfln("could not load dds file %s: %v", path, err)
            return rl.Image{}
        }
        return image
    }

    if strings.has_suffix(path, ".tga") {
        image, err := better.LoadTgaImage(path)
        if err != .None {
            fmt.printfln("could not load tga file: %s", path)
            return rl.Image{}
        }
        return image
    }

    // Nothing we can do
    // Also this shouldn't happen for Victoria 3 or EU5
    return rl.Image{}
}

LoadTexture :: proc(path: string) -> rl.Texture2D {
    image := LoadImage(path)
    if !rl.IsImageValid(image) {
        rl.UnloadImage(image)
        return rl.Texture2D{}
    }

    texture := rl.LoadTextureFromImage(image)
    if !rl.IsTextureValid(texture) {
        rl.UnloadTexture(texture)
        return rl.Texture2D{}
    }

    return texture
}