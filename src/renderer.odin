package pdx_flag_builder

import "core:unicode/utf8"
import "core:strings"
import rl "vendor:raylib"
import mu "vendor:microui"
import "core:fmt"
import "core:mem"
import "core:sync/chan"
import "texture"
import "core:thread"
import "core:os"
import "pdx"
import nfd "nativefiledialog"
import time "core:time"
import "core:math"

RELEASE :: #config(RELEASE, false)

APPLICATION_NAME :: "PDX Flag Builder"
APPLICATION_ICON :: "assets/icon.png"
APPLICATION_SPLASH_LOGO :: "assets/textures/logo.dds"
APPLICATION_SPLASH_SPINNER :: "assets/textures/spinner.dds"

THREAD_IDENTIFIER_TEXTURE_LOADING: int: 1
THREAD_IDENTIFIER_SPLASHSCREEN: int: 2

TEXTURE_RECT_IDENTIFIER: u8: 1
TEXTURE_RECT_IDENTIFIER_TRANSPARENCY: u8: 2
TEXTURE_RECT_IDENTIFIER_FLAG_PREVIEW: u8: 3
TEXTURE_RECT_IDENTIFIER_FLAG_DRAW: u8: 4

createTextureLoadingThread :: proc() -> ^thread.Thread {
    imageLoader :: proc(t: ^thread.Thread) {
        for !state.Done {
            request, ok := chan.try_recv(state.ImageLoadChannel)
            if !ok {
                continue
            }
            if _, exists := state.RenderTextureCache[request.Identifier]; exists {
                fmt.eprintfln("duplicate image request: %s", request.Identifier)
                continue
            }
            path := strings.clone(request.Path)
            image := texture.LoadImage(request.Path)
            textureRequest := TextureRequest{
                Identifier = request.Identifier,
                Path = path,
                Image = image
            }
            delete(request.Path)
            chan.send(state.TextureLoadChannel, textureRequest)
        }
    }
    if imageThread := thread.create(imageLoader); imageThread != nil {
        imageThread.init_context = context
        imageThread.user_index = THREAD_IDENTIFIER_TEXTURE_LOADING
        return imageThread
    } else {
        fmt.eprintfln("Could not create image loading thread")
        os.exit(2)
    }
}

handleTextureRequests :: proc() {
    for {
        request, ok := chan.try_recv(state.TextureLoadChannel)
        if !ok {
            break
        }
        if _, exists := state.RenderTextureCache[request.Identifier]; exists {
            fmt.eprintfln("duplicate texture request: %s", request.Identifier)
            continue
        }
        texture := rl.LoadTextureFromImage(request.Image)
        path := strings.clone(request.Path)
        state.GuiTextureCache[path] = texture
        state.RenderTextureMap[path] = texture
        state.RenderTextureCache[request.Identifier] = texture
        rl.UnloadImage(request.Image)
        delete(request.Path)
    }
}

handleFlagRequests :: proc() {
    for {
        request, ok := chan.try_recv(state.FlagLoadChannel)
        if !ok {
            break
        }
        if _, exists := state.RenderFlagMap[request.Identifier]; exists {
            fmt.eprintfln("duplicate flag request: %s", request.Identifier)
            continue
        }
        flag, exists := state.Flags[request.Name]
        if !exists {
            fmt.eprintfln("non existing flag requested: %s", request.Name)
            delete(request.Name)
            continue
        }
        state.RenderFlagMap[request.Identifier] = flag
        delete(request.Name)
    }
}

handleFlagExportRequests :: proc() {
    for {
    request, ok := chan.try_recv(state.FlagExportChannel)
        if !ok {
            break
        }
        switch request.Type {
        case .Image:
            executeExportFlagImage(request)
        case .Script:
            fmt.eprintln("Not Supported Yet")
        }
    }
}

