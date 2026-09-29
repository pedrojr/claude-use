package main

import (
	_ "embed"
	"flag"
	"fmt"
	"image/color"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

const (
	windowWidth     = 290
	refreshInterval = 5 * time.Minute
	retryInterval   = time.Minute      // sooner retry when offline (e.g. right after waking from sleep)
	tickInterval    = 30 * time.Second // updates the "Resets in" countdown locally
	overlayAlpha    = 215              // ~85% opaque, same with or without click-through
)

// trayIconPNG is the system tray icon: a full-colour image with a transparent background,
// so it reads well on both the light and the dark Windows taskbar.
//
//go:embed assets/tray.png
var trayIconPNG []byte

type overlay struct {
	w       fyne.Window
	content fyne.CanvasObject
	cfg     config

	header, plan, status          *canvas.Text
	session, weekly, opus, sonnet *limitRow

	hwnd           uintptr
	refreshing     bool
	clickThrough   bool
	dragging       bool
	dragDX, dragDY int32 // cursor offset from the window's top-left corner, taken on mouse down
	planCode       string
	lastErr        error
	lastUpdate     time.Time
	retry          *time.Timer
}

func formatPct(v float64) string { return fmt.Sprintf("%.0f%%", v) }

func gap(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, h))
	return r
}

func newOverlay(w fyne.Window, cfg config) *overlay {
	o := &overlay{
		w:       w,
		cfg:     cfg,
		header:  text(T("header"), 15, colText, true),
		plan:    text("", 13, colText, false),
		status:  text(T("loading"), 12, colText, false),
		session: newLimitRow("session"),
		weekly:  newLimitRow("weekly"),
		opus:    newLimitRow("weeklyOpus"),
		sonnet:  newLimitRow("weeklySonnet"),
	}
	o.opus.box.Hide()
	o.sonnet.box.Hide()

	refresh := newTapIcon(theme.ViewRefreshIcon(), o.refresh)
	header := container.NewHBox(o.header, o.plan)
	footer := container.NewBorder(nil, nil, o.status, refresh)

	body := container.New(layout.NewCustomPaddedLayout(12, 12, 16, 16),
		container.NewVBox(
			header, gap(4),
			o.session.box, gap(4),
			o.weekly.box,
			o.opus.box,
			o.sonnet.box, gap(4),
			footer,
		))
	o.content = container.NewStack(newDragHandle(o.pressed, o.dragged, o.dragEnd), body)
	return o
}

// refresh fetches usage in the background. Must be called on the UI goroutine.
func (o *overlay) refresh() {
	if o.refreshing {
		return
	}
	o.refreshing = true
	o.renderStatus()
	go func() {
		u, plan, err := fetchUsage()
		fyne.Do(func() { o.show(u, plan, err) })
	}()
}

func (o *overlay) show(u *Usage, plan string, err error) {
	o.refreshing = false
	o.planCode = plan
	o.renderPlan()
	o.lastErr = err
	if err == nil {
		o.lastUpdate = time.Now()
		o.session.set(u.FiveHour)
		o.weekly.set(u.SevenDay)
		o.opus.set(u.SevenDayOpus)
		o.sonnet.set(u.SevenDaySonnet)
	}
	if o.retry != nil {
		o.retry.Stop()
		o.retry = nil
	}
	if ue, ok := err.(usageError); ok && ue.key == "errOffline" {
		o.retry = time.AfterFunc(retryInterval, func() { fyne.Do(o.refresh) })
	}
	o.renderStatus()
	o.fit()
}

func (o *overlay) renderPlan() {
	if p := planLabel(o.planCode); p != o.plan.Text {
		o.plan.Text = p
		o.plan.Refresh()
	}
}

// renderStatus shows the footer text for the current state in the current language.
func (o *overlay) renderStatus() {
	var s string
	switch {
	case o.refreshing:
		s = T("refreshing")
	case o.lastErr != nil:
		s = o.lastErr.Error()
		if !o.lastUpdate.IsZero() {
			s += " · " + o.lastUpdate.Format("15:04")
		}
	case !o.lastUpdate.IsZero():
		s = T("updatedAt", o.lastUpdate.Format("15:04"))
	default:
		s = T("loading")
	}
	o.status.Text = s
	o.status.Refresh()
}

