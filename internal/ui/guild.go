package ui

import (
	"fmt"
	"slices"
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

// guildListMaxH caps the room the hero list keeps for itself before the tavern is drawn.
const guildListMaxH = 10

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
	for _, h := range m.g.Guild.Heroes {
		where := lang.T("guild.where.town")
		if h.Queue != 0 {
			where = lang.T("guild.where.waiting")
		}
		if e := h.Exp; e != nil {
			where = lang.T("guild.where.floor", h.Floor(e.Level.Depth))
			if e.Retreating {
				where = lang.T("guild.where.back", h.Floor(e.Level.Depth))
			}
		}
		hero := h.Hero
		text := fmt.Sprintf("%-16s %-10s %-7s HP %d/%d  %s", hero.Name, hero.Class.Name(), lang.T("guild.level", hero.Level),
			hero.HP, hero.MaxHP(), where)
		rows = append(rows, prepRow{section: lang.T("guild.section.heroes"), text: text, style: sValue,
			act: func(m *Model) { m.openHero(h) }, desc: lang.T("guild.hero.desc")})
	}
	return rows
}

// viewGuild is the guild's main screen: the heroes on top and the guild log under them.
func (m Model) viewGuild(h int) string {
	logH := max(3, h/4)
	need := min(guildListMaxH, 2+2*len(m.g.Guild.Heroes))
	var scene []string
	if art := m.tavern(nil); h-logH-2-lipgloss.Height(art)-1 >= need {
		scene = []string{lipgloss.PlaceHorizontal(m.w, lipgloss.Center, art), ""}
	}
	listH := h - logH - 2 - len(scene)
	if len(scene) > 0 {
		listH -= lipgloss.Height(scene[0]) - 1
	}
	list := m.viewRows(nil, m.guildRows(), m.w, listH)
	list = lipgloss.NewStyle().Height(listH).MaxHeight(listH).Render(list)
	list = strings.Join(append(scene, list), "\n")
	log := m.renderLog(m.g.Guild.Log, m.w-2, logH)
	if len(m.g.Guild.Log) == 0 {
		log = sMuted.Render(lang.T("guild.log.empty"))
	}
	return strings.Join([]string{list, sSection.Render(lang.T("guild.log.title")), log}, "\n")
}
