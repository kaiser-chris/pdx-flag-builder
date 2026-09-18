// Package uitest drives the real application from Go tests, the way a user
// would, without a window appearing on the desktop and without faking
// operating system input.
//
// The application's widgets report their label, window and rectangle as they
// are laid out (see gui.SetRecorder), so a test can find a widget by what it
// says and click it, type into it, or check its state. The input goes straight
// into Dear ImGui after the platform input for the frame, so the real mouse and
// keyboard play no part.
//
// The tests that use it need an OpenGL context, so they carry the uitest build
// tag and run with "make uitest" rather than with every "go test".
package uitest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/gui"
)

// Item is one widget as the last frame laid it out.
type Item struct {
	gui.RecordedItem
}

// Text is the label as shown on screen, without the ## suffix.
func (i Item) Text() string {
	if before, _, found := strings.Cut(i.Label, "##"); found {
		return before
	}

	return i.Label
}

// clickInset is how far in from its left edge a wide widget is clicked.
const clickInset = 16

// ClickPoint is where a click on the widget lands: halfway down, and halfway
// across for a narrow widget but only a little way in for a wide one. The
// middle of a wide widget is not safe. A table row spans every column, and its
// middle can land on the border between two columns, which a resizable table
// hands to the column resizer instead of the row.
//
// Only the part of the widget that is visible counts. A row that spans every
// column of a table reports the whole row as its rectangle but is clipped to
// the cell it was created in, and the click has to land in that cell.
func (i Item) ClickPoint() imgui.Vec2 {
	low, high := i.visible()

	return imgui.Vec2{
		X: low.X + min((high.X-low.X)/2, clickInset),
		Y: (low.Y + high.Y) / 2,
	}
}

// visible is the part of the widget inside its clip rectangle. It is empty,
// with low past high, for a widget scrolled out of view.
func (i Item) visible() (low, high imgui.Vec2) {
	low = imgui.Vec2{X: max(i.Min.X, i.ClipMin.X), Y: max(i.Min.Y, i.ClipMin.Y)}
	high = imgui.Vec2{X: min(i.Max.X, i.ClipMax.X), Y: min(i.Max.Y, i.ClipMax.Y)}

	return low, high
}

// Reachable reports whether a point lies in the part of the widget's window
// that is shown, where a click could actually land on the widget.
func (i Item) Reachable(point imgui.Vec2) bool {
	return point.X >= i.ClipMin.X && point.X < i.ClipMax.X &&
		point.Y >= i.ClipMin.Y && point.Y < i.ClipMax.Y
}

// Windows that Dear ImGui names itself.
const (
	// MainMenuBar is the window the main menu bar lives in.
	MainMenuBar = "##MainMenuBar"

	// AnyMenu matches the window of whichever menu is open. Menus are named
	// after their label, as in Databases###Menu_00.
	AnyMenu = "###Menu_"
)

// maxWaitFrames bounds WaitFor. Frames run unthrottled in a hidden window, so
// this is a fraction of a second of real time for anything not waiting on disk.
const maxWaitFrames = 2000

// Target is the application under test.
type Target interface {
	// Step runs one frame.
	Step()

	// SetInput installs a function the application calls every frame to feed
	// input into Dear ImGui.
	SetInput(func())
}

// Driver runs an application frame by frame and plays a user.
type Driver struct {
	t      testing.TB
	target Target

	// items is what the last complete frame laid out; recording is what the
	// frame under way has laid out so far.
	items     []Item
	recording []Item

	// The input the next frames will see.
	mouse     imgui.Vec2
	mouseDown bool
	text      string
	keys      []keyEvent
}

type keyEvent struct {
	key  imgui.Key
	down bool
}

// New attaches a driver to an application and runs its first frames.
func New(t testing.TB, target Target) *Driver {
	t.Helper()

	driver := &Driver{
		t:      t,
		target: target,

		// Park the pointer away from everything so that nothing is hovered
		// until a test means it to be.
		mouse: imgui.Vec2{X: -10000, Y: -10000},
	}

	target.SetInput(driver.feed)

	gui.SetRecorder(driver)
	t.Cleanup(func() { gui.SetRecorder(nil) })

	// The first frame lays the dock space out; only the second has every
	// window where it will stay.
	driver.Frames(2)

	if len(driver.items) == 0 {
		t.Fatal("the application laid out no widgets")
	}

	return driver
}

// Record is called by every widget the application lays out.
func (d *Driver) Record(item gui.RecordedItem) {
	d.recording = append(d.recording, Item{item})
}

