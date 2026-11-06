package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const saveVersion = 6

type saveFile struct {
	Version int
	SavedAt time.Time
	Game    *Game
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
	data, err := json.Marshal(saveFile{saveVersion, now, g})
	if err != nil {
		return err
	}
	return writeSave(path, data)
}

// Load returns a nil game when there is no save yet.
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
	return f.Game, f.SavedAt, nil
}
