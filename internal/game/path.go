package game

type pathScratch struct {
	stamp uint32
	seen  []uint32
	prev  []int32
	queue []int32
}

var scratch = pathScratch{
	seen:  make([]uint32, MapW*MapH),
	prev:  make([]int32, MapW*MapH),
	queue: make([]int32, 0, MapW*MapH),
}

func idxPos(i int32) Pos { return Pos{int(i) % MapW, int(i) / MapW} }

// search runs one BFS from start over seen tiles accepted by pass and returns the first step
// toward the nearest tile of the highest-priority goal (lowest index) that is reachable.
// Goal tiles are entered even when not passable but never walked through.
func (l *Level) search(start Pos, pass func(Pos) bool, goals ...func(Pos) bool) (Pos, int) {
	s := &scratch
	s.stamp++
	found := make([]int32, len(goals))
	for i := range found {
		found[i] = -1
	}
	best := len(goals)
	isGoal := func(p Pos) bool {
		for i := range best {
			if goals[i](p) {
				return true
			}
		}
		return false
	}
	si := int32(start.Y*MapW + start.X)
	s.seen[si] = s.stamp
	s.prev[si] = si
	s.queue = append(s.queue[:0], si)
	for head := 0; head < len(s.queue) && best > 0; head++ {
		ci := s.queue[head]
		p := idxPos(ci)
		if ci != si {
			for i := range best {
				if found[i] < 0 && goals[i](p) {
					found[i] = ci
					best = i
					break
				}
			}
			if !pass(p) {
				continue
			}
		}
		for _, d := range Dirs8 {
			n := p.Add(d)
			if !l.In(n) {
				continue
			}
			ni := int32(n.Y*MapW + n.X)
			if s.seen[ni] == s.stamp || l.Seen[ni] == 0 || !l.CanStep(p, n) || (!pass(n) && !isGoal(n)) {
				continue
			}
			s.seen[ni] = s.stamp
			s.prev[ni] = ci
			s.queue = append(s.queue, ni)
		}
	}
	if best == len(goals) {
		return Pos{}, -1
	}
	ci := found[best]
	for s.prev[ci] != si {
		ci = s.prev[ci]
	}
	return idxPos(ci), best
}

func (l *Level) pathStep(start Pos, pass func(Pos) bool, goal func(Pos) bool) (Pos, bool) {
	p, i := l.search(start, pass, goal)
	return p, i >= 0
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
