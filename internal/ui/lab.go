package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"clidle/internal/game"
	"clidle/internal/lang"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const (
	labMap = iota
	labHero
	labGear
	labFloor
	labTactics
	labConfig
	labPages
)

const (
	labMaxLevel = 60
	labMaxDepth = 200
	labMaxTrain = 50
	labMaxCount = 12
	labMaxSteps = 5000
	labBigStep  = 10
	labSeedLen  = 20
	labCellW    = 16
	labLeftMinW = 3*labCellW + 6
	labMinW     = labLeftMinW + 2 + game.MapW
	labMinH     = game.MapH + 9
)

var labSpeeds = []int{25, 50, 100, 200, 500, 1000, 5000}

var sKey = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

// labState is what the playground rows change besides the scenario.
type labState struct {
	floor  int
	col    int
	room   int
	inRoom bool
	typing bool
	buf    string
	warn   string
}

// Lab is the playground: it builds a fight from a scenario and plays it with the game's own rules.
// The embedded Model draws the map, the hero and the rule editor; Model.tab is the playground page.
type Lab struct {
	Model
	sc      *game.Scenario
	st      *labState
	built   string
	running bool
	speed   int
	acc     int64
}

// NewLab opens the playground on the default scenario, in English; it never reads or writes the save.
func NewLab() Lab {
	sc := game.DefaultScenario()
	l := Lab{sc: &sc, st: &labState{floor: 1, col: 1, room: game.LabRooms / 2}, speed: slices.Index(labSpeeds, 100)}
	l.Model = New(game.NewLab(sc))
	l.built = fmt.Sprint(sc)
	return l
}

func (l *Lab) rebuild() {
	effects := l.g.Effects
	l.g = game.NewLab(*l.sc)
	l.g.Effects = effects
	l.built = fmt.Sprint(*l.sc)
	l.running, l.acc, l.fx = false, 0, nil
}

// sync copies saved rules into the scenario and rebuilds the run when the scenario changed.
func (l *Lab) sync(before []game.Gambit) {
	if gs := l.g.Config.Gambits; !slices.Equal(before, gs) {
		l.sc.Gambits = slices.Clone(gs)
	}
	if fmt.Sprint(*l.sc) != l.built {
		l.rebuild()
	}
}

func (l Lab) over() bool { return l.g.Lab().End != game.LabRunning }

func (l Lab) Init() tea.Cmd { return tick() }

func (l Lab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.w, l.h = msg.Width, msg.Height
	case tickMsg:
		l.advance(time.Time(msg))
		return l, tick()
	case tea.KeyMsg:
		if l.key(msg.String()) {
			return l, tea.Quit
		}
	}
	return l, nil
}

func (l *Lab) advance(now time.Time) {
	gap := now.Sub(l.lastTick).Milliseconds()
	l.lastTick = now
	if l.running {
		l.acc += gap * int64(labSpeeds[l.speed]) / 100
		for n := 0; !l.over(); n++ {
			s := l.g.StepMs()
			if l.acc < s {
				break
			}
			if n == labMaxSteps {
				l.acc = 0
				break
			}
			l.acc -= s
			l.g.LabStep()
		}
		l.running = !l.over()
	}
	l.collectFx(now)
}

func (l *Lab) play() {
	if l.g.Exp == nil {
		return
	}
	if l.over() {
		l.rebuild()
	}
	l.running, l.acc = !l.running, 0
}

// key handles a key press and reports whether to quit.
func (l *Lab) key(k string) bool {
	before := slices.Clone(l.g.Config.Gambits)
	defer l.sync(before)
	switch {
	case k == "ctrl+c":
		return true
	case l.st.typing:
		l.typeSeed(k)
		return false
	case l.confirmDo != nil:
		switch k {
		case "y", "Y", "enter":
			do := l.confirmDo
			l.confirmDo = nil
			do(&l.Model)
		case "n", "N", "esc":
			l.confirmDo = nil
		}
		return false
	}
	l.st.warn = ""
	switch k {
	case "q":
		return true
	case "1", "2", "3", "4", "5", "6":
		page := int(k[0] - '1')
		l.st.inRoom = false
		l.leave(func(m *Model) { m.tab, m.cursor, m.listFree = page, 0, false })
	case " ":
		l.play()
	case "p":
		if l.g.Exp != nil && !l.over() {
			l.running = false
			l.g.LabStep()
			l.collectFx(time.Now())
		}
	case "+", "=":
		l.speed = min(len(labSpeeds)-1, l.speed+1)
	case "-", "_":
		l.speed = max(0, l.speed-1)
	case "r":
		l.rebuild()
	case "n":
		l.sc.Seed = uint64(time.Now().UnixNano())
	case "v":
		l.seeAll = !l.seeAll
	default:
		if l.tab != labMap {
			l.listKey(k)
		}
	}
	return false
}

