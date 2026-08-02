package ui

import "core:time"
import mu "vendor:microui"
import strings "core:strings"

WINDOW_TOAST: string: "toast-layer"

ToastType :: enum {
    Info,
    Warning,
    Danger,
}

ToastContainer :: struct {
    Messages: [dynamic]ToastMessage,
}

ToastMessage :: struct {
    Type: ToastType,
    Header: string,
    Message: string,
    Expirary: time.Time
}

CreateToastContainer :: proc() -> ToastContainer {
    return ToastContainer{
        Messages = make([dynamic]ToastMessage)
    }
}
DestroyToastContainer :: proc(container: ^ToastContainer) {
    for _, index in container.Messages {
        DestroyToast(container.Messages[index])
    }
    delete(container.Messages)
}

Toast :: proc(container: ^ToastContainer, header, message: string, type: ToastType = .Info, expiraryMilliseconds: time.Duration = 1000) {
    expirary := time.time_add(time.now(), expiraryMilliseconds * time.Millisecond)
    toast := ToastMessage{
        Type = type,
        Header = strings.clone(header),
        Message = strings.clone(message),
        Expirary = expirary,
    }
    append(&container.Messages, toast)
}
DestroyToast :: proc(toast: ToastMessage) {
    delete(toast.Header)
    delete(toast.Message)
}

InitToast :: proc(ctx: ^mu.Context, toast: ^ToastContainer, destination: mu.Rect, buttonIdCounter: ^i32) {
    // Only open when it has elements
    if len(toast.Messages) <= 0 {
        return
    }

    if mu.window(ctx, WINDOW_TOAST, destination, { .NO_CLOSE, .AUTO_SIZE, .NO_TITLE, .NO_RESIZE, .NO_FRAME }) {
        window := mu.get_current_container(ctx)
        window.rect = destination
        window.rect.x -= ctx.style.padding
        window.rect.y -= ctx.style.padding

        for message, index in toast.Messages {
            msToExpirary := time.duration_milliseconds(time.diff(message.Expirary, time.now()))
            if (msToExpirary > 0) {
                DestroyToast(toast.Messages[index])
                ordered_remove(&toast.Messages, index)
                continue
            }

            mu.layout_row(ctx, { -1 }, 20)

            headerRect := mu.layout_next(ctx)
            headerRect.h += ctx.style.padding
            mu.draw_rect(ctx, headerRect, ctx.style.colors[.TITLE_BG])
            mu.draw_control_text(ctx, message.Header, headerRect, .TEXT)

            font := ctx.style.font
            mu.layout_row(ctx, { -1 }, ctx.text_height(font))
            lines := strings.split_lines(message.Message)
            for line in lines {
                r := mu.layout_next(ctx)
                r.h += ctx.style.padding
                mu.draw_rect(ctx, r, ctx.style.colors[.WINDOW_BG])
                mu.draw_control_text(ctx, line, r, .TEXT)
            }

            mu.layout_row(ctx, { -1 }, 5)
            mu.layout_next(ctx)
        }
    }
}
