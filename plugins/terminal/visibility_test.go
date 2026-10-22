package terminal

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestHiddenTerminalKeepsOutputAndEvents(t *testing.T) {
	loadLib(t)
	var title, notify string
	bells := 0
	term, err := New(Options{Conn: newPipe(), OnTitle: func(s string) { title = s }, OnBell: func() { bells++ }, OnNotify: func(a, b string) { notify = a + ":" + b }})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	var draws atomic.Int64
	draw := func() { draws.Add(1) }
	term.draw.Store(&draw)
	term.Feed([]byte("visible"))
	if draws.Load() != 1 {
		t.Fatal("visible output did not redraw")
	}
	term.SetVisible(false)
	draws.Store(0)
	term.Feed([]byte("\r\nlatest\x1b]2;hidden title\a\a\x1b]777;notify;head;body\a"))
	term.Resize(90, 30)
	if got := draws.Load(); got != 0 {
		t.Errorf("hidden output requested %d frames", got)
	}
	if !strings.Contains(term.Text(), "latest") || term.Title() != "hidden title" || title != "hidden title" || bells != 1 || notify != "head:body" {
		t.Errorf("hidden state/events: text %q title %q callback %q bells %d notify %q", term.Text(), term.Title(), title, bells, notify)
	}
	term.SetVisible(true)
	if got := draws.Load(); got != 1 {
		t.Fatalf("reveal requested %d frames, want 1", got)
	}
	term.SetVisible(true)
	if got := draws.Load(); got != 1 {
		t.Fatalf("unchanged visibility requested another frame: %d", got)
	}
}

func TestHiddenTerminalStopsWindowFrames(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	shown := true
	builds, paints := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Column(c).Fill().Children(func() {
			if shown {
				View(c, term).Fill().AutoFocus()
			}
			ui.Box(c).Size(1, 1).Draw(func(*ui.Painter, ui.Rect) { paints++ })
		})
	}, 400, 200)
	tt.Frame()
	shown = false
	term.SetVisible(false)
	tt.Frame()
	before, painted := builds, paints
	for range 20 {
		term.Feed([]byte("hidden\r\n"))
		if tt.FramePending() {
			tt.Frame()
		}
	}
	tt.Advance(time.Minute)
	if tt.FramePending() {
		tt.Frame()
	}
	if builds != before {
		t.Fatalf("hidden output/timers built window %d times", builds-before)
	}
	if paints != painted {
		t.Fatalf("hidden terminal timer painted window %d times", paints-painted)
	}
	shown = true
	term.SetVisible(true)
	if !tt.FramePending() {
		t.Fatal("reveal did not request frame")
	}
	tt.Frame()
	first := append([]byte(nil), tt.Image().Pix...)
	tt.Frame()
	if !bytes.Equal(first, tt.Image().Pix) {
		t.Fatal("reveal needed another frame for current content")
	}
	if !strings.Contains(term.Text(), "hidden") {
		t.Fatal("reveal lost latest output")
	}
}

func TestTerminalVisibilityConcurrent(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	var draws atomic.Int64
	draw := func() { draws.Add(1) }
	term.draw.Store(&draw)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			term.Feed([]byte("x"))
			term.Resize(80, 24)
		}
	})
	wg.Go(func() {
		for range 100 {
			term.SetVisible(false)
			term.SetVisible(true)
		}
	})
	wg.Wait()
	term.SetVisible(false)
	before := draws.Load()
	term.Feed([]byte("last"))
	if draws.Load() != before {
		t.Fatal("hidden output requested frame")
	}
	term.Close()
	term.SetVisible(true)
}

// Close may race output and host visibility changes. The closed emulator
// must not be reused, and every caller must finish without waiting on UI work.
func TestTerminalVisibilityConcurrentClose(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	var draws atomic.Int64
	draw := func() { draws.Add(1) }
	term.draw.Store(&draw)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		<-start
		for range 100 {
			term.Feed([]byte("output\r\n"))
			term.Resize(80, 24)
		}
	})
	wg.Go(func() {
		<-start
		for range 100 {
			term.SetVisible(false)
			term.SetVisible(true)
		}
	})
	wg.Go(func() {
		<-start
		term.Close()
	})
	close(start)
	wg.Wait()
	term.SetVisible(false)
	before := draws.Load()
	term.Feed([]byte("after close"))
	if draws.Load() != before || term.Text() != "" {
		t.Fatal("hidden closed terminal retained output or requested a frame")
	}
}

// Output callbacks run outside the emulator lock, so a host may hide or
// close its terminal in response to an event without deadlocking Feed.
func TestHiddenTerminalCallbackCanClose(t *testing.T) {
	loadLib(t)
	var term *Terminal
	var closeErr error
	var err error
	term, err = New(Options{Conn: newPipe(), OnTitle: func(string) {
		term.SetVisible(false)
		closeErr = term.Close()
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	var draws atomic.Int64
	draw := func() { draws.Add(1) }
	term.draw.Store(&draw)
	done := make(chan struct{})
	go func() {
		term.Feed([]byte("\x1b]2;close now\a"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Minute): // a hang guard
		t.Fatal("output callback could not hide and close its terminal")
	}
	if closeErr != nil || term.Text() != "" || draws.Load() != 0 {
		t.Fatalf("callback close: error %v, text %q, frames %d", closeErr, term.Text(), draws.Load())
	}
}
