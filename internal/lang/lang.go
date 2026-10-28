// Package lang holds the game texts, one JSON file per language.
package lang

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed *.json
var files embed.FS

const Default = "en"

// Codes lists the available languages in the order the player cycles through them.
var Codes = []string{"en", "pt"}

var (
	tables = map[string]map[string]string{}
	cur    = Default
)

func init() {
	for _, code := range Codes {
		data, err := files.ReadFile(code + ".json")
		if err != nil {
			panic(err)
		}
		t := map[string]string{}
		if err := json.Unmarshal(data, &t); err != nil {
			panic(fmt.Sprintf("lang %s: %v", code, err))
		}
		tables[code] = t
	}
}

// Set switches the current language; unknown codes fall back to the default.
func Set(code string) {
	if _, ok := tables[code]; !ok {
		code = Default
	}
	cur = code
}

func Current() string { return cur }

// Next returns the language after code in Codes.
func Next(code string) string {
	for i, c := range Codes {
		if c == code {
			return Codes[(i+1)%len(Codes)]
		}
	}
	return Default
}

// T translates key into the current language, formatting it with args.
func T(key string, args ...any) string {
	s := In(cur, key)
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// In returns the raw text of key in a given language, falling back to the default language and then to the key.
func In(code, key string) string {
	if s, ok := tables[code][key]; ok {
		return s
	}
	if s, ok := tables[Default][key]; ok {
		return s
	}
	return key
}

// Keys returns the keys defined for a language.
func Keys(code string) []string {
	var keys []string
	for k := range tables[code] {
		keys = append(keys, k)
	}
	return keys
}
