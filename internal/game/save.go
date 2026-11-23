package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const saveVersion = 12

type saveFile struct {
	Version int
	SavedAt time.Time
	Game    *Game  `json:",omitempty"`
	Guild   *Guild `json:",omitempty"`
}

func SavePath() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "clidle", "save.json")
}

func (g *Game) Save(path string, now time.Time) error {
	data, err := json.Marshal(saveFile{Version: saveVersion, SavedAt: now, Guild: g.Guild})
	if err != nil {
		return err
	}
	return writeSave(path, data)
}

// Load returns the guild's first hero, or nil when there is no save yet.
func Load(path string) (*Game, time.Time, error) {
	data, err := readSave(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	var f saveFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, time.Time{}, err
	}
	if f.Version < saveVersion {
		var old struct{ Game legacy }
		if err := json.Unmarshal(data, &old); err != nil {
			return nil, time.Time{}, err
		}
		migrate(&f, &old.Game)
	}
	if f.Guild == nil || len(f.Guild.Heroes) == 0 {
		return nil, time.Time{}, errors.New("save without heroes")
	}
	f.Guild.link()
	return f.Guild.Heroes[0], f.SavedAt, nil
}
