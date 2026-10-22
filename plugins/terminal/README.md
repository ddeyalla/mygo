# terminal

A terminal for [MyGo](https://github.com/egoist/mygo) apps of native UI: a
view that runs the user's shell, or any program, in a pseudo-terminal, with
[Ghostty](https://ghostty.org)'s terminal emulator, libghostty-vt, loaded
at run time with no cgo.

```go
import "github.com/egoist/mygo/plugins/terminal"

term, err := terminal.New(terminal.Options{})
// ...
mygo.NewWindow(mygo.WindowOptions{Title: "Terminal", Content: ui.View(func(c *ui.Context) {
	terminal.View(c, term).Fill().AutoFocus()
})})
```

A host that keeps terminals attached while their views are omitted can call
`term.SetVisible(false)` to suppress redraw requests. Output and callbacks
continue; `term.SetVisible(true)` requests a fresh frame. Terminals are visible
by default. Apply visibility once after composing the frame, so a terminal
that stays visible is not hidden and shown on every frame.

See [the documentation](../../docs/plugins/terminal.md), and
`examples/terminal` for an app.

`go generate` here builds libghostty-vt for every platform with Zig, from
the commit of Ghostty in `internal/libbuild`, into `build/`, and writes
`mygo-plugin.json`, which names the files and their SHA-256 for the CLI and
this package: publish them as the assets of the release that file points
at. It writes `ghostty_themes.go` too, the themes that Ghostty ships, which
`go run ./internal/libbuild -themes` writes alone, without Zig.
