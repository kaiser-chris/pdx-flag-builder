package pdx_flag_builder

import "core:unicode/utf8"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import fmt "core:fmt"

TEXTURE_RECT_IDENTIFIER: u8: 1
TOOLBAR_HEIGHT: i32: 35
FLAG_HEIGHT: i32: 512
FLAG_WIDTH: i32: 768

State :: struct {
    Settings: Settings,
    Databases: [dynamic]DatabaseState,
    Context: mu.Context,
    SettingsWindowOpen: bool,
    DatabaseWindowOpen: bool,
    FlagWindowOpen: bool,
    atlas_texture: rl.Texture2D,
    TextureCache: map[cstring]rl.Texture2D,
    FrameTextures: [dynamic]FrameTexture,
    TransparencyTexture: rl.Texture2D,
    InvalidTexture: rl.Texture2D,
    Flag: Flag,
    ButtonIdentifier: i32
}

state := State{}

DatabaseState :: struct {
    Settings: FlagDatabase,
    BufferName: [128]byte,
    BufferNameLength: int,
    BufferPath: [512]byte,
    BufferPathLength: int,
}

FrameTexture :: struct {
    TransparencyBackground: bool,
    Path: cstring,
    Texture: rl.Texture2D,
}

main :: proc() {
    loadSettings()
    state.FrameTextures = make([dynamic]FrameTexture)
    state.TextureCache = make(map[cstring]rl.Texture2D)
    state.FlagWindowOpen = true

    rl.SetConfigFlags({.WINDOW_RESIZABLE})
    rl.InitWindow(1280, 800, "PDX Flag Editor")
    defer rl.CloseWindow()

    rl.SetWindowMinSize(FLAG_WIDTH, FLAG_HEIGHT + TOOLBAR_HEIGHT)


    pixels := make([][4]u8, mu.DEFAULT_ATLAS_WIDTH*mu.DEFAULT_ATLAS_HEIGHT)
    for alpha, i in mu.default_atlas_alpha {
        pixels[i] = {0xff, 0xff, 0xff, alpha}
    }
    defer delete(pixels)

    image := rl.Image{
        data = raw_data(pixels),
        width   = mu.DEFAULT_ATLAS_WIDTH,
        height  = mu.DEFAULT_ATLAS_HEIGHT,
        mipmaps = 1,
        format  = .UNCOMPRESSED_R8G8B8A8,
    }
    state.atlas_texture = rl.LoadTextureFromImage(image)
    defer rl.UnloadTexture(state.atlas_texture)

    state.TransparencyTexture = rl.LoadTexture("textures/transparency.dds")
    defer rl.UnloadTexture(state.TransparencyTexture)

    state.InvalidTexture = rl.LoadTexture("textures/invalid.dds")
    defer rl.UnloadTexture(state.InvalidTexture)

    ctx := &state.Context
    mu.init(ctx, set_clipboard, get_clipboard)

    ctx.text_width = mu.default_atlas_text_width
    ctx.text_height = mu.default_atlas_text_height

    rl.SetTargetFPS(120)
    main_loop: for !rl.WindowShouldClose() {
        { // text input
            text_input: [512]byte = ---
            text_input_offset := 0
            for text_input_offset < len(text_input) {
                ch := rl.GetCharPressed()
                if ch == 0 {
                    break
                }
                b, w := utf8.encode_rune(ch)
                copy(text_input[text_input_offset:], b[:w])
                text_input_offset += w
            }
            mu.input_text(ctx, string(text_input[:text_input_offset]))
        }

        // mouse coordinates
        mouse_pos := [2]i32{rl.GetMouseX(), rl.GetMouseY()}
        mu.input_mouse_move(ctx, mouse_pos.x, mouse_pos.y)
        mu.input_scroll(ctx, 0, i32(rl.GetMouseWheelMove() * -30))

        // mouse buttons
        @static buttons_to_key := [?]struct{
            rl_button: rl.MouseButton,
            mu_button: mu.Mouse,
        }{
            {.LEFT, .LEFT},
            {.RIGHT, .RIGHT},
            {.MIDDLE, .MIDDLE},
        }
        for button in buttons_to_key {
            if rl.IsMouseButtonPressed(button.rl_button) {
                mu.input_mouse_down(ctx, mouse_pos.x, mouse_pos.y, button.mu_button)
            } else if rl.IsMouseButtonReleased(button.rl_button) {
                mu.input_mouse_up(ctx, mouse_pos.x, mouse_pos.y, button.mu_button)
            }

        }

        // keyboard
        @static keys_to_check := [?]struct{
            rl_key: rl.KeyboardKey,
            mu_key: mu.Key,
        }{
            {.LEFT_SHIFT,    .SHIFT},
            {.RIGHT_SHIFT,   .SHIFT},
            {.LEFT_CONTROL,  .CTRL},
            {.RIGHT_CONTROL, .CTRL},
            {.LEFT_ALT,      .ALT},
            {.RIGHT_ALT,     .ALT},
            {.ENTER,         .RETURN},
            {.V,             .V},
            {.C,             .C},
            {.A,             .A},
            {.X,             .X},
            {.HOME,          .HOME},
            {.END,           .END},
            {.DELETE,        .DELETE},
            {.KP_ENTER,      .RETURN},
            {.BACKSPACE,     .BACKSPACE},
        }
        for key in keys_to_check {
            if rl.IsKeyPressed(key.rl_key) {
                mu.input_key_down(ctx, key.mu_key)
            } else if rl.IsKeyReleased(key.rl_key) {
                mu.input_key_up(ctx, key.mu_key)
            }
        }

        mu.begin(ctx)
        renderGui(ctx)
        mu.end(ctx)

        render(ctx)
        clear(&state.FrameTextures)
        state.ButtonIdentifier = 0
    }

    for key in state.TextureCache {
        rl.UnloadTexture(state.TextureCache[key])
    }
}

