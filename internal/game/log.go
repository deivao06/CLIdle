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
)

type LogEntry struct {
	Text string
	Kind LogKind
}

func (g *Game) logf(kind LogKind, format string, args ...any) {
	g.Log = append(g.Log, LogEntry{fmt.Sprintf(format, args...), kind})
	if len(g.Log) > LogSize {
		g.Log = g.Log[len(g.Log)-LogSize:]
	}
}
