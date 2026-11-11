//go:build js

// The playground in the browser: the same Lab as cmd/playground, drawn on the page's xterm.js.
package main

import (
	"io"
	"syscall/js"

	"clidle/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type termWriter struct{ fn js.Value }

func (w termWriter) Write(p []byte) (int, error) {
	buf := js.Global().Get("Uint8Array").New(len(p))
	js.CopyBytesToJS(buf, p)
	w.fn.Invoke(buf)
	return len(p), nil
}

func main() {
	page := js.Global().Get("clidle")
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	keys := make(chan []byte, 256)
	in, out := io.Pipe()
	go func() {
		for b := range keys {
			out.Write(b)
		}
	}()
	p := tea.NewProgram(ui.NewLab(), tea.WithAltScreen(), tea.WithInput(in), tea.WithOutput(termWriter{page.Get("write")}),
		tea.WithoutSignalHandler())

	page.Set("input", js.FuncOf(func(_ js.Value, args []js.Value) any {
		keys <- []byte(args[0].String())
		return nil
	}))
	resize := func(cols, rows int) { go p.Send(tea.WindowSizeMsg{Width: cols, Height: rows}) }
	page.Set("resize", js.FuncOf(func(_ js.Value, args []js.Value) any {
		resize(args[0].Int(), args[1].Int())
		return nil
	}))
	resize(page.Get("cols").Int(), page.Get("rows").Int())

	if _, err := p.Run(); err != nil {
		page.Call("fail", err.Error())
		return
	}
	page.Call("exit")
}
