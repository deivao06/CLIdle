package ui

import (
	"errors"
	"syscall/js"
)

const canExport = true

// exportSave downloads the save kept in localStorage as a JSON file.
func exportSave(path, name string) error {
	data := js.Global().Get("localStorage").Call("getItem", path)
	if data.IsNull() {
		return errors.New("no save")
	}
	blob := js.Global().Get("Blob").New([]any{data}, map[string]any{"type": "application/json"})
	a := js.Global().Get("document").Call("createElement", "a")
	a.Set("href", js.Global().Get("URL").Call("createObjectURL", blob))
	a.Set("download", name)
	a.Call("click")
	return nil
}