render :: proc(ctx: ^mu.Context) {
    render_texture :: proc(rect: mu.Rect, pos: [2]i32, color: mu.Color) {
        source := rl.Rectangle{
            f32(rect.x),
            f32(rect.y),
            f32(rect.w),
            f32(rect.h),
        }
        position := rl.Vector2{f32(pos.x), f32(pos.y)}

        rl.DrawTextureRec(state.atlas_texture, source, position, transmute(rl.Color)color)
    }

    rl.ClearBackground(transmute(rl.Color)state.Settings.BackgroundColor)

    rl.BeginDrawing()

    renderFlagPreview()

    rl.BeginScissorMode(0, 0, rl.GetScreenWidth(), rl.GetScreenHeight())
    defer rl.EndScissorMode()

    // Load textures needed by MicroUI
    loadFrameTextures()

    command_backing: ^mu.Command
    for variant in mu.next_command_iterator(ctx, &command_backing) {
        switch cmd in variant {
        case ^mu.Command_Text:
            pos := [2]i32{cmd.pos.x, cmd.pos.y}
            for ch in cmd.str do if ch&0xc0 != 0x80 {
                r := min(int(ch), 127)
                rect := mu.default_atlas[mu.DEFAULT_ATLAS_FONT + r]
                render_texture(rect, pos, cmd.color)
                pos.x += rect.w
            }
        case ^mu.Command_Rect:
            if isTextureRect(cmd.color) {
                renderTexture(cmd)
            } else {
                rl.DrawRectangle(cmd.rect.x, cmd.rect.y, cmd.rect.w, cmd.rect.h, transmute(rl.Color)cmd.color)
            }
        case ^mu.Command_Icon:
            rect := mu.default_atlas[cmd.id]
            x := cmd.rect.x + (cmd.rect.w - rect.w)/2
            y := cmd.rect.y + (cmd.rect.h - rect.h)/2
            render_texture(rect, {x, y}, cmd.color)
        case ^mu.Command_Clip:
            rl.EndScissorMode()
            rl.BeginScissorMode(cmd.rect.x, cmd.rect.y, cmd.rect.w, cmd.rect.h)
        case ^mu.Command_Jump:
            unreachable()
        }
    }

    defer rl.EndDrawing()
}

set_clipboard :: proc(user_data: rawptr, text: string) -> bool {
    rl.SetClipboardText(strings.clone_to_cstring(text))
    return true
}

