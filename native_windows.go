//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procGetWindowLongPtrW          = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW          = user32.NewProc("SetWindowLongPtrW")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procSetWindowPos               = user32.NewProc("SetWindowPos")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procMonitorFromPoint           = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procSetWindowRgn               = user32.NewProc("SetWindowRgn")
	procGetDpiForWindow            = user32.NewProc("GetDpiForWindow")
	procCreateRoundRectRgn         = gdi32.NewProc("CreateRoundRectRgn")
)

const (
	wsExTopmost     = 0x00000008
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExAppWindow   = 0x00040000
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000

	lwaAlpha = 0x2

	swpNoSize       = 0x0001
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020

	monitorDefaultToPrimary = 1

	cornerRadiusDIP = 10 // rounded corner radius, in device-independent pixels
	marginDIP       = 12 // gap to the right edge of the screen
)

var (
	gwlExStyle  int32 = -20
	hwndTopmost int32 = -1
)

type winRect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	CbSize        uint32
	Monitor, Work winRect
	Flags         uint32
}

// applyOverlay turns the window into a semi-transparent, always-on-top tool window
// (no taskbar button, never steals focus), rounds its corners and docks it to the
// right edge of the primary monitor, vertically centred.
func applyOverlay(hwnd uintptr, alpha byte, clickThrough bool) {
	if hwnd == 0 {
		return
	}

	ex, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle))
	ex |= wsExLayered | wsExToolWindow | wsExNoActivate | wsExTopmost
	ex &^= wsExAppWindow
	if clickThrough {
		ex |= wsExTransparent
	} else {
		ex &^= wsExTransparent
	}
	procSetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle), ex)
	procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(alpha), lwaAlpha)

	var r winRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := r.Right-r.Left, r.Bottom-r.Top

	dpi := int32(96)
	if procGetDpiForWindow.Find() == nil {
		if d, _, _ := procGetDpiForWindow.Call(hwnd); d != 0 {
			dpi = int32(d)
		}
	}
	scale := func(v int32) int32 { return v * dpi / 96 }

	radius := scale(cornerRadiusDIP) * 2
	rgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(w+1), uintptr(h+1), uintptr(radius), uintptr(radius))
	if rgn != 0 {
		procSetWindowRgn.Call(hwnd, rgn, 1) // the system owns rgn after this call
	}

	mon, _, _ := procMonitorFromPoint.Call(0, monitorDefaultToPrimary)
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	x := mi.Work.Right - w - scale(marginDIP)
	y := mi.Work.Top + (mi.Work.Bottom-mi.Work.Top-h)/2

	procSetWindowPos.Call(hwnd, uintptr(hwndTopmost), uintptr(x), uintptr(y), 0, 0,
		swpNoSize|swpNoActivate|swpFrameChanged)
}
