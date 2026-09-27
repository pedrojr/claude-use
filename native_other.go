//go:build !windows

package main

// applyOverlay is only implemented on Windows.
func applyOverlay(hwnd uintptr, alpha byte, clickThrough bool) {}
