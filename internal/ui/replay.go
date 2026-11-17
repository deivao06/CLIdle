package ui

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	replayMaxMs   = int64(480 * time.Hour / time.Millisecond)
	replaySnapMs  = int64(2 * time.Hour / time.Millisecond)
	replayPieceMs = int64(10 * time.Minute / time.Millisecond)
	replaySyncMs  = int64(20 * time.Minute / time.Millisecond)
	replayTickMs  = int64(3 * time.Minute / time.Millisecond)
)

// replaySpeeds are simulated seconds per real second.
var replaySpeeds = []int{1, 10, 60, 600, 3600}

type replaySnap struct {
	ms   int64
	data []byte
}

// replayState is the playground's replay of a simulated run: it plays the same Runner as tools/simview and keeps
// a snapshot every replaySnapMs, so moving the time slider only replays from the nearest snapshot.
type replayState struct {
	code     string
	runner   *game.Runner
	snaps    []replaySnap
	end      int64
	playing  bool
	speed    int
	fx       []fx
	busy     bool
	progress atomic.Int64
	cancel   atomic.Bool
	cmd      tea.Cmd
}

type replayDoneMsg struct{}

func snapRunner(r *game.Runner) replaySnap {
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	json.NewEncoder(z).Encode(r)
	z.Close()
	return replaySnap{r.Ms, b.Bytes()}
}

func (s replaySnap) runner() *game.Runner {
	z, err := gzip.NewReader(bytes.NewReader(s.data))
	if err != nil {
		return nil
	}
	data, _ := io.ReadAll(z)
	r := &game.Runner{}
	if json.Unmarshal(data, r) != nil {
		return nil
	}
	return r
}

// load starts the run a replay code names, at its first second.
func (rp *replayState) load(code string) bool {
	c, p, seed, ok := game.ParseReplayCode(code)
	if !ok {
		return false
	}
	rp.code = game.ReplayCode(c, p, seed)
	rp.runner = game.NewRunner(c, p, seed)
	rp.snaps = []replaySnap{snapRunner(rp.runner)}
	rp.end, rp.playing, rp.fx = 0, false, nil
	return true
}

func (rp *replayState) last() int64 {
	if rp.end > 0 {
		return rp.end
	}
	return replayMaxMs
}

// seek moves the run to time t: from the current state when it is on the way, otherwise from the nearest
// snapshot before t. Long jumps run off the UI loop and return the command that does them.
func (rp *replayState) seek(t int64) tea.Cmd {
	if rp.runner == nil || rp.busy {
		return nil
	}
	t = clamp64(t, 0, rp.last())
	base := rp.snaps[0]
	for _, s := range rp.snaps {
		if s.ms <= t {
			base = s
		}
	}
	if r := rp.runner; r.Ms > t || r.Ms < base.ms {
		if rr := base.runner(); rr != nil {
			rp.runner = rr
		}
		rp.fx = nil
	}
	if t-rp.runner.Ms <= replaySyncMs {
		rp.advanceTo(t)
		return nil
	}
	rp.busy = true
	rp.progress.Store(0)
	rp.cancel.Store(false)
	from := rp.runner.Ms
	return func() tea.Msg {
		rp.advanceTo(t, func(ms int64) {
			rp.progress.Store((ms - from) * 1000 / max(1, t-from))
			yieldToBrowser()
		})
		return replayDoneMsg{}
	}
}

// advanceTo plays the runner up to t in pieces, keeping a snapshot at each replaySnapMs it passes for the first
// time; it stops early when the run ends or the jump is cancelled.
func (rp *replayState) advanceTo(t int64, report ...func(int64)) {
	r := rp.runner
	for r.Ms < t && !rp.cancel.Load() {
		next := (rp.snaps[len(rp.snaps)-1].ms/replaySnapMs + 1) * replaySnapMs
		stop := min(t, r.Ms+replayPieceMs)
		if next > r.Ms && next < stop {
			stop = next
		}
		before := r.Ms
		r.Advance(stop - r.Ms)
		if r.Ms == before {
			rp.end = r.Ms
			return
		}
		if r.Ms >= next {
			rp.snaps = append(rp.snaps, snapRunner(r))
		}
		for _, f := range report {
			f(r.Ms)
		}
	}
}

