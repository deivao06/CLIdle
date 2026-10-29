package game

import (
	"fmt"

	"clidle/internal/lang"
)

type Trait int

const (
	Str Trait = iota
	Dex
	Int
	Con
	NumTraits
)

func (t Trait) Name() string {
	return lang.T([...]string{"trait.str", "trait.dex", "trait.int", "trait.con"}[t])
}

func (t Trait) Effect() string {
	if t == Con {
		return fmt.Sprintf("+%d HP", TrainHP)
	}
	return "+1 " + lang.T([...]string{"stat.str", "stat.dex", "stat.int"}[t])
}

func (g *Game) InTown() bool { return g.Exp == nil }

// Depart starts an expedition right away when the hero is resting in town.
func (g *Game) Depart() {
	if g.InTown() {
		g.StartExpedition()
	}
}

func (g *Game) ToggleAuto() {
	g.Auto = !g.Auto
	if g.Auto {
		g.AutoDeaths = 0
		g.logf(LogInfo, "%s", lang.T("log.auto_on"))
	} else {
		g.logf(LogInfo, "%s", lang.T("log.auto_off"))
	}
}

func (g *Game) refreshShop() {
	g.Shop = g.Shop[:0]
	for range ShopSize {
		g.Shop = append(g.Shop, newItem(g.RNG, max(1, g.MaxDepth)))
	}
}

func ShopPrice(it *Item) int { return it.Value() * ShopMarkup }

func (g *Game) spend(cost int) bool {
	if g.Gold < cost {
		g.logf(LogBad, "%s", lang.T("log.no_gold", cost))
		return false
	}
	g.Gold -= cost
	return true
}

func (g *Game) BuyShop(i int) {
	if i < 0 || i >= len(g.Shop) {
		return
	}
	if !g.Hero.Class.CanUse(g.Shop[i]) {
		g.logf(LogInfo, "%s", lang.T("log.cannot_use", g.Shop[i].Title()))
		return
	}
	if !g.spend(ShopPrice(g.Shop[i])) {
		return
	}
	it := g.Shop[i]
	g.Shop = append(g.Shop[:i], g.Shop[i+1:]...)
	if g.InTown() {
		g.equipBought(it)
		return
	}
	g.Pending = append(g.Pending, it)
	g.logf(LogInfo, "%s", lang.T("log.bought_later", it.Title()))
}

func (g *Game) equipBought(it *Item) {
	h := g.Hero
	msg := lang.T("log.hero_equips", it.Title(), it.StatLabel())
	if old := h.Equip[it.Slot]; old != nil {
		g.Gold += old.Value()
		msg += lang.T("log.sold_old", old.Title(), old.Value())
	}
	h.Equip[it.Slot] = it
	g.logf(it.Rarity.LogKind(), "%s", msg)
}

// UpgradeTarget is the item the smith will work on and how many upgrades are already queued for it.
func (g *Game) UpgradeTarget(s Slot) (*Item, int) {
	it := g.Hero.Equip[s]
	for _, p := range g.Pending {
		if p.Slot == s {
			it = p
		}
	}
	return it, g.Upgrades[s]
}

func (g *Game) UpgradeCost(s Slot) int {
	it, queued := g.UpgradeTarget(s)
	if it == nil {
		return 0
	}
	return UpgradeCost(it, it.Upgrade+queued)
}

func (g *Game) Upgrade(s Slot) {
	it, _ := g.UpgradeTarget(s)
	if it == nil || !g.spend(g.UpgradeCost(s)) {
		return
	}
	if g.InTown() {
		it.Upgrade++
		g.logf(LogGood, "%s", lang.T("log.smith_done", it.Title(), it.StatLabel()))
		return
	}
	g.Upgrades[s]++
	g.logf(LogInfo, "%s", lang.T("log.smith_queued", it.Label()))
}

func (g *Game) Train(t Trait) {
	if !g.spend(TrainCost(t, g.Hero.Training[t])) {
		return
	}
	g.Hero.Training[t]++
	g.logf(LogGood, "%s", lang.T("log.trained", t.Name(), g.Hero.Training[t], t.Effect()))
}

func (g *Game) buyPotions() {
	h := g.Hero
	n := min(g.Config.PotionsToBring-h.Potions, g.Gold/PotionPrice)
	if n <= 0 {
		return
	}
	g.Gold -= n * PotionPrice
	h.Potions += n
	key := "log.buy_potions"
	if n == 1 {
		key = "log.buy_potion"
	}
	g.logf(LogInfo, "%s", lang.T(key, n, n*PotionPrice))
}

func (g *Game) arriveInTown() {
	g.Hero.HP = g.Hero.MaxHP()
	if n := g.Hero.Potions - MaxPotions; n > 0 {
		g.Hero.Potions = MaxPotions
		g.Gold += n * PotionSellPrice
		key := "log.sell_potions"
		if n == 1 {
			key = "log.sell_potion"
		}
		g.logf(LogInfo, "%s", lang.T(key, n, n*PotionSellPrice))
	}
	for _, it := range g.Pending {
		g.equipBought(it)
	}
	g.Pending = nil
	for s, n := range g.Upgrades {
		if it := g.Hero.Equip[s]; it != nil && n > 0 {
			it.Upgrade += n
			g.logf(LogGood, "%s", lang.T("log.smith_done", it.Title(), it.StatLabel()))
		}
	}
	g.Upgrades = [NumSlots]int{}
	g.refreshShop()
}
