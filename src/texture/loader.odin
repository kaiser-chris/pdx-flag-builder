package texture

import "bcdec"
import rl "vendor:raylib"
import "core:strings"
import fmt "core:fmt"

LoadTexture :: proc(path: string) -> rl.Texture2D {
    // Try Raylib
    {
        cPath := strings.clone_to_cstring(path)
        defer delete(cPath)
        texture := rl.LoadTexture(cPath)
        if texture.format != rl.PixelFormat.UNKNOWN {
        // Raylib can handle it
            return texture
        }
        // Unload raylibs attempt
        rl.UnloadTexture(texture)
    }

    if strings.has_suffix(path, ".dds") {
        texture, err := bcdec.LoadBc7Texture(path)
        if err != nil {
            fmt.printfln("could not load dds file %s: %v", path, err)
            return rl.Texture2D{}
        }
        return texture
    }

    // todo
    return rl.Texture2D{}
}