package game

import (
	"fmt"
	"slices"
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
	Magic
	Rare
	Epic
	Legendary
	NumRarities
)

func (r Rarity) Name() string {
	return lang.T([...]string{"rarity.common", "rarity.magic", "rarity.rare", "rarity.epic", "rarity.legendary"}[r])
}

func (r Rarity) LogKind() LogKind {
	return [...]LogKind{LogLoot, LogMagic, LogRare, LogEpic, LogLegendary}[r]
}

type Effect int

const (
	LifeSteal Effect = iota
	Crit
	Vigor
	GoldFind
	PotionBoost
	XPBoost
	NumEffects
)

type ItemEffect struct {
	Kind  Effect
	Value int
}

func (f ItemEffect) Label() string {
	return lang.T([...]string{"effect.life_steal", "effect.crit", "effect.vigor", "effect.gold_find", "effect.potion_boost", "effect.xp_boost"}[f.Kind], f.Value)
}

type Item struct {
	ID      uint64
	Slot    Slot
	Kind    int
	Rarity  Rarity
	Name    string
	Base    int
	Upgrade int
	Forged  int `json:",omitempty"`
	Floor   int `json:",omitempty"`
	Effects []ItemEffect
}

func (it *Item) Bonus() int { return it.Base + it.Forged }

// ForgeBase is the base the item would have as a common, which the blacksmith works from.
func (it *Item) ForgeBase() int {
	if it.Slot == Boots {
		return it.Base
	}
	return max(1, int(float64(it.Base)/rarityMult[it.Rarity]+0.5))
}

func (it *Item) UpgradeStep() int { return max(1, it.ForgeBase()/UpgradeDivisor) }

func (it *Item) Value() int {
	piles := float64(ItemValuePiles * (int(it.Rarity) + 1))
	for _, e := range it.Effects {
		piles += EffectValuePiles * float64(e.Value) / float64(EffectRange[e.Kind][1])
	}
	return max(1, int(GoldPerPile(ItemDepth(it.Slot, it.Kind, it.Rarity, it.Base))*piles+0.5))
}

func (it *Item) SellPrice() int {
	spent := 0
	for lvl := range it.Upgrade {
		at := *it
		at.Upgrade = lvl
		spent += UpgradeCost(&at, lvl)
	}
	return max(1, it.Value()+spent*UpgradeRefundPct/100)
}

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
	return s + it.ILvl()
}

func (it *Item) ILvl() string {
	if it.Floor == 0 {
		return ""
	}
	return lang.T("item.ilvl", it.Floor)
}

func (it *Item) StatLabel() string {
	s := fmt.Sprintf("%s +%d", lang.T([...]string{"stat.atk", "stat.def", "stat.spd", "stat.def", "stat.def"}[it.Slot]), it.Bonus())
	if it.Upgrade > 0 {
		s += lang.T("stat.base", it.Base)
	}
	for _, f := range it.Effects {
		s += ", " + f.Label()
	}
	return s
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

func newItem(rng *RNG, depth int) *Item { return newItemOf(rng, depth, rollRarity(rng, Legendary)) }

func rollRarity(rng *RNG, top Rarity) Rarity {
	roll := rng.Float()
	for r := top; r > Common; r-- {
		if roll < RarityChance[r] {
			return r
		}
		roll -= RarityChance[r]
	}
	return Common
}

func newItemOf(rng *RNG, depth int, rarity Rarity) *Item {
	slot := Slot(rng.Intn(int(NumSlots)))
	kind := rng.Intn(len(ItemNames[slot]))
	base := ItemRoll(slot, kind, depth, rarity, rng.Float())
	names := ItemNames[slot][kind]
	name := names[0].name
	for _, n := range names {
		if n.depth <= depth {
			name = n.name
		}
	}
	it := &Item{ID: rng.Uint64(), Slot: slot, Kind: kind, Rarity: rarity, Name: name, Base: base, Floor: depth}
	for len(it.Effects) < EffectCount[rarity] {
		k := Effect(rng.Intn(int(NumEffects)))
		if !slices.ContainsFunc(it.Effects, func(f ItemEffect) bool { return f.Kind == k }) {
			it.Effects = append(it.Effects, ItemEffect{k, rng.Range(EffectRange[k][0], EffectRange[k][1])})
		}
	}
	return it
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
