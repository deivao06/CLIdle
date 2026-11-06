//go:build !js

package game

import (
	"os"
	"path/filepath"
)

// writeSave writes atomically so a crash mid-write never corrupts the previous save.
func writeSave(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readSave(path string) ([]byte, error) { return os.ReadFile(path) }

func RemoveSave(path string) error { return os.Remove(path) }
