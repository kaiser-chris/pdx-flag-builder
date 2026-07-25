package bcdec

import "core:c"
import "core:os"

import rl "vendor:raylib"

when ODIN_OS == .Windows {
    foreign import bc7 "bc7_decode.lib"
} else {
    foreign import bc7 "libbc7_decode.a"
}

foreign bc7 {
    bc7_decode_rgba8 :: proc(
    compressed_block: ^u8,
    rgba_out:         ^u8,
    destination_pitch_bytes: c.int,
    ) ---
}

DDS_BC7_Error :: enum {
    None,
    File_Read_Failed,
    File_Too_Small,
    Invalid_Magic,
    Invalid_Header,
    Not_DX10_DDS,
    Not_BC7,
    Unsupported_Texture_Type,
    Truncated_Pixel_Data,
    Allocation_Failed,
}

DDS_MAGIC              :: u32(0x2053_4444) // "DDS "
DDS_HEADER_SIZE        :: u32(124)
DDS_PIXEL_FORMAT_SIZE  :: u32(32)
DDS_FOURCC_DX10        :: u32(0x3031_5844) // "DX10"

DXGI_FORMAT_BC7_UNORM      :: u32(98)
DXGI_FORMAT_BC7_UNORM_SRGB :: u32(99)

D3D10_RESOURCE_DIMENSION_TEXTURE2D :: u32(3)
D3D11_RESOURCE_MISC_TEXTURECUBE     :: u32(0x4)

DDS_FILE_HEADER_BYTES :: 128
DDS_DX10_HEADER_BYTES :: 20
DDS_BC7_DATA_OFFSET   :: DDS_FILE_HEADER_BYTES + DDS_DX10_HEADER_BYTES

read_u32_le :: proc(data: []u8, offset: int) -> (u32, bool) {
    if offset < 0 || offset + 4 > len(data) {
        return 0, false
    }

    value := u32(data[offset + 0]) |
    u32(data[offset + 1]) << 8 |
    u32(data[offset + 2]) << 16 |
    u32(data[offset + 3]) << 24

    return value, true
}

