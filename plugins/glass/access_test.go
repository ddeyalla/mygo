package glass

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// glassOver returns a tester of a window 300 DIPs wide and 200 high filled
// with *behind, with the glass *g over a box 200 by 100 at (50, 50),
// rounded by 20, at a scale of 2.
func glassOver(behind *ui.Color, g *Glass) *ui.Tester {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(*behind).Children(func() {
			ui.Box(c).Absolute().Left(50).Top(50).Size(200, 100).Radius(20).Material(*g)
		})
	}
	tt := ui.NewTester(view, 300, 200)
	tt.SetScale(2)
	return tt
}

// overBlackAndWhite returns the pixel (x, y) of tt over black and over
// white behind.
func overBlackAndWhite(tt *ui.Tester, behind *ui.Color, x, y int) (black, white [3]int) {
	*behind = ui.RGB(0, 0, 0)
	tt.Frame()
	black = pixelAt(tt.Image(), x, y)
	*behind = ui.RGB(255, 255, 255)
	tt.Frame()
	white = pixelAt(tt.Image(), x, y)
	return black, white
}

func TestGlassIsOpaqueWithoutTransparency(t *testing.T) {
	behind := ui.RGB(0, 0, 0)
	g := Glass{}
	tt := glassOver(&behind, &g)
	// The middle of the glass.
	const x, y = 300, 200
	if black, white := overBlackAndWhite(tt, &behind, x, y); black == white {
		t.Fatalf("glass shows nothing of what is behind it: %v", black)
	}
	tt.SetPreferences(ui.Preferences{ReduceTransparency: true, TextScale: 1})
	black, white := overBlackAndWhite(tt, &behind, x, y)
	if black != white {
		t.Errorf("without transparency, glass over black is %v, over white %v", black, white)
	}
	// Light glass over the white window is white.
	if black[0] < 250 || black[1] < 250 || black[2] < 250 {
		t.Errorf("light glass is %v", black)
	}
	tt.SetDark(true)
	black, white = overBlackAndWhite(tt, &behind, x, y)
	if black != white {
		t.Errorf("without transparency, dark glass over black is %v, over white %v", black, white)
	}
	if black[0] < 40 || black[0] > 70 {
		t.Errorf("dark glass is %v, not a dark gray", black)
	}
	tt.SetDark(false)
	// Clear glass too.
	g.Style = Clear
	if black, white = overBlackAndWhite(tt, &behind, x, y); black != white {
		t.Errorf("without transparency, clear glass over black is %v, over white %v", black, white)
	}
	// A tint colors it, as the glass: 88% of an opaque blue.
	g = Glass{Tint: ui.RGB(0, 0, 255)}
	black, white = overBlackAndWhite(tt, &behind, x, y)
	if black != white {
		t.Errorf("without transparency, tinted glass over black is %v, over white %v", black, white)
	}
	if black[2] < 250 || black[0] < 25 || black[0] > 36 {
		t.Errorf("tinted glass is %v", black)
	}
	// Its rim, a device pixel at its left edge, stands out from it.
	g = Glass{}
	tt.Frame()
	if rim := pixelAt(tt.Image(), 100, y); rim[0] >= 250 {
		t.Errorf("the rim of opaque glass is %v", rim)
	}
}

func TestPaintIsOpaqueWithoutTransparency(t *testing.T) {
	behind := ui.RGB(0, 0, 0)
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(behind).Draw(func(p *ui.Painter, r ui.Rect) {
			Paint(p, ui.Rect{X: 20, Y: 20, W: 100, H: 60}, 20, Glass{})
		})
	}
	tt := ui.NewTester(view, 200, 120)
	tt.SetPreferences(ui.Preferences{ReduceTransparency: true, TextScale: 1})
	if black, white := overBlackAndWhite(tt, &behind, 70, 50); black != white || black[0] < 250 {
		t.Errorf("painted without transparency, glass over black is %v, over white %v", black, white)
	}
}

func TestGlassRimInHighContrast(t *testing.T) {
	behind := ui.RGB(120, 120, 120)
	g := Glass{}
	tt := glassOver(&behind, &g)
	// How far the rim at the glass's left edge is from its middle, and
	// how far the middle over black is from it over white.
	measure := func() (rim, shows int) {
		tt.Frame()
		img := tt.Image()
		rim = pixelAt(img, 300, 200)[0] - pixelAt(img, 100, 200)[0]
		black, white := overBlackAndWhite(tt, &behind, 300, 200)
		behind = ui.RGB(120, 120, 120)
		return rim, white[0] - black[0]
	}
	rim, shows := measure()
	tt.SetPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
	hcRim, hcShows := measure()
	if hcRim <= rim+40 {
		t.Errorf("in high contrast, the rim is %d from the middle, not more than %d", hcRim, rim)
	}
	if hcShows >= shows*3/4 {
		t.Errorf("in high contrast, %d of what is behind shows through, not less than %d", hcShows, shows)
	}
	// Opaque glass's rim is stronger in high contrast too.
	tt.SetPreferences(ui.Preferences{ReduceTransparency: true, TextScale: 1})
	solidRim, _ := measure()
	tt.SetPreferences(ui.Preferences{ReduceTransparency: true, HighContrast: true, TextScale: 1})
	if hcSolidRim, _ := measure(); hcSolidRim <= solidRim+40 {
		t.Errorf("in high contrast, opaque glass's rim is %d from the middle, not more than %d", hcSolidRim, solidRim)
	}
}

func TestScrollEdgeIsOpaqueWithoutTransparency(t *testing.T) {
	for _, e := range []ScrollEdge{{}, {Hard: true}, {Bottom: true}} {
		behind := ui.RGB(0, 0, 0)
		view := func(c *ui.Context) {
			ui.Box(c).Fill().Background(behind).Children(func() {
				b := ui.Box(c).Absolute().Left(0).Right(0).Height(40).Material(e)
				if e.Bottom {
					b.Bottom(0)
				} else {
					b.Top(0)
				}
			})
		}
		tt := ui.NewTester(view, 200, 100)
		tt.SetScale(2)
		tt.SetPreferences(ui.Preferences{ReduceTransparency: true, TextScale: 1})
		// The rows near the bar and near the other edge.
		near, far := 2, 77
		if e.Bottom {
			near, far = 197, 122
		}
		for _, y := range []int{near, far} {
			black, white := overBlackAndWhite(tt, &behind, 100, y)
			if black != white || black[0] < 250 {
				t.Errorf("%v without transparency: row %d is %v over black, %v over white", e, y, black, white)
			}
		}
		// The hairline along the other edge shows over white.
		line := 80
		if e.Bottom {
			line = 119
		}
		behind = ui.RGB(255, 255, 255)
		tt.Frame()
		if got := pixelAt(tt.Image(), 100, line)[0]; got > 235 {
			t.Errorf("%v without transparency: the hairline over white is %d", e, got)
		}
	}
}
