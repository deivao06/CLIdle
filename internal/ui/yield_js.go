package ui

import "time"

// yieldToBrowser blocks briefly so the browser can paint; wasm goroutines never return to it otherwise.
func yieldToBrowser() { time.Sleep(time.Millisecond) }
