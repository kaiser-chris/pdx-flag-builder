package texture

import "bcdec"
import rl "vendor:raylib"
import "core:strings"
import "core:fmt"
import "core:image"
import "core:image/tga"

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

    if strings.has_suffix(path, ".tga") {
        texture, ok := LoadTgaTexture(path)
        if !ok {
            fmt.printfln("could not load tga file: %s", path)
            return rl.Texture2D{}
        }
        return texture
    }

    // Nothing we can do
    // Also this shouldn't happen for Victoria 3 or EU5
    return rl.Texture2D{}
}

LoadTgaTexture :: proc(path: string, allocator := context.allocator) -> (rl.Texture2D, bool) {
    img, err := tga.load_from_file(path, allocator = allocator)
    if err != nil {
        fmt.eprintf("failed to load tga: %v\n", err)
        return rl.Texture2D{}, false
    }
    defer image.destroy(img, allocator)

    if img.depth != 8 {
        fmt.eprintf("only 8-bit-per-channel TGA is handled here, got depth=%d\n", img.depth)
        return rl.Texture2D{}, false
    }

    ok := true

    switch img.channels {
    case 3:
        ok = image.alpha_add_if_missing(img, allocator = allocator)
    case 4:
        // already RGBA
    case:
        fmt.eprintf("unsupported channel count: %d\n", img.channels)
        return rl.Texture2D{}, false
    }

    if !ok {
        fmt.eprintf("failed to normalize image to RGBA\n")
        return rl.Texture2D{}, false
    }

    pixel_count := img.width * img.height
    byte_count  := pixel_count * 4

    ray_pixels := make([]u8, byte_count, allocator)
    copy(ray_pixels[0:byte_count], img.pixels.buf[0:byte_count])

    rl_img := rl.Image{
        data    = raw_data(ray_pixels),
        width   = i32(img.width),
        height  = i32(img.height),
        mipmaps = 1,
        format  = .UNCOMPRESSED_R8G8B8A8,
    }

    tex := rl.LoadTextureFromImage(rl_img)

    delete(ray_pixels, allocator)

    if !rl.IsTextureValid(tex) {
        fmt.eprintf("LoadTextureFromImage failed\n")
        return rl.Texture2D{}, false
    }

    return tex, true
}