package game

import (
	"encoding/base64"
	"errors"
	"math/bits"
	"slices"
)

// labCodeVersion is the format of Scenario.Code; it takes the first two bits.
const labCodeVersion = 0

var errLabCode = errors.New("invalid playground code")

type bitWriter struct {
	buf []byte
	n   int
}

func (w *bitWriter) bit(b bool) {
	if w.n%8 == 0 {
		w.buf = append(w.buf, 0)
	}
	if b {
		w.buf[w.n/8] |= 0x80 >> (w.n % 8)
	}
	w.n++
}

func (w *bitWriter) fixed(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bit(v>>i&1 == 1)
	}
}

// num writes v ≥ 0 as an exp-Golomb code: 1 bit for 0, 3 for 1–2, 5 for 3–6 and so on.
func (w *bitWriter) num(v int) {
	u := uint64(v) + 1
	n := bits.Len64(u)
	w.fixed(0, n-1)
	w.fixed(u, n)
}

// signed writes a small signed difference, zigzagged so 0, -1, 1, -2… get the shortest codes.
func (w *bitWriter) signed(v int) { w.num(int(uint(v<<1) ^ uint(v>>63))) }

// big writes v in groups of 4 bits, each followed by a bit telling whether more follow; for seeds of any size.
func (w *bitWriter) big(v uint64) {
	for {
		w.fixed(v&15, 4)
		v >>= 4
		w.bit(v != 0)
		if v == 0 {
			return
		}
	}
}

type bitReader struct {
	buf []byte
	n   int
	err error
}

func (r *bitReader) bit() bool {
	if r.n >= len(r.buf)*8 {
		r.err = errLabCode
		return false
	}
	b := r.buf[r.n/8]&(0x80>>(r.n%8)) != 0
	r.n++
	return b
}

func (r *bitReader) fixed(n int) uint64 {
	var v uint64
	for range n {
		v <<= 1
		if r.bit() {
			v |= 1
		}
	}
	return v
}

func (r *bitReader) num() int {
	zeros := 0
	for !r.bit() {
		if zeros++; zeros > 40 || r.err != nil {
			r.err = errLabCode
			return 0
		}
	}
	return int(1<<zeros|r.fixed(zeros)) - 1
}

func (r *bitReader) signed() int {
	u := r.num()
	return u>>1 ^ -(u & 1)
}

func (r *bitReader) big() uint64 {
	var v uint64
	for shift := 0; ; shift += 4 {
		if shift > 60 {
			r.err = errLabCode
			return 0
		}
		v |= r.fixed(4) << shift
		if !r.bit() {
			return v
		}
	}
}

// upTo reads a number, failing the code and giving 0 when it is max or more.
func (r *bitReader) upTo(max int) int {
	v := r.num()
	if v >= max || r.err != nil {
		r.err = errLabCode
		return 0
	}
	return v
}

// Code packs the scenario into a short text that ParseCode turns back into the same scenario. Values equal to the
// ones before them (gear of the same floor and upgrade, monsters of their floor, the class's default tactics) take
// a bit or two.
func (s Scenario) Code() string {
	w := &bitWriter{}
	w.fixed(labCodeVersion, 2)
	w.big(s.Seed)
	w.num(int(s.Class))
	w.num(s.Level - 1)
	for _, t := range s.Train {
		w.num(t)
	}
	w.num(s.Potions)
	w.num(int(s.Mode))
	w.num(len(s.Floors))
	prev := 0
	for _, f := range s.Floors {
		w.num(f.Depth - prev - 1)
		prev = f.Depth
		on := 0
		for _, b := range f.Rooms {
			w.bit(b)
			if b {
				on++
			}
		}
		if on > 1 {
			w.num(f.Up)
			w.num(f.Down)
		}
		for i, groups := range f.Mons {
			if !f.Rooms[i] {
				continue
			}
			w.num(len(groups))
			for _, m := range groups {
				w.bit(m.Boss)
				w.num(m.Kind)
				w.signed(m.Depth - f.Depth)
				w.num(m.Count - 1)
			}
		}
	}
	depth, up := 1, 0
	if len(s.Floors) > 0 {
		depth = s.Floors[0].Depth
	}
	for _, g := range s.Gear {
		w.bit(g.On)
		if !g.On {
			continue
		}
		w.num(g.Kind)
		w.num(int(g.Rarity))
		w.signed(g.Depth - depth)
		w.signed(g.Upgrade - up)
		depth, up = g.Depth, g.Upgrade
		w.num(len(g.Effects))
		for _, e := range g.Effects {
			w.num(int(e.Kind))
			w.num(e.Value)
		}
	}
	def := slices.Equal(s.Gambits, heroGambits(s.hero()))
	w.bit(def)
	if !def {
		w.num(len(s.Gambits))
		for _, gb := range s.Gambits {
			for _, c := range append(gb.If[:], gb.Until) {
				w.num(int(c.Kind))
				w.num(int(c.Op))
				w.signed(c.N)
			}
			w.num(int(gb.Do))
			w.num(int(gb.Target))
			w.num(int(gb.Exit))
			w.num(gb.Secs)
		}
	}
	return base64.RawURLEncoding.EncodeToString(w.buf)
}

