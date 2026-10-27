package ui

import (
	"time"

	"clidle/internal/game"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	tabDungeon = iota
	tabPrep
	tabHero
)

const (
	tickEvery = 50 * time.Millisecond
	saveEvery = 30 * time.Second
)

type tickMsg time.Time

type Model struct {
	g        *game.Game
	w, h     int
	tab      int
	help     bool
	cursor   int
	lastTick time.Time
	savePath string
	lastSave time.Time
	saveErr  error
}

func New(g *game.Game) Model {
	return Model{g: g, lastTick: time.Now(), lastSave: time.Now()}
}

func NewWithSave(g *game.Game, path string) Model {
	m := New(g)
	m.savePath = path
	return m
}

func (m *Model) save(now time.Time) {
	if m.savePath == "" {
		return
	}
	m.saveErr = m.g.Save(m.savePath, now)
	m.lastSave = now
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
		if now := time.Time(msg); now.Sub(m.lastSave) >= saveEvery {
			m.save(now)
		}
		return m, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.save(time.Now())
			return m, tea.Quit
		case "1":
			m.tab, m.help = tabDungeon, false
		case "2":
			m.tab, m.help = tabPrep, false
		case "3":
			m.tab, m.help = tabHero, false
		case "r":
			m.g.Retreat()
		case "a":
			m.g.ToggleAuto()
		case "e":
			m.g.Depart()
		case "?":
			m.help = !m.help
		default:
			if m.tab == tabPrep && !m.help {
				m.prepKey(msg.String())
			}
		}
	}
	return m, nil
}

func (m *Model) prepKey(key string) {
	rows := m.prepRows()
	m.cursor = clamp(m.cursor, 0, len(rows)-1)
	r := rows[m.cursor]
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(rows)-1, m.cursor+1)
	case "left", "h":
		if r.adjust != nil {
			r.adjust(-1)
		}
	case "right", "l":
		if r.adjust != nil {
			r.adjust(1)
		}
	case "enter", " ":
		if r.enter != nil {
			r.enter()
		}
	}
	m.cursor = clamp(m.cursor, 0, len(m.prepRows())-1)
}
