package game

import (
	"errors"
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
	return fmt.Sprintf("+%d ", TrainGain(t)) + lang.T([...]string{"stat.str", "stat.dex", "stat.int"}[t])
}

func (g *Game) InTown() bool { return g.Exp == nil }

// Depart starts an expedition right away when the hero is resting in town and the mission table has room;
// the player's call skips the heroes waiting their turn.
func (g *Game) Depart() bool {
	if !g.InTown() || g.Guild.Running() >= g.Guild.Slots() {
		return false
	}
	g.StartExpedition()
	return true
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

func (gd *Guild) refreshShop() {
	lvl := gd.Buildings[Market]
	gd.Shop = make([]*Item, 0, ShopSize+MarketItemsAt(lvl))
	for range cap(gd.Shop) {
		gd.Shop = append(gd.Shop, newItemOf(gd.RNG, gd.ShopLevel(), rollRarity(gd.RNG, Epic, MarketRarity(lvl))))
	}
}

func (gd *Guild) RestockPrice() int { return RestockCost(gd.Buildings[Market]) }

// Restock replaces the shop's items for a price.
func (gd *Guild) Restock() bool {
	if gd.Gold < gd.RestockPrice() {
		return false
	}
	gd.Gold -= gd.RestockPrice()
	gd.refreshShop()
	return true
}

func ShopPrice(it *Item) int { return pctOf(it.Value(), 100*ShopMarkup) }

func (g *Game) spend(cost int) bool {
	if g.Guild.Gold < cost {
		g.logf(LogBad, "%s", lang.T("log.no_gold", cost))
		return false
	}
	g.Guild.Gold -= cost
	return true
}

var (
	ErrAway     = errors.New("away")
	ErrOtherUse = errors.New("other class")
)

// Refuse says why the hero cannot take the item now, or nil.
func (g *Game) Refuse(it *Item) error {
	switch {
	case !g.InTown():
		return ErrAway
	case !g.Hero.Class.CanUse(it):
		return ErrOtherUse
	}
	return nil
}

// BuyShop buys the guild shop's i-th item for g, who equips it at once.
func (g *Game) BuyShop(i int) {
	gd := g.Guild
	if i < 0 || i >= len(gd.Shop) || g.Refuse(gd.Shop[i]) != nil || !g.spend(ShopPrice(gd.Shop[i])) {
		return
	}
	it := gd.Shop[i]
	gd.Shop = slices.Delete(gd.Shop, i, i+1)
	g.equipBought(it)
}

func (g *Game) equipBought(it *Item) {
	h := g.Hero
	msg := lang.T("log.hero_equips", it.Title(), it.StatLabel())
	if old := h.Equip[it.Slot]; old != nil {
		g.Guild.Stash = append(g.Guild.Stash, old)
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
	return g.Guild.discounted(UpgradeCost(it, it.Upgrade), Smithy)
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

func (g *Game) BagCap() int { return g.Guild.BagCap() }

func (gd *Guild) BagCap() int { return BagSize + gd.BagLevel*BagSlotsPerLevel }

// NextBag is the bag size after the next upgrade and its cost; ok is false at the maximum.
func (gd *Guild) NextBag() (size, cost int, ok bool) {
	lvl := gd.BagLevel
	return BagSize + (lvl+1)*BagSlotsPerLevel, BagCost(lvl), lvl < BagMaxLevel
}

// UpgradeBag makes every hero's bag bigger.
func (gd *Guild) UpgradeBag() bool {
	_, cost, ok := gd.NextBag()
	if !ok || gd.Gold < cost {
		return false
	}
	gd.Gold -= cost
	gd.BagLevel++
	return true
}

func (g *Game) Train(t Trait) {
	if !g.InTown() || !g.spend(g.TrainPrice(t)) {
		return
	}
	g.Hero.Training[t]++
	g.logf(LogGood, "%s", lang.T("log.trained", t.Name(), g.Hero.Training[t], t.Effect()))
}

func (g *Game) BuyPotion() {
	price := g.PotionCost()
	if !g.InTown() || g.Hero.Potions >= MaxPotions || !g.spend(price) {
		return
	}
	g.Hero.Potions++
	g.logf(LogInfo, "%s", lang.T("log.buy_potion", 1, Num(price)))
}

func (g *Game) buyPotions() {
	h := g.Hero
	price := g.PotionCost()
	n := min(g.Config.PotionsToBring-h.Potions, g.Guild.Gold/price)
	if n <= 0 {
		return
	}
	g.Guild.Gold -= n * price
	h.Potions += n
	key := "log.buy_potions"
	if n == 1 {
		key = "log.buy_potion"
	}
	g.logf(LogInfo, "%s", lang.T(key, n, Num(n*price)))
}

func (g *Game) arriveInTown() {
	g.Hero.HP = g.Hero.MaxHP()
	if n := g.Hero.Potions - MaxPotions; n > 0 {
		g.Hero.Potions = MaxPotions
		g.Guild.Gold = addGold(g.Guild.Gold, n*PotionSellPrice)
		key := "log.sell_potions"
		if n == 1 {
			key = "log.sell_potion"
		}
		g.logf(LogInfo, "%s", lang.T(key, n, Num(n*PotionSellPrice)))
	}
}

func (g *Game) EquipStash(i int) {
	if !g.InTown() || i < 0 || i >= len(g.Guild.Stash) {
		return
	}
	it := g.Guild.Stash[i]
	if !g.Hero.Class.CanUse(it) {
		g.logf(LogInfo, "%s", lang.T("log.cannot_use", it.Title()))
		return
	}
	if old := g.Hero.Equip[it.Slot]; old != nil {
		g.Guild.Stash[i] = old
	} else {
		g.Guild.Stash = slices.Delete(g.Guild.Stash, i, i+1)
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
	g.Guild.Stash = append(g.Guild.Stash, it)
	g.logf(LogInfo, "%s", lang.T("log.unequip", it.Title()))
}

func (gd *Guild) SellStash(i int) {
	if i < 0 || i >= len(gd.Stash) {
		return
	}
	it := gd.Stash[i]
	gd.Stash = slices.Delete(gd.Stash, i, i+1)
	gd.Gold = addGold(gd.Gold, it.SellPrice())
}
