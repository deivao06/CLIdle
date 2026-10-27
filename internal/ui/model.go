package ui

import (
	"time"

	"clidle/internal/game"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	tabDungeon = iota
	tabTown
	tabHero
)

const tickEvery = 50 * time.Millisecond

type tickMsg time.Time

type Model struct {
	g        *game.Game
	w, h     int
	tab      int
	help     bool
	lastTick time.Time
}

func New(g *game.Game) Model {
	return Model{g: g, lastTick: time.Now()}
}

func tick() tea.Cmd {
	return tea.Tick(tickEvery, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tickMsg:
		ms := time.Time(msg).Sub(m.lastTick).Milliseconds()
		m.g.Advance(ms)
		m.lastTick = m.lastTick.Add(time.Duration(ms) * time.Millisecond)
		return m, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.tab = tabDungeon
		case "2":
			m.tab = tabTown
		case "3":
			m.tab = tabHero
		case "?":
			m.help = !m.help
		}
	}
	return m, nil
}
