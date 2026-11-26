package ui

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

// priceRowW keeps prices near the item names on wide terminals.
const (
	priceRowW  = 72
	priceNameW = 26
)

// pickDraft is the hero picker: what is being handed out, its price and what giving it to a hero does.
type pickDraft struct {
	item   *game.Item
	potion bool
	price  int
	give   func(h *game.Game)
}

// say shows a short message in the footer; ok tells good news from a refusal.
func (m *Model) say(text string, ok bool) {
	m.notice, m.noticeOK, m.noticeAt = text, ok, time.Now()
}

func (m *Model) needGold(price int) {
	m.say(lang.T("pick.need_gold", game.Num(price), game.Num(m.g.Guild.Gold)), false)
}

func (m Model) shopRows() []prepRow {
	gd := m.g.Guild
	left := max(1, int((game.ShopRefreshMs-gd.ShopMs+59999)/60000))
	sec := lang.T("shop.section", gd.ShopLevel(), left)
	var rows []prepRow
	for _, it := range gd.Shop {
		price := game.ShopPrice(it)
		rows = append(rows, prepRow{section: sec, card: priceCard(string(it.Slot.Glyph())+" "+it.Title(), it.StatLabel(), m.priced(rarityStyle(it.Rarity), price), goldText(price), m.goldFor(price)),
			act: func(m *Model) {
				m.pickFor(it, false, price, func(h *game.Game) { h.BuyShop(slices.Index(gd.Shop, it)) })
			}, desc: lang.T("shop.item.desc", it.Rarity.Name())})
	}
	if len(gd.Shop) == 0 {
		rows = append(rows, prepRow{section: sec, text: lang.T("shop.sold_out"), style: sMuted})
	}
	cost := gd.RestockPrice()
	restock := prepRow{section: sec, card: priceCard(lang.T("shop.restock"), "", m.buyable(cost), goldText(cost), m.goldFor(cost)),
		desc: lang.T("shop.restock.desc", game.RestockPct, left)}
	if cost > gd.Gold {
		restock.act = func(m *Model) { m.needGold(cost) }
	} else {
		restock.confirm = lang.T("confirm.restock", game.Num(cost))
		restock.do = func(m *Model) {
			if gd.Restock() {
				m.cursor = len(gd.Shop)
				m.say(lang.T("shop.restocked", len(gd.Shop), game.Num(cost)), true)
			}
		}
	}
	rows = append(rows, restock)

	potions := lang.T("shop.section.potions")
	price := m.g.PotionCost()
	rows = append(rows, prepRow{section: potions, card: priceCard(lang.T("shop.potion"), "", m.buyable(price), goldText(price), m.goldFor(price)),
		act: func(m *Model) { m.pickFor(nil, true, price, (*game.Game).BuyPotion) }, desc: lang.T("shop.potion.desc", game.MaxPotions)})

	bag := lang.T("shop.section.bag")
	if size, cost, ok := gd.NextBag(); ok {
		rows = append(rows, prepRow{section: bag, card: priceCard(lang.T("shop.bag", size-game.BagSlotsPerLevel, size), "", m.buyable(cost), goldText(cost), m.goldFor(cost)),
			act: func(m *Model) {
				if !gd.UpgradeBag() {
					m.needGold(cost)
					return
				}
				m.say(lang.T("shop.bag_done", gd.BagCap(), game.Num(cost)), true)
			}, desc: lang.T("shop.bag.desc")})
	} else {
		rows = append(rows, prepRow{section: bag, text: lang.T("shop.bag_max", size-game.BagSlotsPerLevel), style: sMuted, desc: lang.T("shop.bag.desc")})
	}
	return rows
}

// chestOrder is the chest as the tab shows it: by slot, then the deepest and strongest items first.
func chestOrder(stash []*game.Item) []*game.Item {
	items := slices.Clone(stash)
	slices.SortStableFunc(items, func(a, b *game.Item) int {
		return cmp.Or(cmp.Compare(a.Slot, b.Slot), cmp.Compare(b.Floor, a.Floor), cmp.Compare(b.Bonus(), a.Bonus()))
	})
	return items
}

