package game

import (
	"math"
	"time"
)

const (
	HeroBaseAtk   = 5
	HeroBaseDef   = 1
	HeroBaseSpd   = 4
	ExploreDexDiv = 2

	MissChance     = 0.1
	CombatRoundMs  = 1000
	MonsterWindow  = 4
	MonsterAtkMult = 1.3

	TownWaitMs = 5000
	LogSize    = 200
)

// StepsPerSec converts the speed attribute into exploration pace.
func StepsPerSec(spd int) float64 { return 1 + float64(spd)*0.5 }

func XPToNext(level int) int { return int(20 * math.Pow(1.5, float64(level-1))) }

func MonsterScale(depthsAboveMin int) float64 { return math.Pow(1.15, float64(depthsAboveMin)) }

func MonstersPerLevel(depth int) int { return min(max(5, 3+depth/2), 12) }

const (
	BossEvery   = 10
	BossHPMult  = 5
	BossAtkMult = 1.4
	BossDefMult = 1.3
	BossXPMult  = 8
	BossMinSpd  = 5

	BossWakeDist = 1

	DeathKeepGoldPct = 50
	DeathLogLines    = 6
)

var BossKinds = []MonsterKind{
	{Glyph: 'G', Name: "boss.goblin_king", Color: "46"},
	{Glyph: 'O', Name: "boss.orc_warlord", Color: "34"},
	{Glyph: 'L', Name: "boss.lich", Color: "51", Undead: true},
	{Glyph: 'D', Name: "boss.ancient_dragon", Color: "196"},
}

var MonsterKinds = []MonsterKind{
	{'r', "monster.rat", "180", 6, 3, 0, 4, 3, 1, false},
	{'b', "monster.bat", "137", 5, 3, 0, 7, 4, 1, false},
	{'k', "monster.kobold", "172", 10, 4, 1, 4, 6, 1, false},
	{'g', "monster.goblin", "70", 14, 6, 2, 4, 9, 4, false},
	{'w', "monster.wolf", "247", 16, 7, 1, 6, 12, 6, false},
	{'z', "monster.zombie", "107", 28, 8, 3, 2, 15, 8, true},
	{'o', "monster.orc", "28", 30, 11, 5, 4, 22, 11, false},
	{'e', "monster.skeleton", "255", 26, 13, 6, 4, 28, 14, true},
	{'l', "monster.werewolf", "130", 45, 17, 7, 6, 40, 18, false},
	{'d', "monster.young_dragon", "160", 70, 22, 10, 5, 60, 24, false},
}

const (
	BagSize           = 8
	StashSize         = 20
	StartPotions      = 2
	PotionHealPct     = 35
	PotionChance      = 0.2
	GoldPilesMin      = 2
	GoldPilesMax      = 4
	ItemsMin          = 0
	ItemsMax          = 2
	UpgradeDivisor    = 10
	ItemValuePerBonus = 4
)

func GoldPerPile(depth int) float64 { return 5 * math.Pow(1.12, float64(depth-1)) }

func GoldPerKill(xp int) int { return max(1, xp/2) }

var EffectCount = [...]int{Epic: 1, Legendary: 2}

var EffectRange = [NumEffects][2]int{
	LifeSteal:   {5, 15},
	Crit:        {5, 15},
	Vigor:       {5, 15},
	GoldFind:    {10, 30},
	PotionBoost: {10, 30},
	XPBoost:     {5, 15},
}

var RarityChance = [...]float64{Magic: 0.2, Rare: 0.08, Epic: 0.03, Legendary: 0.01}

var rarityMult = [...]float64{1, 1.5, 1.85, 2.2, 2.8}

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

type classGear struct {
	Weapon int
	Armor  []int
	Shield bool
}

var ClassGear = [NumClasses]classGear{
	Warrior: {Blade, []int{Light, Medium, Heavy}, true},
	Mage:    {Staff, []int{Light}, false},
	Rogue:   {Dagger, []int{Light, Medium}, false},
	Cleric:  {Mace, []int{Medium, Heavy}, true},
}

type abilityInfo struct {
	Class Class
	Level int
	Mana  int
	Key   string
}

var AbilityInfo = [NumAbilities]abilityInfo{
	PowerStrike:  {Warrior, 1, 0, "ability.power_strike"},
	SecondWind:   {Warrior, 5, 0, "ability.second_wind"},
	Whirlwind:    {Warrior, 10, 0, "ability.whirlwind"},
	SneakAttack:  {Rogue, 1, 0, "ability.sneak_attack"},
	Dodge:        {Rogue, 5, 0, "ability.dodge"},
	TreasureNose: {Rogue, 8, 0, "ability.treasure_nose"},
	MagicMissile: {Mage, 1, 2, "spell.magic_missile"},
	Sleep:        {Mage, 4, 3, "spell.sleep"},
	Fireball:     {Mage, 7, 5, "spell.fireball"},
	Blink:        {Mage, 10, 4, "spell.blink"},
	FrostRay:     {Mage, 14, 4, "spell.frost_ray"},
	Heal:         {Cleric, 1, 4, "spell.heal"},
	Bless:        {Cleric, 4, 3, "spell.bless"},
	TurnUndead:   {Cleric, 8, 3, "spell.turn_undead"},
}

const (
	PowerStrikeEvery  = 4
	SecondWindPct     = 30
	SecondWindHealPct = 40

	SneakMult      = 3
	DodgePerDex    = 0.03
	DodgeCap       = 0.35
	NoseGoldMult   = 1.5
	NoseItemChance = 0.5

	SpellRange      = 5
	SleepMinHPPct   = 50
	SleepRounds     = 6
	SleepBossRounds = 2
	FrostSlowRounds = 5
	BlinkHPPct      = 35

	HealPct         = 35
	BlessRounds     = 8
	BlessMissChance = 0.03
	BlessDef        = 3
	TurnUndeadMult  = 3
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
	ShopMarkup      = 3
	PotionPrice     = 10
	PotionSellPrice = 5
	MaxPotions      = 10
	TrainHP         = 5

	BagSlotsPerLevel = 2
	BagMaxLevel      = 8
)

func UpgradeCost(it *Item, level int) int {
	return int(float64(it.Value()+10) * math.Pow(1.6, float64(level)))
}

var trainBase = [NumTraits]float64{Str: 40, Dex: 120, Int: 40, Con: 30}

type classHP struct{ Base, PerLevel int }

var ClassMana = [NumClasses]int{Mage: 10, Cleric: 8}

const (
	ManaPerInt       = 2
	ManaRegenPerStep = 1
	ManaRegenDiv     = 4
)

var ClassHP = [NumClasses]classHP{Warrior: {30, 8}, Mage: {20, 5}, Rogue: {25, 6}, Cleric: {28, 7}}

// ClassStats growth is per 2 levels.
type classStats struct{ Start, Growth [NumTraits]int }

var ClassStats = [NumClasses]classStats{
	Warrior: {Growth: [NumTraits]int{Str: 3}},
	Mage:    {Start: [NumTraits]int{Int: 2}, Growth: [NumTraits]int{Int: 3}},
	Rogue:   {Start: [NumTraits]int{Dex: 1}, Growth: [NumTraits]int{Dex: 1}},
	Cleric:  {Growth: [NumTraits]int{Str: 2, Int: 1}},
}

func BagCost(level int) int {
	return int(100 * math.Pow(1.8, float64(level)))
}

func TrainCost(t Trait, level int) int {
	return int(trainBase[t] * math.Pow(1.5, float64(level)))
}

const OfflineCap = 12 * time.Hour
