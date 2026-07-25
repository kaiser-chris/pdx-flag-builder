package pdx_flag_builder

import "core:unicode/utf8"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "core:fmt"
import "core:mem"
import "core:sync/chan"
import "texture"

TEXTURE_RECT_IDENTIFIER: u8: 1
TEXTURE_RECT_IDENTIFIER_TRANSPARENCY: u8: 2

State :: struct {
    Settings: Settings,
    Databases: [dynamic]DatabaseState,
    Context: mu.Context,
    SettingsWindowOpen: bool,
    DatabaseWindowOpen: bool,
    DatabaseSearch: string,
    SidebarOpen: bool,
    SidebarWidth: i32,
    AtlasTexture: rl.Texture2D,
    RenderTextureCache: map[int]rl.Texture2D,
    GuiTextureCache: map[string]rl.Texture2D,
    TextureMap: map[string]int,
    TextureLoadChannel: chan.Chan(TextureRequest),
    TransparencyTexture: rl.Texture2D,
    InvalidTexture: rl.Texture2D,
    Flag: Flag,
    ButtonIdentifier: i32,
    NextTextureIdentifier: int,
    SelectedFlagElement: SelectedFlagElement,
}

setupState :: proc() {
    state.Databases = make([dynamic]DatabaseState)
    state.RenderTextureCache = make(map[int]rl.Texture2D)
    state.GuiTextureCache = make(map[string]rl.Texture2D)
    state.TextureMap = make(map[string]int)
    channel, err := chan.create(chan.Chan(TextureRequest), 2048, context.allocator)
    if err != nil {
        fmt.eprintfln("Could not create TextureLoadChannel: %v", err)
    }
    state.TextureLoadChannel = channel
    state.SidebarOpen = true
    state.SidebarWidth = 300
    state.Flag = createFlag()
    state.NextTextureIdentifier = 1
}
destroyState :: proc() {
    for _, index in state.Databases {
        destroyNamedColors(&state.Databases[index])
    }
    for key in state.RenderTextureCache {
        rl.UnloadTexture(state.RenderTextureCache[key])
    }
    delete(state.RenderTextureCache)
    for key in state.GuiTextureCache {
        delete(key)
    }
    delete(state.GuiTextureCache)
    for key in state.TextureMap {
        delete(key)
    }
    delete(state.TextureMap)
    delete(state.Databases)
    delete(state.DatabaseSearch)
    chan.destroy(state.TextureLoadChannel)
    destroyFlag(state.Flag)
}

state := State{}

DatabaseState :: struct {
    Settings: FlagDatabase,
    BufferName: [128]byte,
    BufferNameLength: int,
    BufferPath: [512]byte,
    BufferPathLength: int,
    NamedColors: [dynamic]FlagColorVariant,
}

TextureRequest :: struct {
    Identifier: int,
    Path: string,
}

FrameTexture :: struct {
    TransparencyBackground: bool,
    Path: cstring,
    Texture: rl.Texture2D,
}

main :: proc() {
    track: mem.Tracking_Allocator
    mem.tracking_allocator_init(&track, context.allocator)
    context.allocator = mem.tracking_allocator(&track)

    // Check for leaks
    defer {
        if len(track.allocation_map) > 0 {
            fmt.printfln("Real Memory Leak! %v allocations not freed.", len(track.allocation_map))
            for _, entry in track.allocation_map {
                fmt.printfln("- %v bytes at %v", entry.size, entry.location)
            }
        } else {
            fmt.println("No leaks detected! Memory is safely pooled by the allocator.")
        }
    }

    setupParsing()
    defer destroyParsing()
    setupState()
    defer destroyState()
    loadSettings()
    defer destroySettings()

    rl.SetConfigFlags({.WINDOW_RESIZABLE})
    rl.InitWindow(1280, 800, "PDX Flag Editor")
    defer rl.CloseWindow()

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
    state.AtlasTexture = rl.LoadTextureFromImage(image)
    defer rl.UnloadTexture(state.AtlasTexture)

    state.TransparencyTexture = texture.LoadTexture("textures/transparency.dds")
    defer rl.UnloadTexture(state.TransparencyTexture)

    state.InvalidTexture = texture.LoadTexture("textures/invalid.dds")
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

        state.ButtonIdentifier = 0
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

        rl.DrawTextureRec(state.AtlasTexture, source, position, transmute(rl.Color)color)
    }

    rl.ClearBackground(transmute(rl.Color)state.Settings.BackgroundColor)

    // Load textures needed by MicroUI
    loadRequestedTextures()

    rl.BeginDrawing()

    rl.BeginScissorMode(0, 0, rl.GetScreenWidth(), rl.GetScreenHeight())
    defer rl.EndScissorMode()

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
            switch {
            case isTextureRect(cmd.color):
                renderTexture(cmd)
            case isTextureWithTransparencyRect(cmd.color):
                renderTransparentTexture(cmd)
            case:
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

loadRequestedTextures :: proc() {
    for {
        request, ok := chan.try_recv(state.TextureLoadChannel)
        if !ok {
            break
        }
        if _, exists := state.RenderTextureCache[request.Identifier]; exists {
            fmt.eprintfln("duplicate request: %s", request.Identifier)
            continue
        }
        texture := texture.LoadTexture(request.Path)
        state.GuiTextureCache[request.Path] = texture
        state.RenderTextureCache[request.Identifier] = texture
    }
}

// When the UI needs to draw an image:
getTextureIdentifier :: proc(path: string) -> (int, bool) {
    if identifier, ok := state.TextureMap[path]; ok {
        return identifier, true
    }

    // It's a new texture! Generate an ID.
    identifier := state.NextTextureIdentifier
    state.NextTextureIdentifier += 1

    // Store it in the UI cache
    state.TextureMap[strings.clone(path)] = identifier

    // Send a load request to the render thread.
    // We clone the string so the render thread can safely use and delete it.
    request := TextureRequest{
        Identifier = identifier,
        Path = strings.clone(path),
    }
    ok := chan.try_send(state.TextureLoadChannel, request)
    if !ok {
        fmt.eprintfln("Could not request texture load: %s", path)
        state.NextTextureIdentifier -= 1
    }

    return identifier, !ok
}

isTextureRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER
}
isTextureWithTransparencyRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER_TRANSPARENCY
}

