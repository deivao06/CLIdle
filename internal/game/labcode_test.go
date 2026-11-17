package game

import (
	"fmt"
	"testing"
)

func TestLabCodeRoundTrip(t *testing.T) {
	custom := DefaultScenario()
	custom.Seed = 1<<64 - 1
	custom.SetClass(Mage)
	custom.SetLevel(30)
	custom.Train = [NumTraits]int{Int: 12, Con: 3}
	custom.Gear[Armor] = LabGear{On: true, Kind: Light, Rarity: Legendary, Depth: 7, Upgrade: 4,
		Effects: []ItemEffect{{LifeSteal, 9}, {Crit, 15}}}
	custom.Gambits = append(custom.Gambits[1:], Gambit{If: [2]Cond{{CondHP, OpLess, -3}}, Do: ActFlee, Exit: ExitTime, Secs: 40})
	custom.Mode = DescendFast
	f := custom.EditFloor(12)
	f.Toggle(0)
	f.Toggle(1)
	f.Toggle(4)
	f.Down = 4
	f.Mons[1] = []LabMon{{Kind: 3, Depth: 9, Count: 4}, {Kind: 2, Boss: true, Depth: 20, Count: 1}}
	for _, s := range []Scenario{DefaultScenario(), bossScenario(Cleric, 60, 3, Rare, true), custom} {
		t.Logf("%2d chars: %s", len(s.Code()), s.Code())
		got, err := ParseCode(s.Code())
		if err != nil || fmt.Sprint(got) != fmt.Sprint(s) {
			t.Fatalf("code %q: %v\n got %v\nwant %v", s.Code(), err, got, s)
		}
	}
	if c := bossScenario(Warrior, 60, 1, Common, false).Code(); len(c) > 24 {
		t.Fatalf("boss fight code %q is %d long", c, len(c))
	}
}

func TestLabCodeRejectsGarbage(t *testing.T) {
	code := bossScenario(Rogue, 30, 2, Common, false).Code()
	for _, bad := range []string{"", "!!", "_____", code[:len(code)/2], "A" + code} {
		if _, err := ParseCode(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	for i := range 2000 {
		b := []byte(code)
		b[i%len(b)] = "AZaz09-_"[i%8]
		ParseCode(string(b))
	}
}
