//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procCreateMutexW = kernel32.NewProc("CreateMutexW")

	procGetWindowLongPtrW          = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW          = user32.NewProc("SetWindowLongPtrW")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procSetWindowPos               = user32.NewProc("SetWindowPos")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procMonitorFromPoint           = user32.NewProc("MonitorFromPoint")
	procMonitorFromRect            = user32.NewProc("MonitorFromRect")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procSetWindowRgn               = user32.NewProc("SetWindowRgn")
	procGetDpiForWindow            = user32.NewProc("GetDpiForWindow")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
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

	swHide           = 0
	swShowNoActivate = 4

	swpNoSize       = 0x0001
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020

	monitorDefaultToPrimary = 1
	monitorDefaultToNearest = 2

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

type winPoint struct{ X, Y int32 }

// applyOverlay turns the window into a semi-transparent, always-on-top tool window
// (no taskbar button, never steals focus) and rounds its corners. It is placed at pos
// (kept inside the work area of the nearest monitor) or, when pos is nil, docked to the
// right edge of the primary monitor, vertically centred.
func applyOverlay(hwnd uintptr, alpha byte, clickThrough bool, pos *windowPos) {
	if hwnd == 0 {
		return
	}

	ex, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle))
	wasToolWindow := ex&wsExToolWindow != 0
	ex |= wsExLayered | wsExToolWindow | wsExNoActivate | wsExTopmost
	ex &^= wsExAppWindow
	if clickThrough {
		ex |= wsExTransparent
	} else {
		ex &^= wsExTransparent
	}
	procSetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle), ex)
	if !wasToolWindow {
		// The taskbar only re-reads the extended style when a window is shown, so
		// hide and re-show it once to drop the button created before the style change.
		if v, _, _ := procIsWindowVisible.Call(hwnd); v != 0 {
			procShowWindow.Call(hwnd, swHide)
			procShowWindow.Call(hwnd, swShowNoActivate)
		}
	}
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

	var x, y int32
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	if pos != nil {
		want := winRect{pos.X, pos.Y, pos.X + w, pos.Y + h}
		mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&want)), monitorDefaultToNearest)
		procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		x = clamp(pos.X, mi.Work.Left, mi.Work.Right-w)
		y = clamp(pos.Y, mi.Work.Top, mi.Work.Bottom-h)
	} else {
		mon, _, _ := procMonitorFromPoint.Call(0, monitorDefaultToPrimary)
		procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		x = mi.Work.Right - w - scale(marginDIP)
		y = mi.Work.Top + (mi.Work.Bottom-mi.Work.Top-h)/2
	}

	procSetWindowPos.Call(hwnd, uintptr(hwndTopmost), uintptr(x), uintptr(y), 0, 0,
		swpNoSize|swpNoActivate|swpFrameChanged)
}

// clamp keeps v in [lo, hi]; lo wins when the window is larger than the work area.
func clamp(v, lo, hi int32) int32 {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

// cursorPos returns the mouse position in screen pixels.
func cursorPos() (int32, int32) {
	var p winPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p.X, p.Y
}

// windowPosition returns the top-left corner of the window in screen pixels.
func windowPosition(hwnd uintptr) (int32, int32, bool) {
	if hwnd == 0 {
		return 0, 0, false
	}
	var r winRect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return 0, 0, false
	}
	return r.Left, r.Top, true
}

// moveWindow moves the window without resizing, activating or changing its z-order.
func moveWindow(hwnd uintptr, x, y int32) {
	if hwnd == 0 {
		return
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
}

// singleInstanceMutex stays open for the whole process lifetime; Windows releases it on exit.
var singleInstanceMutex uintptr

// acquireSingleInstance reports whether this is the only running instance of the overlay.
func acquireSingleInstance() bool {
	name, _ := syscall.UTF16PtrFromString(`Local\com.github.claude-use`)
	h, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return true // could not create the mutex; do not block startup
	}
	if err == syscall.ERROR_ALREADY_EXISTS {
		return false
	}
	singleInstanceMutex = h
	return true
}

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "claude-use"
)

// autostartSupported reports whether "start with Windows" can be toggled on this platform.
const autostartSupported = true

// autostartEnabled reports whether the overlay is registered to start with Windows.
func autostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValueName)
	return err == nil
}

// setAutostart registers (or removes) the current executable in the user's Run key.
func setAutostart(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		if err := k.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(runValueName, `"`+exe+`"`)
}
