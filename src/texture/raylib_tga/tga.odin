package raylib_tga

import "core:mem"
import "core:os"
import rl "vendor:raylib"

TGA_Error :: enum {
    None,
    File_Read_Failed,
    File_Too_Small,
    Unsupported_Color_Map,
    Unsupported_Image_Type,
    Unsupported_Pixel_Depth,
    Invalid_Dimensions,
    Truncated_Data,
    Allocation_Failed,
    Invalid_RLE_Data,
}

TGA_HEADER_SIZE :: 18

read_u16_le :: proc(data: []u8, offset: int) -> (u16, bool) {
    if offset < 0 || offset + 2 > len(data) {
        return 0, false
    }

    value := u16(data[offset + 0]) |
    u16(data[offset + 1]) << 8

    return value, true
}

write_rgba_pixel :: proc(
pixels: []u8,
width: int,
height: int,
source_x: int,
source_y: int,
b: u8,
g: u8,
r: u8,
a: u8,
origin_top: bool,
origin_right: bool,
) {
// TGA data can be stored from either upper/lower and left/right origins.
    target_x := source_x
    target_y := source_y

    if origin_right {
        target_x = width - 1 - source_x
    }

    if !origin_top {
        target_y = height - 1 - source_y
    }

    offset := (target_y * width + target_x) * 4

    pixels[offset + 0] = r
    pixels[offset + 1] = g
    pixels[offset + 2] = b
    pixels[offset + 3] = a
}

LoadTgaImage :: proc(path: string) -> (rl.Image, TGA_Error) {
    file_data, err := os.read_entire_file(path, context.allocator)
    if err != nil {
        return rl.Image{}, .File_Read_Failed
    }
    defer delete(file_data)

    if len(file_data) < TGA_HEADER_SIZE {
        return rl.Image{}, .File_Too_Small
    }

    id_length       := int(file_data[0])
    color_map_type  := file_data[1]
    image_type      := file_data[2]

    width_u16, ok := read_u16_le(file_data, 12)
    if !ok {
        return rl.Image{}, .File_Too_Small
    }

    height_u16, ok14 := read_u16_le(file_data, 14)
    if !ok14 {
        return rl.Image{}, .File_Too_Small
    }

    bits_per_pixel := file_data[16]
    descriptor     := file_data[17]

    // This loader deliberately supports only direct-color images.
    // Types 2 and 10 require there to be no palette/color map.
    if color_map_type != 0 {
        return rl.Image{}, .Unsupported_Color_Map
    }

    is_uncompressed := image_type == 2
    is_rle          := image_type == 10

    if !is_uncompressed && !is_rle {
        return rl.Image{}, .Unsupported_Image_Type
    }

    if bits_per_pixel != 24 && bits_per_pixel != 32 {
        return rl.Image{}, .Unsupported_Pixel_Depth
    }

    width  := int(width_u16)
    height := int(height_u16)

    if width <= 0 || height <= 0 {
        return rl.Image{}, .Invalid_Dimensions
    }

    bytes_per_pixel := int(bits_per_pixel / 8)
    pixel_count := width * height
    rgba_size := pixel_count * 4

    // TGA header is 18 bytes, followed by its optional image-ID field.
    source_offset := TGA_HEADER_SIZE + id_length
    if source_offset > len(file_data) {
        return rl.Image{}, .Truncated_Data
    }

    // This allocation is owned by raylib. Therefore, callers MUST use:
    // rl.UnloadImage(image)
    pixel_memory := rl.MemAlloc(u32(rgba_size))
    if pixel_memory == nil {
        return rl.Image{}, .Allocation_Failed
    }

    // Make the raylib allocation writable using normal Odin slice indexing.
    pixels := mem.slice_ptr((^u8)(pixel_memory), rgba_size)

    // If an error happens after allocation, avoid leaking it.
    success := false
    defer if !success {
        rl.MemFree(pixel_memory)
    }

    // Bit 5: 1 = first source row is the top row.
    // Bit 4: 1 = first source pixel is the right-most pixel.
    origin_top   := (descriptor & 0x20) != 0
    origin_right := (descriptor & 0x10) != 0

    pixel_index := 0

    if is_uncompressed {
        required_bytes := pixel_count * bytes_per_pixel
        if source_offset + required_bytes > len(file_data) {
            return rl.Image{}, .Truncated_Data
        }

        for pixel_index < pixel_count {
            b := file_data[source_offset + 0]
            g := file_data[source_offset + 1]
            r := file_data[source_offset + 2]
            a := u8(255)

            if bits_per_pixel == 32 {
                a = file_data[source_offset + 3]
            }

            source_x := pixel_index % width
            source_y := pixel_index / width

            write_rgba_pixel(
            pixels, width, height,
            source_x, source_y,
            b, g, r, a,
            origin_top, origin_right,
            )

            source_offset += bytes_per_pixel
            pixel_index += 1
        }
    } else {
        // TGA RLE packets:
        // high bit set   -> repeat one pixel (count = low 7 bits + 1)
        // high bit clear -> copy following raw pixels
        for pixel_index < pixel_count {
            if source_offset >= len(file_data) {
                return rl.Image{}, .Truncated_Data
            }

            packet_header := file_data[source_offset]
            source_offset += 1

            run_length := int(packet_header & 0x7f) + 1
            is_rle_packet := (packet_header & 0x80) != 0

            if pixel_index + run_length > pixel_count {
                return rl.Image{}, .Invalid_RLE_Data
            }

            if is_rle_packet {
                if source_offset + bytes_per_pixel > len(file_data) {
                    return rl.Image{}, .Truncated_Data
                }

                b := file_data[source_offset + 0]
                g := file_data[source_offset + 1]
                r := file_data[source_offset + 2]
                a := u8(255)

                if bits_per_pixel == 32 {
                    a = file_data[source_offset + 3]
                }

                source_offset += bytes_per_pixel

                for _ in 0..<run_length {
                    source_x := pixel_index % width
                    source_y := pixel_index / width

                    write_rgba_pixel(
                    pixels, width, height,
                    source_x, source_y,
                    b, g, r, a,
                    origin_top, origin_right,
                    )

                    pixel_index += 1
                }
            } else {
                bytes_needed := run_length * bytes_per_pixel
                if source_offset + bytes_needed > len(file_data) {
                    return rl.Image{}, .Truncated_Data
                }

                for _ in 0..<run_length {
                    b := file_data[source_offset + 0]
                    g := file_data[source_offset + 1]
                    r := file_data[source_offset + 2]
                    a := u8(255)

                    if bits_per_pixel == 32 {
                        a = file_data[source_offset + 3]
                    }

                    source_x := pixel_index % width
                    source_y := pixel_index / width

                    write_rgba_pixel(
                    pixels, width, height,
                    source_x, source_y,
                    b, g, r, a,
                    origin_top, origin_right,
                    )

                    source_offset += bytes_per_pixel
                    pixel_index += 1
                }
            }
        }
    }

    image := rl.Image{
        data    = pixel_memory,
        width   = i32(width),
        height  = i32(height),
        mipmaps = 1,
        format  = .UNCOMPRESSED_R8G8B8A8,
    }

    success = true
    return image, .None
}