guranteeBounds :: proc(ctx: ^mu.Context) {
    window := mu.get_current_container(ctx)
    if (window.rect.h + TOOLBAR_HEIGHT) > rl.GetScreenHeight() {
        window.rect.h = rl.GetScreenHeight() - TOOLBAR_HEIGHT
    }
    if window.rect.w > rl.GetScreenWidth() {
        window.rect.w = rl.GetScreenWidth()
    }
    if (window.rect.x + window.rect.w) > rl.GetScreenWidth() {
        window.rect.x = rl.GetScreenWidth() - window.rect.w
    }
    if (window.rect.y + window.rect.h) > rl.GetScreenHeight() {
        window.rect.y = rl.GetScreenHeight() - window.rect.h
    }
    if window.rect.x < 0 {
        window.rect.x = 0
    }
    if window.rect.y < TOOLBAR_HEIGHT {
        window.rect.y = TOOLBAR_HEIGHT
    }
}

colorSlider :: proc(ctx: ^mu.Context, val: ^u8, lo, hi: u8) -> (res: mu.Result_Set) {
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
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}
    source := rl.Rectangle{0, 0, f32(state.TransparencyTexture.width), f32(state.TransparencyTexture.height)}
    rl.DrawTexturePro(state.TransparencyTexture, source, destination, { 0 , 0 }, 0, rl.WHITE)
    if texture, ok := state.RenderTextureCache[index]; ok {
        source := rl.Rectangle{0, 0, f32(texture.width), f32(texture.height)}
        rl.DrawTexturePro(texture, source, destination, { 0, 0 }, 0, rl.WHITE)
    }
}

renderTransparentTexture :: proc(cmd: ^mu.Command_Rect) {
    index := (int(cmd.color.r) << 16) | (int(cmd.color.g) << 8) | int(cmd.color.b)
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}
    source := rl.Rectangle{0, 0, f32(state.TransparencyTexture.width), f32(state.TransparencyTexture.height)}
    if texture, ok := state.RenderTextureCache[index]; ok {
        source := rl.Rectangle{0, 0, f32(texture.width), f32(texture.height)}
        rl.DrawTexturePro(texture, source, destination, { 0, 0 }, 0, rl.WHITE)
    }
}

drawTexture :: proc(ctx: ^mu.Context, texturePath: string, width, height: i32) {
    index, ok := getTextureIdentifier(texturePath)
    if !ok {
        return
    }

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

drawTransparentTexture :: proc(ctx: ^mu.Context, texturePath: string, width: i32 = 0, height: i32 = 0) {
    index, ok := getTextureIdentifier(texturePath)
    if !ok {
        return
    }

    magicColor := mu.Color{
        r = u8((index >> 16) & 0xFF),
        g = u8((index >> 8) & 0xFF),
        b = u8(index & 0xFF),
        a = TEXTURE_RECT_IDENTIFIER_TRANSPARENCY,
    }

    rect := mu.layout_next(ctx)
    if width != 0 {
        rect.w = width
    }
    if width != 0 {
        rect.h = height
    }
    mu.draw_rect(ctx, rect, magicColor)
}

setButtonIdentifier :: proc(ctx: ^mu.Context) {
    mu.push_id(ctx, fmt.tprintf("%i", state.ButtonIdentifier))
    state.ButtonIdentifier += 1
}

createFlag :: proc() -> Flag {
    return Flag{
        Colors = make([dynamic]FlagColorVariant),
        Layers = make([dynamic]FlagLayerVariant),
    }
}