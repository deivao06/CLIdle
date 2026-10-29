package game

import "fmt"

type LogKind int

const (
	LogInfo LogKind = iota
	LogGood
	LogBad
	LogLoot
	LogRare
	LogEpic
	LogMagic
	LogLegendary
)

type LogEntry struct {
	Text  string
	Kind  LogKind
	Count int
}

func (g *Game) logf(kind LogKind, format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	if n := len(g.Log); n > 0 && g.Log[n-1].Text == text && g.Log[n-1].Kind == kind {
		g.Log[n-1].Count++
		return
	}
	g.Log = append(g.Log, LogEntry{text, kind, 1})
	if len(g.Log) > LogSize {
		g.Log = g.Log[len(g.Log)-LogSize:]
	}
}
