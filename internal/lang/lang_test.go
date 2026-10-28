package lang

import (
	"slices"
	"strings"
	"testing"
)

func TestLanguagesHaveTheSameKeys(t *testing.T) {
	want := Keys(Default)
	slices.Sort(want)
	for _, code := range Codes {
		got := Keys(code)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Fatalf("%s keys differ from %s", code, Default)
		}
	}
}

func TestLanguagesHaveTheSameVerbs(t *testing.T) {
	for _, k := range Keys(Default) {
		for _, code := range Codes {
			if a, b := strings.Count(In(Default, k), "%"), strings.Count(In(code, k), "%"); a != b {
				t.Errorf("%s: %s has %d verbs, %s has %d", k, Default, a, code, b)
			}
		}
	}
}
