package game

import (
	"slices"

	"clidle/internal/lang"
)

// migrate brings an older save up to saveVersion, one version at a time.
func migrate(f *saveFile) {
	if f.Version < 2 {
		migrateV2(f.Game)
		migrateV2Kinds(f.Game)
	}
	if f.Version < 3 {
		migrateV3Rarity(f.Game)
	}
	f.Version = saveVersion
}

// migrateV2 maps the v1 training order (Vigor, Força, Resistência, Agilidade) onto the attributes.
func migrateV2(g *Game) {
	old := g.Hero.Training
	g.Hero.Training = [NumTraits]int{Str: old[1], Dex: old[3], Con: old[0] + old[2]}
}

func savedItems(g *Game) []*Item {
	items := append(append(append(g.Hero.Equip[:], g.Shop...), g.Pending...), g.Stash...)
	if e := g.Exp; e != nil {
		items = append(append(items, e.StartEquip[:]...), e.Bag...)
		for _, f := range e.Level.Items {
			items = append(items, f.Item)
		}
	}
	return slices.DeleteFunc(items, func(it *Item) bool { return it == nil })
}

// migrateV2Kinds deduces Item.Kind, which v1 did not have, from the item name.
func migrateV2Kinds(g *Game) {
	for _, it := range savedItems(g) {
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
func migrateV3Rarity(g *Game) {
	for _, it := range savedItems(g) {
		if it.Rarity == 2 {
			it.Rarity = Epic
		}
	}
}
