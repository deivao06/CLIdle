package game

import (
	"math"
	"time"
)

const (
	HeroBaseAtk = 5
	HeroBaseDef = 1
	HeroBaseSpd = 4
	SpdLevelDiv = 3
	SpdDexDiv   = 5

	MissEven       = 0.20
	CritTop        = 0.30
	DexSoften      = 10
	CombatRoundMs  = 1000
	MonsterWindow  = 4
	MonsterAtkMult = 1.3

	TownWaitMs = 5000
	LogSize    = 200
)

// StepsPerSec converts the speed attribute into exploration pace.
func StepsPerSec(spd int) float64 { return 1 + float64(spd)*0.5 }

func XPToNext(level int) int { return int(20 * math.Pow(1.5, float64(level-1))) }

const MonsterGrowthLin = 0.45

// MonsterScale grows a monster linearly with the floors below its first floor.
func MonsterScale(depth, minDepth int) float64 {
	return 1 + MonsterGrowthLin*float64(max(0, depth-minDepth))
}

// MissChance and CritChance compare the attacker's DEX with the defender's, the same for heroes and monsters.
func MissChance(atk, def int) float64 {
	a, d := float64(atk+DexSoften), float64(def+DexSoften)
	return 2 * MissEven * d / (a + d)
}

func CritChance(atk, def int) float64 {
	a, d := float64(atk+DexSoften), float64(def+DexSoften)
	return CritTop * max(0, a-d) / (a + d)
}

// MonsterMissChance is the chance that a monster's attack misses the hero, its own miss or the hero's dodge.
func MonsterMissChance(dex int, h *Hero) float64 {
	m := MissChance(dex, h.Attr(Dex))
	return m + (1-m)*h.DodgeChance()
}

// MonsterDexEvery is how many floors below its first floor a monster needs to gain 1 dexterity.
const MonsterDexEvery = 2

func MonstersPerLevel(depth int) int { return min(max(5, 3+depth/2), 12) }

const (
	BossEvery      = 10
	BossHPMult     = 5
	BossHPPerBoss  = 1
	BossAtkMult    = 1.4
	BossAtkPerBoss = 0.1
	BossDefMult    = 1.3
	BossXPMult     = 8
	BossMinSpd     = 5
	BossDexBonus   = 5

	BossWakeDist = 1

	BossEpicChance      = 0.47
	BossLegendaryChance = 0.03

	DeathKeepGoldPct = 25
	BigHitDiv        = 4
	DeathLogLines    = 6
)

var BossKinds = []MonsterKind{
	{Glyph: 'G', Name: "boss.goblin_king", Color: "46"},
	{Glyph: 'O', Name: "boss.orc_warlord", Color: "34"},
	{Glyph: 'L', Name: "boss.lich", Color: "51", Undead: true},
	{Glyph: 'D', Name: "boss.ancient_dragon", Color: "196"},
}

var MonsterKinds = []MonsterKind{
	{'r', "monster.rat", "180", 6, 3, 0, 4, 3, 1, false, 3},
	{'b', "monster.bat", "137", 5, 3, 0, 7, 4, 1, false, 8},
	{'k', "monster.kobold", "166", 10, 4, 1, 4, 6, 1, false, 2},
	{'g', "monster.goblin", "70", 14, 6, 2, 4, 9, 4, false, 4},
	{'w', "monster.wolf", "247", 16, 7, 1, 6, 12, 6, false, 6},
	{'z', "monster.zombie", "107", 28, 8, 3, 2, 15, 8, true, 0},
	{'o', "monster.orc", "28", 30, 11, 5, 4, 22, 11, false, 2},
	{'e', "monster.skeleton", "189", 26, 13, 6, 4, 28, 14, true, 4},
	{'l', "monster.werewolf", "130", 45, 17, 7, 6, 40, 18, false, 8},
	{'d', "monster.young_dragon", "160", 70, 22, 10, 5, 60, 24, false, 6},
}

const (
	BagSize           = 8
	StartPotions      = 5
	PotionHealPct     = 35
	PotionChance      = 0.2
	GoldPilesMin      = 1
	GoldPilesMax      = 3
	ItemsMin          = 0
	ItemsMax          = 2
	UpgradeDivisor    = 10
	UpgradeMax        = 10
	ItemValuePerBonus = 4
	ItemValuePiles    = 1
	EffectValuePiles  = 1
	UpgradeRefundPct  = 20
)

// GoldPerPile grows 8% a floor up to the first dungeon's last floor, then by the same amount each floor.
func GoldPerPile(depth int) float64 {
	top := 5 * math.Pow(1.08, float64(min(depth, DungeonFloors)-1))
	return top * (1 + 0.08*float64(max(0, depth-DungeonFloors)))
}

