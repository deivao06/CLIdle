package game

import (
	"encoding/json"
	"math/rand/v2"
)

type RNG struct {
	pcg *rand.PCG
	r   *rand.Rand
}

func NewRNG(seed uint64) *RNG {
	pcg := rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)
	return &RNG{pcg: pcg, r: rand.New(pcg)}
}

func (g *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return g.r.IntN(n)
}

// Range returns a value in [lo, hi].
func (g *RNG) Range(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + g.r.IntN(hi-lo+1)
}

func (g *RNG) Float() float64 { return g.r.Float64() }

func (g *RNG) Chance(p float64) bool { return g.r.Float64() < p }

func (g *RNG) Uint64() uint64 { return g.r.Uint64() }

func (g *RNG) MarshalJSON() ([]byte, error) {
	b, err := g.pcg.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return json.Marshal(b)
}

func (g *RNG) UnmarshalJSON(data []byte) error {
	var b []byte
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	pcg := &rand.PCG{}
	if err := pcg.UnmarshalBinary(b); err != nil {
		return err
	}
	g.pcg = pcg
	g.r = rand.New(pcg)
	return nil
}
