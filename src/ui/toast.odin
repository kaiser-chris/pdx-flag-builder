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

    if mu.window(ctx, WINDOW_TOAST, destination, { .NO_CLOSE, .NO_TITLE, .NO_RESIZE, .AUTO_SIZE, .NO_FRAME }) {
        window := mu.get_current_container(ctx)
        window.rect = destination

        mu.layout_row(ctx, { -1 }, 50)
        for message, index in toast.Messages {
            msToExpirary := time.duration_milliseconds(time.diff(message.Expirary, time.now()))
            if (msToExpirary > 0) {
                DestroyToast(toast.Messages[index])
                ordered_remove(&toast.Messages, index)
                continue
            }

            toastRect := mu.layout_next(ctx)
            toastRect.x -= ctx.style.padding
            toastRect.w += 2 * ctx.style.padding
            toastRect.y -= ctx.style.padding

            headerRect := mu.Rect{
                x = toastRect.x,
                y = toastRect.y,
                w = toastRect.w,
                h = 20
            }
            mu.draw_rect(ctx, headerRect, ctx.style.colors[.TITLE_BG])
            mu.draw_control_text(ctx, message.Header, headerRect, .TEXT)

            bodyRect := mu.Rect{
                x = toastRect.x,
                y = toastRect.y + headerRect.h,
                w = toastRect.w,
                h = 30
            }
            mu.draw_rect(ctx, bodyRect, ctx.style.colors[.WINDOW_BG])
            textRect := mu.Rect{
                x = bodyRect.x + ctx.style.padding,
                y = bodyRect.y + ctx.style.padding,
                w = bodyRect.w - (2 * ctx.style.padding),
                h = bodyRect.h - (2 * ctx.style.padding),
            }
            mu.draw_control_text(ctx, message.Message, textRect, .TEXT)
        }
    }
}