func (l *Lab) typeSeed(k string) {
	st := l.st
	switch r := []rune(k); {
	case k == "enter":
		if n, err := strconv.ParseUint(st.buf, 10, 64); err == nil {
			l.sc.Seed = n
		}
		st.typing = false
	case k == "esc":
		st.typing = false
	case k == "backspace":
		if rs := []rune(st.buf); len(rs) > 0 {
			st.buf = string(rs[:len(rs)-1])
		}
	case len(r) == 1 && unicode.IsDigit(r[0]) && len(st.buf) < labSeedLen:
		st.buf += k
	}
}

func (l *Lab) listKey(k string) {
	rows := l.rows()
	if len(rows) == 0 {
		return
	}
	l.cursor = clamp(l.cursor, 0, len(rows)-1)
	r := rows[l.cursor]
	nudge := func(d int) {
		if r.adjust != nil {
			r.adjust(d)
		}
	}
	switch k {
	case "up", "k":
		l.cursor, l.listFree = max(0, l.cursor-1), false
	case "down", "j":
		l.cursor, l.listFree = min(len(rows)-1, l.cursor+1), false
	case "left", "h":
		nudge(-1)
	case "right", "l":
		nudge(1)
	case "pgdown":
		if l.tab != labTactics {
			nudge(-labBigStep)
		}
	case "pgup":
		if l.tab != labTactics {
			nudge(labBigStep)
		}
	case "x":
		if r.sell != nil {
			r.sell()
		}
	case "esc":
		switch {
		case l.tab == labFloor && l.st.inRoom:
			l.st.leaveRoom(&l.Model)
		case l.edit > 0:
			l.leave(func(*Model) {})
		}
	case "enter":
		switch {
		case r.confirm != "":
			l.ask(r.confirm, r.do)
		case r.act != nil:
			r.act(&l.Model)
		case r.enter != nil:
			r.enter()
		}
	}
	if l.tab == labFloor && !l.st.inRoom && l.cursor >= labGridRow && l.cursor < labGridRow+3 {
		l.st.room = (l.cursor-labGridRow)*3 + l.st.col
	}
	l.cursor = clamp(l.cursor, 0, len(l.rows())-1)
}

func (l Lab) rows() []prepRow {
	switch l.tab {
	case labHero:
		return l.heroRows()
	case labGear:
		return l.gearRows()
	case labFloor:
		return l.floorRows()
	case labTactics:
		if l.editing() {
			return l.ruleRows(l.edit - 1)
		}
		return l.tacticsRows(0)
	case labConfig:
		return l.configRows()
	}
	return nil
}

// hints renders key/description pairs with the keys in the theme's orange.
func hints(w int, pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, sKey.Render(pairs[i])+" "+sMuted.Render(pairs[i+1]))
	}
	sep := sMuted.Render(" · ")
	for len(parts) > 1 && lipgloss.Width(strings.Join(parts, sep)) > w {
		parts = parts[:len(parts)-1]
	}
	return truncate(strings.Join(parts, sep), w)
}

func (l Lab) View() string { return zone.Scan(l.view()) }

func (l Lab) view() string {
	if l.w == 0 {
		return ""
	}
	if l.w < labMinW || l.h < labMinH {
		msg := sWarn.Render(lang.T("ui.too_small")) + "\n" +
			sMuted.Render(lang.T("ui.too_small_size", labMinW, labMinH, l.w, l.h))
		return lipgloss.Place(l.w, l.h, lipgloss.Center, lipgloss.Center, msg)
	}
	bodyH := l.h - 3
	var body string
	if l.confirmDo != nil {
		title := sTitle.Width(min(lipgloss.Width(l.confirmText), l.w-12)).Render(l.confirmText)
		body = lipgloss.Place(l.w, bodyH, lipgloss.Center, lipgloss.Center, sBox.Render(title+"\n\n"+l.confirmButtons()))
	} else {
		leftW := l.w - game.MapW - 2
		left := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(l.viewPage(leftW, bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, sPanel.Height(bodyH).Render(l.viewMap(bodyH)))
	}
	body = lipgloss.NewStyle().Width(l.w).Height(bodyH).MaxHeight(bodyH).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, l.header()+"\n", body, l.footer())
}

// viewMap is the right column: the run, or on the floor page a preview of the floor being edited.
func (l Lab) viewMap(h int) string {
	m := l.Model
	m.w = game.MapW
	if l.tab == labFloor {
		g := game.LabPreview(*l.sc, l.st.floor)
		if g == nil {
			return sMuted.Render(lang.T("lab.floor.none", l.st.floor))
		}
		m.g, m.fx, m.seeAll = g, nil, true
		g.Effects = l.g.Effects
		return sTitle.Render(lang.T("lab.preview", l.st.floor)) + "\n" + m.renderMap(game.MapW, min(game.MapH, h-1))
	}
	if l.g.Exp == nil {
		return sMuted.Render(lang.T("lab.no_floors"))
	}
	m.seeAll = l.seeAll
	return m.viewDungeon(h)
}

