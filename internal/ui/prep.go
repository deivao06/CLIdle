package ui

import (
	"fmt"
	"slices"
	"strings"

	"clidle/internal/game"

	"github.com/charmbracelet/lipgloss"
)

type prepRow struct {
	section string
	text    string
	style   lipgloss.Style
	adjust  func(delta int)
	enter   func()
}

func onOff(b bool) string {
	if b {
		return "sim"
	}
	return "não"
}

func step(v *int, delta, by, lo, hi int) { *v = clamp(*v+delta*by, lo, hi) }

func (m Model) prepRows() []prepRow {
	g := m.g
	c := &g.Config
	var rows []prepRow
	if g.InTown() {
		rows = append(rows, prepRow{section: "Próxima expedição", text: "▶ Partir agora", style: sTitle, enter: g.Depart})
	}
	cfg := func(label, value string, adjust func(int)) {
		rows = append(rows, prepRow{
			section: "Próxima expedição",
			text:    fmt.Sprintf("%-22s %s", label, value),
			style:   sPlain,
			adjust:  adjust,
			enter:   func() { adjust(1) },
		})
	}
	auto := "desligada"
	if g.Auto {
		auto = "ligada"
	}
	cfg("Auto-expedição", auto, func(int) { g.ToggleAuto() })
	mode := "explorar tudo"
	if c.Mode == game.DescendFast {
		mode = "descer direto"
	}
	cfg("Modo", mode, func(int) { c.Mode = 1 - c.Mode })
	cfg("Levar poções", fmt.Sprintf("%d (%d ouro cada)", c.PotionsToBring, game.PotionPrice),
		func(d int) { step(&c.PotionsToBring, d, 1, 0, 30) })
	cfg("Beber poção com HP <", fmt.Sprintf("%d%%", c.PotionAtPct), func(d int) { step(&c.PotionAtPct, d, 5, 5, 90) })
	retreat := "nunca"
	if c.RetreatHPPct > 0 {
		retreat = fmt.Sprintf("%d%%", c.RetreatHPPct)
	}
	cfg("Recuar sem poção, HP <", retreat, func(d int) { step(&c.RetreatHPPct, d, 5, 0, 90) })
	cfg("Recuar mochila cheia", onOff(c.RetreatBagFull), func(int) { c.RetreatBagFull = !c.RetreatBagFull })
	last := "sem limite"
	if c.LastFloor > 0 {
		last = fmt.Sprintf("andar %d", c.LastFloor)
	}
	cfg("Último andar", last, func(d int) { step(&c.LastFloor, d, 1, 0, 999) })
	floors := g.ShortcutList()
	at := max(0, slices.Index(floors, c.StartFloor))
	cfg("Começar no andar", fmt.Sprintf("%d (%d atalhos)", floors[at], len(floors)-1), func(d int) {
		c.StartFloor = floors[(at+d+len(floors))%len(floors)]
	})
	cfg("Bip em morte/chefe", onOff(g.Bell), func(int) { g.Bell = !g.Bell })

	for i, it := range g.Shop {
		price := game.ShopPrice(it)
		text := fmt.Sprintf("%c %-22s %-8s %6d ouro%s", it.Slot.Glyph(), it.Title(), it.StatLabel(), price, m.compare(it))
		rows = append(rows, prepRow{section: "Loja", text: text, style: m.priced(rarityStyle(it.Rarity), price),
			enter: func() { g.BuyShop(i) }})
	}
	if len(g.Shop) == 0 {
		rows = append(rows, prepRow{section: "Loja", text: "esgotada: renova quando o herói voltar", style: sMuted})
	}
	for _, it := range g.Pending {
		rows = append(rows, prepRow{section: "Loja", text: fmt.Sprintf("na fila: %s (%s)", it.Title(), it.StatLabel()), style: sMuted})
	}

	for s := range game.NumSlots {
		it, queued := g.UpgradeTarget(s)
		if it == nil {
			rows = append(rows, prepRow{section: "Ferreiro", text: fmt.Sprintf("%c sem %s", s.Glyph(), slotNames[s]), style: sMuted})
			continue
		}
		next := *it
		next.Upgrade += queued + 1
		cost := g.UpgradeCost(s)
		text := fmt.Sprintf("%c %-22s → %-8s %6d ouro", s.Glyph(), it.Name+fmt.Sprintf(" +%d", it.Upgrade+queued), next.StatLabel(), cost)
		if queued > 0 {
			text += fmt.Sprintf("  (+%d na fila)", queued)
		}
		rows = append(rows, prepRow{section: "Ferreiro", text: text, style: m.priced(sPlain, cost),
			enter: func() { g.Upgrade(s) }})
	}

	for t := range game.NumTraits {
		lvl := g.Hero.Training[t]
		cost := game.TrainCost(t, lvl)
		text := fmt.Sprintf("%-12s nv %-3d %-8s %6d ouro", t.Name(), lvl, t.Effect(), cost)
		rows = append(rows, prepRow{section: "Treino", text: text, style: m.priced(sPlain, cost),
			enter: func() { g.Train(t) }})
	}
	return rows
}

func (m Model) priced(st lipgloss.Style, cost int) lipgloss.Style {
	if cost > m.g.Gold {
		return sMuted
	}
	return st
}

// compare shows how a shop item changes the stat of the currently equipped one.
func (m Model) compare(it *game.Item) string {
	cur := m.g.Hero.Equip[it.Slot]
	diff := it.Bonus()
	if cur != nil {
		diff -= cur.Bonus()
	}
	if diff > 0 {
		return fmt.Sprintf("  ▲%d", diff)
	}
	return fmt.Sprintf("  ▼%d", -diff)
}

func (m Model) viewPrep(h int) string {
	g := m.g
	rows := m.prepRows()
	where := "Herói na cidade: compras valem agora."
	if !g.InTown() {
		where = "Herói na dungeon: config vale na próxima expedição, equipamento entra quando ele voltar."
	}
	head := []string{
		sGold.Render(fmt.Sprintf("Ouro %d", g.Gold)) + sMuted.Render(" · ") + sMuted.Render(where),
		sMuted.Render("↑↓ escolher · ←→ ajustar · enter comprar/alternar"),
	}
	var lines []string
	cursorLine := 0
	section := ""
	for i, r := range rows {
		if r.section != section {
			section = r.section
			lines = append(lines, "", sSection.Render(section))
		}
		text := truncate(r.text, m.w-4)
		if i == m.cursor {
			cursorLine = len(lines)
			lines = append(lines, sTabOn.Render("›")+" "+r.style.Render(text))
		} else {
			lines = append(lines, "  "+r.style.Render(text))
		}
	}
	avail := max(1, h-len(head))
	top := clamp(cursorLine-avail/2, 0, max(0, len(lines)-avail))
	lines = lines[top:min(len(lines), top+avail)]
	return strings.Join(append(head, lines...), "\n")
}
