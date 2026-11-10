//go:build js

package main

import (
	"io"
	"syscall/js"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
	"clidle/internal/ui"
	"clidle/internal/update"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const savePath = "clidle/save.json"

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

	g, savedAt, err := game.Load(savePath)
	if err != nil {
		page.Call("fail", lang.T("main.load_error", savePath, err))
		return
	}
	var away time.Duration
	fresh := g == nil
	if fresh {
		g = game.New(uint64(time.Now().UnixNano()), game.Warrior)
	} else {
		away = time.Since(savedAt)
	}
	lang.Set(g.Lang)
	update.Site = js.Global().Get("location").Get("href").String()

	keys := make(chan []byte, 256)
	in, out := io.Pipe()
	go func() {
		for b := range keys {
			out.Write(b)
		}
	}()
	p := tea.NewProgram(ui.NewWithSave(g, savePath, away, fresh),
		tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithInput(in), tea.WithOutput(termWriter{page.Get("write")}),
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

	final, err := p.Run()
	if err != nil {
		page.Call("fail", err.Error())
		return
	}
	if m, ok := final.(ui.Model); ok && m.Restarting() {
		page.Call("reload")
		return
	}
	page.Call("exit")
}
