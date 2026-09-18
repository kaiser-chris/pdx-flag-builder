package app

import (
	"image/color"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// colorFromFloats converts the 0..1 channels Dear ImGui's colour editor works
// with back into the 8 bit channels the settings file stores.
func colorFromFloats(channels [4]float32) color.RGBA {
	return color.RGBA{
		R: uint8(channels[0]*255 + 0.5),
		G: uint8(channels[1]*255 + 0.5),
		B: uint8(channels[2]*255 + 0.5),
		A: uint8(channels[3]*255 + 0.5),
	}
}

// colorKind is the spelling of a colour slot the editor offers.
type colorKind int

const (
	kindNamed colorKind = iota
	kindRGB
	kindHSV
	kindSlot
	kindList
)

func (k colorKind) String() string {
	switch k {
	case kindNamed:
		return "Named"
	case kindRGB:
		return "RGB"
	case kindHSV:
		return "HSV"
	case kindSlot:
		return "Slot"
	case kindList:
		return "Random"
	}

	return "?"
}

func kindOf(value pdx.ColorValue) colorKind {
	switch value.(type) {
	case pdx.RGBColor:
		return kindRGB
	case pdx.HSVColor:
		return kindHSV
	case pdx.SlotColor:
		return kindSlot
	case pdx.ListColor:
		return kindList
	}

	return kindNamed
}

// Widths in a colour row. The value editor takes the rest of the line, but never
// less than this.
const (
	kindComboWidth     = 80
	minColorValueWidth = 90
)

// colorEditor edits a set of colour slots.
//
// slots is what a slot may refer back to: the flag's colours, whether the set
// being edited is the flag's own or a layer's. A layer can therefore borrow a
// colour of its flag, and the flag can reuse one of its own.
func (a *App) colorEditor(id string, colors *pdx.Colors, slots pdx.Colors) {
	imgui.PushIDStr(id)
	defer imgui.PopID()

	remove := ""

	for index := range *colors {
		entry := &(*colors)[index]

		imgui.PushIDStr(entry.Slot)

		a.swatch(*entry, slots)

		imgui.SameLine()
		imgui.AlignTextToFramePadding()
		imgui.TextUnformatted(entry.Slot)

		imgui.SameLine()
		a.kindCombo(entry, slots)

		// The value takes whatever the line has left beside the remove button.
		imgui.SameLine()
		style := imgui.CurrentStyle()
		removeWidth := imgui.CalcTextSize("Remove").X + style.FramePadding().X*2
		width := max(imgui.ContentRegionAvail().X-removeWidth-style.ItemSpacing().X, gui.Scaled(minColorValueWidth))
		a.colorValueEditor(entry, slots, width)

		imgui.SameLine()
		if gui.SmallButton("Remove") {
			remove = entry.Slot
		}

		imgui.PopID()
	}

	if remove != "" && colors.Remove(remove) {
		a.changed()
	}

	next := colors.NextFreeSlot()

	imgui.BeginDisabledV(next == "")
	if gui.Button("Add Colour") {
		colors.Set(pdx.Color{Slot: next, Value: a.defaultColor()})
		a.changed()
	}
	imgui.EndDisabled()
}

// defaultColor is what a newly added slot starts as: white, by name when the
// game defines it.
func (a *App) defaultColor() pdx.ColorValue {
	if _, ok := a.state.library.palette["white"]; ok {
		return pdx.NamedColor{Name: "white"}
	}

	return pdx.RGBColor{R: 255, G: 255, B: 255}
}

func (a *App) swatch(entry pdx.Color, slots pdx.Colors) {
	resolved, known := entry.Resolve(a.state.library.palette, slots)

	imgui.ColorButtonV("##swatch",
		imgui.Vec4{X: float32(resolved.R) / 255, Y: float32(resolved.G) / 255, Z: float32(resolved.B) / 255, W: 1},
		imgui.ColorEditFlagsNoTooltip|imgui.ColorEditFlagsNoDragDrop,
		imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()},
	)

	if imgui.IsItemHovered() && imgui.BeginTooltip() {
		imgui.TextUnformatted(entry.Slot + " = " + entry.Value.Describe())

		if !known {
			imgui.TextDisabled("This colour could not be resolved.")
		}

		imgui.EndTooltip()
	}
}

// kindCombo switches a slot between spellings. The colour is carried across as
// closely as the new spelling allows, so that switching does not change what
// the flag looks like more than it has to.
func (a *App) kindCombo(entry *pdx.Color, slots pdx.Colors) {
	current := kindOf(entry.Value)

	imgui.SetNextItemWidth(gui.Scaled(kindComboWidth))

	if !gui.BeginCombo("##kind", current.String()) {
		return
	}
	defer imgui.EndCombo()

	for _, kind := range []colorKind{kindNamed, kindRGB, kindHSV, kindSlot} {
		if kind == kindSlot && len(otherSlots(entry.Slot, slots)) == 0 {
			// Nothing to refer to.
			continue
		}

		if gui.Selectable(kind.String(), kind == current, 0) && kind != current {
			entry.Value = a.convertColor(*entry, kind, slots)
			a.changed()
		}
	}
}

