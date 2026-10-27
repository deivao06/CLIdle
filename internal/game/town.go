package game

import "fmt"

type Trait int

const (
	Vigor Trait = iota
	Strength
	Toughness
	Agility
	NumTraits
)

func (t Trait) Name() string {
	return [...]string{"Vigor", "Força", "Resistência", "Agilidade"}[t]
}

func (t Trait) Effect() string {
	return [...]string{fmt.Sprintf("+%d HP", TrainHP), "+1 Atq", "+1 Def", "+1 Vel"}[t]
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
		g.logf(LogInfo, "Auto-expedição ligada.")
	} else {
		g.logf(LogInfo, "Auto-expedição desligada: o herói fica na cidade ao voltar.")
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
		g.logf(LogBad, "Ouro insuficiente (precisa de %d).", cost)
		return false
	}
	g.Gold -= cost
	return true
}

func (g *Game) BuyShop(i int) {
	if i < 0 || i >= len(g.Shop) || !g.spend(ShopPrice(g.Shop[i])) {
		return
	}
	it := g.Shop[i]
	g.Shop = append(g.Shop[:i], g.Shop[i+1:]...)
	if g.InTown() {
		g.equipBought(it)
		return
	}
	g.Pending = append(g.Pending, it)
	g.logf(LogInfo, "Você compra %s. Ele recebe quando voltar.", it.Title())
}

func (g *Game) equipBought(it *Item) {
	h := g.Hero
	msg := fmt.Sprintf("O herói equipa %s (%s).", it.Title(), it.StatLabel())
	if old := h.Equip[it.Slot]; old != nil {
		g.Gold += old.Value()
		msg += fmt.Sprintf(" %s vendido por %d.", old.Title(), old.Value())
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
		g.logf(LogGood, "O ferreiro melhora %s (%s).", it.Title(), it.StatLabel())
		return
	}
	g.Upgrades[s]++
	g.logf(LogInfo, "Melhoria de %s na fila: o ferreiro trabalha quando o herói voltar.", it.Name)
}

func (g *Game) Train(t Trait) {
	if !g.spend(TrainCost(t, g.Hero.Training[t])) {
		return
	}
	g.Hero.Training[t]++
	g.logf(LogGood, "Treino de %s: nível %d (%s).", t.Name(), g.Hero.Training[t], t.Effect())
}

func (g *Game) buyPotions() {
	h := g.Hero
	n := min(g.Config.PotionsToBring-h.Potions, g.Gold/PotionPrice)
	if n <= 0 {
		return
	}
	g.Gold -= n * PotionPrice
	h.Potions += n
	noun := "poções"
	if n == 1 {
		noun = "poção"
	}
	g.logf(LogInfo, "Compra %d %s (%d ouro).", n, noun, n*PotionPrice)
}

func (g *Game) arriveInTown() {
	g.Hero.HP = g.Hero.MaxHP()
	for _, it := range g.Pending {
		g.equipBought(it)
	}
	g.Pending = nil
	for s, n := range g.Upgrades {
		if it := g.Hero.Equip[s]; it != nil && n > 0 {
			it.Upgrade += n
			g.logf(LogGood, "O ferreiro melhora %s (%s).", it.Title(), it.StatLabel())
		}
	}
	g.Upgrades = [NumSlots]int{}
	g.refreshShop()
}
