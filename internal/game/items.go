package game

import "fmt"

type Slot int

const (
	Weapon Slot = iota
	Armor
	Boots
	NumSlots
)

type Rarity int

const (
	Common Rarity = iota
	Rare
	Epic
)

func (r Rarity) Name() string {
	return [...]string{"comum", "raro", "épico"}[r]
}

func (r Rarity) LogKind() LogKind {
	return [...]LogKind{LogLoot, LogRare, LogEpic}[r]
}

type Item struct {
	Slot    Slot
	Rarity  Rarity
	Name    string
	Base    int
	Upgrade int
}

func (it *Item) Bonus() int { return it.Base + it.Upgrade*max(1, it.Base/UpgradeDivisor) }

func (it *Item) Value() int { return it.Bonus() * ItemValuePerBonus * (int(it.Rarity) + 1) }

func (it *Item) Title() string {
	s := it.Name
	if it.Upgrade > 0 {
		s = fmt.Sprintf("%s +%d", s, it.Upgrade)
	}
	return s
}

func (it *Item) StatLabel() string {
	return fmt.Sprintf("%s +%d", [...]string{"Atq", "Def", "Vel"}[it.Slot], it.Bonus())
}

func (s Slot) Glyph() rune { return [...]rune{')', '[', ']'}[s] }

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
	slot := Slot(rng.Intn(int(NumSlots)))
	rarity := Common
	switch roll := rng.Float(); {
	case roll < EpicChance:
		rarity = Epic
	case roll < EpicChance+RareChance:
		rarity = Rare
	}
	base := ItemBase(slot, depth, rarity) * (0.8 + 0.4*rng.Float())
	names := ItemNames[slot]
	name := names[0].name
	for _, n := range names {
		if n.depth <= depth {
			name = n.name
		}
	}
	return &Item{Slot: slot, Rarity: rarity, Name: name, Base: max(1, int(base+0.5))}
}

func (l *Level) placeLoot(rng *RNG, startRoom int) {
	room := func() int {
		r := rng.Intn(len(l.Rooms) - 1)
		if r >= startRoom {
			r++
		}
		return r
	}
	for range rng.Range(GoldPilesMin, GoldPilesMax) {
		gold := max(1, int(GoldPerPile(l.Depth)*(0.5+rng.Float())))
		l.Items = append(l.Items, &FloorItem{Pos: l.RandomFloor(rng, room()), Gold: gold})
	}
	if rng.Chance(PotionChance) {
		l.Items = append(l.Items, &FloorItem{Pos: l.RandomFloor(rng, room()), Potion: true})
	}
	for range rng.Range(ItemsMin, ItemsMax) {
		l.Items = append(l.Items, &FloorItem{Pos: l.RandomFloor(rng, room()), Item: newItem(rng, l.Depth)})
	}
}