func (a *App) convertColor(entry pdx.Color, kind colorKind, slots pdx.Colors) pdx.ColorValue {
	resolved, _ := entry.Resolve(a.state.library.palette, slots)

	switch kind {
	case kindRGB:
		return pdx.RGBColor{R: resolved.R, G: resolved.G, B: resolved.B}

	case kindHSV:
		hue, saturation, value := pdx.RGBToHSV(resolved)

		return pdx.HSVColor{H: hue, S: saturation, V: value}

	case kindSlot:
		return pdx.SlotColor{Slot: otherSlots(entry.Slot, slots)[0]}
	}

	if name, ok := a.state.library.palette.Nearest(resolved); ok {
		return pdx.NamedColor{Name: name}
	}

	return pdx.NamedColor{Name: "white"}
}

// otherSlots lists the slots a slot may refer to: every filled one but itself,
// since a slot pointing at itself resolves to nothing.
func otherSlots(slot string, slots pdx.Colors) []string {
	var names []string

	for _, candidate := range slots {
		if candidate.Slot != slot {
			names = append(names, candidate.Slot)
		}
	}

	return names
}

func (a *App) colorValueEditor(entry *pdx.Color, slots pdx.Colors, width float32) {
	switch value := entry.Value.(type) {
	case pdx.NamedColor:
		a.namedColorEditor(entry, value, width)

	case pdx.RGBColor:
		channels := [3]float32{float32(value.R) / 255, float32(value.G) / 255, float32(value.B) / 255}

		imgui.SetNextItemWidth(width)
		if gui.ColorEditRGB("##rgb", &channels) {
			rgba := colorFromFloats([4]float32{channels[0], channels[1], channels[2], 1})
			entry.Value = pdx.RGBColor{R: rgba.R, G: rgba.G, B: rgba.B}
			a.changed()
		}

	case pdx.HSVColor:
		a.hsvEditor(entry, value, width)

	case pdx.SlotColor:
		imgui.SetNextItemWidth(width)
		if gui.BeginCombo("##slot", value.Slot) {
			for _, candidate := range otherSlots(entry.Slot, slots) {
				if gui.Selectable(candidate, candidate == value.Slot, 0) && candidate != value.Slot {
					entry.Value = pdx.SlotColor{Slot: candidate}
					a.changed()
				}
			}

			imgui.EndCombo()
		}

	case pdx.ListColor:
		imgui.AlignTextToFramePadding()
		imgui.TextDisabled("random from " + value.List)
	}
}

// hsvEditor edits hue in degrees and the other two as percentages, the way the
// hsv360 spelling in the files writes them.
func (a *App) hsvEditor(entry *pdx.Color, value pdx.HSVColor, width float32) {
	saturation := value.S * 100
	brightness := value.V * 100

	width = (width - imgui.CurrentStyle().ItemInnerSpacing().X*2) / 3
	changed := false

	imgui.SetNextItemWidth(width)
	changed = gui.DragFloat("##hue", &value.H, 1, 0, 360, "H %.0f") || changed

	imgui.SameLine()
	imgui.SetNextItemWidth(width)
	changed = gui.DragFloat("##saturation", &saturation, 0.5, 0, 100, "S %.0f%%") || changed

	imgui.SameLine()
	imgui.SetNextItemWidth(width)
	changed = gui.DragFloat("##value", &brightness, 0.5, 0, 100, "V %.0f%%") || changed

	if changed {
		entry.Value = pdx.HSVColor{H: value.H, S: saturation / 100, V: brightness / 100}
		a.changed()
	}
}

// namedColorEditor picks one of the game's named colours, with a search box
// because there are close to two hundred of them.
func (a *App) namedColorEditor(entry *pdx.Color, value pdx.NamedColor, width float32) {
	imgui.SetNextItemWidth(width)

	if !gui.BeginCombo("##named", value.Name) {
		return
	}
	defer imgui.EndCombo()

	if imgui.IsWindowAppearing() {
		a.state.colorSearch = ""
		imgui.SetKeyboardFocusHere()
	}

	imgui.SetNextItemWidth(-1)
	gui.InputText("##colour-search", "Search", &a.state.colorSearch)

	query := strings.ToLower(strings.TrimSpace(a.state.colorSearch))

	for _, name := range a.state.library.paletteNames {
		if query != "" && !strings.Contains(strings.ToLower(name), query) {
			continue
		}

		imgui.PushIDStr(name)

		a.swatch(pdx.Color{Slot: name, Value: pdx.NamedColor{Name: name}}, nil)
		imgui.SameLine()

		if gui.Selectable(name, name == value.Name, 0) && name != value.Name {
			entry.Value = pdx.NamedColor{Name: name}
			a.changed()
		}

		imgui.PopID()
	}

	if len(a.state.library.paletteNames) == 0 {
		imgui.TextDisabled("No named colours loaded.")
	}
}