executeExportFlagImage :: proc(request: FlagExportRequest) {
    path: cstring
    filters := [1]nfd.Filter_Item { { "Image", "png" } }
    args := nfd.Save_Dialog_Args {
        filter_list = raw_data(filters[:]),
        filter_count = len(filters)
    }

    result := nfd.SaveDialogU8_With(&path, &args)
    if result == .Cancel {
        return
    }
    if result == .Error {
        fmt.eprintln("Export dialog error:", nfd.GetError())
        return
    }

    target := rl.LoadRenderTexture(request.Size[0], request.Size[1]);
    destination := rl.Rectangle{0, 0, f32(request.Size[0]), f32(request.Size[1])}
    rl.BeginTextureMode(target);
    rl.ClearBackground(rl.WHITE);
    renderFlag(request.Flag, destination)
    rl.EndTextureMode();

    image := rl.LoadImageFromTexture(target.texture);
    success := rl.ExportImage(image, path);
    rl.UnloadImage(image);
    rl.UnloadRenderTexture(target);
    if !success {
        fmt.eprintfln("Could not export image")
    } else {
        fmt.printfln("Successfully exported")
    }
    nfd.FreePathU8(path)
}

createSplashScreenThread :: proc() -> ^thread.Thread {
    loader :: proc(t: ^thread.Thread) {
        pdx.setupParsing()
        CreateState()
        time.sleep(time.Millisecond * 500)
    }
    if loaderThread := thread.create(loader); loaderThread != nil {
        loaderThread.init_context = context
        loaderThread.user_index = THREAD_IDENTIFIER_SPLASHSCREEN
        return loaderThread
    } else {
        fmt.eprintfln("Could not create init thread")
        os.exit(2)
    }
}

showSplashScreen :: proc() {
    rl.SetConfigFlags({ .WINDOW_UNDECORATED, .WINDOW_TOPMOST, .WINDOW_TRANSPARENT })
    rl.InitWindow(212, 212, APPLICATION_NAME)

    icon := rl.LoadImage(APPLICATION_ICON)
    rl.ImageFormat(&icon, .UNCOMPRESSED_R8G8B8A8)
    rl.SetWindowIcon(icon)
    rl.UnloadImage(icon)
    logo := rl.LoadTexture(APPLICATION_SPLASH_LOGO)
    spinner := rl.LoadTexture(APPLICATION_SPLASH_SPINNER)

    loaderThread := createSplashScreenThread()
    thread.start(loaderThread)

    rl.SetTargetFPS(60)
    rotation: f32
    splashLoop: for !rl.WindowShouldClose() {
        if thread.is_done(loaderThread) {
            break
        }
        rl.BeginDrawing()
        rl.ClearBackground(rl.BLANK)
        rl.BeginScissorMode(0, 0, rl.GetScreenWidth(), rl.GetScreenHeight())

        source := rl.Rectangle{
            x = 0,
            y = 0,
            width = f32(logo.width),
            height = f32(logo.height),
        }
        destination := rl.Rectangle{
            x = f32(rl.GetScreenWidth() / 2),
            y = f32(rl.GetScreenHeight() / 2),
            width = f32(logo.width),
            height = f32(logo.height),
        }
        origin := rl.Vector2{
            destination.width * 0.5,
            destination.height * 0.5,
        }

        rl.DrawTexturePro(logo, source, destination, origin, 0, rl.WHITE)
        rl.DrawTexturePro(spinner, source, destination, origin, rotation, rl.WHITE)


        rl.EndScissorMode()
        rl.EndDrawing()
        rotation += 1
        if rotation > 360 {
            rotation = 0
        }
    }

    state.Done = false
    thread.destroy(loaderThread)

    rl.CloseWindow()
}

