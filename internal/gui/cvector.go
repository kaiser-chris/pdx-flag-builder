package gui

// Lists Dear ImGui keeps, read the way cimgui-go's generated code should.
//
// cimgui-go's accessors for a list of pointers, such as DrawData.Textures and
// Context.Windows, dereference the C array to its first element and wrap that
// alone, so Slice reads past the end of a single Go value as soon as the list
// holds two entries. These functions read the same lists through the C
// accessors cimgui-go already compiles in.
//
// The declarations below restate those accessors with an equivalent type
// rather than including cimgui-go's header, which lives in the module cache
// where cgo cannot point portably. The layout of an ImVector (a size, a
// capacity, a pointer to the elements) does not change.

/*
typedef struct {
	int Size;
	int Capacity;
	void **Data;
} gui_pointer_list;

extern gui_pointer_list *wrap_ImDrawData_GetTextures(void *self);
extern gui_pointer_list wrap_ImGuiContext_GetWindows(void *self);
*/
import "C"

import (
	"unsafe"

	"github.com/AllenDang/cimgui-go/imgui"
)

// drawDataTextures lists the textures Dear ImGui wants serviced this frame.
//
// hide empties the list in place, so that a backend walking it afterwards
// finds nothing, and returns the function that puts it back.
func drawDataTextures(drawData *imgui.DrawData) (textures []*imgui.TextureData, hide func() (restore func())) {
	list := C.wrap_ImDrawData_GetTextures(unsafe.Pointer(drawData.CData))
	if list == nil || list.Size == 0 || list.Data == nil {
		return nil, func() func() { return func() {} }
	}

	for _, entry := range unsafe.Slice(list.Data, list.Size) {
		if entry != nil {
			textures = append(textures, imgui.NewTextureDataFromC(unsafe.Pointer(entry)))
		}
	}

	hide = func() func() {
		size := list.Size
		list.Size = 0

		return func() { list.Size = size }
	}

	return textures, hide
}

// contextWindows lists every window of the current context, open or not,
// including the internal ones: menus, popups, tooltips and child windows.
func contextWindows() []*imgui.Window {
	list := C.wrap_ImGuiContext_GetWindows(unsafe.Pointer(imgui.CurrentContext().CData))
	if list.Size == 0 || list.Data == nil {
		return nil
	}

	windows := make([]*imgui.Window, 0, list.Size)

	for _, entry := range unsafe.Slice(list.Data, list.Size) {
		if entry != nil {
			windows = append(windows, imgui.NewWindowFromC(unsafe.Pointer(entry)))
		}
	}

	return windows
}

// cPointer turns an address cimgui-go hands back as a uintptr into a pointer.
//
// Turning a uintptr into a pointer is only unsafe for Go memory, which the
// collector may move or free in between. These addresses point into memory
// Dear ImGui allocated in C, which the Go runtime never touches, so the value
// is reinterpreted in place instead of converted.
func cPointer(address uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&address))
}
