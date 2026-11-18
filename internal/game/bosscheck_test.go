package game

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

// bossCheckLevels is the median level heroes had on first reaching each boss floor in the 480h dungeon simulation
// (Balanço 2, 2026-11-18).
var bossCheckLevels = map[int]int{10: 4, 20: 9, 30: 16, 40: 25, 50: 39, 60: 55, 70: 69, 80: 81, 90: 95, 100: 118}

// bossCheckTrain is the median total training of each class on first reaching each boss floor in the same simulation.
var bossCheckTrain = [NumClasses]map[int]int{
	Warrior: {10: 4, 20: 10, 30: 19, 40: 36, 50: 59, 60: 85, 70: 99, 80: 116, 90: 136, 100: 169},
	Mage:    {10: 0, 20: 6, 30: 12, 40: 23, 50: 42, 60: 75, 70: 94, 80: 121, 90: 145, 100: 178},
	Rogue:   {10: 3, 20: 8, 30: 16, 40: 33, 50: 44, 60: 66, 70: 82, 80: 112, 90: 131, 100: 159},
	Cleric:  {10: 4, 20: 8, 30: 17, 40: 25, 50: 42, 60: 69, 70: 93, 80: 97, 90: 131, 100: 164},
}

// bossTraining splits the total training between the class's main trained attribute and CON, the extra point
// going to the main one, as the careful profile buys them.
func bossTraining(c Class, total int) [NumTraits]int {
	var t [NumTraits]int
	t[auditTraits[c][0]] = total - total/2
	t[Con] += total / 2
	return t
}

// bossSeed is the first fight's seed; the others step from it.
const bossSeed = 31415926

const (
	bossCheckFights = 20
	bossCheckSteps  = 20000
)

// bossScenario is a hero of the usual level for the boss floor, untrained unless trained (typical training), with gear of that floor and rarity at +10
// (no item effects), five potions and the default tactics, in one room with the boss.
func bossScenario(c Class, floor int, seed uint64, rarity Rarity, trained bool) Scenario {
	s := DefaultScenario()
	s.Seed = seed
	s.SetClass(c)
	s.SetLevel(bossCheckLevels[floor])
	s.Potions = 5
	if trained {
		s.Train = bossTraining(c, bossCheckTrain[c][floor])
	}
	gear := ClassGear[c]
	kinds := [NumSlots]int{Weapon: gear.Weapon[0], Armor: gear.Armor[len(gear.Armor)-1]}
	for slot := range NumSlots {
		s.Gear[slot] = LabGear{On: Slot(slot) != Shield || gear.Shield, Kind: kinds[slot], Rarity: rarity, Depth: floor, Upgrade: UpgradeMax}
	}
	s.Floors = nil
	f := s.EditFloor(floor)
	f.Toggle(LabRooms / 2)
	f.Mons[LabRooms/2] = []LabMon{{Kind: (floor/BossEvery - 1) % len(BossKinds), Boss: true, Depth: floor, Count: 1}}
	return s
}

// bossFight plays one fight and returns whether the hero killed the boss and the boss HP left, in percent.
func bossFight(s Scenario) (bool, int) {
	g := NewLab(s)
	boss := g.Exp.Level.boss()
	for i := 0; i < bossCheckSteps && g.Lab().End == LabRunning && boss.HP > 0; i++ {
		g.LabStep()
	}
	if boss.HP <= 0 {
		return true, 0
	}
	return false, boss.HP * 100 / boss.MaxHP
}

// TestBossCheck fights every boss of the first dungeon bossCheckFights times for one class (SIM_CLASS),
// writing the win rate per boss to tools/simview with the playground code of the first lost fight (or the first
// fight when none was lost).
func TestBossCheck(t *testing.T) {
	cs := os.Getenv("SIM_CLASS")
	if cs == "" || os.Getenv("BOSS_CHECK") == "" {
		t.Skip()
	}
	ci, _ := strconv.Atoi(cs)
	c := Class(ci)
	rarity := map[string]Rarity{"": Common, "common": Common, "rare": Rare}[os.Getenv("BOSS_RARITY")]
	if rarity == Common && os.Getenv("BOSS_RARITY") != "" && os.Getenv("BOSS_RARITY") != "common" {
		t.Fatal("BOSS_RARITY must be common or rare")
	}
	rec := newSimRecorder()
	rec.file.Kind = "boss"
	trained := os.Getenv("BOSS_TRAIN") == "typical"
	ser := rec.series(c, "", 0)
	for floor := BossEvery; floor <= DungeonFloors; floor += BossEvery {
		p := bossPoint{Floor: floor, Level: bossCheckLevels[floor], Fights: bossCheckFights}
		left := 0
		for i := range uint64(bossCheckFights) {
			seed := bossSeed + i*7919
			won, l := bossFight(bossScenario(c, floor, seed, rarity, trained))
			if won {
				p.Wins++
			}
			if p.Seed == 0 || p.Won && !won {
				p.Seed, p.Won = seed, won
			}
			left += l
		}
		p.Code = bossScenario(c, floor, p.Seed, rarity, trained).Code()
		p.Pct = p.Wins * 100 / p.Fights
		p.BossLeft = left / p.Fights
		ser.Fights = append(ser.Fights, p)
		rec.flush()
		fmt.Printf("BOSS %-8s floor %3d level %2d: won %2d/%d (%3d%%), boss HP left %3d%%\n", c.Name(), floor, p.Level, p.Wins, p.Fights, p.Pct, p.BossLeft)
	}
	rec.done(ser)
}

func TestBossFightCodeReplaysTheSameFight(t *testing.T) {
	for _, c := range []Class{Warrior, Mage} {
		s := bossScenario(c, 20, 3, Rare, true)
		parsed, err := ParseCode(s.Code())
		if err != nil {
			t.Fatal(err)
		}
		won, left := bossFight(s)
		won2, left2 := bossFight(parsed)
		if won != won2 || left != left2 {
			t.Fatalf("%v: direct %v/%d, from code %v/%d", c, won, left, won2, left2)
		}
	}
}
