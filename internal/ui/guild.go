package ui

import (
	"fmt"
	"slices"
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

const (
	// heroCardMaxW keeps a hero's HP and gold near the name on wide terminals.
	heroCardMaxW = 60
	// placeMinW is the least room worth showing a cut dungeon name in.
	placeMinW      = 6
	corridorStepMs = 333
)

// tabs lists the screens of the current view: the guild's, or the open hero's.
func (m Model) tabs() []int {
	if m.guild {
		return []int{tabGuild, tabBuildings, tabRecruit, tabSettings}
	}
	if m.g.Guild.Nerd {
		return []int{tabDungeon, tabPrep, tabHero, tabNerd}
	}
	return []int{tabDungeon, tabPrep, tabHero}
}

func (m *Model) openGuild() {
	m.guild, m.tab, m.help = true, tabGuild, false
	m.cursor = max(0, slices.Index(m.g.Guild.Heroes, m.g))
}

func (m *Model) openHero(h *game.Game) {
	m.g, m.guild, m.tab, m.cursor, m.heroTop, m.help = h, false, tabDungeon, 0, 0, false
	h.Events = h.Events[:0]
	*m.chestPage = 0
	m.lastDeaths, m.lastBosses = h.Deaths, h.BossKills
	m.spendFree(h)
}

// spendFree asks for the hero's free attribute points, if any are left.
func (m *Model) spendFree(h *game.Game) {
	if h.Hero.FreePoints > 0 {
		m.alloc, m.cursor = &allocDraft{recruit: -1, name: h.Hero.Name, class: h.Hero.Class, hero: h}, 0
	}
}

func (m Model) guildRows() []prepRow {
	var rows []prepRow
	for i, h := range m.g.Guild.Heroes {
		card := func(w int) []string {
			return heroCard(h, i, min(w, heroCardMaxW), m.lastTick.UnixMilli()/corridorStepMs)
		}
		rows = append(rows, prepRow{section: lang.T("guild.section.heroes"), card: card, act: func(m *Model) { m.openHero(h) }})
	}
	return rows
}

// heroCard is the i-th hero's two lines in the guild: who they are, then where they are and the gold they found.
func heroCard(h *game.Game, i, w int, frame int64) []string {
	hero := h.Hero
	where := lang.T("guild.where.town")
	if !h.Auto {
		where = lang.T("guild.where.idle")
	}
	if h.Queue != 0 {
		where = lang.T("guild.where.waiting")
	}
	gold := ""
	goldSt := sGold
	if h.LastGold > 0 {
		gold, goldSt = lang.T("guild.gold.last", game.Group(int64(h.LastGold))), sGold.Faint(true)
	}
	if e := h.Exp; e != nil {
		where = lang.T("guild.where.floor", h.Floor(e.Level.Depth))
		if e.Retreating {
			where = lang.T("guild.where.back", h.Floor(e.Level.Depth))
		}
		gold, goldSt = lang.T("guild.gold.trip", game.Group(int64(e.Gold))), sGold
	}
	who := corridor(h, i, frame) + " " + sTitle.Render(hero.Name) + "  " + sMuted.Render(hero.Class.Name()+" "+lang.T("guild.level", hero.Level))
	hp := hpStyle(hero.HP, hero.MaxHP()).Render(fmt.Sprintf("HP %d/%d", hero.HP, hero.MaxHP()))
	tier := " " + lang.T("guild.tier", h.Dungeon+1)
	head := where + " · "
	room := w - 4 - lipgloss.Width(head+tier) - 1 - lipgloss.Width(gold)
	place := ellipsis(h.DungeonPlace(), room)
	if room < placeMinW {
		place, tier = "", strings.TrimSpace(tier)
	}
	return []string{ends(who, hp, w), "    " + ends(sMuted.Render(head+place+tier), goldSt.Render(gold), w-4)}
}

var sFoe = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

// corridor is the three cells before a hero's name: the @ walks down a corridor on an expedition, back the
// other way when coming home, fights a † that blinks, sits at a table at home or waits at the door.
func corridor(h *game.Game, i int, frame int64) string {
	at := heroStyle(i).Render("@")
	floor := sMuted.Render(".")
	e := h.Exp
	switch {
	case e != nil && e.Fighting:
		foe := " "
		if frame%2 == 0 {
			foe = sFoe.Render("†")
		}
		return floor + at + foe
	case e != nil:
		pos := int(frame % 3)
		if e.Retreating {
			pos = 2 - pos
		}
		cells := []string{floor, floor, floor}
		cells[pos] = at
		return strings.Join(cells, "")
	case h.Queue != 0:
		return at + sDoor.Render("+") + " "
	case !h.Auto:
		return tableStyle.Render("π") + sResting.Render("@") + " "
	}
	return tableStyle.Render("π") + at + " "
}

// ends puts left and right on the ends of a w-wide line, cutting left when both do not fit.
func ends(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		left = ellipsis(left, w-1-lipgloss.Width(right))
		gap = w - lipgloss.Width(left) - lipgloss.Width(right)
	}
	return left + strings.Repeat(" ", max(1, gap)) + right
}

func ellipsis(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return truncate(s, max(0, w-1)) + "…"
}

// viewGuild is the guild's main screen: the heroes, the tavern above them when there is room, and the guild
// log with the rest of the height.
func (m Model) viewGuild(h int) string {
	const logMin = 3
	avail := h - 1
	need := 2 + 2*len(m.g.Guild.Heroes)
	var scene []string
	if art := m.tavern(nil); avail-lipgloss.Height(art)-need >= logMin {
		scene = []string{lipgloss.PlaceHorizontal(m.w, lipgloss.Center, art)}
	}
	sceneH := 0
	if len(scene) > 0 {
		sceneH = lipgloss.Height(scene[0])
	}
	listH := max(1, min(need, avail-sceneH-logMin))
	logH := max(1, avail-sceneH-listH)
	list := m.viewRows(nil, m.guildRows(), m.w, listH)
	list = lipgloss.NewStyle().Height(listH).MaxHeight(listH).Render(list)
	list = strings.Join(append(scene, list), "\n")
	log := m.renderLog(m.g.Guild.Log, m.w-2, logH)
	if len(m.g.Guild.Log) == 0 {
		log = sMuted.Render(lang.T("guild.log.empty"))
	}
	return strings.Join([]string{list, sSection.Render(lang.T("guild.log.title")), log}, "\n")
}
