package ui

import (
	"os"
	"sync/atomic"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"

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
	wakeGap   = 5 * time.Second
)

type tickMsg time.Time

type catchUpDoneMsg game.Summary

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

	away       time.Duration
	catching   bool
	progress   *atomic.Int64
	summary    *game.Summary
	lastDeaths int
	lastBosses int

	confirmText string
	confirmDo   func()
}

func (m *Model) ask(text string, do func()) {
	m.confirmText, m.confirmDo = text, do
}

func New(g *game.Game) Model {
	return Model{g: g, lastTick: time.Now(), lastSave: time.Now(), progress: &atomic.Int64{},
		lastDeaths: g.Deaths, lastBosses: g.BossKills}
}

// NewWithSave catches up on the time spent away before resuming live play.
func NewWithSave(g *game.Game, path string, away time.Duration) Model {
	m := New(g)
	m.savePath = path
	m.away = away
	m.catching = away >= wakeGap
	return m
}

// catchUp runs the offline simulation off the UI loop; while m.catching is set the model must not touch the game.
func (m Model) catchUp(away time.Duration) tea.Cmd {
	m.progress.Store(0)
	g, progress := m.g, m.progress
	return func() tea.Msg {
		return catchUpDoneMsg(g.CatchUp(away, func(f float64) { progress.Store(int64(f * 1000)) }))
	}
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

func (m Model) Init() tea.Cmd {
	if m.catching {
		return tea.Batch(m.catchUp(m.away), tick())
	}
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case catchUpDoneMsg:
		m.catching = false
		m.lastTick = time.Now()
		m.lastDeaths, m.lastBosses = m.g.Deaths, m.g.BossKills
		if s := game.Summary(msg); s.Eventful() {
			m.summary = &s
		}
		m.save(time.Now())
	case tickMsg:
		if m.catching {
			return m, tick()
		}
		now := time.Time(msg)
		gap := now.Sub(m.lastTick)
		if gap >= wakeGap {
			m.lastTick = now
			m.catching = true
			return m, tea.Batch(m.catchUp(gap), tick())
		}
		ms := gap.Milliseconds()
		m.g.Advance(ms)
		m.lastTick = m.lastTick.Add(time.Duration(ms) * time.Millisecond)
		m.bell()
		if now.Sub(m.lastSave) >= saveEvery {
			m.save(now)
		}
		return m, tick()
	case tea.KeyMsg:
		if m.catching {
			return m, nil
		}
		if m.summary != nil {
			m.summary = nil
			return m, nil
		}
		if m.confirmDo != nil {
			switch msg.String() {
			case "s", "S", "y", "Y", "enter":
				m.confirmDo()
				m.confirmDo = nil
			case "n", "N", "esc":
				m.confirmDo = nil
			}
			return m, nil
		}
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
			if e := m.g.Exp; e != nil && !e.Retreating {
				m.ask(lang.T("confirm.retreat"), m.g.Retreat)
			}
		case "a":
			m.g.ToggleAuto()
		case "e":
			if m.g.InTown() {
				m.ask(lang.T("confirm.depart"), m.g.Depart)
			}
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

func (m *Model) bell() {
	g := m.g
	if g.Bell && (g.Deaths > m.lastDeaths || g.BossKills > m.lastBosses) {
		os.Stdout.WriteString("\a")
	}
	m.lastDeaths, m.lastBosses = g.Deaths, g.BossKills
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
		switch {
		case r.confirm != "":
			m.ask(r.confirm, r.enter)
		case r.enter != nil:
			r.enter()
		}
	}
	m.cursor = clamp(m.cursor, 0, len(m.prepRows())-1)
}
