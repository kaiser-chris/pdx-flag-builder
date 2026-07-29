package ui

import mu "vendor:microui"
import strings "core:strings"

WINDOW_DROPDOWN: string: "dropdown-layer"

DropdownAction :: #type proc(ctx: ^mu.Context)

DropdownElement :: struct {
    Label: string,
    Action: DropdownAction,
}

Dropdown :: struct {
    Location: mu.Vec2,
    Width: i32,
    Elements: [dynamic]DropdownElement,
    IgnoreUntilMouseUp: bool,
}

CreateDropdownElement :: proc(label: string, action: DropdownAction) -> DropdownElement {
    return DropdownElement{
        strings.clone(label),
        action,
    }
}
DestroyDropdownElement :: proc(element: ^DropdownElement) {
    delete(element.Label)
}

CreateDropdown :: proc() -> Dropdown {
    return Dropdown{
        Elements = make([dynamic]DropdownElement)
    }
}
DestroyDropdown :: proc(dropdown: ^Dropdown) {
    for _, index in dropdown.Elements {
        DestroyDropdownElement(&dropdown.Elements[index])
    }
    delete(dropdown.Elements)
}

OpenDropdown :: proc(dropdown: ^Dropdown, location: mu.Vec2, elements: []DropdownElement, width: i32 = 0) {
    dropdown.IgnoreUntilMouseUp = true
    dropdown.Width = width
    dropdown.Location = location
    append(&dropdown.Elements, ..elements)
}

CloseDropdown :: proc(dropdown: ^Dropdown) {
    for _, index in dropdown.Elements {
        DestroyDropdownElement(&dropdown.Elements[index])
    }
    clear(&dropdown.Elements)
}

InitDropdown :: proc(ctx: ^mu.Context, dropdown: ^Dropdown, buttonIdCounter: ^i32) {
    // Only open when it has elements
    if len(dropdown.Elements) <= 0 {
        return
    }

    if mu.window(ctx, WINDOW_DROPDOWN, { 0, 0, 0, 0 }, {.NO_CLOSE, .NO_TITLE, .NO_RESIZE, .AUTO_SIZE}) {
        window := mu.get_current_container(ctx)
        window.rect.x = dropdown.Location.x
        window.rect.y = dropdown.Location.y
        if dropdown.Width != 0 {
            window.rect.w = dropdown.Width
            mu.layout_row(ctx, {-1})
        }
        for element in dropdown.Elements {
            SetButtonIdentifier(ctx, buttonIdCounter)
            if .SUBMIT in mu.button(ctx, element.Label) {
                element.Action(ctx)
                CloseDropdown(dropdown)
            }
            mu.pop_id(ctx)
        }
    }

    // Close when clicking somewhere else
    cnt := mu.get_container(ctx, WINDOW_DROPDOWN)
    if dropdown.IgnoreUntilMouseUp {
        if !(.LEFT in ctx.mouse_down_bits) {
            dropdown.IgnoreUntilMouseUp = false
        }
    } else if cnt != nil {
        if ctx.hover_root != cnt && .LEFT in ctx.mouse_down_bits {
            CloseDropdown(dropdown)
        }
    }

}