// feed is called by the application every frame, after the platform input and
// before the interface is built, so what it sets is what the interface sees.
func (d *Driver) feed() {
	io := imgui.CurrentIO()

	io.SetMousePos(d.mouse)
	io.SetMouseButtonDown(0, d.mouseDown)

	for _, event := range d.keys {
		io.AddKeyEvent(event.key, event.down)
	}
	d.keys = d.keys[:0]

	if d.text != "" {
		io.AddInputCharactersUTF8(d.text)
		d.text = ""
	}
}

// Frame runs one frame and records the widgets it laid out.
func (d *Driver) Frame() {
	d.recording = nil
	d.target.Step()
	d.items = d.recording
}

// Frames runs several frames.
func (d *Driver) Frames(count int) {
	for range count {
		d.Frame()
	}
}

// WaitFor runs frames until a condition holds, failing the test if it never
// does. what describes the condition for the failure message.
func (d *Driver) WaitFor(what string, condition func() bool) {
	d.t.Helper()

	for range maxWaitFrames {
		if condition() {
			return
		}

		d.Frame()
	}

	d.t.Fatalf("gave up waiting for %s after %d frames", what, maxWaitFrames)
}

// Items returns what the last frame laid out.
func (d *Driver) Items() []Item {
	return append([]Item(nil), d.items...)
}

// Dump logs every widget of the last frame. It is meant for writing a test, to
// find out what a widget is called and which window it is in.
func (d *Driver) Dump() {
	d.t.Helper()

	var dump strings.Builder

	for _, item := range d.items {
		fmt.Fprintf(&dump, "%-24q in %-28q at (%.0f,%.0f)-(%.0f,%.0f)\n",
			item.Label, item.Window, item.Min.X, item.Min.Y, item.Max.X, item.Max.Y)
	}

	d.t.Log("widgets in the last frame:\n" + dump.String())
}

// FindAll returns every widget with the given label in the given window. An
// empty window searches everywhere, AnyMenu matches whichever menu is open, and
// a window ending in a slash is matched as a prefix, which is how to reach the
// child window a scrolling table lives in, as in "Flag Database/". The label
// is compared with both the visible text and the full label, so "Save" and
// "##name" both work.
func (d *Driver) FindAll(window, label string) []Item {
	var found []Item

	for _, item := range d.items {
		if !matchesWindow(item.Window, window) {
			continue
		}

		if item.Text() == label || item.Label == label {
			found = append(found, item)
		}
	}

	return found
}

// Exists reports whether a widget is on screen.
func (d *Driver) Exists(window, label string) bool {
	return len(d.FindAll(window, label)) > 0
}

// Find returns the one widget with the given label, failing the test when there
// is none or more than one.
func (d *Driver) Find(window, label string) Item {
	d.t.Helper()

	found := d.FindAll(window, label)

	switch len(found) {
	case 0:
		d.Dump()
		d.t.Fatalf("no widget %q in %s", label, describeWindow(window))

	case 1:
		return found[0]
	}

	var windows []string
	for _, item := range found {
		windows = append(windows, item.Window)
	}

	d.t.Fatalf("%d widgets %q in %s, in windows %q; name the window to pick one",
		len(found), label, describeWindow(window), windows)

	return Item{}
}

// Click clicks the one widget with the given label.
func (d *Driver) Click(window, label string) {
	d.t.Helper()

	d.ClickItem(d.Find(window, label))
}

// ClickItem clicks a widget found earlier.
//
// A click takes three frames, the way a real one does: the pointer arrives,
// the button goes down, the button comes up. Buttons act on the last of those.
// One more frame lets the interface catch up with whatever the click changed.
func (d *Driver) ClickItem(item Item) {
	d.t.Helper()

	d.aim(item)

	d.mouseDown = true
	d.Frame()

	d.mouseDown = false
	d.Frame()

	d.Frame()
}

// maxAimFrames bounds how long aim follows a moving widget.
const maxAimFrames = 8