// viewPage is the left column: the hero on the map page, the page's list on the others.
func (l Lab) viewPage(w, h int) string {
	if l.tab == labMap {
		pw := min(heroColW, w-2)
		return l.heroPanel(pw, h, true) + l.heroAbilityNames(pw)
	}
	keys := []string{"↑↓", lang.T("lab.key.choose"), "←→", lang.T("lab.key.adjust"), "enter", lang.T("lab.key.enter")}
	switch {
	case l.tab == labTactics && l.editing():
		i := l.edit - 1
		head := []string{sTitle.Render(l.draft[i].Label()), hints(w, "↑↓", lang.T("lab.key.choose"), "←→", lang.T("lab.key.adjust"),
			"esc", lang.T("lab.key.back"))}
		return l.viewRows(head, l.ruleRows(i), w, h)
	case l.tab == labFloor && l.st.inRoom:
		keys = append(keys, "x", lang.T("lab.key.remove"), "esc", lang.T("lab.key.back"))
	}
	if l.tab != labTactics && l.tab != labConfig {
		keys = append(keys, "pgup/pgdn", "±10")
	}
	head := []string{hints(w, keys...)}
	if l.tab != labConfig {
		head = append(head, sMuted.Render(lang.T("lab.keys.reset")))
	}
	if l.tab == labFloor && l.st.inRoom {
		head = append(head, "", sSection.Render(lang.T("lab.section.rooms")+lang.T("lab.state.floor", l.st.floor)))
		for y := range 3 {
			head = append(head, "  "+l.gridLine(y))
		}
	}
	if l.st.warn != "" {
		head = append(head, sWarn.Render(l.st.warn))
	}
	return l.viewRows(head, l.rows(), w, h)
}

func (l Lab) runState() string {
	speed := fmt.Sprintf("%g×", float64(labSpeeds[l.speed])/100)
	seed := sMuted.Render(lang.T("lab.state.seed", l.sc.Seed))
	switch {
	case l.g.Exp == nil:
		return sMuted.Render(lang.T("lab.state.empty")) + seed
	case l.over():
		return sWarn.Render(lang.T([...]string{"", "lab.state.dead", "lab.state.out", "lab.state.down"}[l.g.Lab().End])) + seed
	case l.running:
		return sHPGood.Render(lang.T("lab.state.run", speed)) + seed
	}
	return sTitle.Render(lang.T("lab.state.pause", speed)) + seed
}

func (l Lab) header() string {
	h := l.g.Hero
	right := hpStyle(h.HP, h.MaxHP()).Render(fmt.Sprintf("HP %d/%d", h.HP, h.MaxHP()))
	if e := l.g.Exp; e != nil {
		right += sMuted.Render(lang.T("lab.state.floor", e.Level.Depth))
	}
	right += "  " + l.runState() + " "
	keys := []string{"lab.tab.map", "lab.tab.hero", "lab.tab.gear", "lab.tab.floor", "lab.tab.tactics", "lab.tab.config"}
	var parts []string
	for i, k := range keys {
		if i == l.tab {
			parts = append(parts, sTabOn.Render(fmt.Sprintf(" %d %s ", i+1, lang.T(k))))
			continue
		}
		parts = append(parts, " "+sKey.Render(fmt.Sprint(i+1))+" "+sTabOff.Render(lang.T(k))+" ")
	}
	left := strings.Join(parts, "")
	if gap := l.w - lipgloss.Width(left) - lipgloss.Width(right); gap >= 1 {
		return left + strings.Repeat(" ", gap) + right
	}
	return truncate(left, l.w)
}

func (l Lab) footer() string {
	switch {
	case l.st.typing:
		return hints(l.w, lang.T("lab.key.type"), lang.T("lab.key.seed"), "enter", lang.T("lab.key.confirm"), "esc", lang.T("lab.key.cancel"))
	case l.confirmDo != nil:
		return hints(l.w, "y/enter", lang.T("lab.key.confirm"), "n/esc", lang.T("lab.key.cancel"))
	}
	return hints(l.w, lang.T("lab.key.space"), lang.T("lab.key.play"), "p", lang.T("lab.key.step"), "+/-", lang.T("lab.key.speed"),
		"r", lang.T("lab.key.restart"), "n", lang.T("lab.key.new_seed"), "v", lang.T("lab.key.see_all", onOff(l.seeAll)),
		"1-6", lang.T("lab.key.screens"), "q", lang.T("lab.key.quit"))
}