get_clipboard :: proc(user_data: rawptr) -> (string, bool) {
    return string(rl.GetClipboardText()), true
}

loadFrameTextures :: proc() {
    for _, index in state.FrameTextures {
        texture, exists := state.TextureCache[state.FrameTextures[index].Path]
        if !exists {
            texture = rl.LoadTexture(state.FrameTextures[index].Path)
            state.TextureCache[state.FrameTextures[index].Path] = texture
            fmt.printfln("%i", texture.format)
            if texture.format == rl.PixelFormat.UNKNOWN {
                texture = state.InvalidTexture
                state.TextureCache[state.FrameTextures[index].Path] = texture
            }
        }
        state.FrameTextures[index].Texture = texture
    }
}

isTextureRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER
}

u8_slider :: proc(ctx: ^mu.Context, val: ^u8, lo, hi: u8) -> (res: mu.Result_Set) {
    mu.push_id(ctx, uintptr(val))

    @static tmp: mu.Real
    tmp = mu.Real(val^)
    res = mu.slider(ctx, &tmp, mu.Real(lo), mu.Real(hi), 0, "%.0f", {.ALIGN_CENTER})
    val^ = u8(tmp)
    mu.pop_id(ctx)
    return
}

renderTexture :: proc(cmd: ^mu.Command_Rect) {
    index := (int(cmd.color.r) << 16) | (int(cmd.color.g) << 8) | int(cmd.color.b)
    if len(state.FrameTextures) <= index {
        // Texture does not exist
        fmt.eprintfln("Texture index out of bounds")
        return
    }
    texture := state.FrameTextures[index]
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}
    if texture.TransparencyBackground {
        source := rl.Rectangle{0, 0, f32(state.TransparencyTexture.width), f32(state.TransparencyTexture.height)}
        rl.DrawTexturePro(state.TransparencyTexture, source, destination, { 0 , 0 }, 0, rl.WHITE)
    }
    source := rl.Rectangle{0, 0, f32(texture.Texture.width), f32(texture.Texture.height)}
    rl.DrawTexturePro(texture.Texture, source, destination, { 0 , 0 }, 0, rl.WHITE)
}

drawTexture :: proc(ctx: ^mu.Context, texturePath: string, width, height: i32, transparencyBackground: bool = false) {
    texture := FrameTexture{
        TransparencyBackground = transparencyBackground,
        Path = strings.clone_to_cstring(texturePath),
    }
    index := len(state.FrameTextures)
    append(&state.FrameTextures, texture)

    magicColor := mu.Color{
        r = u8((index >> 16) & 0xFF),
        g = u8((index >> 8) & 0xFF),
        b = u8(index & 0xFF),
        a = TEXTURE_RECT_IDENTIFIER,
    }

    rect := mu.layout_next(ctx)
    rect.w = width
    rect.h = height
    mu.draw_rect(ctx, rect, magicColor)
}

renderFlagPreview :: proc() {
    destination := rl.Rectangle{
        x = f32(rl.GetScreenWidth() - state.TransparencyTexture.width) / 2,
        y = f32(rl.GetScreenHeight() - state.TransparencyTexture.height + TOOLBAR_HEIGHT) / 2,
        width = f32(FLAG_WIDTH),
        height = f32(FLAG_HEIGHT),
    }
    // Draw Background
    {
        source := rl.Rectangle{
            x = 0,
            y = 0,
            width = f32(state.TransparencyTexture.width),
            height = f32(state.TransparencyTexture.height),
        }
        rl.DrawTexturePro(state.TransparencyTexture, source, destination, { 0, 0 }, 0, rl.WHITE)
    }
    if state.Flag.Pattern.Name != "" {
        texture := state.TextureCache[state.Flag.Pattern.Texture]
        source := rl.Rectangle{
            x = 0,
            y = 0,
            width = f32(texture.width),
            height = f32(texture.height),
        }
        rl.DrawTexturePro(texture, source, destination, { 0, 0 }, 0, rl.WHITE)
    }
}

setButtonIdentifier :: proc(ctx: ^mu.Context) {
    mu.push_id(ctx, fmt.tprintf("%i", state.ButtonIdentifier))
    state.ButtonIdentifier += 1
}