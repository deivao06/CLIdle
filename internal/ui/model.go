package ui

import (
	"os"
	"sync/atomic"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
	"clidle/internal/update"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	tabDungeon = iota
	tabPrep
	tabHero
	tabSettings
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
	heroTop  int
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
	confirmDo   func(*Model)
	confirmTown bool

	choosing bool
	crt      bool

	confirmKeep bool
	cmd         tea.Cmd
	wasInTown   bool
	checking    bool
	lastCheck   time.Time
	release     *update.Release
	updating    bool
	updateErr   string
	updateErrAt time.Time
	restart     bool
}

func (m *Model) ask(text string, do func(*Model)) {
	m.confirmText, m.confirmDo, m.confirmTown, m.confirmKeep = text, do, m.g.InTown(), false
}

func New(g *game.Game) Model {
	return Model{g: g, lastTick: time.Now(), lastSave: time.Now(), progress: &atomic.Int64{},
		lastDeaths: g.Deaths, lastBosses: g.BossKills, crt: g.CRT}
}

// NewWithSave catches up on the time spent away before resuming live play; a fresh game first asks for the class.
func NewWithSave(g *game.Game, path string, away time.Duration, fresh bool) Model {
	m := New(g)
	m.choosing = fresh
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

// resetSave erases the save and starts a fresh game at the class screen.
func (m *Model) resetSave() {
	if m.savePath != "" {
		os.Remove(m.savePath)
	}
	g := game.New(uint64(time.Now().UnixNano()), game.Warrior)
	g.Lang = lang.Current()
	w, h := m.w, m.h
	*m = NewWithSave(g, m.savePath, 0, true)
	m.w, m.h = w, h
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
	if !m.catching {
		m.crt = m.g.CRT
	}
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
		if m.choosing {
			m.lastTick = now
			return m, tick()
		}
		gap := now.Sub(m.lastTick)
		if gap >= wakeGap {
			m.lastTick = now
			m.catching = true
			return m, tea.Batch(m.catchUp(gap), tick())
		}
		ms := gap.Milliseconds()
		m.g.Advance(ms)
		m.lastTick = m.lastTick.Add(time.Duration(ms) * time.Millisecond)
		if m.confirmDo != nil && !m.confirmKeep && m.g.InTown() != m.confirmTown {
			m.confirmDo = nil
		}
		m.bell()
		if now.Sub(m.lastSave) >= saveEvery {
			m.save(now)
		}
		return m, tea.Batch(tick(), m.townArrival(now))
	case updateFoundMsg:
		m.updateFound(msg.r)
	case updateDoneMsg:
		return m, m.updateDone(msg.err)
	case tea.KeyMsg:
		if m.catching {
			return m, nil
		}
		if m.summary != nil {
			m.summary = nil
			return m, nil
		}
		if m.choosing && msg.String() != "ctrl+c" && msg.String() != "q" {
			m.chooseKey(msg.String())
			return m, nil
		}
		if m.confirmDo != nil {
			switch msg.String() {
			case "s", "S", "y", "Y", "enter":
				do := m.confirmDo
				m.confirmDo = nil
				do(&m)
				cmd := m.cmd
				m.cmd = nil
				return m, cmd
			case "n", "N", "esc":
				m.confirmDo = nil
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			if !m.choosing {
				m.save(time.Now())
			}
			return m, tea.Quit
		case "1":
			m.tab, m.help, m.cursor = tabDungeon, false, 0
		case "2":
			m.tab, m.help, m.cursor = tabPrep, false, 0
		case "3":
			m.tab, m.help = tabHero, false
		case "4":
			m.tab, m.help, m.cursor = tabSettings, false, 0
		case "r":
			if e := m.g.Exp; e != nil && !e.Retreating {
				m.ask(lang.T("confirm.retreat"), func(m *Model) { m.g.Retreat() })
			}
		case "a":
			if !m.g.Auto && m.g.InTown() {
				m.ask(lang.T("confirm.auto"), func(m *Model) { m.g.ToggleAuto(); m.g.Depart() })
			} else {
				m.g.ToggleAuto()
			}
		case "e":
			if m.g.InTown() {
				m.ask(lang.T("confirm.depart"), func(m *Model) { m.g.Depart() })
			}
		case "u":
			m.askUpdate()
		case "?":
			m.help = !m.help
		case "esc":
			m.help = false
		default:
			if (m.tab == tabPrep || m.tab == tabSettings || m.tab == tabDungeon && m.g.InTown()) && !m.help {
				m.prepKey(msg.String())
			}
			if m.tab == tabHero && !m.help {
				m.heroKey(msg.String())
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

func (m *Model) heroKey(key string) {
	switch key {
	case "up", "k":
		m.heroTop--
	case "down", "j":
		m.heroTop++
	}
	m.heroTop = clamp(m.heroTop, 0, max(0, len(m.heroLines())-(m.h-2)))
}

func (m *Model) prepKey(key string) {
	rows := m.rows()
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
	case "x":
		if r.sell != nil {
			r.sell()
		}
	case "enter", " ":
		switch {
		case r.confirm != "":
			do := r.do
			if do == nil {
				enter := r.enter
				do = func(*Model) { enter() }
			}
			m.ask(r.confirm, do)
		case r.enter != nil:
			r.enter()
		}
	}
	m.cursor = clamp(m.cursor, 0, len(m.rows())-1)
}
