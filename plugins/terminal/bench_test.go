package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkOutput measures how fast the terminal takes a program's output:
// lines of text with some colors.
func BenchmarkOutput(b *testing.B) {
	if err := Load(); err != nil {
		b.Skip(err)
	}
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		b.Fatal(err)
	}
	defer term.Close()
	var sb strings.Builder
	for i := range 1000 {
		fmt.Fprintf(&sb, "\x1b[32m%6d\x1b[0m  the quick brown fox jumps over the lazy dog \x1b[1;34m%x\x1b[0m\r\n", i, i*7919)
	}
	data := []byte(sb.String())
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		term.write(data)
	}
}

// BenchmarkFrame measures a frame of a 1600×1000 window whose every row
// changed, as while a program scrolls.
func BenchmarkFrame(b *testing.B) {
	if err := Load(); err != nil {
		b.Skip(err)
	}
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		b.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 1600, 1000)
	tt.SetScale(2)
	i := 0
	for b.Loop() {
		i++
		term.write([]byte(fmt.Sprintf("\x1b[32m%6d\x1b[0m  the quick brown fox jumps over the lazy dog \x1b[1;34m%x\x1b[0m %s\r\n", i, i*7919, strings.Repeat("=", i%100))))
		for range 60 {
			term.write([]byte("line " + strings.Repeat("abc ", i%40) + "\r\n"))
		}
		tt.Frame()
	}
}

// BenchmarkTailText measures reading the text of the last rows of a
// terminal with a full scrollback of 4 MB, against reading all of its text.
func BenchmarkTailText(b *testing.B) {
	if err := Load(); err != nil {
		b.Skip(err)
	}
	term, err := New(Options{Conn: newPipe(), Scrollback: 4 << 20})
	if err != nil {
		b.Fatal(err)
	}
	defer term.Close()
	term.Resize(120, 50)
	var sb strings.Builder
	for i := 0; sb.Len() < 16<<20; i++ {
		fmt.Fprintf(&sb, "%6d  the quick brown fox jumps over the lazy dog %x\r\n", i, i*7919)
	}
	term.write([]byte(sb.String()))
	b.Run("tail", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			term.TailText(100)
		}
	})
	b.Run("text", func(b *testing.B) {
		b.ReportAllocs()
		n := 0
		for b.Loop() {
			n = len(term.Text())
		}
		b.ReportMetric(float64(n), "text-bytes")
	})
}
