package gui

import (
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
)

// colorMatch marks the part of a name a search matched: a warm tint that
// stands out against the cool accent the rest of the interface uses for
// selection and hover.
var colorMatch = imgui.Vec4{X: 0.85, Y: 0.62, Z: 0.18, W: 0.45}

// MatchRanges finds every place query occurs in text, ignoring case, as byte
// ranges of text. An empty query matches nothing.
func MatchRanges(text, query string) [][2]int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	lowered := strings.ToLower(text)

	// Lowering some characters changes how many bytes they take, after which
	// offsets into the lowered text no longer point into the original. Names
	// in the game files are plain ASCII, so such a name just goes unmarked.
	if len(lowered) != len(text) {
		return nil
	}

	var ranges [][2]int

	for offset := 0; ; {
		found := strings.Index(lowered[offset:], query)
		if found < 0 {
			return ranges
		}

		start := offset + found
		offset = start + len(query)
		ranges = append(ranges, [2]int{start, offset})
	}
}

// HighlightedText is a line of text with every match of query marked.
func HighlightedText(text, query string) {
	markMatches(imgui.CursorScreenPos(), text, query)
	imgui.TextUnformatted(text)
}

// HighlightedSelectable is a selectable row labelled with text, with every
// match of query marked. The row is height tall, or one line for zero, and
// the text is centred in it vertically.
func HighlightedSelectable(text, query string, selected bool, flags imgui.SelectableFlags, height float32) bool {
	start := imgui.CursorScreenPos()

	clicked := imgui.SelectableBoolV("##"+text, selected, flags, imgui.Vec2{Y: height})
	record(text, selected)

	position := imgui.Vec2{X: start.X, Y: start.Y + max(height-imgui.TextLineHeight(), 0)/2}

	markMatches(position, text, query)
	imgui.WindowDrawList().AddTextVec2(position, imgui.ColorU32Col(imgui.ColText), text)

	return clicked
}

// markMatches paints the background behind the matched parts of a line of
// text that is about to be drawn at position. It has to come first so that
// the text ends up on top.
func markMatches(position imgui.Vec2, text, query string) {
	ranges := MatchRanges(text, query)
	if len(ranges) == 0 {
		return
	}

	drawList := imgui.WindowDrawList()
	color := imgui.ColorU32Vec4(colorMatch)
	height := imgui.TextLineHeight()
	rounding := Scaled(2)

	for _, match := range ranges {
		left := position.X + textWidth(text[:match[0]])
		right := left + textWidth(text[match[0]:match[1]])

		drawList.AddRectFilledV(
			imgui.Vec2{X: left, Y: position.Y},
			imgui.Vec2{X: right, Y: position.Y + height},
			color, rounding, 0,
		)
	}
}

// textWidth measures text as drawn, including any "##" in it, which
// CalcTextSize would otherwise treat as the start of a hidden id.
func textWidth(text string) float32 {
	return imgui.CalcTextSizeV(text, false, -1).X
}