main :: proc() {
    when !RELEASE {
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
    }
    rl.SetTraceLogLevel(.WARNING)

    nfd.Init()
    defer nfd.Quit()

    showSplashScreen()
    defer pdx.destroyParsing()
    defer DestroyState()

    rl.SetConfigFlags({ .WINDOW_RESIZABLE })
    rl.InitWindow(1280, 800, APPLICATION_NAME)
    rl.ClearWindowState({ .WINDOW_UNDECORATED, .WINDOW_TOPMOST })
    defer rl.CloseWindow()

    icon := rl.LoadImage(APPLICATION_ICON)
    rl.ImageFormat(&icon, .UNCOMPRESSED_R8G8B8A8)
    rl.SetWindowIcon(icon)
    rl.UnloadImage(icon)

    state.RecolorShader = texture.LoadRecolorShader(SHADER_RECOLOR)
    defer rl.UnloadShader(state.RecolorShader.Shader)

    pixels := make([][4]u8, mu.DEFAULT_ATLAS_WIDTH * mu.DEFAULT_ATLAS_HEIGHT)
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

    state.TransparencyTexture = texture.LoadTexture(TEXTURE_TRANSPARENCY)
    defer rl.UnloadTexture(state.TransparencyTexture)

    state.InvalidTexture = texture.LoadTexture(TEXTURE_INVALID)
    defer rl.UnloadTexture(state.InvalidTexture)

    ctx := &state.Context
    mu.init(ctx, set_clipboard, get_clipboard)

    ctx.text_width = mu.default_atlas_text_width
    ctx.text_height = mu.default_atlas_text_height

    thread.start(state.TextureLoadingThread)

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
            {.LEFT,          .LEFT},
            {.RIGHT,         .RIGHT},
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
        handleTextureRequests()
        handleFlagRequests()
        handleFlagExportRequests()
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
            case isFlagRect(cmd.color):
                renderFlagDraw(cmd)
            case isFlagPreviewRect(cmd.color):
                renderFlagPreviewTexture(cmd)
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

getTextureIdentifier :: proc(path: string) -> (int, bool) {
    if identifier, ok := state.TextureMap[path]; ok {
        return identifier, true
    }
    identifier := state.NextTextureIdentifier
    state.NextTextureIdentifier += 1
    state.TextureMap[strings.clone(path)] = identifier
    request := ImageRequest{
        Identifier = identifier,
        Path = strings.clone(path),
    }
    ok := chan.try_send(state.ImageLoadChannel, request)
    if !ok {
        state.NextTextureIdentifier -= 1
        delete(request.Path)
    }
    return identifier, !ok
}

getFlagIdentifier :: proc(name: string) -> (int, bool) {
    if identifier, ok := state.GuiFlagMap[name]; ok {
        return identifier, true
    }
    // Request Flag textures
    flag, exists := state.Flags[name]
    if exists {
        if flag.Pattern.Path != "" {
            getTextureIdentifier(flag.Pattern.Path)
        }
        for variant in flag.Layers {
            switch layer in variant {
            case ^pdx.FlagLayerTexturedEmblem:
                if layer.Texture.Path != "" {
                    getTextureIdentifier(layer.Texture.Path)
                }
            case ^pdx.FlagLayerColoredEmblem:
                if layer.Texture.Path != "" {
                    getTextureIdentifier(layer.Texture.Path)
                }
            }
        }
    }
    identifier := state.NextFlagIdentifier
    state.NextFlagIdentifier += 1
    state.GuiFlagMap[strings.clone(name)] = identifier
    request := FlagRequest{
        Identifier = identifier,
        Name = strings.clone(name),
    }
    ok := chan.try_send(state.FlagLoadChannel, request)
    if !ok {
        state.NextFlagIdentifier -= 1
        delete(request.Name)
    }
    return identifier, !ok
}

isTextureRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER
}
isTextureWithTransparencyRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER_TRANSPARENCY
}
isFlagRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER_FLAG_DRAW
}
isFlagPreviewRect :: proc(textureId: mu.Color) -> bool {
    return textureId.a == TEXTURE_RECT_IDENTIFIER_FLAG_PREVIEW
}