// retranslate re-renders every visible text after a language change.
func (o *overlay) retranslate() {
	o.w.SetTitle(T("windowTitle"))
	o.header.Text = T("header")
	o.header.Refresh()
	o.renderPlan()
	for _, r := range []*limitRow{o.session, o.weekly, o.opus, o.sonnet} {
		r.retranslate()
	}
	o.renderStatus()
	o.fit()
}

func (o *overlay) tick() {
	resetPassed := false
	for _, r := range []*limitRow{o.session, o.weekly, o.opus, o.sonnet} {
		r.tick()
		// A limit whose reset time passed after the last update is stale: fetch the new
		// values now instead of showing "Resetting…" until the next scheduled refresh.
		// Comparing with lastUpdate refreshes once per reset, even if the API lags.
		if t := r.limit.ResetTime(); !t.IsZero() && t.After(o.lastUpdate) && time.Now().After(t) {
			resetPassed = true
		}
	}
	if resetPassed && o.lastErr == nil {
		o.refresh()
	}
}

// fit sizes the window to its content, then re-applies the native overlay styling
// (region and position depend on the final pixel size).
func (o *overlay) fit() {
	o.w.Resize(fyne.NewSize(windowWidth, o.content.MinSize().Height))
	o.applyNative()
	time.AfterFunc(250*time.Millisecond, func() { fyne.Do(o.applyNative) })
}

func (o *overlay) applyNative() {
	if o.dragging {
		return // dragEnd re-applies it at the new position
	}
	if o.hwnd == 0 {
		if nw, ok := o.w.(driver.NativeWindow); ok {
			nw.RunNative(func(ctx any) {
				if wc, ok := ctx.(driver.WindowsWindowContext); ok {
					o.hwnd = wc.HWND
				}
			})
		}
	}
	applyOverlay(o.hwnd, overlayAlpha, o.clickThrough, o.cfg.Position)
}

// pressed remembers where the window was grabbed, so it does not slip when the drag
// starts (Fyne only reports a drag after the mouse has already moved a few pixels).
func (o *overlay) pressed() {
	cx, cy := cursorPos()
	if wx, wy, ok := windowPosition(o.hwnd); ok {
		o.dragDX, o.dragDY = cx-wx, cy-wy
	}
}

// dragged moves the window with the mouse. Screen coordinates are used (not the
// event's window-relative ones) because the window moves under the cursor.
func (o *overlay) dragged() {
	if o.hwnd == 0 {
		return
	}
	o.dragging = true
	cx, cy := cursorPos()
	moveWindow(o.hwnd, cx-o.dragDX, cy-o.dragDY)
}

// dragEnd keeps the window inside the screen and remembers the new position.
func (o *overlay) dragEnd() {
	if !o.dragging {
		return
	}
	o.dragging = false
	x, y, ok := windowPosition(o.hwnd)
	if !ok {
		return
	}
	o.cfg.Position = &windowPos{X: x, Y: y}
	o.applyNative()
	if x, y, ok := windowPosition(o.hwnd); ok {
		o.cfg.Position = &windowPos{X: x, Y: y}
	}
	o.saveConfig()
}

func (o *overlay) resetPosition() {
	o.cfg.Position = nil
	o.saveConfig()
	o.applyNative()
}

func (o *overlay) saveConfig() {
	if err := o.cfg.save(); err != nil {
		fyne.LogError("Unable to save settings", err)
	}
}

func (o *overlay) loop() {
	refresh := time.NewTicker(refreshInterval)
	tick := time.NewTicker(tickInterval)
	for {
		select {
		case <-refresh.C:
			fyne.Do(o.refresh)
		case <-tick.C:
			fyne.Do(o.tick)
		}
	}
}

