package main

import (
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
	windowWidth     = 250
	refreshInterval = 5 * time.Minute
	tickInterval    = 30 * time.Second // updates the "Reinicia em" countdown locally
	alphaNormal     = 215              // ~85% opaque
	alphaPassThru   = 150              // ~60% opaque while clicks pass through
)

type overlay struct {
	w       fyne.Window
	content fyne.CanvasObject

	plan                           *canvas.Text
	status                         *canvas.Text
	session, weekly, opus, sonnet *limitRow

	hwnd         uintptr
	refreshing   bool
	clickThrough bool
	lastUpdate   time.Time
}

func formatPct(v float64) string { return fmt.Sprintf("%.0f%%", v) }

func gap(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, h))
	return r
}

func newOverlay(w fyne.Window) *overlay {
	o := &overlay{
		w:       w,
		plan:    text("", 11, colSubtle, false),
		status:  text("Carregando…", 10, colSubtle, false),
		session: newLimitRow("Sessão atual"),
		weekly:  newLimitRow("Semanal · todos os modelos"),
		opus:    newLimitRow("Semanal · Opus"),
		sonnet:  newLimitRow("Semanal · Sonnet"),
	}
	o.opus.box.Hide()
	o.sonnet.box.Hide()

	refresh := newTapIcon(theme.ViewRefreshIcon(), o.refresh)
	header := container.NewHBox(text("Seus limites de uso", 13, colText, true), o.plan)
	footer := container.NewBorder(nil, nil, o.status, refresh)

	o.content = container.New(layout.NewCustomPaddedLayout(10, 10, 14, 14),
		container.NewVBox(
			header, gap(4),
			o.session.box, gap(4),
			o.weekly.box,
			o.opus.box,
			o.sonnet.box, gap(4),
			footer,
		))
	return o
}

// refresh fetches usage in the background. Must be called on the UI goroutine.
func (o *overlay) refresh() {
	if o.refreshing {
		return
	}
	o.refreshing = true
	o.setStatus("Atualizando…", colSubtle)
	go func() {
		u, plan, err := fetchUsage()
		fyne.Do(func() { o.show(u, plan, err) })
	}()
}

func (o *overlay) show(u *Usage, plan string, err error) {
	o.refreshing = false
	if p := planLabel(plan); p != o.plan.Text {
		o.plan.Text = p
		o.plan.Refresh()
	}
	if err != nil {
		msg := err.Error()
		if !o.lastUpdate.IsZero() {
			msg += " · " + o.lastUpdate.Format("15:04")
		}
		o.setStatus(msg, colError)
		o.fit()
		return
	}
	o.lastUpdate = time.Now()
	o.session.set(u.FiveHour)
	o.weekly.set(u.SevenDay)
	o.opus.set(u.SevenDayOpus)
	o.sonnet.set(u.SevenDaySonnet)
	o.setStatus("Atualizado às "+o.lastUpdate.Format("15:04"), colSubtle)
	o.fit()
}

func (o *overlay) setStatus(s string, c color.Color) {
	o.status.Text = s
	o.status.Color = c
	o.status.Refresh()
}

func (o *overlay) tick() {
	for _, r := range []*limitRow{o.session, o.weekly, o.opus, o.sonnet} {
		r.tick()
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
	if o.hwnd == 0 {
		if nw, ok := o.w.(driver.NativeWindow); ok {
			nw.RunNative(func(ctx any) {
				if wc, ok := ctx.(driver.WindowsWindowContext); ok {
					o.hwnd = wc.HWND
				}
			})
		}
	}
	alpha := byte(alphaNormal)
	if o.clickThrough {
		alpha = alphaPassThru
	}
	applyOverlay(o.hwnd, alpha, o.clickThrough)
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

// printUsage writes the current usage to stdout (handy for testing without the UI).
func printUsage() {
	u, plan, err := fetchUsage()
	if err != nil {
		fmt.Println("erro:", err)
		os.Exit(1)
	}
	fmt.Println("plano:", planLabel(plan))
	for _, l := range []struct {
		name string
		l    *Limit
	}{{"sessão", u.FiveHour}, {"semanal", u.SevenDay}, {"opus", u.SevenDayOpus}, {"sonnet", u.SevenDaySonnet}} {
		if l.l != nil {
			fmt.Printf("%-8s %5s  %s\n", l.name, formatPct(l.l.Utilization), resetText(l.l.ResetTime()))
		}
	}
}

func main() {
	printOnly := flag.Bool("print", false, "imprime o uso no terminal e sai")
	flag.Parse()
	if *printOnly {
		printUsage()
		return
	}

	a := app.NewWithID("com.github.claude-use")
	a.Settings().SetTheme(overlayTheme{theme.DefaultTheme()})

	var w fyne.Window
	if drv, ok := a.Driver().(desktop.Driver); ok {
		w = drv.CreateSplashWindow() // borderless
	} else {
		w = a.NewWindow("Uso do Claude")
	}
	w.SetTitle("Uso do Claude")

	o := newOverlay(w)
	w.SetContent(o.content)
	w.Resize(fyne.NewSize(windowWidth, o.content.MinSize().Height))

	if desk, ok := a.(desktop.App); ok {
		visible := true
		passThru := fyne.NewMenuItem("Ignorar cliques (atravessar)", nil)
		toggle := fyne.NewMenuItem("Ocultar", nil)
		menu := fyne.NewMenu("Uso do Claude",
			fyne.NewMenuItem("Atualizar agora", o.refresh),
			passThru,
			toggle,
		)
		passThru.Action = func() {
			o.clickThrough = !o.clickThrough
			passThru.Checked = o.clickThrough
			menu.Refresh()
			o.applyNative()
		}
		toggle.Action = func() {
			visible = !visible
			if visible {
				w.Show()
				o.fit()
				toggle.Label = "Ocultar"
			} else {
				w.Hide()
				toggle.Label = "Mostrar"
			}
			menu.Refresh()
		}
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(theme.HistoryIcon())
	}

	a.Lifecycle().SetOnStarted(func() {
		o.fit()
		o.refresh()
		go o.loop()
	})
	w.ShowAndRun()
}
