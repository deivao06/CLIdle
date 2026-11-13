package game

// addGold sums gold without wrapping past MaxGold.
func addGold(a, b int) int {
	if b > MaxGold-a {
		return MaxGold
	}
	return a + b
}

// pctOf is n*pct/100 for n ≥ 0, stopping at MaxGold instead of wrapping.
func pctOf(n, pct int) int {
	q, r := n/100, n%100
	if pct > 0 && q > (MaxGold-r*pct/100)/pct {
		return MaxGold
	}
	return q*pct + r*pct/100
}

// goldOf truncates a float to whole gold, stopping at MaxGold.
func goldOf(f float64) int {
	if f >= float64(MaxGold) {
		return MaxGold
	}
	return int(f)
}
