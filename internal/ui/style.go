package ui

import (
	"clidle/internal/game"

	"github.com/charmbracelet/lipgloss"
)

var (
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	sWall   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	sFloor  = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	sDoor   = lipgloss.NewStyle().Foreground(lipgloss.Color("136"))
	sStairs = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	sHero   = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	sPlain  = lipgloss.NewStyle()

	sTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sTabOn   = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("214")).Bold(true)
	sTabOff  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	sPanel   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("238")).PaddingLeft(1)
	sSection = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

var logStyles = map[game.LogKind]lipgloss.Style{
	game.LogInfo: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	game.LogGood: lipgloss.NewStyle().Foreground(lipgloss.Color("114")),
	game.LogBad:  lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
	game.LogLoot: lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
	game.LogRare: lipgloss.NewStyle().Foreground(lipgloss.Color("75")),
	game.LogEpic: lipgloss.NewStyle().Foreground(lipgloss.Color("177")),
}

func tileStyle(t game.Tile) *lipgloss.Style {
	switch t {
	case game.HWall, game.VWall:
		return &sWall
	case game.Door:
		return &sDoor
	case game.StairsDown, game.StairsUp:
		return &sStairs
	case game.Corridor, game.Floor:
		return &sFloor
	}
	return &sPlain
}