const (
	DungeonFloors    = 100
	DungeonStep      = 75
	DungeonAttrBonus = 5
	DungeonGoldPiles = 10
)

var (
	dungeonNouns = []string{"dungeon.noun.cave", "dungeon.noun.crypt", "dungeon.noun.dungeon", "dungeon.noun.catacombs", "dungeon.noun.fortress",
		"dungeon.noun.grotto", "dungeon.noun.tomb", "dungeon.noun.citadel", "dungeon.noun.tower", "dungeon.noun.mine"}
	dungeonAdjs = []string{"dungeon.adj.infernal", "dungeon.adj.forgotten", "dungeon.adj.shadowed", "dungeon.adj.cursed", "dungeon.adj.deep",
		"dungeon.adj.burning", "dungeon.adj.frozen", "dungeon.adj.bloody", "dungeon.adj.silent", "dungeon.adj.lost", "dungeon.adj.venomous", "dungeon.adj.ancient"}
)

const GoldKillDiv = 8

func GoldPerKill(xp int) int { return max(1, xp/GoldKillDiv) }

var EffectCount = [...]int{Rare: 1, Epic: 2, Legendary: 3}

var EffectCap = [NumEffects]int{
	LifeSteal:   30,
	Crit:        75,
	Vigor:       50,
	GoldFind:    100,
	PotionBoost: 60,
	XPBoost:     50,
}

var EffectRange = [NumEffects][2]int{
	LifeSteal:   {5, 15},
	Crit:        {5, 15},
	Vigor:       {5, 15},
	GoldFind:    {10, 30},
	PotionBoost: {10, 30},
	XPBoost:     {5, 15},
}

var RarityChance = [...]float64{Magic: 0.2, Rare: 0.08, Epic: 0.01, Legendary: 0.002}

var rarityMult = [...]float64{1, 1.5, 1.85, 2.3, 2.9}

var armorWeightMult = [...]float64{Light: 0.8, Medium: 1, Heavy: 1.25}

func ItemBase(slot Slot, kind, depth int, r Rarity) float64 {
	switch slot {
	case Weapon:
		return (2 + float64(depth)*1.2) * rarityMult[r]
	case Armor:
		return (1 + float64(depth)*0.8) * rarityMult[r] * armorWeightMult[kind]
	case Helmet, Shield:
		return (1 + float64(depth)*0.8) * rarityMult[r] / 2
	}
	return 1 + float64(depth)/6 + float64(r)
}

// ItemDepth is the floor where an item's base is the expected roll for its slot, kind and rarity.
func ItemDepth(slot Slot, kind int, r Rarity, base int) int {
	b := float64(base)
	var d float64
	switch slot {
	case Weapon:
		d = (b/rarityMult[r] - 2) / 1.2
	case Armor:
		d = (b/(rarityMult[r]*armorWeightMult[kind]) - 1) / 0.8
	case Helmet, Shield:
		d = (2*b/rarityMult[r] - 1) / 0.8
	default:
		d = (b - 1 - float64(r)) * 6
	}
	return max(1, int(d+0.5))
}

const ItemSpread = 0.1

// ItemRoll keeps a higher rarity at least as strong as a lower one from the same floor.
func ItemRoll(slot Slot, kind, depth int, r Rarity, roll float64) int {
	base := ItemBase(slot, kind, depth, r)
	if slot != Boots {
		base *= 1 - ItemSpread + 2*ItemSpread*roll
	}
	return max(1, int(base+0.5))
}

type classGear struct {
	Weapon []int
	Armor  []int
	Shield bool
}

var ClassGear = [NumClasses]classGear{
	Warrior: {[]int{Blade, Mace}, []int{Light, Medium, Heavy}, true},
	Mage:    {[]int{Staff}, []int{Light}, false},
	Rogue:   {[]int{Dagger}, []int{Light, Medium}, false},
	Cleric:  {[]int{Mace, Blade}, []int{Medium, Heavy}, true},
}

type abilityInfo struct {
	Class Class
	Level int
	Mana  int
	Key   string
}

var AbilityInfo = [NumAbilities]abilityInfo{
	PowerStrike:  {Warrior, 1, 0, "ability.power_strike"},
	SecondWind:   {Warrior, 12, 0, "ability.second_wind"},
	Whirlwind:    {Warrior, 25, 0, "ability.whirlwind"},
	SneakAttack:  {Rogue, 1, 0, "ability.sneak_attack"},
	Dodge:        {Rogue, 12, 0, "ability.dodge"},
	TreasureNose: {Rogue, 25, 0, "ability.treasure_nose"},
	MagicMissile: {Mage, 1, 2, "spell.magic_missile"},
	Sleep:        {Mage, 8, 3, "spell.sleep"},
	Fireball:     {Mage, 15, 5, "spell.fireball"},
	Blink:        {Mage, 22, 4, "spell.blink"},
	FrostRay:     {Mage, 30, 4, "spell.frost_ray"},
	Heal:         {Cleric, 1, 4, "spell.heal"},
	Bless:        {Cleric, 12, 3, "spell.bless"},
	TurnUndead:   {Cleric, 25, 3, "spell.turn_undead"},
}

