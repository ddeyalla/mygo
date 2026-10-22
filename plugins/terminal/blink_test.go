package terminal

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestBlinkPaintsOnly: the cursor of a focused terminal blinks by painting
// the window again, every 600 ms, without building the view, and looks as
// it did: shown, then hidden, in phase with the last key typed or the
// focus, until blinkTimeout goes by and it rests shown without drawing,
// and steady while the window has no focus.
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
	b, p := builds, paints
	tt.Advance(blinkPeriod / 2)
	for i, want := range []bool{true, false, true, false} {
		if got := shown(); got != want {
			t.Fatalf("half period %d: the cursor shows %v, want %v", i, got, want)
		}
		tt.Advance(blinkPeriod)
	}

	// It blinks, painting only, until blinkTimeout went by since the focus:
	// a paint at each of the 16 turns, and one at the timeout that shows
	// the cursor, since 10 s falls in a hidden half.
	tt.Advance(blinkTimeout - 4*blinkPeriod - blinkPeriod/2 + 10*time.Millisecond)
	t.Logf("a focused terminal blinking for %v: %d builds, %d paints", blinkTimeout, builds-b, paints-p)
	if builds != b {
		t.Errorf("the view built %d times as the cursor blinked, want 0", builds-b)
	}
	if want := int(blinkTimeout/blinkPeriod) + 1; paints-p != want {
		t.Errorf("the window painted %d times as the cursor blinked, want %d", paints-p, want)
	}
	if !shown() {
		t.Error("the cursor does not show once it stops blinking")
	}

	// Then it rests shown, and a minute at rest neither builds nor paints.
	restsShown := func(after string) {
		t.Helper()
		b, p := builds, paints
		for range 100 {
			tt.Advance(blinkPeriod)
			if !shown() {
				t.Fatalf("the cursor hid at rest, %s", after)
			}
		}
		t.Logf("a focused terminal at rest for a minute, %s: %d builds, %d paints", after, builds-b, paints-p)
		if builds != b || paints != p {
			t.Errorf("at rest for a minute, %s, the view built %d times and painted %d times, want none", after, builds-b, paints-p)
		}
	}
	restsShown("after the focus")

	// A key starts the blink again, in phase with the key.
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
	tt.Advance(blinkTimeout)
	if !shown() {
		t.Error("the cursor does not show once it stops blinking after the key")
	}
	restsShown("after a key")

	// So does the focus coming back.
	tt.SetFocused(false)
	tt.SetFocused(true)
	tt.Advance(blinkPeriod + 10*time.Millisecond)
	if shown() {
		t.Error("the cursor still shows a period after the focus came back")
	}
	tt.Advance(blinkTimeout)
	restsShown("after the focus came back")

	// Without the focus, it is steady, and nothing draws.
	tt.SetFocused(false)
	b, p = builds, paints
	tt.Advance(10 * time.Second)
	if builds != b || paints != p {
		t.Errorf("without the focus, the view built %d times and painted %d times in 10 s, want none", builds-b, paints-p)
	}
}
