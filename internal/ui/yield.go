//go:build !js

package ui

import "time"

const wakeGap = 5 * time.Second

func yieldToBrowser() {}
