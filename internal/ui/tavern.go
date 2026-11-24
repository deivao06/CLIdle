package ui

import (
	"math"
	"strings"

	"clidle/internal/game"

	"github.com/charmbracelet/lipgloss"
)

// tavernRows is the guild's tavern: & the fireplace, s a seat, π a table, ▤ the quest board, = and ║ the counter,
// 0 a keg, + the door; the two k mark the ends of the keeper's path behind the counter.
var tavernRows = []string{
	" /-----------------------\\ ",
	"/&..sπs...............▤▤▤.\\",
	"|.........sπs.....sπs.....|",
	"|..sπs....................+",
	"|......................0..+",
	"|============..........000|",
	"\\k.........k║.........0000/",
	" \\-----------------------/ ",
}

var (
	heroColors  = []string{"226", "51", "213", "118", "208", "141", "45", "203"}
	tavernColor = map[rune]string{'π': "130", '=': "94", '║': "94", '0': "136", '▤': "180"}
	sKeeper     = lipgloss.NewStyle().Foreground(lipgloss.Color("173"))
	sResting    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	tableStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(tavernColor['π']))
)

const keeperStepMs = 900

// heroStyle is the color of the i-th hero's @ in the tavern and in the guild's hero list.
func heroStyle(i int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(heroColors[i%len(heroColors)])).Bold(true)
}

// tavern draws the guild's tavern with the heroes who are home, each in their own seat; when hl is set, only
// that hero stands out, and in the guild the hero under the cursor does.
func (m Model) tavern(hl *game.Game) string {
	ms := m.lastTick.UnixMilli()
	frame := ms / campfireFrameMs
	radius := 32 + flicker(frame, 0)*3
	picked := -1
	if m.guild && m.tab == tabGuild {
		picked = m.cursor
	}
	var kx, ky, kFrom, kTo int
	for y, row := range tavernRows {
		for x, r := range []rune(row) {
			if r == 'k' {
				if kFrom == 0 {
					kFrom = x
				}
				kTo, ky = x, y
			}
		}
	}
	span := kTo - kFrom
	step := int(ms/keeperStepMs) % (2 * span)
	kx = kFrom + min(step, 2*span-step)

	var b, run strings.Builder
	var runStyle lipgloss.Style
	runKey := ""
	flush := func() {
		if run.Len() > 0 {
			b.WriteString(runStyle.Render(run.String()))
			run.Reset()
		}
	}
	put := func(key string, st lipgloss.Style, ch rune) {
		if key != runKey {
			flush()
			runKey, runStyle = key, st
		}
		run.WriteRune(ch)
	}
	seat := 0
	for y, row := range tavernRows {
		for x, r := range []rune(row) {
			dx, dy := float64(x-1), float64(y-1)*2
			d := math.Sqrt(dx*dx + dy*dy)
			lit := d <= radius
			light := lightColors[min(len(lightColors)-1, int(math.Pow(d/radius, 1.5)*float64(len(lightColors))))]
			switch {
			case r == '&':
				c := fireColors[int(flicker(frame, 1)*float64(len(fireColors)))]
				put("fire", lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true), '&')
				continue
			case x == kx && y == ky:
				put("keeper", sKeeper, 'k')
				continue
			case r == 'k':
				r = '.'
			case r == 's':
				i := seat
				seat++
				if i < len(m.g.Guild.Heroes) && m.g.Guild.Heroes[i].Exp == nil {
					st, key := heroStyle(i), "hero"+heroColors[i%len(heroColors)]
					switch {
					case hl != nil && m.g.Guild.Heroes[i] != hl:
						st, key = sResting, "resting"
					case hl != nil:
						st, key = sHero, "hl"
					case i == picked:
						st, key = st.Reverse(true), "picked"
					}
					put(key, st, '@')
					continue
				}
				r = '.'
			}
			if !lit {
				put("dark", sDark, r)
				continue
			}
			if c, ok := tavernColor[r]; ok {
				put("obj"+c, lipgloss.NewStyle().Foreground(lipgloss.Color(c)), r)
				continue
			}
			put(light, lipgloss.NewStyle().Foreground(lipgloss.Color(light)), r)
		}
		flush()
		runKey = ""
		if y < len(tavernRows)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
