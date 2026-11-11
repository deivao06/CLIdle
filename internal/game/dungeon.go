package game

const (
	MapW = 80
	MapH = 21
)

type Tile byte

const (
	Rock Tile = iota
	Floor
	HWall
	VWall
	Door
	Corridor
	StairsDown
	StairsUp
)

func (t Tile) Walkable() bool {
	switch t {
	case Floor, Door, Corridor, StairsDown, StairsUp:
		return true
	}
	return false
}

func (t Tile) Glyph() rune {
	switch t {
	case Floor:
		return '.'
	case HWall:
		return '-'
	case VWall:
		return '|'
	case Door:
		return '+'
	case Corridor:
		return '#'
	case StairsDown:
		return '>'
	case StairsUp:
		return '<'
	}
	return ' '
}

type Pos struct{ X, Y int }

func (p Pos) Add(d Pos) Pos { return Pos{p.X + d.X, p.Y + d.Y} }

func (p Pos) Dist(o Pos) int { return max(abs(p.X-o.X), abs(p.Y-o.Y)) }

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var Dirs8 = []Pos{{0, -1}, {1, 0}, {0, 1}, {-1, 0}, {1, -1}, {1, 1}, {-1, 1}, {-1, -1}}

// Room is the rectangle including its walls.
type Room struct{ X, Y, W, H int }

func (r Room) Contains(p Pos) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

func (r Room) Inside(p Pos) bool {
	return p.X > r.X && p.X < r.X+r.W-1 && p.Y > r.Y && p.Y < r.Y+r.H-1
}

type Level struct {
	Depth int
	Tiles []byte
	Seen  []byte
	Rooms []Room
	Up    Pos
	Down  Pos
	Items []*FloorItem
	Mons  []*Monster
}

func (l *Level) In(p Pos) bool { return p.X >= 0 && p.Y >= 0 && p.X < MapW && p.Y < MapH }

func (l *Level) At(p Pos) Tile {
	if !l.In(p) {
		return Rock
	}
	return Tile(l.Tiles[p.Y*MapW+p.X])
}

func (l *Level) Set(p Pos, t Tile) { l.Tiles[p.Y*MapW+p.X] = byte(t) }

func (l *Level) IsSeen(p Pos) bool { return l.In(p) && l.Seen[p.Y*MapW+p.X] == 1 }

func (l *Level) RoomAt(p Pos) int {
	for i, r := range l.Rooms {
		if r.Contains(p) {
			return i
		}
	}
	return -1
}

// CanStep forbids diagonal moves into or out of doors, like Rogue.
func (l *Level) CanStep(from, to Pos) bool {
	if !l.At(to).Walkable() {
		return false
	}
	if from.X != to.X && from.Y != to.Y {
		if l.At(from) == Door || l.At(to) == Door {
			return false
		}
	}
	return true
}

func (l *Level) MonsterAt(p Pos) *Monster {
	for _, m := range l.Mons {
		if m.Pos == p && m.HP > 0 {
			return m
		}
	}
	return nil
}

func (l *Level) ItemAt(p Pos) int {
	for i, it := range l.Items {
		if it.Pos == p {
			return i
		}
	}
	return -1
}

// Visible follows Rogue: a lit room is fully visible from inside, otherwise only adjacent tiles.
func (l *Level) Visible(from, p Pos) bool {
	if from.Dist(p) <= 1 {
		return true
	}
	ri := l.RoomAt(from)
	return ri >= 0 && l.Rooms[ri].Contains(p) && l.At(from) != Door
}

func (l *Level) Reveal(from Pos) {
	for _, d := range append(Dirs8, Pos{}) {
		p := from.Add(d)
		if l.In(p) {
			l.Seen[p.Y*MapW+p.X] = 1
		}
	}
	ri := l.RoomAt(from)
	if ri < 0 || l.At(from) == Door {
		return
	}
	r := l.Rooms[ri]
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			l.Seen[y*MapW+x] = 1
		}
	}
}

const floorTries = 100

// RandomFloor picks a free floor tile in the room, or the first free one in this or the next rooms when it is crowded.
func (l *Level) RandomFloor(rng *RNG, room int) Pos {
	free := func(p Pos) bool { return l.At(p) == Floor && l.MonsterAt(p) == nil && l.ItemAt(p) < 0 }
	r := l.Rooms[room]
	for range floorTries {
		p := Pos{rng.Range(r.X+1, r.X+r.W-2), rng.Range(r.Y+1, r.Y+r.H-2)}
		if free(p) {
			return p
		}
	}
	for i := range l.Rooms {
		r := l.Rooms[(room+i)%len(l.Rooms)]
		for y := r.Y + 1; y <= r.Y+r.H-2; y++ {
			for x := r.X + 1; x <= r.X+r.W-2; x++ {
				if p := (Pos{x, y}); free(p) {
					return p
				}
			}
		}
	}
	return Pos{r.X + 1, r.Y + 1}
}

