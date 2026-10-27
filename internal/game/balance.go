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
