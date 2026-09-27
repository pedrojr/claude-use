//go:build !windows

package main

// applyOverlay is only implemented on Windows.
func applyOverlay(hwnd uintptr, alpha byte, clickThrough bool) {}

// acquireSingleInstance is only enforced on Windows.
func acquireSingleInstance() bool { return true }

// autostartSupported reports whether "start with Windows" can be toggled on this platform.
const autostartSupported = false

func autostartEnabled() bool     { return false }
func setAutostart(on bool) error { return nil }
