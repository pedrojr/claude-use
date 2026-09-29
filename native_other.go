//go:build !windows

package main

// applyOverlay is only implemented on Windows.
func applyOverlay(hwnd uintptr, alpha byte, clickThrough bool, pos *windowPos) {}

// Window dragging is only implemented on Windows.
func cursorPos() (int32, int32)                        { return 0, 0 }
func windowPosition(hwnd uintptr) (int32, int32, bool) { return 0, 0, false }
func moveWindow(hwnd uintptr, x, y int32)              {}

// acquireSingleInstance is only enforced on Windows.
func acquireSingleInstance() bool { return true }

// autostartSupported reports whether "start with Windows" can be toggled on this platform.
const autostartSupported = false

func autostartEnabled() bool     { return false }
func setAutostart(on bool) error { return nil }
