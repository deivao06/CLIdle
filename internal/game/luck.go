package game

import (
	"math"
	"strconv"
	"strings"

	"clidle/internal/lang"
)

// LuckStreakMin is the streak length from which the log shows its odds.
const LuckStreakMin = 3

// Streak is a run of misses in a row and the chance of the whole run.
type Streak struct {
	Run    int     `json:",omitempty"`
	Chance float64 `json:",omitempty"`
}

// add counts one more miss of chance p, keeps best as the least likely streak, and reports whether the odds show.
func (s *Streak) add(p float64, best *Streak) bool {
	if s.Run == 0 {
		s.Chance = 1
	}
	s.Run++
	s.Chance *= p
	if s.Run < LuckStreakMin {
		return false
	}
	if best.Run == 0 || s.Chance < best.Chance {
		*best = *s
	}
	return true
}

// Odds reads like "(1 in 64 · 1.6%)".
func (s Streak) Odds() string {
	pct := s.Chance * 100
	dec := 1
	if pct < 1 {
		dec = min(8, int(math.Ceil(-math.Log10(pct)))+1)
	}
	p := strings.Replace(strconv.FormatFloat(pct, 'f', dec, 64), ".", lang.T("num.decimal"), 1)
	return lang.T("luck.odds", group(int64(math.Round(1/s.Chance))), p)
}

func group(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + lang.T("num.thousands") + s[i:]
	}
	return s
}
