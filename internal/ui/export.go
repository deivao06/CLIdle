//go:build !js

package ui

const canExport = false

func exportSave(string, string) error { return nil }
