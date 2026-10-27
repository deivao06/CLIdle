package game

const (
	HeroBaseSpd = 4

	TownWaitMs = 5000
	LogSize    = 200
)

// StepsPerSec converts the speed attribute into exploration pace.
func StepsPerSec(spd int) float64 { return 1 + float64(spd)*0.5 }
