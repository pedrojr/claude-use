package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	colBackground = color.NRGBA{0x1c, 0x1c, 0x1c, 0xff}
	colText       = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	colBarTrack   = color.NRGBA{0x12, 0x2a, 0x4a, 0xff}
	colBarBlue    = color.NRGBA{0x2f, 0x80, 0xe8, 0xff}
	colBarOrange  = color.NRGBA{0xe8, 0x9a, 0x2f, 0xff}
	colBarRed     = color.NRGBA{0xe0, 0x4f, 0x4f, 0xff}
)

func barColor(pct float64) color.Color {
	switch {
	case pct >= 90:
		return colBarRed
	case pct >= 75:
		return colBarOrange
	}
	return colBarBlue
}

// overlayTheme forces the dark variant with a compact text size and white foreground.
type overlayTheme struct{ fyne.Theme }

func (t overlayTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return colBackground
	case theme.ColorNameForeground:
		return colText
	}
	return t.Theme.Color(n, theme.VariantDark)
}

func (t overlayTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 3
	}
	return t.Theme.Size(n)
}

// usageBar is a thin rounded progress bar like the one in Claude's usage screen.
type usageBar struct {
	widget.BaseWidget
	value      float64 // 0..1
	track, bar *canvas.Rectangle
}

func newUsageBar() *usageBar {
	b := &usageBar{
		track: canvas.NewRectangle(colBarTrack),
		bar:   canvas.NewRectangle(colBarBlue),
	}
	b.track.CornerRadius = 3
	b.bar.CornerRadius = 3
	b.ExtendBaseWidget(b)
	return b
}

func (b *usageBar) set(pct float64) {
	v := pct / 100
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	b.value = v
	b.bar.FillColor = barColor(pct)
	b.Refresh()
}

func (b *usageBar) CreateRenderer() fyne.WidgetRenderer { return &usageBarRenderer{b} }

type usageBarRenderer struct{ b *usageBar }

func (r *usageBarRenderer) Layout(size fyne.Size) {
	r.b.track.Resize(size)
	w := size.Width * float32(r.b.value)
	if w < size.Height { // always draw at least a dot, like the original
		w = size.Height
	}
	r.b.bar.Resize(fyne.NewSize(w, size.Height))
}

func (r *usageBarRenderer) MinSize() fyne.Size { return fyne.NewSize(60, 7) }
func (r *usageBarRenderer) Refresh() {
	r.Layout(r.b.Size())
	r.b.track.Refresh()
	r.b.bar.Refresh()
}
func (r *usageBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.b.track, r.b.bar}
}
func (r *usageBarRenderer) Destroy() {}

// tapIcon is a small, discreet clickable icon (used for refresh).
type tapIcon struct {
	widget.Icon
	onTap func()
}

func newTapIcon(res fyne.Resource, onTap func()) *tapIcon {
	t := &tapIcon{onTap: onTap}
	t.SetResource(res)
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapIcon) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}
func (t *tapIcon) Cursor() desktop.Cursor { return desktop.PointerCursor }
func (t *tapIcon) MinSize() fyne.Size     { return fyne.NewSize(16, 16) }

func text(s string, size float32, c color.Color, bold bool) *canvas.Text {
	t := canvas.NewText(s, c)
	t.TextSize = size
	t.TextStyle.Bold = bold
	return t
}

// dragHandle is an invisible layer behind the content; dragging it moves the window.
type dragHandle struct {
	widget.BaseWidget
	onPress, onDrag, onEnd func()
}

func newDragHandle(onPress, onDrag, onEnd func()) *dragHandle {
	d := &dragHandle{onPress: onPress, onDrag: onDrag, onEnd: onEnd}
	d.ExtendBaseWidget(d)
	return d
}

func (d *dragHandle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}
func (d *dragHandle) MouseDown(*desktop.MouseEvent) { d.onPress() }
func (d *dragHandle) MouseUp(*desktop.MouseEvent)   {}
func (d *dragHandle) Dragged(*fyne.DragEvent)       { d.onDrag() }
func (d *dragHandle) DragEnd()                      { d.onEnd() }

// limitRow shows: title .... NN% / bar / "Resets in ...".
type limitRow struct {
	box                *fyne.Container
	titleKey           string
	title, pct, resets *canvas.Text
	bar                *usageBar
	limit              *Limit
}

func newLimitRow(titleKey string) *limitRow {
	r := &limitRow{
		titleKey: titleKey,
		title:    text(T(titleKey), 14, colText, true),
		pct:      text("—", 13, colText, false),
		resets:   text("", 12, colText, false),
		bar:      newUsageBar(),
	}
	r.box = container.NewVBox(
		container.NewBorder(nil, nil, r.title, r.pct),
		r.bar,
		r.resets,
	)
	return r
}

func (r *limitRow) set(l *Limit) {
	r.limit = l
	if l == nil {
		r.box.Hide()
		return
	}
	r.box.Show()
	r.pct.Text = T("used", formatPct(l.Utilization))
	r.pct.Refresh()
	r.bar.set(l.Utilization)
	r.tick()
}

// retranslate re-renders the texts after a language change.
func (r *limitRow) retranslate() {
	r.title.Text = T(r.titleKey)
	r.title.Refresh()
	if r.limit != nil {
		r.set(r.limit)
	}
}

// tick re-renders the countdown without calling the API.
func (r *limitRow) tick() {
	if r.limit == nil {
		return
	}
	r.resets.Text = resetText(r.limit.ResetTime())
	r.resets.Refresh()
}
