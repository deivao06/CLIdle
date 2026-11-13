package game

import (
	"slices"

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
		g.Shop = append(g.Shop, newItemOf(g.RNG, max(1, g.MaxDepth), rollRarity(g.RNG, Epic)))
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
	if !g.InTown() || i < 0 || i >= len(g.Shop) {
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
	g.equipBought(it)
}

func (g *Game) equipBought(it *Item) {
	h := g.Hero
	msg := lang.T("log.hero_equips", it.Title(), it.StatLabel())
	if old := h.Equip[it.Slot]; old != nil {
		g.Stash = append(g.Stash, old)
		msg += lang.T("log.kept_old", old.Title())
	}
	h.Equip[it.Slot] = it
	h.HP = min(h.HP, h.MaxHP())
	g.logf(it.Rarity.LogKind(), "%s", msg)
}

func (g *Game) UpgradeCost(s Slot) int {
	it := g.Hero.Equip[s]
	if it == nil {
		return 0
	}
	return UpgradeCost(it, it.Upgrade)
}

func (g *Game) Upgrade(s Slot) {
	it := g.Hero.Equip[s]
	if !g.InTown() || it == nil || it.Upgrade >= UpgradeMax || !g.spend(g.UpgradeCost(s)) {
		return
	}
	it.Upgrade++
	it.Forged += it.UpgradeStep()
	g.logf(LogGood, "%s", lang.T("log.smith_done", it.Title(), it.StatLabel()))
}

func (g *Game) BagCap() int { return BagSize + g.BagLevel*BagSlotsPerLevel }

// NextBag is the bag size after the next upgrade and its cost; ok is false at the maximum.
func (g *Game) NextBag() (size, cost int, ok bool) {
	lvl := g.BagLevel
	return BagSize + (lvl+1)*BagSlotsPerLevel, BagCost(lvl), lvl < BagMaxLevel
}

func (g *Game) UpgradeBag() {
	_, cost, ok := g.NextBag()
	if !g.InTown() || !ok || !g.spend(cost) {
		return
	}
	g.BagLevel++
	g.logf(LogGood, "%s", lang.T("log.bag_done", g.BagCap()))
}

func (g *Game) Train(t Trait) {
	if !g.InTown() || !g.spend(TrainCost(g.Hero.Training[t])) {
		return
	}
	g.Hero.Training[t]++
	g.logf(LogGood, "%s", lang.T("log.trained", t.Name(), g.Hero.Training[t], t.Effect()))
}

func (g *Game) BuyPotion() {
	if !g.InTown() || g.Hero.Potions >= MaxPotions || !g.spend(PotionPrice) {
		return
	}
	g.Hero.Potions++
	g.logf(LogInfo, "%s", lang.T("log.buy_potion", 1, PotionPrice))
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
	g.refreshShop()
}

func (g *Game) EquipStash(i int) {
	if !g.InTown() || i < 0 || i >= len(g.Stash) {
		return
	}
	it := g.Stash[i]
	if !g.Hero.Class.CanUse(it) {
		g.logf(LogInfo, "%s", lang.T("log.cannot_use", it.Title()))
		return
	}
	if old := g.Hero.Equip[it.Slot]; old != nil {
		g.Stash[i] = old
	} else {
		g.Stash = slices.Delete(g.Stash, i, i+1)
	}
	g.Hero.Equip[it.Slot] = it
	g.Hero.HP = min(g.Hero.HP, g.Hero.MaxHP())
	g.logf(it.Rarity.LogKind(), "%s", lang.T("log.hero_equips", it.Title(), it.StatLabel()))
}

func (g *Game) Unequip(s Slot) {
	it := g.Hero.Equip[s]
	if !g.InTown() || it == nil {
		return
	}
	g.Hero.Equip[s] = nil
	g.Hero.HP = min(g.Hero.HP, g.Hero.MaxHP())
	g.Stash = append(g.Stash, it)
	g.logf(LogInfo, "%s", lang.T("log.unequip", it.Title()))
}

func (g *Game) SellStash(i int) {
	if !g.InTown() || i < 0 || i >= len(g.Stash) {
		return
	}
	it := g.Stash[i]
	g.Stash = slices.Delete(g.Stash, i, i+1)
	g.Gold += it.SellPrice()
	g.logf(LogInfo, "%s", lang.T("log.sell_item", it.Title(), it.SellPrice()))
}
