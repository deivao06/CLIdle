package game

import (
	"slices"

	"clidle/internal/lang"
)

// queueV3 holds the purchases v3 queued while the hero was away; v4 dropped the queue.
type queueV3 struct {
	Pending   []*Item
	Upgrades  [NumSlots]int
	BagQueued int
}

// legacy holds the fields older saves had that the current structs dropped.
type legacy struct {
	queueV3
	Config configV5
}

// migrate brings an older save up to saveVersion, one version at a time.
func migrate(f *saveFile, old *legacy) {
	q := &old.queueV3
	if f.Version < 2 {
		migrateV2(f.Game)
		migrateV2Kinds(f.Game, q)
	}
	if f.Version < 3 {
		migrateV3Rarity(f.Game, q)
	}
	if f.Version < 4 {
		migrateV4Refund(f.Game, q)
	}
	if f.Version < 5 {
		f.Game.Config.KeepLoot = f.Game.Config.ManualEquip
	}
	if f.Version < 6 {
		f.Game.Config.Gambits = classGambits(gambitsV5(old.Config), f.Game.Hero.Class)
	}
	if f.Version < 7 {
		f.Game.Config.Gambits = validGambits(f.Game.Config.Gambits)
		f.Game.TacticsChanged()
	}
	if f.Version < 8 {
		migrateV8Forged(f.Game, q)
	}
	if f.Version < 9 {
		migrateV9Gold(f.Game)
	}
	f.Version = saveVersion
}

// migrateV2 maps the v1 training order (Vigor, Força, Resistência, Agilidade) onto the attributes.
func migrateV2(g *Game) {
	old := g.Hero.Training
	g.Hero.Training = [NumTraits]int{Str: old[1], Dex: old[3], Con: old[0] + old[2]}
}

func savedItems(g *Game, q *queueV3) []*Item {
	items := append(append(append(g.Hero.Equip[:], g.Shop...), q.Pending...), g.Stash...)
	if e := g.Exp; e != nil {
		items = append(append(items, e.StartEquip[:]...), e.Bag...)
		for _, f := range e.Level.Items {
			items = append(items, f.Item)
		}
	}
	return slices.DeleteFunc(items, func(it *Item) bool { return it == nil })
}

// migrateV2Kinds deduces Item.Kind, which v1 did not have, from the item name.
func migrateV2Kinds(g *Game, q *queueV3) {
	for _, it := range savedItems(g, q) {
		for kind, names := range ItemNames[it.Slot] {
			for _, n := range names {
				if it.Name == n.name || it.Name == lang.In("pt", n.name) {
					it.Kind = kind
				}
			}
		}
	}
}

// migrateV3Rarity moves v2's epic (2) past the new magic rarity.
func migrateV3Rarity(g *Game, q *queueV3) {
	for _, it := range savedItems(g, q) {
		if it.Rarity == 2 {
			it.Rarity = Epic
		}
	}
}

// migrateV4Refund gives back the gold of everything still queued, since purchases now only happen in town.
func migrateV4Refund(g *Game, q *queueV3) {
	for _, it := range q.Pending {
		g.Gold += ShopPrice(it)
	}
	for s, n := range q.Upgrades {
		it := g.Hero.Equip[s]
		for _, p := range q.Pending {
			if p.Slot == Slot(s) {
				it = p
			}
		}
		for i := range n {
			if it != nil {
				g.Gold += UpgradeCost(it, it.Upgrade+i)
			}
		}
	}
	for i := range q.BagQueued {
		g.Gold += BagCost(g.BagLevel + i)
	}
}

// migrateV8Forged keeps the blacksmith bonus v7 computed from the upgrade count, which v8 stores on the item.
func migrateV8Forged(g *Game, q *queueV3) {
	for _, it := range savedItems(g, q) {
		it.Forged = it.Upgrade * max(1, it.Base/10)
	}
}

// migrateV9Gold gives back the gold that wrapped below zero before v9 capped it at MaxGold.
func migrateV9Gold(g *Game) {
	if g.Gold < 0 {
		g.Gold = MaxGold
	}
	if g.Exp != nil && g.Exp.Gold < 0 {
		g.Exp.Gold = MaxGold
	}
}