guranteeBounds :: proc(ctx: ^mu.Context) {
    window := mu.get_current_container(ctx)
    sidebarWidth := state.SidebarWidth + SIDEBAR_HANDLE_WIDTH

    if (window.rect.h + TOOLBAR_HEIGHT) > rl.GetScreenHeight() {
        window.rect.h = rl.GetScreenHeight() - TOOLBAR_HEIGHT
    }
    if (window.rect.w + sidebarWidth) > rl.GetScreenWidth() {
        window.rect.w = rl.GetScreenWidth() - sidebarWidth
    }
    if (window.rect.x + window.rect.w) > rl.GetScreenWidth() {
        window.rect.x = rl.GetScreenWidth() - window.rect.w
    }
    if (window.rect.y + window.rect.h) > rl.GetScreenHeight() {
        window.rect.y = rl.GetScreenHeight() - window.rect.h
    }
    if window.rect.x > (rl.GetScreenWidth() - sidebarWidth - window.rect.w - 2) {
        window.rect.x = rl.GetScreenWidth() - sidebarWidth - window.rect.w - 2
    }
    if window.rect.x < 0 {
        window.rect.x = 0
    }
    if window.rect.y < TOOLBAR_HEIGHT {
        window.rect.y = TOOLBAR_HEIGHT
    }
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

renderFlagDraw :: proc(cmd: ^mu.Command_Rect) {
    index := (int(cmd.color.r) << 16) | (int(cmd.color.g) << 8) | int(cmd.color.b)
    flag, exists := state.RenderFlagMap[index]
    if !exists {
        return
    }
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}
    renderFlag(flag, destination)
}

renderFlagPreviewTexture :: proc(cmd: ^mu.Command_Rect) {
    destination := rl.Rectangle{f32(cmd.rect.x), f32(cmd.rect.y), f32(cmd.rect.w), f32(cmd.rect.h)}
    renderFlag(state.Flag, destination)
}

renderFlag :: proc(flag: pdx.Flag, destination: rl.Rectangle) {
    rl.BeginScissorMode(i32(destination.x), i32(destination.y), i32(destination.width), i32(destination.height))

    if flag.Pattern.Name != "" {
        pattern, ok := state.RenderTextureMap[flag.Pattern.Path]

        if ok {
            source := rl.Rectangle{0, 0, f32(pattern.width), f32(pattern.height)}

            colorMappings := make([dynamic]texture.ColorRecolor)
            defer delete(colorMappings)
            for color in flag.Colors {
                fillColorMapping(&colorMappings, color, flag, pdx.PATTERN_REPLACE_COLORS)
            }

            texture.DrawRecoloredTexture(
                pattern,
                source,
                destination,
                colorMappings[:],
                state.RecolorShader,
            )
        }
    }

    for variant in flag.Layers {
        switch layer in variant {
        case ^pdx.FlagLayerColoredEmblem:
            renderColoredEmblemInstances(layer, flag, destination)
        case ^pdx.FlagLayerTexturedEmblem:
            renderTexturedEmblemInstances(layer, destination)
        }
    }

    rl.EndScissorMode()
}

renderColoredEmblemInstances :: proc(layer: ^pdx.FlagLayerColoredEmblem, flag: pdx.Flag, destination: rl.Rectangle) {
    emblem, ok := state.RenderTextureMap[layer.Texture.Path]
    if !ok {
        return
    }
    source := rl.Rectangle{0, 0, f32(emblem.width), f32(emblem.height)}

    colorMappings := make([dynamic]texture.ColorRecolor)
    defer delete(colorMappings)
    for color in layer.Colors {
        fillColorMapping(&colorMappings, color, flag, pdx.COLORED_EMBLEM_REPLACE_COLORS)
    }

    for instance in layer.Instances {
        target := calculateInstanceDestination(emblem, instance, destination)
        texture.DrawRecoloredTexture(
            emblem,
            source,
        target,
            colorMappings[:],
            state.RecolorShader,
            calculateInstanceOrigin(emblem, instance, target),
            f32(instance.Rotation),
        )
    }

    if len(layer.Instances) == 0 {
        instance := pdx.LayerInstance{
            Rotation = 0,
            Position = pdx.DEFAULT_POSITION,
            Scale = pdx.DEFAULT_SCALE,
        }
        target := calculateInstanceDestination(emblem, &instance, destination)
        texture.DrawRecoloredTexture(
            emblem,
            source,
            target,
            colorMappings[:],
            state.RecolorShader,
            calculateInstanceOrigin(emblem, &instance, target),
            f32(instance.Rotation),
        )
    }
}

