package game

import "math"

const (
	HeroBaseHP      = 30
	HeroHPPerLevel  = 8
	HeroBaseAtk     = 5
	HeroAtkPerLevel = 3 // per 2 levels
	HeroBaseDef     = 1
	HeroBaseSpd     = 4

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

var MonsterKinds = []MonsterKind{
	{'r', "rato", "180", 6, 3, 0, 4, 3, 1},
	{'b', "morcego", "137", 5, 3, 0, 7, 4, 1},
	{'k', "kobold", "172", 10, 4, 1, 4, 6, 2},
	{'g', "goblin", "70", 14, 6, 2, 4, 9, 4},
	{'w', "lobo", "247", 16, 7, 1, 6, 12, 6},
	{'z', "zumbi", "107", 28, 8, 3, 2, 15, 8},
	{'o', "orc", "28", 30, 11, 5, 4, 22, 11},
	{'e', "esqueleto", "255", 26, 13, 6, 4, 28, 14},
	{'l', "lobisomem", "130", 45, 17, 7, 6, 40, 18},
	{'d', "dragão jovem", "160", 70, 22, 10, 5, 60, 24},
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
	Weapon: {{1, "Adaga"}, {4, "Espada Curta"}, {8, "Maça"}, {12, "Espada Longa"}, {18, "Machado de Guerra"}, {25, "Montante"}},
	Armor:  {{1, "Gibão de Couro"}, {5, "Couro Batido"}, {10, "Cota de Malha"}, {16, "Brunea"}, {24, "Armadura de Placas"}},
	Boots:  {{1, "Botas de Couro"}, {6, "Botas Reforçadas"}, {14, "Botas Ligeiras"}, {22, "Botas Élficas"}},
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

var trainBase = [NumTraits]float64{30, 40, 40, 120}

func TrainCost(t Trait, level int) int {
	return int(trainBase[t] * math.Pow(1.5, float64(level)))
}