const (
	PowerStrikeEvery  = 4
	SecondWindPct     = 30
	SecondWindHealPct = 40

	SneakMult      = 3
	DodgePerDex    = 0.005
	DodgeCap       = 0.15
	NoseGoldMult   = 1.5
	NoseItemChance = 0.5

	SpellRange      = 5
	SleepMinHPPct   = 50
	SleepRounds     = 6
	SleepBossRounds = 2
	FrostSlowRounds = 5
	BlinkHPPct      = 35

	HealPct        = 35
	BlessRounds    = 8
	BlessDef       = 3
	TurnUndeadMult = 3
)

var SpellPower = [NumAbilities]int{MagicMissile: 4, Fireball: 4, FrostRay: 6}

type itemName struct {
	depth int
	name  string
}

var ItemNames = [NumSlots][][]itemName{
	Weapon: {
		Blade:  {{1, "item.short_sword"}, {8, "item.long_sword"}, {16, "item.war_axe"}, {25, "item.greatsword"}},
		Staff:  {{1, "item.quarterstaff"}, {8, "item.oak_staff"}, {16, "item.runed_staff"}, {25, "item.archmage_staff"}},
		Dagger: {{1, "item.dagger"}, {8, "item.stiletto"}, {16, "item.kris"}, {25, "item.shadow_blade"}},
		Mace:   {{1, "item.mace"}, {8, "item.flail"}, {16, "item.morning_star"}, {25, "item.warhammer"}},
	},
	Armor: {
		Light:  {{1, "item.leather_jerkin"}, {8, "item.studded_leather"}, {18, "item.shadow_leather"}},
		Medium: {{1, "item.scale_mail"}, {8, "item.chain_mail"}, {18, "item.brigandine"}},
		Heavy:  {{1, "item.banded_mail"}, {10, "item.splint_mail"}, {20, "item.plate_armor"}},
	},
	Boots:  {{{1, "item.leather_boots"}, {6, "item.reinforced_boots"}, {14, "item.swift_boots"}, {22, "item.elven_boots"}}},
	Helmet: {{{1, "item.leather_cap"}, {8, "item.iron_helm"}, {18, "item.great_helm"}}},
	Shield: {{{1, "item.buckler"}, {8, "item.kite_shield"}, {18, "item.tower_shield"}}},
}

const (
	ShopSize        = 3
	ShopMarkup      = 5
	PotionPrice     = 10
	PotionSellPrice = 5
	MaxPotions      = 10
	TrainHP         = 5
	ArmorHPPct      = 50

	BagSlotsPerLevel = 2
	BagMaxLevel      = 8
)

func UpgradeCost(it *Item, level int) int {
	value := (it.ForgeBase() + level*it.UpgradeStep()) * ItemValuePerBonus * (int(it.Rarity) + 1)
	return int(float64(value+10) * math.Pow(1.6, float64(level)))
}

const TrainBase = 40

type classHP struct{ Base, PerLevel int }

var ClassMana = [NumClasses]int{Mage: 10, Cleric: 8}

// ClassManaPerInt and ClassManaRegenPerInt are the max mana and the mana regen each INT point gives.
var (
	ClassManaPerInt      = [NumClasses]int{Mage: 3, Cleric: 2}
	ClassManaRegenPerInt = [NumClasses]int{Mage: 6, Cleric: 4}
)

const (
	ManaRegenPerStep = 25
	ManaRegenDiv     = 100
)

var ClassHP = [NumClasses]classHP{Warrior: {30, 8}, Mage: {26, 5}, Rogue: {25, 6}, Cleric: {28, 7}}

// ClassStats growth is per 2 levels.
type classStats struct{ Start, Growth [NumTraits]int }

var ClassStats = [NumClasses]classStats{
	Warrior: {Growth: [NumTraits]int{Str: 3}},
	Mage:    {Start: [NumTraits]int{Int: 2}, Growth: [NumTraits]int{Int: 4}},
	Rogue:   {Start: [NumTraits]int{Dex: 1}, Growth: [NumTraits]int{Dex: 1}},
	Cleric:  {Growth: [NumTraits]int{Str: 2, Int: 1}},
}

func BagCost(level int) int {
	return int(100 * math.Pow(1.8, float64(level)))
}

func TrainCost(level int) int {
	return int(TrainBase * math.Pow(1.5, float64(level)))
}

const OfflineCap = 12 * time.Hour

const MaxGold = math.MaxInt64
