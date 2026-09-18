package app

import (
	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
)

// credit is the attribution a work bundled into the application asks for.
type credit struct {
	// work names the work and says what the application uses it for.
	work string

	// notice is the attribution in the words its licence asks for.
	notice string

	// link is where the work, or its licence, can be found.
	link, url string
}

// credits lists every bundled work that asks to be credited. The Odin version
// kept these in its README; they belong where users of the application see
// them too. Its other icons are not bundled any more and so not listed.
var credits = []credit{
	{
		work:   "Application icon",
		notice: "\"waving flag\" by Suncheli Project from Noun Project, licensed under CC BY 3.0.",
		link:   "thenounproject.com",
		url:    "https://thenounproject.com/browse/icons/term/waving-flag/",
	},
	{
		work: "Roboto, the interface font",
		notice: "Designed by Christian Robertson. Copyright 2011 The Roboto Project Authors. " +
			"Licensed under the SIL Open Font License, Version 1.1. Roboto is a trademark of Google.",
		link: "openfontlicense.org",
		url:  "https://openfontlicense.org",
	},
}

func (c credit) show() {
	gui.PushStrongFont()
	imgui.TextUnformatted(c.work)
	gui.PopFont()

	dimmedWrapped(c.notice)
	imgui.TextLinkOpenURLV(c.link, c.url)
}
