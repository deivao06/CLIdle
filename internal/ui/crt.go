package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var ansi16 = [16][3]int{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0}, {0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

func xtermRGB(n int) (int, int, int) {
	switch {
	case n < 16:
		c := ansi16[max(n, 0)]
		return c[0], c[1], c[2]
	case n < 232:
		lv := [6]int{0, 95, 135, 175, 215, 255}
		n -= 16
		return lv[n/36], lv[n/6%6], lv[n%6]
	default:
		v := 8 + (min(n, 255)-232)*10
		return v, v, v
	}
}

// crtShade ages a color: a bit less saturated, a bit darker on scanlines and, when flicker is on,
// darker still if it is a gray (secondary text, walls, borders).
func crtShade(r, g, b, dim int, flick bool) string {
	l := (299*r + 587*g + 114*b) / 1000
	gray := max(r, g, b)-min(r, g, b) < crtGrayDiff
	f := 1 - crtDimStep*float64(dim)
	if flick && gray {
		f -= crtFlickerDim
	}
	age := func(c int) string {
		v := float64(c) + float64(l-c)*crtDesat
		return strconv.Itoa(clamp(int(v*f), 0, 255))
	}
	if lipgloss.ColorProfile() == termenv.TrueColor {
		return "2;" + age(r) + ";" + age(g) + ";" + age(b)
	}
	ar, _ := strconv.Atoi(age(r))
	ag, _ := strconv.Atoi(age(g))
	ab, _ := strconv.Atoi(age(b))
	return "5;" + strconv.Itoa(nearestXterm(ar, ag, ab))
}

func nearestXterm(r, g, b int) int {
	best, bestD := 16, 1<<30
	for n := 16; n < 256; n++ {
		xr, xg, xb := xtermRGB(n)
		if d := (xr-r)*(xr-r) + (xg-g)*(xg-g) + (xb-b)*(xb-b); d < bestD {
			best, bestD = n, d
		}
	}
	return best
}

var crtCache = map[string]string{}

// crtSGR rewrites the colors of one SGR sequence (the part between "\x1b[" and "m");
// the terminal's default colors are left alone.
func crtSGR(params string, dim int, flick bool) string {
	key := params + "|" + strconv.Itoa(dim) + strconv.FormatBool(flick)
	if s, ok := crtCache[key]; ok {
		return s
	}
	p := strings.Split(params, ";")
	var out []string
	num := func(i int) int {
		if i >= len(p) {
			return 0
		}
		n, _ := strconv.Atoi(p[i])
		return n
	}
	for i := 0; i < len(p); i++ {
		n := num(i)
		switch {
		case (n == 38 || n == 48) && num(i+1) == 5:
			r, g, b := xtermRGB(num(i + 2))
			out = append(out, strconv.Itoa(n)+";"+crtShade(r, g, b, dim, flick))
			i += 2
		case (n == 38 || n == 48) && num(i+1) == 2:
			out = append(out, strconv.Itoa(n)+";"+crtShade(num(i+2), num(i+3), num(i+4), dim, flick))
			i += 4
		case n >= 30 && n <= 37, n >= 90 && n <= 97:
			r, g, b := xtermRGB(n%10 + 8*(n/90))
			out = append(out, "38;"+crtShade(r, g, b, dim, flick))
		case n >= 40 && n <= 47, n >= 100 && n <= 107:
			r, g, b := xtermRGB(n%10 + 8*(n/100))
			out = append(out, "48;"+crtShade(r, g, b, dim, flick))
		default:
			out = append(out, p[i])
		}
	}
	s := "\x1b[" + strings.Join(out, ";") + "m"
	crtCache[key] = s
	return s
}

var crtTear = strings.NewReplacer("│", "╎", "─", "╌")

// crt makes a rendered frame look like an old terminal without changing its text or layout:
// slightly faded colors, darker odd lines (scanlines), rare flicker of the gray elements and,
// on rare frames, a torn line in the interface borders.
func crt(s string, frame int64) string {
	lines := strings.Split(s, "\n")
	flick := flicker(frame, crtFlickerSalt) < crtFlickerChance
	tearY := -1
	if flicker(frame, crtGlitchSalt) < crtGlitchChance {
		tearY = int(flicker(frame, crtGlitchSalt+1) * float64(len(lines)))
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/2)
	for y, line := range lines {
		if y > 0 {
			b.WriteByte('\n')
		}
		if y == tearY {
			line = crtTear.Replace(line)
		}
		dim := y % 2
		for {
			i := strings.Index(line, "\x1b[")
			j := -1
			if i >= 0 {
				j = strings.IndexByte(line[i:], 'm')
			}
			if j < 0 {
				b.WriteString(line)
				break
			}
			b.WriteString(line[:i])
			b.WriteString(crtSGR(line[i+2:i+j], dim, flick))
			line = line[i+j+1:]
		}
	}
	return b.String()
}
