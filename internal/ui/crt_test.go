package ui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
)

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

func TestCRTKeepsTextAndLayout(t *testing.T) {
	in := "\x1b[1;38;5;208mfire\x1b[0m plain │ text\n\x1b[93m@\x1b[0m and \x1b[48;5;214mtab\x1b[0m ───"
	untear := strings.NewReplacer("╎", "│", "╌", "─")
	for frame := range int64(500) {
		out := crt(in, frame)
		if untear.Replace(sgr.ReplaceAllString(out, "")) != sgr.ReplaceAllString(in, "") {
			t.Fatalf("frame %d changed the text: %q", frame, out)
		}
	}
	if plain := "no colors here"; crt(plain, 0) != plain {
		t.Fatal("text in the terminal's default color should be left alone")
	}
}

func TestCRTKeepsHue(t *testing.T) {
	out := crtSGR("38;2;220;30;30", 1, true)
	var n int
	if _, err := fmt.Sscanf(out, "\x1b[38;5;%dm", &n); err != nil {
		t.Fatalf("unexpected %q", out)
	}
	if r, g, b := xtermRGB(n); r <= g || r <= b {
		t.Fatalf("red became %d,%d,%d", r, g, b)
	}
}

func TestCRTSettingTogglesAndPersists(t *testing.T) {
	g := game.New(1, game.Warrior)
	m := New(g)
	m.tab = tabSettings
	for i, r := range m.settingsRows() {
		if strings.HasPrefix(r.text, lang.T("prep.crt")) {
			m.cursor = i
		}
	}
	m.prepKey("enter")
	if !g.NoCRT {
		t.Fatal("enter did not turn CRT off")
	}
	path := filepath.Join(t.TempDir(), "save.json")
	if err := g.Save(path, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, _, err := game.Load(path)
	if err != nil || !got.NoCRT {
		t.Fatalf("CRT off not saved: %v", err)
	}
	if New(game.New(1, game.Warrior)).crtOff {
		t.Fatal("CRT should be on by default")
	}
}