// aim puts the pointer on a widget and keeps it there until the widget stops
// moving, then returns where the pointer ended up.
//
// A widget found in the last frame is not necessarily still where it was. A
// window that has just appeared can move on its next frames: Dear ImGui sizes
// a modal on its first frame and centres it on the second. So the widget is
// looked up again by id after every frame, and the pointer follows it.
func (d *Driver) aim(item Item) imgui.Vec2 {
	d.t.Helper()

	for range maxAimFrames {
		d.mouse = item.ClickPoint()
		d.Frame()

		current, found := d.byID(item.ID)
		if !found {
			d.t.Fatalf("widget %q disappeared before it could be clicked", item.Label)
		}

		if current.ClickPoint() == d.mouse {
			// Dear ImGui lays out a widget that is scrolled away just the same,
			// and a click at where it would be lands on whatever covers it.
			if !current.Reachable(d.mouse) {
				d.t.Fatalf("widget %q at %v is outside the visible part of %s (%v to %v)",
					item.Label, d.mouse, describeWindow(current.Window), current.ClipMin, current.ClipMax)
			}

			return d.mouse
		}

		item = current
	}

	d.t.Fatalf("widget %q kept moving for %d frames", item.Label, maxAimFrames)

	return d.mouse
}

func (d *Driver) byID(id imgui.ID) (Item, bool) {
	for _, item := range d.items {
		if item.ID == id {
			return item, true
		}
	}

	return Item{}, false
}

// OffScreen returns the widgets of the last frame that reach past the edge of
// the window, which a user could not see or click.
func (d *Driver) OffScreen() []Item {
	size := imgui.CurrentIO().DisplaySize()

	var outside []Item

	for _, item := range d.items {
		if item.Min.X < 0 || item.Min.Y < 0 || item.Max.X > size.X || item.Max.Y > size.Y {
			outside = append(outside, item)
		}
	}

	return outside
}

// dragSteps is how many frames a drag is spread over. Dear ImGui only starts a
// drag once the pointer has moved past a small threshold, and a drag that
// arrives in one jump looks nothing like a user's.
const dragSteps = 6

// Drag presses on a widget, moves the pointer by the given distance and lets
// go. On a drag field this changes the value; the whole drag is one gesture,
// which is what the undo history records it as.
func (d *Driver) Drag(item Item, byX, byY float32) {
	d.t.Helper()

	start := d.aim(item)

	d.mouseDown = true
	d.Frame()

	for step := 1; step <= dragSteps; step++ {
		fraction := float32(step) / dragSteps
		d.mouse = imgui.Vec2{X: start.X + byX*fraction, Y: start.Y + byY*fraction}
		d.Frame()
	}

	d.mouseDown = false
	d.Frame()

	d.Frame()
}

// Menu opens a menu in the main menu bar and clicks its way down the path, so
// Menu("File", "Export", "To Image...") picks an item from a submenu.
func (d *Driver) Menu(path ...string) {
	d.t.Helper()

	if len(path) == 0 {
		return
	}

	d.Click(MainMenuBar, path[0])

	for _, label := range path[1:] {
		// Menus open in windows of their own. Of several matches the last is
		// the most recently opened, which is the one the path leads into.
		found := d.FindAll(AnyMenu, label)
		if len(found) == 0 {
			d.Dump()
			d.t.Fatalf("menu item %q is not in the open menu", label)
		}

		d.ClickItem(found[len(found)-1])
	}
}

// Type types text into whatever has keyboard focus.
func (d *Driver) Type(text string) {
	d.text = text
	d.Frame()
	d.Frame()
}

// Press presses and releases a key.
func (d *Driver) Press(key imgui.Key) {
	d.keys = append(d.keys, keyEvent{key: key, down: true})
	d.Frame()

	d.keys = append(d.keys, keyEvent{key: key, down: false})
	d.Frame()
}

// Shortcut presses a key with a modifier held, such as Ctrl+S.
func (d *Driver) Shortcut(modifier, key imgui.Key) {
	d.keys = append(d.keys, keyEvent{key: modifier, down: true}, keyEvent{key: key, down: true})
	d.Frame()

	d.keys = append(d.keys, keyEvent{key: key, down: false}, keyEvent{key: modifier, down: false})
	d.Frame()
}

// Fill replaces the contents of a text field: it clicks into it, selects what
// is there and types over it.
func (d *Driver) Fill(item Item, text string) {
	d.t.Helper()

	d.ClickItem(item)
	d.Shortcut(imgui.ModCtrl, imgui.KeyA)
	d.Type(text)
	d.Press(imgui.KeyEnter)
}

func matchesWindow(name, want string) bool {
	switch {
	case want == "":
		return true
	case want == AnyMenu:
		return strings.Contains(name, AnyMenu)
	case strings.HasSuffix(want, "_"), strings.HasSuffix(want, "/"):
		return strings.HasPrefix(name, want)
	default:
		return name == want
	}
}

func describeWindow(window string) string {
	if window == "" {
		return "any window"
	}

	return fmt.Sprintf("window %q", window)
}
