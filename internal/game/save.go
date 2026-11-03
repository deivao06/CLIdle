package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const saveVersion = 4

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

// Save writes atomically so a crash mid-write never corrupts the previous save.
func (g *Game) Save(path string, now time.Time) error {
	data, err := json.Marshal(saveFile{saveVersion, now, g})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load returns a nil game when there is no save yet.
func Load(path string) (*Game, time.Time, error) {
	data, err := os.ReadFile(path)
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
		var old struct{ Game queueV3 }
		if err := json.Unmarshal(data, &old); err != nil {
			return nil, time.Time{}, err
		}
		migrate(&f, &old.Game)
	}
	return f.Game, f.SavedAt, nil
}