const (
	gridCols, gridRows = 3, 3
	gridRooms          = gridCols * gridRows
)

func GenerateLevel(rng *RNG, depth int) *Level {
	var all [gridRooms]bool
	for i := range all {
		all[i] = true
	}
	l := carveLevel(rng, depth, all)
	up := rng.Intn(len(l.Rooms))
	down := rng.Intn(len(l.Rooms) - 1)
	if down >= up {
		down++
	}
	l.Up = l.RandomFloor(rng, up)
	l.Set(l.Up, StairsUp)
	l.Down = l.RandomFloor(rng, down)
	l.Set(l.Down, StairsDown)
	return l
}

// carveLevel lays out the 3x3 grid; rooms that are off keep their rectangle but stay solid rock.
func carveLevel(rng *RNG, depth int, on [gridRooms]bool) *Level {
	l := &Level{
		Depth: depth,
		Tiles: make([]byte, MapW*MapH),
		Seen:  make([]byte, MapW*MapH),
	}
	cw, ch := MapW/gridCols, MapH/gridRows
	for gy := range gridRows {
		for gx := range gridCols {
			w := rng.Range(6, cw-2)
			h := rng.Range(4, ch-1)
			x := gx*cw + rng.Range(0, cw-1-w)
			y := gy*ch + rng.Range(0, ch-1-h)
			if r := (Room{x, y, w, h}); on[len(l.Rooms)] {
				l.carveRoom(r)
			} else {
				l.Rooms = append(l.Rooms, r)
			}
		}
	}

	type edge struct{ a, b int }
	var edges []edge
	for gy := range gridRows {
		for gx := range gridCols {
			i := gy*gridCols + gx
			if gx+1 < gridCols {
				edges = append(edges, edge{i, i + 1})
			}
			if gy+1 < gridRows {
				edges = append(edges, edge{i, i + gridCols})
			}
		}
	}
	for i := len(edges) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		edges[i], edges[j] = edges[j], edges[i]
	}
	parent := make([]int, len(l.Rooms))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	for _, e := range edges {
		if !on[e.a] || !on[e.b] {
			continue
		}
		if find(e.a) != find(e.b) {
			parent[find(e.a)] = find(e.b)
			l.connect(rng, e.a, e.b)
		} else if rng.Chance(0.25) {
			l.connect(rng, e.a, e.b)
		}
	}
	return l
}

func (l *Level) carveRoom(r Room) {
	l.Rooms = append(l.Rooms, r)
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			t := Floor
			if y == r.Y || y == r.Y+r.H-1 {
				t = HWall
			} else if x == r.X || x == r.X+r.W-1 {
				t = VWall
			}
			l.Set(Pos{x, y}, t)
		}
	}
}

// connect joins two grid-adjacent rooms; a is always left of or above b.
func (l *Level) connect(rng *RNG, a, b int) {
	ra, rb := l.Rooms[a], l.Rooms[b]
	if b == a+1 {
		da := Pos{ra.X + ra.W - 1, rng.Range(ra.Y+1, ra.Y+ra.H-2)}
		db := Pos{rb.X, rng.Range(rb.Y+1, rb.Y+rb.H-2)}
		mid := rng.Range(da.X+1, db.X-1)
		l.Set(da, Door)
		l.Set(db, Door)
		l.carveH(da.Y, da.X+1, mid)
		l.carveV(mid, da.Y, db.Y)
		l.carveH(db.Y, mid, db.X-1)
		return
	}
	da := Pos{rng.Range(ra.X+1, ra.X+ra.W-2), ra.Y + ra.H - 1}
	db := Pos{rng.Range(rb.X+1, rb.X+rb.W-2), rb.Y}
	mid := rng.Range(da.Y+1, db.Y-1)
	l.Set(da, Door)
	l.Set(db, Door)
	l.carveV(da.X, da.Y+1, mid)
	l.carveH(mid, da.X, db.X)
	l.carveV(db.X, mid, db.Y-1)
}

func (l *Level) carveH(y, x0, x1 int) {
	for x := min(x0, x1); x <= max(x0, x1); x++ {
		if l.At(Pos{x, y}) == Rock {
			l.Set(Pos{x, y}, Corridor)
		}
	}
}

func (l *Level) carveV(x, y0, y1 int) {
	for y := min(y0, y1); y <= max(y0, y1); y++ {
		if l.At(Pos{x, y}) == Rock {
			l.Set(Pos{x, y}, Corridor)
		}
	}
}
