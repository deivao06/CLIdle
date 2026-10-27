package game

// pathStep runs a BFS from start over tiles accepted by pass and returns the first
// step toward the nearest tile accepted by goal.
func (l *Level) pathStep(start Pos, pass func(Pos) bool, goal func(Pos) bool) (Pos, bool) {
	prev := make([]int32, MapW*MapH)
	for i := range prev {
		prev[i] = -1
	}
	idx := func(p Pos) int { return p.Y*MapW + p.X }
	queue := []Pos{start}
	prev[idx(start)] = int32(idx(start))
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p != start && goal(p) {
			for {
				q := Pos{int(prev[idx(p)]) % MapW, int(prev[idx(p)]) / MapW}
				if q == start {
					return p, true
				}
				p = q
			}
		}
		for _, d := range Dirs8 {
			n := p.Add(d)
			if !l.In(n) || prev[idx(n)] >= 0 || !l.CanStep(p, n) {
				continue
			}
			if !goal(n) && !pass(n) {
				continue
			}
			prev[idx(n)] = int32(idx(p))
			queue = append(queue, n)
		}
	}
	return Pos{}, false
}

// distMap returns walking distances from start over walkable tiles, ignoring monsters.
func (l *Level) distMap(start Pos) []int {
	dist := make([]int, MapW*MapH)
	for i := range dist {
		dist[i] = -1
	}
	dist[start.Y*MapW+start.X] = 0
	queue := []Pos{start}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range Dirs8 {
			n := p.Add(d)
			if !l.In(n) || dist[n.Y*MapW+n.X] >= 0 || !l.CanStep(p, n) {
				continue
			}
			dist[n.Y*MapW+n.X] = dist[p.Y*MapW+p.X] + 1
			queue = append(queue, n)
		}
	}
	return dist
}