func (m Model) chestRows() []prepRow {
	gd := m.g.Guild
	chest := lang.T("prep.section.chest", len(gd.Stash))
	if len(gd.Stash) == 0 {
		return []prepRow{{section: chest, text: lang.T("chest.empty"), style: sMuted, desc: lang.T("chest.empty.desc")}}
	}
	var rows []prepRow
	pages := max(1, (len(gd.Stash)+chestPageSize-1)/chestPageSize)
	page := clamp(*m.chestPage, 0, pages-1)
	*m.chestPage = page
	if pages > 1 {
		row := cfgRow(chest, "prep.chest_page", fmt.Sprintf("%d/%d", page+1, pages), func(d int) {
			*m.chestPage = (page + d + pages) % pages
		})
		row.desc = lang.T("prep.chest_page.desc", chestPageSize)
		rows = append(rows, row)
	}
	items := chestOrder(gd.Stash)
	first := page * chestPageSize
	for _, it := range items[first:min(len(items), first+chestPageSize)] {
		sells := game.Num(it.SellPrice())
		row := prepRow{section: chest, card: priceCard(string(it.Slot.Glyph())+" "+it.Title(), it.StatLabel(), rarityStyle(it.Rarity), lang.T("chest.sells", sells), sGold.Faint(true)),
			act: func(m *Model) {
				m.pickFor(it, false, 0, func(h *game.Game) { h.EquipStash(slices.Index(gd.Stash, it)) })
			},
			sell:    func() { gd.SellStash(slices.Index(gd.Stash, it)) },
			sellMsg: lang.T("chest.sold", it.Title(), sells),
			desc:    lang.T("chest.item.desc", it.Rarity.Name(), sells)}
		if it.Rarity == game.Legendary {
			row.sellAsk = lang.T("confirm.sell_legendary", it.Title(), sells)
		}
		rows = append(rows, row)
	}
	return rows
}

// priceCard is a row with its price on the right: one line when name and stat fit, else the stat goes below.
func priceCard(name, stat string, st lipgloss.Style, gold string, goldSt lipgloss.Style) func(w int) []string {
	return func(w int) []string {
		w = min(w, priceRowW)
		gold := goldSt.Render(gold)
		line := name
		if stat != "" {
			line = fmt.Sprintf("%-*s  %s", priceNameW, name, stat)
		}
		if lipgloss.Width(line)+2+lipgloss.Width(gold) <= w || stat == "" {
			return []string{ends(st.Render(ellipsis(line, w-2-lipgloss.Width(gold))), gold, w)}
		}
		return []string{ends(st.Render(ellipsis(name, w-2-lipgloss.Width(gold))), gold, w), "  " + st.Render(ellipsis(stat, w-2))}
	}
}

// pickFor opens the hero picker for an item (nil for a potion), unless the guild cannot pay for it.
func (m *Model) pickFor(it *game.Item, potion bool, price int, give func(h *game.Game)) {
	if price > m.g.Guild.Gold {
		m.needGold(price)
		return
	}
	m.pick, m.pickBack = &pickDraft{item: it, potion: potion, price: price, give: give}, m.cursor
	m.cursor = m.bestPick()
}

// bestPick is the hero the picker starts on: the one the item helps most, or the one with the fewest potions.
func (m Model) bestPick() int {
	best, score := 0, 0
	found := false
	for i, h := range m.g.Guild.Heroes {
		if m.refuse(h) != "" {
			continue
		}
		s := -h.Hero.Potions
		if m.pick.item != nil {
			s = statDiff(h.Hero, m.pick.item)
		}
		if !found || s > score {
			best, score, found = i, s, true
		}
	}
	return best
}

// statDiff is how much the item raises its stat over what the hero has in that slot.
func statDiff(h *game.Hero, it *game.Item) int {
	if cur := h.Equip[it.Slot]; cur != nil {
		return it.Bonus() - cur.Bonus()
	}
	return it.Bonus()
}

