package ui

import "time"

// wakeGap is long in the browser because background tabs only get timer callbacks about once a minute.
const wakeGap = 2 * time.Minute

// yieldToBrowser blocks briefly so the browser can paint; wasm goroutines never return to it otherwise.
func yieldToBrowser() { time.Sleep(time.Millisecond) }
