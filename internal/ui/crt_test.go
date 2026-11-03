package ui

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
)

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

func TestCRTKeepsText(t *testing.T) {
	in := "\x1b[1;38;5;208mfire\x1b[0m plain    text\n\x1b[93m@\x1b[0m and \x1b[48;5;214mtab\x1b[0m"
	plain := func(s string) string {
		var lines []string
		for _, l := range strings.Split(sgr.ReplaceAllString(s, ""), "\n") {
			lines = append(lines, strings.TrimLeft(strings.ReplaceAll(l, "·", " "), " "))
		}
		return strings.Join(lines, "\n")
	}
	for frame := range int64(500) {
		out := crt(in, frame)
		if plain(out) != plain(in) {
			t.Fatalf("frame %d changed the text: %q", frame, out)
		}
		if !strings.Contains(out, "\x1b[0;38;") {
			t.Fatalf("reset does not restore the CRT tint: %q", out)
		}
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