// ParseCode reads a code made by Scenario.Code.
func ParseCode(code string) (Scenario, error) {
	buf, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil {
		return Scenario{}, errLabCode
	}
	r := &bitReader{buf: buf}
	if r.fixed(2) != labCodeVersion {
		return Scenario{}, errLabCode
	}
	var s Scenario
	s.Seed = r.big()
	s.Class = Class(r.upTo(int(NumClasses)))
	s.Level = r.upTo(1000) + 1
	for t := range s.Train {
		s.Train[t] = r.upTo(100000)
	}
	s.Potions = r.upTo(1000)
	s.Mode = ExploreMode(r.upTo(2))
	floors := r.upTo(1000)
	prev := 0
	for range floors {
		if r.err != nil {
			return Scenario{}, errLabCode
		}
		f := LabFloor{Depth: prev + 1 + r.upTo(100000)}
		prev = f.Depth
		on := []int{}
		for i := range f.Rooms {
			if f.Rooms[i] = r.bit(); f.Rooms[i] {
				on = append(on, i)
			}
		}
		if len(on) == 0 || !f.Linked() {
			return Scenario{}, errLabCode
		}
		f.Up, f.Down = on[0], on[0]
		if len(on) > 1 {
			f.Up, f.Down = r.upTo(LabRooms), r.upTo(LabRooms)
			if !f.Rooms[f.Up] || !f.Rooms[f.Down] {
				return Scenario{}, errLabCode
			}
		}
		for _, i := range on {
			groups := r.upTo(100)
			for range groups {
				m := LabMon{Boss: r.bit()}
				kinds := len(MonsterKinds)
				if m.Boss {
					kinds = len(BossKinds)
				}
				m.Kind = r.upTo(kinds)
				m.Depth = f.Depth + r.signed()
				m.Count = r.upTo(1000) + 1
				if m.Depth < 0 || r.err != nil {
					return Scenario{}, errLabCode
				}
				f.Mons[i] = append(f.Mons[i], m)
			}
		}
		s.Floors = append(s.Floors, f)
	}
	depth, up := 1, 0
	if len(s.Floors) > 0 {
		depth = s.Floors[0].Depth
	}
	for slot := range s.Gear {
		g := &s.Gear[slot]
		if g.On = r.bit(); !g.On {
			continue
		}
		g.Kind = r.upTo(len(ItemNames[slot]))
		g.Rarity = Rarity(r.upTo(int(NumRarities)))
		g.Depth = depth + r.signed()
		g.Upgrade = up + r.signed()
		depth, up = g.Depth, g.Upgrade
		if g.Depth < 1 || g.Upgrade < 0 || g.Upgrade > UpgradeMax {
			return Scenario{}, errLabCode
		}
		for range r.upTo(EffectCount[Legendary] + 1) {
			g.Effects = append(g.Effects, ItemEffect{Effect(r.upTo(int(NumEffects))), r.upTo(1000)})
		}
	}
	if r.err != nil {
		return Scenario{}, errLabCode
	}
	if r.bit() {
		s.Gambits = heroGambits(s.hero())
	} else {
		for range r.upTo(MaxGambits + 1) {
			var gb Gambit
			conds := []*Cond{&gb.If[0], &gb.If[1], &gb.Until}
			for _, c := range conds {
				c.Kind = CondKind(r.upTo(int(NumConds)))
				c.Op = Op(r.upTo(int(NumOps)))
				c.N = r.signed()
			}
			gb.Do = Action(r.upTo(int(NumActions)))
			gb.Target = Target(r.upTo(int(NumTargets)))
			gb.Exit = ExitKind(r.upTo(int(NumExits)))
			gb.Secs = r.upTo(100000)
			s.Gambits = append(s.Gambits, gb)
		}
	}
	if r.err != nil {
		return Scenario{}, errLabCode
	}
	return s, nil
}
