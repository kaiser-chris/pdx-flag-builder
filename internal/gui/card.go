package gui

import "github.com/AllenDang/cimgui-go/imgui"

// cardPadding is the room between a card's edge and what is in it, in the
// units the interface was designed in.
const cardPadding = 6

// Card frames a group of widgets that belong together, such as the fields of
// one placement in a list of them, so that one group does not run into the
// next.
//
// The frame is drawn behind the widgets once their size is known. Drawing it
// first is not possible, and drawing it last would cover them, so the widgets
// go onto a second draw list channel and the frame onto the first. A child
// window would give a frame too, but would also put the widgets in a window of
// their own, which neither scrolls with the panel nor keeps their names.
type Card struct {
	drawList *imgui.DrawList
	start    imgui.Vec2
	width    float32
	padding  float32
}

// BeginCard starts a card across the width that is left on the line. The
// frame reaches a little into the window's padding on both sides, so that
// fields inside keep the width they would have had outside a card.
func BeginCard() Card {
	card := Card{
		drawList: imgui.WindowDrawList(),
		start:    imgui.CursorScreenPos(),
		width:    imgui.ContentRegionAvail().X,
		padding:  Scaled(cardPadding),
	}

	card.drawList.ChannelsSplit(2)
	card.drawList.ChannelsSetCurrent(1)

	imgui.SetCursorScreenPos(imgui.Vec2{X: card.start.X, Y: card.start.Y + card.padding})
	imgui.BeginGroup()

	return card
}

// End closes the card and draws its frame. A highlighted card is outlined in
// the accent colour, the way a selected item is marked everywhere else.
func (c Card) End(highlighted bool) {
	imgui.EndGroup()

	bottom := imgui.ItemRectMax().Y + c.padding

	low := imgui.Vec2{X: c.start.X - c.padding, Y: c.start.Y}
	high := imgui.Vec2{X: c.start.X + c.width + c.padding, Y: bottom}

	border, thickness := colorBorder, float32(1)
	if highlighted {
		border, thickness = colorAccent, Scaled(1.5)
	}

	rounding := imgui.CurrentStyle().FrameRounding()

	c.drawList.ChannelsSetCurrent(0)
	c.drawList.AddRectFilledV(low, high, imgui.ColorU32Vec4(colorChildBg), rounding, 0)
	c.drawList.AddRectV(low, high, imgui.ColorU32Vec4(border), rounding, thickness, 0)
	c.drawList.ChannelsMerge()

	// Moving the cursor alone does not tell the window the card is there; a
	// widget, even an empty one, does.
	imgui.SetCursorScreenPos(imgui.Vec2{X: c.start.X, Y: bottom})
	imgui.Dummy(imgui.Vec2{})
}