func clamp64(v, lo, hi int64) int64 { return max(lo, min(hi, v)) }

// replayTime writes a simulated time as 10h 05min, or 10h 05min 30s when seconds matter.
func replayTime(ms int64, secs bool) string {
	s := ms / 1000
	out := fmt.Sprintf("%dh %02dmin", s/3600, s/60%60)
	if secs {
		out += fmt.Sprintf(" %02ds", s%60)
	}
	return out
}

// parseReplayTime reads 10:30, 10h30, 10h or 10.
func parseReplayTime(s string) (int64, bool) {
	s = strings.NewReplacer("h", ":", "H", ":", " ", "").Replace(strings.TrimSpace(s))
	hs, ms, _ := strings.Cut(s, ":")
	h, err := strconv.Atoi(hs)
	m := 0
	if ms != "" {
		m, err = strconv.Atoi(ms)
	}
	if err != nil || h < 0 || m < 0 || m > 59 {
		return 0, false
	}
	return (int64(h)*60 + int64(m)) * 60000, true
}

// replaySlider draws the time bar: the part already computed is brighter than the rest.
func (rp *replayState) slider(w int) string {
	w = max(10, w)
	at := int(rp.runner.Ms * int64(w-1) / replayMaxMs)
	known := int(rp.snaps[len(rp.snaps)-1].ms * int64(w-1) / replayMaxMs)
	var b strings.Builder
	for i := range w {
		switch {
		case i == at:
			b.WriteString(sTitle.Render("●"))
		case i < at:
			b.WriteString(sHPGood.Render("━"))
		case i <= known:
			b.WriteString(sPlain.Render("─"))
		default:
			b.WriteString(sMuted.Render("─"))
		}
	}
	return b.String()
}

func (l Lab) replayRows() []prepRow {
	rp, st := l.rp, l.st
	sec := lang.T("lab.section.replay")
	code := rp.code
	if code == "" {
		code = lang.T("lab.none")
	}
	if st.typing && st.field == fieldReplay {
		code = st.buf + "▏"
	}
	rows := []prepRow{{section: sec, text: fmt.Sprintf("%-24s %s", lang.T("lab.replay.code"), code), style: sPlain,
		desc: lang.T("lab.replay.code.desc"), act: func(*Model) { st.typing, st.field, st.buf = true, fieldReplay, "" }}}
	if rp.runner == nil {
		return rows
	}
	when := replayTime(rp.runner.Ms, true) + " / " + replayTime(rp.last(), false)
	if st.typing && st.field == fieldTime {
		when = st.buf + "▏"
	}
	speed := replaySpeeds[rp.speed]
	rows = append(rows,
		prepRow{section: sec, text: fmt.Sprintf("%-24s %s", lang.T("lab.replay.time"), when), style: sPlain,
			desc: lang.T("lab.replay.time.desc"),
			adjust: func(d int) {
				step := int64(time.Minute / time.Millisecond)
				if d >= labBigStep || d <= -labBigStep {
					step = int64(time.Hour / time.Millisecond)
				}
				rp.cmd = rp.seek(rp.runner.Ms + int64(sign(d))*step)
			},
			act: func(*Model) { st.typing, st.field, st.buf = true, fieldTime, "" }},
		labRow(sec, lang.T("lab.replay.speed"), replaySpeed(speed), lang.T("lab.replay.speed.desc"), func(d int) {
			rp.speed = clamp(rp.speed+sign(d), 0, len(replaySpeeds)-1)
		}),
	)
	return rows
}

func replaySpeed(s int) string {
	switch {
	case s >= 3600:
		return lang.T("lab.replay.per_sec", fmt.Sprintf("%dh", s/3600))
	case s >= 60:
		return lang.T("lab.replay.per_sec", fmt.Sprintf("%dmin", s/60))
	}
	return fmt.Sprintf("%d×", s)
}