// refuse says why the hero cannot take what the picker hands out, or "".
func (m Model) refuse(h *game.Game) string {
	p := m.pick
	switch {
	case !h.InTown():
		return lang.T("pick.away")
	case p.item != nil && !h.Hero.Class.CanUse(p.item):
		return lang.T("pick.other_class")
	case p.potion && h.Hero.Potions >= game.MaxPotions:
		return lang.T("pick.full")
	}
	return ""
}

// pickHead is the line under the keys: what is being handed out and its price.
func (m Model) pickHead() string {
	p := m.pick
	if p.item == nil {
		return sPotion.Render(lang.T("shop.potion")) + "  " + sGold.Render(goldText(p.price))
	}
	head := rarityStyle(p.item.Rarity).Render(string(p.item.Slot.Glyph())+" "+p.item.Title()) + "  " + sStats.Render(p.item.StatLabel())
	if p.price > 0 {
		head += "  " + sGold.Render(goldText(p.price))
	}
	return head
}

func (m Model) pickRows() []prepRow {
	sec := lang.T("pick.who")
	var rows []prepRow
	for i, h := range m.g.Guild.Heroes {
		why := m.refuse(h)
		card := func(w int) []string { return m.pickCard(h, i, why, w) }
		row := prepRow{section: sec, card: card, desc: lang.T("pick.desc")}
		if why == "" {
			row.act = func(m *Model) { m.giveTo(h) }
		} else {
			row.act = func(m *Model) { m.say(lang.T("pick.cannot", h.Hero.Name, why), false) }
		}
		rows = append(rows, row)
	}
	return rows
}

// pickCard is a hero in the picker: who, then what they have now in the item's place.
func (m Model) pickCard(h *game.Game, i int, why string, w int) []string {
	hero, p := h.Hero, m.pick
	name, at, right := sTitle.Render(hero.Name), heroStyle(i).Render("@"), ""
	if why != "" {
		name, at, right = sMuted.Render(hero.Name), sMuted.Render("@"), sMuted.Render(why)
	}
	now := lang.T("pick.potions", hero.Potions, game.MaxPotions)
	if p.item != nil {
		now = lang.T("pick.empty_slot")
		if cur := hero.Equip[p.item.Slot]; cur != nil {
			now = lang.T("pick.now", cur.Title(), cur.StatLabel())
		}
		if why == "" {
			right = diffLabel(statDiff(hero, p.item), p.item.Slot.StatName())
		}
	}
	who := at + " " + name + "  " + sMuted.Render(hero.Class.Name()+" "+lang.T("guild.level", hero.Level))
	w = min(w, heroCardMaxW)
	return []string{ends(who, right, w), "    " + sMuted.Render(ellipsis(now, w-4))}
}

func diffLabel(diff int, stat string) string {
	switch {
	case diff > 0:
		return sHPGood.Render(fmt.Sprintf("%s ▲%d", stat, diff))
	case diff < 0:
		return sWarn.Render(fmt.Sprintf("%s ▼%d", stat, -diff))
	}
	return sMuted.Render(lang.T("pick.same", stat))
}

// giveTo hands the picked item or potion to h. An item closes the picker; potions keep it open for more.
func (m *Model) giveTo(h *game.Game) {
	p := m.pick
	gold := m.g.Guild.Gold
	var old *game.Item
	if p.item != nil {
		old = h.Hero.Equip[p.item.Slot]
	}
	p.give(h)
	spent := gold - m.g.Guild.Gold
	if p.potion {
		if spent == 0 {
			m.needGold(p.price)
			return
		}
		m.say(lang.T("pick.gave_potion", h.Hero.Name, h.Hero.Potions, game.MaxPotions, game.Num(spent)), true)
		if m.refuse(h) != "" {
			m.cursor = m.bestPick()
		}
		return
	}
	m.pick, m.cursor = nil, m.pickBack
	if h.Hero.Equip[p.item.Slot] != p.item {
		m.needGold(p.price)
		return
	}
	msg := lang.T("pick.gave", h.Hero.Name, p.item.Title())
	if spent > 0 {
		msg += lang.T("pick.spent", game.Num(spent))
	}
	if old != nil {
		msg += lang.T("pick.old_to_chest", old.Title())
	}
	m.say(msg, true)
}
