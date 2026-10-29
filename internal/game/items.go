package game

import (
	"fmt"
	"strings"

	"clidle/internal/lang"
)

type Slot int

const (
	Weapon Slot = iota
	Armor
	Boots
	Helmet
	Shield
	NumSlots
)

const (
	Blade = iota
	Staff
	Dagger
	Mace
)

const (
	Light = iota
	Medium
	Heavy
)

type Rarity int

const (
	Common Rarity = iota
	Rare
	Magic
	Epic
	Legendary
)

func (r Rarity) Name() string {
	return lang.T([...]string{"rarity.common", "rarity.rare", "rarity.magic", "rarity.epic", "rarity.legendary"}[r])
}

func (r Rarity) LogKind() LogKind {
	return [...]LogKind{LogLoot, LogRare, LogMagic, LogEpic, LogLegendary}[r]
}

type Item struct {
	ID      uint64
	Slot    Slot
	Kind    int
	Rarity  Rarity
	Name    string
	Base    int
	Upgrade int
}

func (it *Item) Bonus() int { return it.Base + it.Upgrade*max(1, it.Base/UpgradeDivisor) }

func (it *Item) Value() int { return it.Bonus() * ItemValuePerBonus * (int(it.Rarity) + 1) }

// Label translates the item name; saves from before translation hold the Portuguese name itself.
func (it *Item) Label() string {
	if strings.HasPrefix(it.Name, "item.") {
		return lang.T(it.Name)
	}
	for _, kinds := range ItemNames {
		for _, names := range kinds {
			for _, n := range names {
				if lang.In("pt", n.name) == it.Name {
					return lang.T(n.name)
				}
			}
		}
	}
	return it.Name
}

func (it *Item) Title() string {
	s := it.Label()
	if it.Upgrade > 0 {
		s = fmt.Sprintf("%s +%d", s, it.Upgrade)
	}
	return s
}

func (it *Item) StatLabel() string {
	return fmt.Sprintf("%s +%d", lang.T([...]string{"stat.atk", "stat.def", "stat.spd", "stat.def", "stat.def"}[it.Slot]), it.Bonus())
}

func (s Slot) Glyph() rune { return [...]rune{')', '[', ']', '^', '}'}[s] }

type FloorItem struct {
	Pos    Pos
	Gold   int
	Potion bool
	Item   *Item
}

func (f *FloorItem) Glyph() rune {
	switch {
	case f.Gold > 0:
		return '$'
	case f.Potion:
		return '!'
	}
	return f.Item.Slot.Glyph()
}

func newItem(rng *RNG, depth int) *Item {
	rarity := Common
	roll := rng.Float()
	for r := Legendary; r > Common; r-- {
		if roll < RarityChance[r] {
			rarity = r
			break
		}
		roll -= RarityChance[r]
	}
	return newItemOf(rng, depth, rarity)
}

func newItemOf(rng *RNG, depth int, rarity Rarity) *Item {
	slot := Slot(rng.Intn(int(NumSlots)))
	kind := rng.Intn(len(ItemNames[slot]))
	base := ItemBase(slot, kind, depth, rarity) * (0.8 + 0.4*rng.Float())
	names := ItemNames[slot][kind]
	name := names[0].name
	for _, n := range names {
		if n.depth <= depth {
			name = n.name
		}
	}
	return &Item{ID: rng.Uint64(), Slot: slot, Kind: kind, Rarity: rarity, Name: name, Base: max(1, int(base+0.5))}
}

func (l *Level) placeLoot(rng *RNG, startRoom int, nose bool) {
	room := func() int {
		r := rng.Intn(len(l.Rooms) - 1)
		if r >= startRoom {
			r++
		}
		return r
	}
	for range rng.Range(GoldPilesMin, GoldPilesMax) {
		gold := GoldPerPile(l.Depth) * (0.5 + rng.Float())
		if nose {
			gold *= NoseGoldMult
		}
		l.Items = append(l.Items, &FloorItem{Pos: l.RandomFloor(rng, room()), Gold: max(1, int(gold))})
	}
	if rng.Chance(PotionChance) {
		l.Items = append(l.Items, &FloorItem{Pos: l.RandomFloor(rng, room()), Potion: true})
	}
	items := rng.Range(ItemsMin, ItemsMax)
	if nose && rng.Chance(NoseItemChance) {
		items++
	}
	for range items {
		l.Items = append(l.Items, &FloorItem{Pos: l.RandomFloor(rng, room()), Item: newItem(rng, l.Depth)})
	}
}
