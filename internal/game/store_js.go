package game

import (
	"fmt"
	"os"
	"syscall/js"
)

func storage(method string, args ...any) (v js.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("localStorage: %v", r)
		}
	}()
	return js.Global().Get("localStorage").Call(method, args...), nil
}

func writeSave(path string, data []byte) error {
	_, err := storage("setItem", path, string(data))
	return err
}

func readSave(path string) ([]byte, error) {
	v, err := storage("getItem", path)
	if err != nil {
		return nil, err
	}
	if v.IsNull() {
		return nil, os.ErrNotExist
	}
	return []byte(v.String()), nil
}

func RemoveSave(path string) error {
	_, err := storage("removeItem", path)
	return err
}