// setupTray builds the system tray icon and its menu.
func (o *overlay) setupTray(a fyne.App, desk desktop.App) {
	visible := true
	refresh := fyne.NewMenuItem("", o.refresh)
	passThru := fyne.NewMenuItem("", nil)
	toggle := fyne.NewMenuItem("", nil)
	resetPos := fyne.NewMenuItem("", o.resetPosition)
	autostart := fyne.NewMenuItem("", nil)
	autostart.Checked = autostartEnabled()
	language := fyne.NewMenuItem("", nil)
	quit := fyne.NewMenuItem("", a.Quit)
	quit.IsQuit = true

	items := []*fyne.MenuItem{refresh, passThru, toggle, resetPos}
	if autostartSupported {
		items = append(items, autostart)
	}
	items = append(items, language, fyne.NewMenuItemSeparator(), quit)
	menu := fyne.NewMenu("", items...)

	langItems := make([]*fyne.MenuItem, len(languages))
	relabel := func() {
		menu.Label = T("windowTitle")
		refresh.Label = T("menuRefresh")
		passThru.Label = T("menuPassThru")
		if visible {
			toggle.Label = T("menuHide")
		} else {
			toggle.Label = T("menuShow")
		}
		resetPos.Label = T("menuResetPos")
		autostart.Label = T("menuAutostart")
		language.Label = T("menuLanguage")
		quit.Label = T("menuQuit")
		for i, l := range languages {
			langItems[i].Checked = l.code == languageCode()
		}
	}
	for i, l := range languages {
		code := l.code
		langItems[i] = fyne.NewMenuItem(l.name, func() {
			if code == languageCode() {
				return
			}
			setLanguage(code)
			o.cfg.Language = code
			o.saveConfig()
			o.retranslate()
			relabel()
			menu.Refresh()
		})
	}
	language.ChildMenu = fyne.NewMenu("", langItems...)

	autostart.Action = func() {
		if err := setAutostart(!autostart.Checked); err != nil {
			o.lastErr = usageError{key: "errAutostart"}
			o.renderStatus()
			return
		}
		autostart.Checked = autostartEnabled()
		menu.Refresh()
	}
	passThru.Action = func() {
		o.clickThrough = !o.clickThrough
		passThru.Checked = o.clickThrough
		menu.Refresh()
		o.applyNative()
	}
	toggle.Action = func() {
		visible = !visible
		if visible {
			o.w.Show()
			o.fit()
		} else {
			o.w.Hide()
		}
		relabel()
		menu.Refresh()
	}

	relabel()
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(fyne.NewStaticResource("tray.png", trayIconPNG))
}

// printUsage writes the current usage to stdout (handy for testing without the UI).
func printUsage() {
	u, plan, err := fetchUsage()
	if err != nil {
		fmt.Println(T("cliError"), err)
		os.Exit(1)
	}
	fmt.Println(T("cliPlan"), planLabel(plan))
	for _, l := range []struct {
		name string
		l    *Limit
	}{{T("cliSession"), u.FiveHour}, {T("cliWeekly"), u.SevenDay}, {"opus", u.SevenDayOpus}, {"sonnet", u.SevenDaySonnet}} {
		if l.l != nil {
			fmt.Printf("%-8s %5s  %s\n", l.name, formatPct(l.l.Utilization), resetText(l.l.ResetTime()))
		}
	}
}

func main() {
	printOnly := flag.Bool("print", false, "print the usage to the terminal and exit")
	autostartFlag := flag.String("autostart", "", `"on" or "off": change start with Windows for the current user and exit (used by the installer)`)
	flag.Parse()

	cfg := loadConfig()
	setLanguage(cfg.Language)

	switch *autostartFlag {
	case "":
	case "on", "off":
		if err := setAutostart(*autostartFlag == "on"); err != nil {
			os.Exit(1)
		}
		return
	default:
		fmt.Fprintln(os.Stderr, `-autostart must be "on" or "off"`)
		os.Exit(2)
	}
	if *printOnly {
		printUsage()
		return
	}
	if !acquireSingleInstance() {
		return // another overlay is already running
	}

	a := app.NewWithID("com.github.claude-use")
	a.Settings().SetTheme(overlayTheme{theme.DefaultTheme()})
	a.SetIcon(fyne.NewStaticResource("tray.png", trayIconPNG))

	var w fyne.Window
	if drv, ok := a.Driver().(desktop.Driver); ok {
		w = drv.CreateSplashWindow() // borderless
	} else {
		w = a.NewWindow(T("windowTitle"))
	}
	w.SetTitle(T("windowTitle"))

	o := newOverlay(w, cfg)
	w.SetContent(o.content)
	w.Resize(fyne.NewSize(windowWidth, o.content.MinSize().Height))

	if desk, ok := a.(desktop.App); ok {
		o.setupTray(a, desk)
	}

	a.Lifecycle().SetOnStarted(func() {
		o.fit()
		o.refresh()
		go o.loop()
	})
	w.ShowAndRun()
}