renderTexturedEmblemInstances :: proc(layer: ^pdx.FlagLayerTexturedEmblem, destination: rl.Rectangle) {
    emblem, ok := state.RenderTextureMap[layer.Texture.Path]
    if !ok {
        return
    }
    source := rl.Rectangle{0, 0, f32(emblem.width), f32(emblem.height)}

    for instance in layer.Instances {
        target := calculateInstanceDestination(emblem, instance, destination)
        rl.DrawTexturePro(
            emblem,
            source,
            target,
            calculateInstanceOrigin(emblem, instance, target),
            f32(instance.Rotation),
            rl.WHITE
        )
    }

    if len(layer.Instances) == 0 {
        instance := pdx.LayerInstance{
            Rotation = 0,
            Position = pdx.DEFAULT_POSITION,
            Scale = pdx.DEFAULT_SCALE,
        }
        target := calculateInstanceDestination(emblem, &instance, destination)
        rl.DrawTexturePro(
            emblem,
            source,
            target,
            calculateInstanceOrigin(emblem, &instance, target),
            f32(instance.Rotation),
            rl.WHITE
        )
    }
}

calculateInstanceOrigin :: proc(texture: rl.Texture2D, instance: ^pdx.LayerInstance, target: rl.Rectangle) -> rl.Vector2 {
    return rl.Vector2{
        target.width * 0.5,
        target.height * 0.5,
    }
}

calculateInstanceDestination :: proc(texture: rl.Texture2D, instance: ^pdx.LayerInstance, target: rl.Rectangle) -> rl.Rectangle {
    // handle buggy stretching/squishing
    // seems slightly off, may need future adjustment
    angle := math.mod(f32(instance.Rotation), 180.0)
    if angle < 0 {
        angle += 180.0
    }
    stretch := 1.0 + 0.5 * (1.0 - math.abs((angle - 90.0) / 90.0))
    squish := 1.0 - 0.25 * (1.0 - math.abs((angle - 90.0) / 90.0))

    width: f32 = target.width * instance.Scale.X * squish
    height: f32 = target.height * instance.Scale.Y * stretch

    // Default position 0.5 is centered
    // The position then moves the instance by the flag width/height
    x := target.x + ((target.width - width) / 2) + (target.width * (instance.Position.X - 0.5)) + (width * 0.5)
    y := target.y + ((target.height - height) / 2) + (target.height * (instance.Position.Y - 0.5)) + (height * 0.5)

    return rl.Rectangle{ x, y, width, height }
}

drawTexture :: proc(ctx: ^mu.Context, texturePath: string, width: i32 = 0, height: i32 = 0) {
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
    if width != 0 {
        rect.w = width
    }
    if width != 0 {
        rect.h = height
    }
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

drawFlagPreview :: proc(ctx: ^mu.Context, width: i32 = 0, height: i32 = 0) {
    magicColor := mu.Color{
        r = 0,
        g = 0,
        b = 0,
        a = TEXTURE_RECT_IDENTIFIER_FLAG_PREVIEW,
    }

    rect := mu.layout_next(ctx)
    rect.x = rect.x - ctx.style.padding
    rect.y = rect.y - ctx.style.padding
    if width != 0 {
        rect.w = width
    }
    if width != 0 {
        rect.h = height
    }
    mu.draw_rect(ctx, rect, magicColor)
}

drawFlag :: proc(ctx: ^mu.Context, flagName: string, width: i32 = 0, height: i32 = 0) {
    index, ok := getFlagIdentifier(flagName)
    if !ok {
        return
    }

    magicColor := mu.Color{
        r = u8((index >> 16) & 0xFF),
        g = u8((index >> 8) & 0xFF),
        b = u8(index & 0xFF),
        a = TEXTURE_RECT_IDENTIFIER_FLAG_DRAW,
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

exportFlagImage :: proc(ctx: ^mu.Context, flag: pdx.Flag, width: i32 = 0, height: i32 = 0) {
    request := FlagExportRequest{
        Flag = flag,
        Size = { FLAG_WIDTH, FLAG_HEIGHT },
        Type = .Image
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