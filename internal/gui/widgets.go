package gui

import (
	"math"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
)

// The interactive widgets the application uses.
//
// Each one calls Dear ImGui and then reports what it drew to the recorder, if
// one is installed. Dear ImGui has hooks of its own for exactly this, for its
// test engine, but they only exist when Dear ImGui itself is compiled with a
// define, and cimgui-go ships it precompiled. Reporting from here gives the
// interface tests the same view: every widget's label, window and rectangle,
// which is what they need to find a widget and click it.
//
// Without a recorder the cost is a nil check per widget.

// RecordedItem is one widget as it was laid out.
type RecordedItem struct {
	ID imgui.ID

	// Label is what the widget was created with, including any ## suffix
	// that keeps its id unique.
	Label string

	// Window is the name of the window the widget sits in. Menus open in
	// windows named ##Menu_00 and so on, the main menu bar is ##MainMenuBar,
	// and a scrolling table lives in a child window named after its parent.
	Window string

	Min, Max imgui.Vec2

	// ClipMin and ClipMax bound the part of the window the widget can be seen
	// in. A widget scrolled out of view is still laid out, just clipped away.
	ClipMin, ClipMax imgui.Vec2

	// Checked is set for a ticked menu item or check box.
	Checked bool
}

// Recorder receives every widget the interface lays out.
type Recorder interface {
	Record(item RecordedItem)
}

var recorder Recorder

// SetRecorder installs a recorder, or removes it when given nil. Only the
// interface tests install one.
func SetRecorder(r Recorder) {
	recorder = r
}

func record(label string, checked bool) {
	if recorder == nil {
		return
	}

	recordIn(currentPlace(), label, checked)
}

// recordIn is record for a widget that may have changed the current window by
// the time it returns, such as a menu that has just opened.
func recordIn(where place, label string, checked bool) {
	if recorder == nil {
		return
	}

	recorder.Record(RecordedItem{
		ID:      imgui.ItemID(),
		Label:   label,
		Window:  where.window,
		Min:     imgui.ItemRectMin(),
		Max:     imgui.ItemRectMax(),
		ClipMin: where.clipMin,
		ClipMax: where.clipMax,
		Checked: checked,
	})
}

// Button is imgui.Button.
func Button(label string) bool {
	pressed := imgui.Button(label)
	record(label, false)

	return pressed
}

// Checkbox is imgui.Checkbox.
func Checkbox(label string, value *bool) bool {
	changed := imgui.Checkbox(label, value)
	record(label, *value)

	return changed
}

// BeginMenu is imgui.BeginMenu. The recorded item is the menu's entry in the
// menu bar, which is what opens it.
func BeginMenu(label string) bool {
	// An open menu makes its own window current, so the window the entry sits
	// in has to be read before.
	var where place
	if recorder != nil {
		where = currentPlace()
	}

	open := imgui.BeginMenu(label)
	recordIn(where, label, false)

	return open
}

// place is where a widget is being laid out: the window and the part of it
// that is visible.
type place struct {
	window           string
	clipMin, clipMax imgui.Vec2
}

func currentPlace() place {
	current := imgui.InternalCurrentWindowRead()
	if current == nil {
		return place{}
	}

	drawList := imgui.WindowDrawList()

	return place{window: current.Name(), clipMin: drawList.ClipRectMin(), clipMax: drawList.ClipRectMax()}
}

// MenuItem is a menu item that runs an action.
func MenuItem(label, shortcut string, enabled bool) bool {
	clicked := imgui.MenuItemBoolV(label, shortcut, false, enabled)
	record(label, false)

	return clicked
}

// MenuToggle is a menu item that ticks and unticks a setting.
func MenuToggle(label, shortcut string, value *bool) bool {
	clicked := imgui.MenuItemBoolPtr(label, shortcut, value)
	record(label, *value)

	return clicked
}

// Selectable is imgui.SelectableBoolV.
func Selectable(label string, selected bool, flags imgui.SelectableFlags) bool {
	clicked := imgui.SelectableBoolV(label, selected, flags, imgui.Vec2{})
	record(label, selected)

	return clicked
}

// InputText is a text field with a hint shown while it is empty.
func InputText(label, hint string, text *string) bool {
	changed := imgui.InputTextWithHint(fieldLabel(label), hint, text, 0, nil)
	record(label, false)

	return changed
}

// SelectableSized is Selectable with a size, for a row that has to leave room
// for buttons beside it or line up with rows that have them. Zero for either
// side keeps the default: the rest of the line, one line of text.
func SelectableSized(label string, selected bool, width, height float32) bool {
	clicked := imgui.SelectableBoolV(label, selected, 0, imgui.Vec2{X: width, Y: height})
	record(label, selected)

	return clicked
}

