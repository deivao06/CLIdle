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

// crtShade tints a color slightly toward phosphor green; dim darkens it a little per step.
func crtShade(r, g, b, dim int) string {
	f := 1 - crtDimStep*float64(dim)
	r = int(float64(r) * (1 - crtTint) * f)
	g = int(min(255, float64(g)+crtGreenLift) * f)
	b = int(float64(b) * (1 - crtTint) * f)
	if lipgloss.ColorProfile() == termenv.TrueColor {
		return "2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b)
	}
	return "5;" + strconv.Itoa(nearestXterm(r, g, b))
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

// crtSGR rewrites the colors of one SGR sequence (the part between "\x1b[" and "m").
func crtSGR(params string, dim int) string {
	key := params + "|" + strconv.Itoa(dim)
	if s, ok := crtCache[key]; ok {
		return s
	}
	base := "38;" + crtShade(crtDefaultFg, crtDefaultFg, crtDefaultFg, dim)
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
		case p[i] == "" || n == 0:
			out = append(out, "0", base)
		case n == 39:
			out = append(out, base)
		case (n == 38 || n == 48) && num(i+1) == 5:
			r, g, b := xtermRGB(num(i + 2))
			out = append(out, strconv.Itoa(n)+";"+crtShade(r, g, b, dim))
			i += 2
		case (n == 38 || n == 48) && num(i+1) == 2:
			out = append(out, strconv.Itoa(n)+";"+crtShade(num(i+2), num(i+3), num(i+4), dim))
			i += 4
		case n >= 30 && n <= 37, n >= 90 && n <= 97:
			r, g, b := xtermRGB(n%10 + 8*(n/90))
			out = append(out, "38;"+crtShade(r, g, b, dim))
		case n >= 40 && n <= 47, n >= 100 && n <= 107:
			r, g, b := xtermRGB(n%10 + 8*(n/100))
			out = append(out, "48;"+crtShade(r, g, b, dim))
		default:
			out = append(out, p[i])
		}
	}
	s := "\x1b[" + strings.Join(out, ";") + "m"
	crtCache[key] = s
	return s
}

// crt tints a rendered frame slightly green, dims every other line a little like scanlines and,
// on rare frames, dims everything a little as a flicker.
func crt(s string, frame int64) string {
	flick := 0
	if flicker(frame, crtFlickerSalt) < crtFlickerChance {
		flick = 1
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/2)
	for y, line := range strings.Split(s, "\n") {
		if y > 0 {
			b.WriteByte('\n')
		}
		dim := flick + y%2
		b.WriteString(crtSGR("", dim))
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
			b.WriteString(crtSGR(line[i+2:i+j], dim))
			line = line[i+j+1:]
		}
		b.WriteString("\x1b[0m")
	}
	return b.String()
}
