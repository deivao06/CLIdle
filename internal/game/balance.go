package game

import (
	"math"
	"time"
)

const (
	HeroBaseAtk = 5
	HeroBaseDef = 1
	HeroBaseSpd = 4

	MissChance    = 0.1
	CombatRoundMs = 1000
	MonsterWindow = 4

	TownWaitMs = 5000
	LogSize    = 200
)

// StepsPerSec converts the speed attribute into exploration pace.
func StepsPerSec(spd int) float64 { return 1 + float64(spd)*0.5 }

func XPToNext(level int) int { return int(20 * math.Pow(1.5, float64(level-1))) }

func MonsterScale(depthsAboveMin int) float64 { return math.Pow(1.1, float64(depthsAboveMin)) }

func MonstersPerLevel(depth int) int { return min(3+depth/2, 12) }

const (
	BossEvery   = 10
	BossHPMult  = 5
	BossAtkMult = 1.4
	BossDefMult = 1.3
	BossXPMult  = 8
	BossMinSpd  = 5
)

var BossKinds = []MonsterKind{
	{Glyph: 'G', Name: "boss.goblin_king", Color: "46"},
	{Glyph: 'O', Name: "boss.orc_warlord", Color: "34"},
	{Glyph: 'L', Name: "boss.lich", Color: "51"},
	{Glyph: 'D', Name: "boss.ancient_dragon", Color: "196"},
}

var MonsterKinds = []MonsterKind{
	{'r', "monster.rat", "180", 6, 3, 0, 4, 3, 1},
	{'b', "monster.bat", "137", 5, 3, 0, 7, 4, 1},
	{'k', "monster.kobold", "172", 10, 4, 1, 4, 6, 2},
	{'g', "monster.goblin", "70", 14, 6, 2, 4, 9, 4},
	{'w', "monster.wolf", "247", 16, 7, 1, 6, 12, 6},
	{'z', "monster.zombie", "107", 28, 8, 3, 2, 15, 8},
	{'o', "monster.orc", "28", 30, 11, 5, 4, 22, 11},
	{'e', "monster.skeleton", "255", 26, 13, 6, 4, 28, 14},
	{'l', "monster.werewolf", "130", 45, 17, 7, 6, 40, 18},
	{'d', "monster.young_dragon", "160", 70, 22, 10, 5, 60, 24},
}

const (
	BagSize           = 8
	StartPotions      = 2
	PotionHealPct     = 50
	PotionChance      = 0.5
	GoldPilesMin      = 2
	GoldPilesMax      = 4
	ItemsMin          = 1
	ItemsMax          = 2
	RareChance        = 0.2
	EpicChance        = 0.05
	UpgradeDivisor    = 10
	ItemValuePerBonus = 4
)

func GoldPerPile(depth int) float64 { return 5 * math.Pow(1.12, float64(depth-1)) }

func GoldPerKill(xp int) int { return max(1, xp/2) }

var rarityMult = [...]float64{1, 1.5, 2.2}

func ItemBase(slot Slot, depth int, r Rarity) float64 {
	switch slot {
	case Weapon:
		return (2 + float64(depth)*1.2) * rarityMult[r]
	case Armor:
		return (1 + float64(depth)*0.8) * rarityMult[r]
	}
	return 1 + float64(depth)/6 + float64(r)
}

type itemName struct {
	depth int
	name  string
}

var ItemNames = [NumSlots][]itemName{
	Weapon: {{1, "item.dagger"}, {4, "item.short_sword"}, {8, "item.mace"}, {12, "item.long_sword"}, {18, "item.war_axe"}, {25, "item.greatsword"}},
	Armor:  {{1, "item.leather_jerkin"}, {5, "item.studded_leather"}, {10, "item.chain_mail"}, {16, "item.brigandine"}, {24, "item.plate_armor"}},
	Boots:  {{1, "item.leather_boots"}, {6, "item.reinforced_boots"}, {14, "item.swift_boots"}, {22, "item.elven_boots"}},
}

const (
	ShopSize    = 3
	ShopMarkup  = 3
	PotionPrice = 10
	TrainHP     = 5
)

func UpgradeCost(it *Item, level int) int {
	return int(float64(it.Value()+10) * math.Pow(1.6, float64(level)))
}

var trainBase = [NumTraits]float64{Str: 40, Dex: 120, Int: 40, Con: 30}

type classHP struct{ Base, PerLevel int }

var ClassHP = [NumClasses]classHP{Warrior: {30, 8}, Mage: {20, 5}, Rogue: {25, 6}, Cleric: {28, 7}}

// ClassStats growth is per 2 levels.
type classStats struct{ Start, Growth [NumTraits]int }

var ClassStats = [NumClasses]classStats{
	Warrior: {Growth: [NumTraits]int{Str: 3}},
	Mage:    {Start: [NumTraits]int{Int: 2}, Growth: [NumTraits]int{Int: 3}},
	Rogue:   {Start: [NumTraits]int{Dex: 1}, Growth: [NumTraits]int{Dex: 2}},
	Cleric:  {Growth: [NumTraits]int{Str: 2, Int: 1}},
}

func TrainCost(t Trait, level int) int {
	return int(trainBase[t] * math.Pow(1.5, float64(level)))
}

const OfflineCap = 12 * time.Hour
