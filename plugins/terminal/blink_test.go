package terminal

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestBlinkPaintsOnly: the cursor of a focused terminal blinks by painting
// the window again, every 600 ms, without building the view, and looks as
// it did: shown, then hidden, in phase with the last key typed, and
// steady while the window has no focus.
func TestBlinkPaintsOnly(t *testing.T) {
	loadLib(t)
	th := &Theme{
		Foreground: ui.Hex("#c0c0c0"), Background: ui.Hex("#101010"),
		Cursor: ui.Hex("#ff0000"), CursorText: ui.Hex("#00ff00"),
	}
	term, err := New(Options{Conn: newPipe(), Theme: th, Cursor: CursorBlock})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	builds, paints := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Column(c).Fill().Children(func() {
			View(c, term).Fill().AutoFocus()
			// Painted whenever the window is, built or not.
			ui.Box(c).Size(1, 1).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) { paints++ })
		})
	}, 400, 200)
	tt.Frame()
	if !term.v.focused || !term.v.cursor.Blinking || !term.v.cursor.Visible {
		t.Fatalf("the cursor is focused %v, blinking %v, visible %v", term.v.focused, term.v.cursor.Blinking, term.v.cursor.Visible)
	}
	shown := func() bool {
		x, y := cellCenter(term, 0, 0)
		s := term.v.scale
		c := tt.Image().RGBAAt(int(x*s), int(y*s))
		return c.R > 0xe0 && c.G < 0x20 && c.B < 0x20
	}
	// Each half period shows the cursor, then hides it, from the focus.
	tt.Advance(blinkPeriod / 2)
	for i, want := range []bool{true, false, true, false} {
		if got := shown(); got != want {
			t.Fatalf("half period %d: the cursor shows %v, want %v", i, got, want)
		}
		tt.Advance(blinkPeriod)
	}

	// A minute of a focused terminal at rest only paints.
	b, p := builds, paints
	tt.Advance(time.Minute)
	t.Logf("a focused terminal at rest for a minute: %d builds, %d paints", builds-b, paints-p)
	if builds != b {
		t.Errorf("the view built %d times in a minute at rest, want 0", builds-b)
	}
	if want := int(time.Minute / blinkPeriod); paints-p < want-1 || paints-p > want+1 {
		t.Errorf("the window painted %d times in a minute at rest, want %d", paints-p, want)
	}

	// A key shows the cursor at once, and the blink goes on from it.
	tt.Advance(blinkPeriod) // into a hidden half
	if shown() {
		t.Fatal("the cursor shows in the hidden half")
	}
	tt.Type("a")
	if !shown() {
		t.Error("the cursor does not show after a key")
	}
	tt.Advance(blinkPeriod - 10*time.Millisecond)
	if !shown() {
		t.Error("the cursor hid before a period went by since the key")
	}
	tt.Advance(20 * time.Millisecond)
	if shown() {
		t.Error("the cursor still shows a period after the key")
	}

	// Without the focus, it is steady, and nothing draws.
	tt.SetFocused(false)
	b, p = builds, paints
	tt.Advance(10 * time.Second)
	if builds != b || paints != p {
		t.Errorf("without the focus, the view built %d times and painted %d times in 10 s, want none", builds-b, paints-p)
	}
}