LoadBc7Texture :: proc(
    path: string,
    allocator := context.allocator,
) -> (rl.Texture2D, DDS_BC7_Error) {
    file_data, err := os.read_entire_file(path, allocator)
    if err != nil {
        return rl.Texture2D{}, .File_Read_Failed
    }
    defer delete(file_data)

    if len(file_data) < DDS_BC7_DATA_OFFSET {
        return rl.Texture2D{}, .File_Too_Small
    }

    magic, magicOk := read_u32_le(file_data, 0)
    if !magicOk || magic != DDS_MAGIC {
        return rl.Texture2D{}, .Invalid_Magic
    }

    // DDS_HEADER starts immediately after the 4-byte "DDS " magic.
    header_size, headerOk := read_u32_le(file_data, 4)
    if !headerOk || header_size != DDS_HEADER_SIZE {
        return rl.Texture2D{}, .Invalid_Header
    }

    height_u32, heightOk := read_u32_le(file_data, 12)
    if !heightOk || height_u32 == 0 {
        return rl.Texture2D{}, .Invalid_Header
    }

    width_u32, widthOk := read_u32_le(file_data, 16)
    if !widthOk || width_u32 == 0 {
        return rl.Texture2D{}, .Invalid_Header
    }

    // DDS_PIXELFORMAT starts at byte 76 in the file:
    // 4-byte magic + 72-byte offset inside DDS_HEADER.
    pixel_format_size, sizeOk := read_u32_le(file_data, 76)
    if !sizeOk || pixel_format_size != DDS_PIXEL_FORMAT_SIZE {
        return rl.Texture2D{}, .Invalid_Header
    }

    fourcc, fourccOk := read_u32_le(file_data, 84)
    if !fourccOk || fourcc != DDS_FOURCC_DX10 {
        return rl.Texture2D{}, .Not_DX10_DDS
    }

    // DDS_HEADER_DXT10 begins after magic + DDS_HEADER, at byte 128.
    dxgi_format, dxgiOk := read_u32_le(file_data, 128)
    if !dxgiOk {
        return rl.Texture2D{}, .Invalid_Header
    }

    if dxgi_format != DXGI_FORMAT_BC7_UNORM &&
    dxgi_format != DXGI_FORMAT_BC7_UNORM_SRGB {
        return rl.Texture2D{}, .Not_BC7
    }

    resource_dimension, domainOk := read_u32_le(file_data, 132)
    if !domainOk || resource_dimension != D3D10_RESOURCE_DIMENSION_TEXTURE2D {
        return rl.Texture2D{}, .Unsupported_Texture_Type
    }

    misc_flag, miscOk := read_u32_le(file_data, 136)
    if !miscOk {
        return rl.Texture2D{}, .Invalid_Header
    }

    // Cubemaps need six faces and are intentionally not handled here.
    if (misc_flag & D3D11_RESOURCE_MISC_TEXTURECUBE) != 0 {
        return rl.Texture2D{}, .Unsupported_Texture_Type
    }

    array_size, arrayOk := read_u32_le(file_data, 140)
    if !arrayOk || array_size != 1 {
        return rl.Texture2D{}, .Unsupported_Texture_Type
    }

    width  := int(width_u32)
    height := int(height_u32)

    // BC7 has 16 bytes per 4x4 pixel block.
    blocks_x := (width + 3) / 4
    blocks_y := (height + 3) / 4
    compressed_size := blocks_x * blocks_y * 16

    if DDS_BC7_DATA_OFFSET + compressed_size > len(file_data) {
        return rl.Texture2D{}, .Truncated_Pixel_Data
    }

    rgba_size := width * height * 4
    if width <= 0 || height <= 0 || rgba_size <= 0 {
        return rl.Texture2D{}, .Invalid_Header
    }

    rgba := make([]u8, rgba_size, allocator)
    if len(rgba) != rgba_size {
        return rl.Texture2D{}, .Allocation_Failed
    }
    defer delete(rgba)

    compressed := file_data[DDS_BC7_DATA_OFFSET: DDS_BC7_DATA_OFFSET + compressed_size]

    // A temporary full 4x4 RGBA block prevents writes past the end for
    // textures whose width or height is not divisible by four.
    block_rgba: [64]u8

    for block_y in 0..<blocks_y {
        for block_x in 0..<blocks_x {
            block_index := block_y * blocks_x + block_x
            source_offset := block_index * 16

            bc7_decode_rgba8(
            &compressed[source_offset],
            &block_rgba[0],
            4 * 4, // 4 RGBA pixels per temporary row
            )

            pixel_x := block_x * 4
            pixel_y := block_y * 4

            copy_width  := min(4, width  - pixel_x)
            copy_height := min(4, height - pixel_y)

            for local_y in 0..<copy_height {
                source_row := local_y * 4 * 4
                target_row := ((pixel_y + local_y) * width + pixel_x) * 4

                for local_x in 0..<copy_width {
                    source_pixel := source_row + local_x * 4
                    target_pixel := target_row + local_x * 4

                    rgba[target_pixel + 0] = block_rgba[source_pixel + 0]
                    rgba[target_pixel + 1] = block_rgba[source_pixel + 1]
                    rgba[target_pixel + 2] = block_rgba[source_pixel + 2]
                    rgba[target_pixel + 3] = block_rgba[source_pixel + 3]
                }
            }
        }
    }

    image := rl.Image{
        data    = raw_data(rgba),
        width   = c.int(width),
        height  = c.int(height),
        mipmaps = 1,
        format  = .UNCOMPRESSED_R8G8B8A8,
    }

    texture := rl.LoadTextureFromImage(image)
    if texture.id == 0 {
        return rl.Texture2D{}, .Allocation_Failed
    }

    return texture, .None
}