// replayHead is what the replay page shows above its rows: the keys, the slider and where the hero is.
func (l Lab) replayHead(w int) []string {
	rp := l.rp
	head := []string{hints(w, "↑↓", lang.T("lab.key.choose"), "←→", lang.T("lab.replay.key.minute"), "pgup/pgdn",
		lang.T("lab.replay.key.hour"), "enter", lang.T("lab.key.enter"))}
	if rp.runner == nil {
		return append(head, "", sMuted.Width(w).Render(lang.T("lab.replay.empty")))
	}
	g := rp.runner.Game
	h := g.Hero
	head = append(head, "", rp.slider(w-2),
		sMuted.Render(lang.T("lab.replay.known", replayTime(rp.snaps[len(rp.snaps)-1].ms, false))), "")
	where := lang.T("lab.replay.town")
	if g.Exp != nil {
		where = lang.T("lab.replay.exp", g.Exp.Num, g.Floor(g.Exp.Level.Depth))
	}
	if rp.end > 0 && rp.runner.Ms >= rp.end {
		where = lang.T("lab.replay.ended")
	}
	head = append(head, sTitle.Render(where),
		fmt.Sprintf("%s %d (%s) · HP %d/%d · %s %d · %s %d", h.Class.Name(), h.Level, rp.runner.Profile.Name(), h.HP, h.MaxHP(),
			lang.T("lab.replay.deaths"), g.Deaths, lang.T("lab.replay.gold"), g.Gold))
	if g.Exp != nil {
		head = append(head, sMuted.Render("▸ "+g.Doing()))
	}
	return head
}

// replayMap is the right column of the replay page: the hero's floor, or a note while in town.
func (l Lab) replayMap(h int) string {
	rp := l.rp
	if rp.runner == nil {
		return sMuted.Render(lang.T("lab.replay.no_run"))
	}
	g := rp.runner.Game
	if g.Exp == nil {
		return sMuted.Render(lang.T("lab.replay.town_map"))
	}
	m := l.Model
	m.w, m.g, m.fx = game.MapW, g, rp.fx
	g.Effects = l.g.Effects
	return m.viewDungeon(h)
}

// replayTick plays the replay at its speed and gathers its map effects.
func (l *Lab) replayTick(gap int64, now time.Time) {
	rp := l.rp
	if rp.runner == nil || rp.busy {
		return
	}
	if rp.playing {
		rp.advanceTo(min(rp.last(), rp.runner.Ms+min(replayTickMs, gap*int64(replaySpeeds[rp.speed]))))
		rp.playing = rp.runner.Ms < rp.last()
	}
	m := l.Model
	m.g, m.fx = rp.runner.Game, rp.fx
	m.collectFx(now)
	rp.fx = m.fx
}

// replayKey handles the keys the replay page uses differently, reporting whether it did.
func (l *Lab) replayKey(k string) bool {
	rp := l.rp
	if rp.runner == nil {
		return false
	}
	switch k {
	case " ":
		if rp.runner.Ms >= rp.last() {
			rp.cmd = rp.seek(0)
		}
		rp.playing = !rp.playing
	case "p":
		rp.playing = false
		rp.advanceTo(rp.runner.Ms + rp.runner.Game.StepMs())
	case "+", "=":
		rp.speed = min(len(replaySpeeds)-1, rp.speed+1)
	case "-", "_":
		rp.speed = max(0, rp.speed-1)
	case "home":
		rp.cmd = rp.seek(0)
	default:
		return false
	}
	return true
}

func (l Lab) replayBusy() string {
	msg := sTitle.Render(lang.T("lab.replay.busy", l.rp.code)) + "\n\n" +
		bar(int(l.rp.progress.Load()), 1000, 40, sHPGood) + fmt.Sprintf(" %d%%", l.rp.progress.Load()/10) + "\n\n" +
		sMuted.Render(lang.T("lab.replay.busy.esc"))
	return lipgloss.Place(l.w, l.h, lipgloss.Center, lipgloss.Center, msg)
}

// replayStatus is the header's right side on the replay page.
func (l Lab) replayStatus() string {
	rp := l.rp
	if rp.runner == nil {
		return sMuted.Render(lang.T("lab.replay.none"))
	}
	t := replayTime(rp.runner.Ms, true)
	if rp.playing {
		return sHPGood.Render(lang.T("lab.replay.playing", t, replaySpeed(replaySpeeds[rp.speed])))
	}
	return sTitle.Render(lang.T("lab.replay.paused", t, replaySpeed(replaySpeeds[rp.speed])))
}