// SmallButton is imgui.SmallButton.
func SmallButton(label string) bool {
	pressed := imgui.SmallButton(label)
	record(label, false)

	return pressed
}

// ArrowButton is imgui.ArrowButton. It has no visible text, so it is recorded
// under its id.
func ArrowButton(id string, direction imgui.Dir) bool {
	pressed := imgui.ArrowButton(id, direction)
	record(id, false)

	return pressed
}

// DragFloat edits a number by dragging, or by typing after a double click.
func DragFloat(label string, value *float32, speed, low, high float32, format string) bool {
	changed := imgui.DragFloatV(fieldLabel(label), value, speed, low, high, format, imgui.SliderFlagsAlwaysClamp)
	record(label, false)

	return changed
}

// DragPair edits two numbers side by side, such as a position.
func DragPair(label string, x, y *float32, speed, low, high float32, format string) bool {
	pair := [2]float32{*x, *y}

	changed := imgui.DragFloat2V(fieldLabel(label), &pair, speed, low, high, format, imgui.SliderFlagsAlwaysClamp)
	record(label, false)

	if changed {
		*x, *y = pair[0], pair[1]
	}

	return changed
}

// ColorEditRGB edits a colour without alpha: one field per channel and a
// swatch that opens the colour picker.
func ColorEditRGB(label string, value *[3]float32) bool {
	changed := imgui.ColorEdit3V(label, value, imgui.ColorEditFlagsNoLabel)
	record(label, false)

	// The swatch at the end of the row opens the picker, which nothing about
	// a plain square of colour says, so it carries an eyedropper.
	high := imgui.ItemRectMax()
	swatch := imgui.Vec2{X: high.X - imgui.FrameHeight(), Y: imgui.ItemRectMin().Y}
	drawPickerIcon(swatch, high, *value)

	return changed
}

// BeginCombo is imgui.BeginCombo. The recorded item is the closed combo box,
// which is what opens it; the choices inside are recorded as they are drawn.
func BeginCombo(label, preview string) bool {
	var where place
	if recorder != nil {
		where = currentPlace()
	}

	open := imgui.BeginCombo(fieldLabel(label), preview)
	recordIn(where, label, false)

	return open
}

// labelColumn is how far in from the left the fields of a form start, in the
// units the interface was designed in. Labels longer than that push their own
// field further right.
const labelColumn = 110

// fieldLabel lays out a form row: the label on the left, in a column shared
// by every row, and the field filling the rest of the line. Dear ImGui puts
// labels to the right of their field by default, which reads backwards in a
// form. It returns the id the field is created with, which hides the label
// Dear ImGui would otherwise draw a second time.
//
// A label with no visible text, such as "##search", is left alone.
func fieldLabel(label string) string {
	visible, _, _ := strings.Cut(label, "##")
	if visible == "" {
		return label
	}

	Label(visible)

	// -FLT_MIN is Dear ImGui's way of saying "up to the right edge".
	imgui.SetNextItemWidth(-math.SmallestNonzeroFloat32)

	return "##" + label
}

// Label draws the label of a form row and moves the cursor to where the row's
// value starts, for a row whose value is not a single field.
func Label(text string) {
	start := imgui.CursorPosX()

	imgui.AlignTextToFramePadding()
	imgui.TextUnformatted(text)

	spacing := imgui.CurrentStyle().ItemSpacing().X
	imgui.SameLineV(start+max(Scaled(labelColumn), imgui.CalcTextSize(text).X+spacing), 0)
}

// TableHeadersRow is imgui.TableHeadersRow, with every header recorded under
// its column's name. Clicking a header sorts the table by that column.
func TableHeadersRow() {
	imgui.TableNextRowV(imgui.TableRowFlagsHeaders, 0)

	for column := range imgui.TableGetColumnCount() {
		if !imgui.TableSetColumnIndex(column) {
			continue
		}

		name := imgui.TableGetColumnNameIntV(column)

		// Dear ImGui's own version pushes the column, so that two columns of
		// the same name still have headers of their own.
		imgui.PushIDInt(column)
		imgui.TableHeader(name)
		record(name, false)
		imgui.PopID()
	}
}

// FindWindow looks a window up by name. cimgui-go wraps whatever Dear ImGui
// returns, a null pointer included, so a missing window would otherwise look
// found and crash on first use.
func FindWindow(name string) (*imgui.Window, bool) {
	window := imgui.InternalFindWindowByName(name)
	if window == nil || window.CData == nil {
		return nil, false
	}

	return window, true
}
