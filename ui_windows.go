// Tray icon + panel: click the icon for a Windows 11-style flyout showing whether PhoneDrop can
// receive on this network and which phones are paired; a settings page holds the save folder,
// clipboard and start-at-sign-in options. Drawn with GDI+ into a per-pixel-alpha layered window:
// no browser, no framework. Nothing is drawn and no timer runs while the panel is hidden.
package main

import (
	"fmt"
	"math"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmTray    = 0x8001 // WM_APP+1
	wmRefresh = 0x8002

	tAnim, tFlash = 1, 2

	// Layout, in 96-DPI pixels.
	sh      = 24  // transparent margin around the panel for its shadow
	panelW  = 360 // panel width
	pad     = 12
	gap     = 8
	footerH = 48
	radius  = 8
	statusH = 72
	labelH  = 32
	rowH    = 64
)

type hit struct {
	r  rectF
	id string
}

type ui struct {
	hwnd  uintptr
	th    theme
	set   settings
	scale float32

	cv             *canvas
	visible        bool
	view           string // "main" | "settings"
	hits           []hit
	hover, press   string
	tracking       bool
	hiddenAt       time.Time
	animStart      time.Time
	hiding         bool
	alpha          float32
	x, y, edge     int // panel window position (px) and the taskbar edge it slides from
	trayIcon       uintptr
	taskbarCreated uint32
	quitAt         time.Time // first click on the footer's quit button (second within 3 s quits)
	startAtLogin   bool
	recheck        chan struct{}

	mu         sync.Mutex // guards the fields below (written by the watcher and server goroutines)
	net        netState
	netKnown   bool
	listening  []netip.Addr
	listenErr  string
	flash      string
	flashUntil time.Time
}

var theUI *ui

// ---------- state from other goroutines ----------

func (u *ui) setNetwork(st netState, listening []netip.Addr) {
	u.mu.Lock()
	u.net, u.netKnown, u.listening = st, true, listening
	if len(listening) > 0 || len(st.listen) == 0 {
		u.listenErr = ""
	}
	u.mu.Unlock()
	if u.hwnd != 0 {
		postMsg.Call(u.hwnd, wmRefresh, 0, 0)
	}
}

func (u *ui) setListenErr(err error) {
	u.mu.Lock()
	u.listenErr = err.Error()
	u.mu.Unlock()
	if u.hwnd != 0 {
		postMsg.Call(u.hwnd, wmRefresh, 0, 0)
	}
}

type snap struct {
	net       netState
	known     bool
	listening []netip.Addr
	listenErr string
	flash     string
}

func (u *ui) snapshot() snap {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := snap{net: u.net, known: u.netKnown, listening: u.listening, listenErr: u.listenErr}
	if time.Now().Before(u.flashUntil) {
		s.flash = u.flash
	}
	return s
}

func (u *ui) flashMsg(text string) {
	u.mu.Lock()
	u.flash, u.flashUntil = text, time.Now().Add(4*time.Second)
	u.mu.Unlock()
	win(user32, "SetTimer", u.hwnd, tFlash, 4100, 0) // repaint when it lapses
}

// status: what the status card says, from the network state.
type status struct {
	icon        [2]rune
	title, sub  string
	ready, warn bool
	settings    bool // clicking opens Windows' network settings
}

func (u *ui) statusOf(s snap) status {
	privateNames := func() []string {
		var out []string
		for _, n := range s.net.networks {
			if n.category != catPublic && len(n.addrs) > 0 {
				out = append(out, n.name)
			}
		}
		return out
	}
	switch {
	case !s.known:
		return status{icon: icWifi, title: "Checking the network…", sub: "One moment"}
	case s.listenErr != "" && len(s.listening) == 0:
		return status{icon: icWarning, title: "Can't receive", sub: u.listenErrText(s.listenErr), warn: true}
	case len(s.listening) > 0:
		names := privateNames()
		where := "this network"
		if len(names) > 0 {
			where = "“" + names[0] + "”"
			if len(names) > 1 {
				where += fmt.Sprintf(" +%d", len(names)-1)
			}
		}
		return status{icon: icShieldCheck, title: "Ready to receive", sub: "On " + where + " · paired phones only", ready: true}
	case s.net.err != nil:
		return status{icon: icWarning, title: "Can't read the network", sub: "Windows didn't say which networks are private", warn: true}
	}
	for _, n := range s.net.networks {
		if n.category == catPublic && len(n.addrs) > 0 {
			return status{icon: icWifiWarning, title: "Paused on this network",
				sub: "“" + n.name + "” is a Public network", warn: true, settings: true}
		}
	}
	return status{icon: icWifiOff, title: "Not connected", sub: "Join your home Wi-Fi to receive", settings: true}
}

func (u *ui) listenErrText(e string) string {
	if strings.Contains(e, "Only one usage") || strings.Contains(strings.ToLower(e), "address already in use") {
		return "Another app is using port " + strconv.Itoa(u.set.Port)
	}
	return "Couldn't open port " + strconv.Itoa(u.set.Port)
}

func (u *ui) trayTip() string {
	st := u.statusOf(u.snapshot())
	if st.ready {
		return appName + " · Ready"
	}
	return appName + " · " + st.title
}

// ---------- drawing ----------

func (u *ui) mainHeight() float32 {
	return pad + statusH + labelH + rowH + pad + footerH
}

func (u *ui) settingsHeight() float32 {
	return 48 + labelH + 64 + 48 + labelH + 48 + 12 + 20 + pad
}

func (u *ui) height() float32 {
	if u.view == "settings" {
		return u.settingsHeight()
	}
	return u.mainHeight()
}

func (u *ui) addHit(r rectF, id string) { u.hits = append(u.hits, hit{r, id}) }

func (u *ui) hitAt(x, y float32) string {
	for i := len(u.hits) - 1; i >= 0; i-- {
		if u.hits[i].r.has(x, y) {
			return u.hits[i].id
		}
	}
	return ""
}

func (u *ui) shadow(c *canvas, r rectF, strength float32) {
	const layers = 14
	a := strength / layers
	if !u.th.dark {
		a *= 0.45
	}
	for i := layers; i >= 1; i-- {
		g := float32(i)
		c.fill(rectF{r.X - g, r.Y - g + 6, r.W + 2*g, r.H + 2*g}, radius+g, alpha(u.th.shadow, a))
	}
}

func (u *ui) render(c *canvas) {
	s := u.snapshot()
	u.hits = u.hits[:0]
	c.clear()
	panel := rectF{sh, sh, panelW, u.height()}
	u.shadow(c, panel, 0.42)
	c.fill(panel, radius, u.th.bg)
	if u.view == "settings" {
		u.drawSettings(c, panel)
	} else {
		u.drawMain(c, panel, s)
	}
	c.stroke(panel, radius, 1, u.th.border)
}

func (u *ui) drawCard(c *canvas, r rectF, fill uint32) {
	c.fill(r, 6, fill)
	if !u.th.dark {
		c.stroke(r, 6, 1, u.th.cardLine)
	}
}

func (u *ui) hoverFill(id string) uint32 {
	switch {
	case id == "":
		return u.th.card
	case u.press == id:
		return u.th.cardPress
	case u.hover == id:
		return u.th.cardHover
	}
	return u.th.card
}

func (u *ui) drawMain(c *canvas, p rectF, s snap) {
	t := u.th
	x, w := p.X+pad, float32(panelW-2*pad)
	y := p.Y + pad

	// Status card.
	st := u.statusOf(s)
	r := rectF{x, y, w, statusH}
	id := ""
	if st.settings {
		id = "btn:netsettings"
	}
	u.drawCard(c, r, u.hoverFill(id))
	chip, fg := t.chip, t.text
	switch {
	case st.ready:
		chip, fg = t.accent, t.onText
	case st.warn:
		chip, fg = alpha(t.warn, 0.18), t.warn
	}
	c.circle(r.X+34, r.Y+statusH/2, 20, chip)
	c.icon(st.icon, st.ready, fontIcon, 20, rectF{r.X + 14, r.Y + statusH/2 - 20, 40, 40}, fg)
	right := r.W - 76
	if id != "" {
		right -= 28
		c.icon(icOpen, false, fontIconSm, 16, rectF{r.X + r.W - 40, r.Y, 28, statusH}, t.text2)
		u.addHit(r, id)
	}
	c.text(st.title, fontSemi, rectF{r.X + 66, r.Y + 16, right, 20}, t.text, 0, 1)
	c.text(st.sub, fontSmall, rectF{r.X + 66, r.Y + 37, right, 18}, t.text2, 0, 1)
	y += statusH

	// Phones.
	c.text("Phones", fontSmallSemi, rectF{x + 2, y + 10, w, labelH - 10}, t.text2, 0, 1)
	y += labelH
	r = rectF{x, y, w, rowH}
	u.drawCard(c, r, t.card)
	c.circle(r.X+30, r.Y+rowH/2, 16, t.chip)
	c.icon(icPhoneAdd, false, fontIcon, 20, rectF{r.X + 14, r.Y + rowH/2 - 16, 32, 32}, t.text)
	bw := c.measure("Pair", fontText) + 32
	c.text("No phones paired yet", fontText, rectF{r.X + 58, r.Y + 13, r.W - 58 - bw - 24, 20}, t.text, 0, 1)
	c.text("Scan a QR code with your iPhone", fontSmall, rectF{r.X + 58, r.Y + 33, r.W - 58 - bw - 24, 18}, t.text2, 0, 1)
	u.accentButton(c, rectF{r.X + r.W - 14 - bw, r.Y + (rowH-32)/2, bw, 32}, "Pair", "btn:pair")

	// Footer: where files go, plus folder, settings and quit buttons.
	fy := p.Y + p.H - footerH
	c.fill(rectF{p.X, fy, p.W, footerH}, radius, t.footer)
	c.fill(rectF{p.X, fy, p.W, radius + 2}, 0, t.footer)
	c.fill(rectF{p.X, fy, p.W, 1}, 0, t.divider)
	msg, col := "Saving to "+filepath.Base(u.set.saveDir()), t.text2
	switch {
	case u.confirmQuit():
		msg, col = "Click again to quit", t.danger
	case s.flash != "":
		msg, col = s.flash, t.text
	}
	u.iconButton(c, rectF{p.X + p.W - pad - 36, fy + 8, 36, 32}, icPower, "btn:quit")
	u.iconButton(c, rectF{p.X + p.W - pad - 76, fy + 8, 36, 32}, icSettings, "btn:settings")
	u.iconButton(c, rectF{p.X + p.W - pad - 116, fy + 8, 36, 32}, icFolderOpen, "btn:folder")
	c.text(msg, fontSmall, rectF{p.X + pad + 4, fy, panelW - 2*pad - 136, footerH}, col, 0, 1)
}

func (u *ui) accentButton(c *canvas, r rectF, label, id string) {
	t := u.th
	fill := t.accent
	switch {
	case u.press == id:
		fill = t.accentPress
	case u.hover == id:
		fill = t.accentHover
	}
	c.fill(r, 4, fill)
	c.text(label, fontText, r, t.onText, 1, 1)
	u.addHit(r, id)
}

func (u *ui) button(c *canvas, r rectF, label, id string) {
	t := u.th
	fill := alpha(t.text, 0.06)
	if !t.dark {
		fill = 0xFFFDFDFD
	}
	switch {
	case u.press == id:
		fill = alpha(t.text, 0.03)
	case u.hover == id:
		fill = alpha(t.text, 0.09)
	}
	c.fill(r, 4, fill)
	c.stroke(r, 4, 1, alpha(t.text, 0.09))
	c.text(label, fontText, r, t.text, 1, 1)
	u.addHit(r, id)
}

func (u *ui) iconButton(c *canvas, r rectF, ic [2]rune, id string) {
	if id == "btn:quit" && u.confirmQuit() {
		c.fill(r, 4, alpha(u.th.danger, 0.18))
		c.icon(ic, false, fontIconSm, 16, r, u.th.danger)
		u.addHit(r, id)
		return
	}
	if u.press == id {
		c.fill(r, 4, alpha(u.th.text, 0.04))
	} else if u.hover == id {
		c.fill(r, 4, alpha(u.th.text, 0.07))
	}
	c.icon(ic, false, fontIconSm, 16, r, u.th.text)
	u.addHit(r, id)
}

func (u *ui) toggleSwitch(c *canvas, x, cy float32, on, hover bool) {
	t := u.th
	r := rectF{x, cy - 10, 40, 20}
	knob := float32(6)
	if hover {
		knob = 7
	}
	if on {
		fill := t.accent
		if hover {
			fill = t.accentHover
		}
		c.fill(r, 10, fill)
		c.circle(x+30, cy, knob, t.onText)
	} else {
		if hover {
			c.fill(r, 10, alpha(t.text, 0.05))
		}
		c.stroke(r, 10, 1, t.text2)
		c.circle(x+10, cy, knob, t.text2)
	}
}

func (u *ui) switchRow(c *canvas, r rectF, label, id string, on bool) {
	if u.hover == id {
		c.fill(rectF{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}, 5, alpha(u.th.text, 0.03))
	}
	c.text(label, fontText, rectF{r.X + 14, r.Y, r.W - 80, r.H}, u.th.text, 0, 1)
	u.toggleSwitch(c, r.X+r.W-54, r.Y+r.H/2, on, u.hover == id)
	u.addHit(r, id)
}

func (u *ui) drawSettings(c *canvas, p rectF) {
	t := u.th
	x, w := p.X+pad, float32(panelW-2*pad)
	u.iconButton(c, rectF{x - 4, p.Y + 8, 36, 32}, icBack, "btn:back")
	c.text("Settings", fontTitle, rectF{x + 40, p.Y + 8, w - 40, 32}, t.text, 0, 1)
	y := p.Y + 48

	c.text("Received items", fontSmallSemi, rectF{x + 2, y + 10, w, labelH - 10}, t.text2, 0, 1)
	y += labelH
	u.drawCard(c, rectF{x, y, w, 64 + 48}, t.card)
	bw := c.measure("Change", fontText) + 28
	c.text("Save files to", fontText, rectF{x + 14, y + 13, w - bw - 40, 20}, t.text, 0, 1)
	c.text(u.set.saveDir(), fontSmall, rectF{x + 14, y + 33, w - bw - 40, 18}, t.text2, 0, 1)
	u.button(c, rectF{x + w - 14 - bw, y + 16, bw, 32}, "Change", "btn:savedir")
	y += 64
	c.fill(rectF{x + 1, y, w - 2, 1}, 0, t.divider)
	u.switchRow(c, rectF{x, y, w, 48}, "Copy received text to the clipboard", "sw:autocopy", u.set.AutoCopy)
	y += 48

	c.text("General", fontSmallSemi, rectF{x + 2, y + 10, w, labelH - 10}, t.text2, 0, 1)
	y += labelH
	u.drawCard(c, rectF{x, y, w, 48}, t.card)
	u.switchRow(c, rectF{x, y, w, 48}, "Start when I sign in", "sw:login", u.startAtLogin)
	y += 48 + 12

	c.icon(icLock, false, fontIconSm, 14, rectF{x, y, 18, 20}, t.text2)
	c.text("Listens on port "+strconv.Itoa(u.set.Port)+", on Private networks only",
		fontSmall, rectF{x + 22, y, w - 22, 20}, t.text2, 0, 1)
}

// ---------- window plumbing ----------

func easeOut(x float64) float32 { return float32(1 - math.Pow(1-math.Min(x, 1), 3)) }

func (u *ui) ensureCanvas(cv **canvas, w, h float32) *canvas {
	pw, ph := int(math.Ceil(float64((w+2*sh)*u.scale))), int(math.Ceil(float64((h+2*sh)*u.scale)))
	if *cv == nil || (*cv).w != pw || (*cv).h != ph || (*cv).scale != u.scale {
		if *cv != nil {
			(*cv).free()
		}
		*cv = newCanvas(pw, ph, u.scale)
	}
	return *cv
}

func present(hwnd uintptr, c *canvas, x, y int, a float32) {
	pt := struct{ x, y int32 }{int32(x), int32(y)}
	sz := struct{ cx, cy int32 }{int32(c.w), int32(c.h)}
	src := struct{ x, y int32 }{}
	blend := uint32(min(255, max(0, int(a*255+0.5))))<<16 | 1<<24 // SourceConstantAlpha, AC_SRC_ALPHA
	win(user32, "UpdateLayeredWindow", hwnd, 0, ptr(&pt), ptr(&sz), c.dc, ptr(&src), 0, ptr(&blend), 2)
}

// paint redraws the panel and puts it on screen at the current animation state.
func (u *ui) paint() {
	if !u.visible {
		return
	}
	c := u.ensureCanvas(&u.cv, panelW, u.height())
	u.render(c)
	u.place(c)
}

func (u *ui) place(c *canvas) {
	el := time.Since(u.animStart).Seconds()
	var off float32
	if u.hiding {
		u.alpha = 1 - easeOut(el/0.12)
	} else {
		p := easeOut(el / 0.2)
		u.alpha, off = min(1, float32(el/0.12)), (1-p)*14*u.scale
	}
	x, y := u.x, u.y
	switch u.edge {
	case 0:
		y += int(off)
	case 1:
		y -= int(off)
	case 2:
		x -= int(off)
	case 3:
		x += int(off)
	}
	present(u.hwnd, c, x, y, u.alpha)
}

func (u *ui) trayRect() rect {
	id := struct {
		size uint32
		hwnd uintptr
		uid  uint32
		guid [16]byte
	}{hwnd: u.hwnd, uid: 1}
	id.size = uint32(unsafe.Sizeof(id))
	var r rect
	if win(shell32, "Shell_NotifyIconGetRect", ptr(&id), ptr(&r)) != 0 || r.right == 0 {
		var pt struct{ x, y int32 }
		win(user32, "GetCursorPos", ptr(&pt))
		r = rect{pt.x - 8, pt.y - 8, pt.x + 8, pt.y + 8}
	}
	return r
}

// position: next to the tray icon, 12 px clear of the taskbar, on whichever edge it lives.
func (u *ui) position(anchor rect, w, h float32) {
	pt := uint64(uint32((anchor.left+anchor.right)/2)) | uint64(uint32((anchor.top+anchor.bottom)/2))<<32
	mon := win(user32, "MonitorFromPoint", uintptr(pt), 2)
	mi := struct {
		size      uint32
		mon, work rect
		flags     uint32
	}{}
	mi.size = uint32(unsafe.Sizeof(mi))
	win(user32, "GetMonitorInfoW", mon, ptr(&mi))
	var dx, dy uint32 = 96, 96
	if shcore.Load() == nil {
		win(shcore, "GetDpiForMonitor", mon, 0, ptr(&dx), ptr(&dy))
	}
	u.scale = float32(dx) / 96
	s := u.scale
	m, sp := int(12*s), int(sh*s)
	pw, ph := int(w*s), int(h*s)
	cx, cy := int(anchor.left+anchor.right)/2, int(anchor.top+anchor.bottom)/2
	wk, mn := mi.work, mi.mon
	clamp := func(v, lo, hi int) int { return max(lo, min(v, hi)) }
	var px, py int
	switch {
	case wk.top > mn.top:
		u.edge, py, px = 1, int(wk.top)+m, clamp(cx-pw/2, int(wk.left)+m, int(wk.right)-m-pw)
	case wk.left > mn.left:
		u.edge, px, py = 2, int(wk.left)+m, clamp(cy-ph/2, int(wk.top)+m, int(wk.bottom)-m-ph)
	case wk.right < mn.right:
		u.edge, px, py = 3, int(wk.right)-m-pw, clamp(cy-ph/2, int(wk.top)+m, int(wk.bottom)-m-ph)
	default:
		u.edge, py, px = 0, int(wk.bottom)-m-ph, clamp(cx-pw/2, int(wk.left)+m, int(wk.right)-m-pw)
	}
	u.x, u.y = px-sp, py-sp
}

func (u *ui) show(view string) {
	u.th = loadTheme()
	u.startAtLogin = runAtLogin()
	u.view, u.hover, u.press, u.hiding = view, "", "", false
	u.position(u.trayRect(), panelW, u.height())
	u.visible = true
	u.animStart = time.Now()
	c := u.ensureCanvas(&u.cv, panelW, u.height())
	u.render(c)
	present(u.hwnd, c, u.x, u.y, 0)
	win(user32, "ShowWindow", u.hwnd, 5)
	win(user32, "SetForegroundWindow", u.hwnd)
	win(user32, "SetTimer", u.hwnd, tAnim, 10, 0)
	select { // look at the network again, in case a change slipped past the watcher
	case u.recheck <- struct{}{}:
	default:
	}
}

func (u *ui) hide() {
	if !u.visible || u.hiding {
		return
	}
	u.hiding, u.animStart, u.hiddenAt = true, time.Now(), time.Now()
	win(user32, "ReleaseCapture")
	win(user32, "SetTimer", u.hwnd, tAnim, 10, 0)
}

// relayout after switching views: keep the edge nearest the taskbar where it is.
func (u *ui) relayout() {
	u.position(u.trayRect(), panelW, u.height())
	u.paint()
}

func (u *ui) click(id string) {
	switch id {
	case "btn:quit":
		if u.confirmQuit() {
			u.quit()
		}
		u.quitAt = time.Now()
		win(user32, "SetTimer", u.hwnd, tFlash, 3100, 0) // repaint when the confirm lapses
	case "btn:settings":
		u.view, u.hover = "settings", ""
		u.relayout()
		return
	case "btn:back":
		u.view, u.hover = "main", ""
		u.relayout()
		return
	case "btn:folder":
		os.MkdirAll(u.set.saveDir(), 0o755)
		shellOpen(u.set.saveDir())
		u.hide()
		return
	case "btn:netsettings":
		shellOpen("ms-settings:network-status")
		u.hide()
		return
	case "btn:pair":
		u.flashMsg("Pairing arrives in the next build")
	case "btn:savedir":
		u.hide()
		if dir, ok := pickFolder(u.set.saveDir()); ok {
			u.set.SaveDir = dir
			u.set.save()
		}
		u.show("settings")
		return
	case "sw:autocopy":
		u.set.AutoCopy = !u.set.AutoCopy
		u.set.save()
	case "sw:login":
		if exe, err := os.Executable(); err == nil && setRunAtLogin(!u.startAtLogin, exe) {
			u.startAtLogin = !u.startAtLogin
		}
	}
	u.paint()
}

// pickFolder shows the standard Windows folder picker (IFileOpenDialog with FOS_PICKFOLDERS).
func pickFolder(start string) (string, bool) {
	clsid := guid{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iid := guid{0xD57C7288, 0xD4AD, 0x4768, [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	var dlg unsafe.Pointer
	if r := win(ole32, "CoCreateInstance", ptr(&clsid), 0, 1 /*CLSCTX_INPROC_SERVER*/, ptr(&iid),
		uintptr(unsafe.Pointer(&dlg))); int32(r) < 0 {
		return "", false
	}
	defer comRelease(dlg)
	const (
		show, setOptions, getOptions, setFolder, setTitle, getResult = 3, 9, 10, 12, 17, 20
		shellItemGetDisplayName                                      = 5
	)
	var opts uint32
	comCall(dlg, getOptions, ptr(&opts))
	comCall(dlg, setOptions, uintptr(opts|0x20|0x40)) // FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM
	comCall(dlg, setTitle, ptr(wstr("Save received files to")))
	iidShellItem := guid{0x43826D1E, 0xE718, 0x42EE, [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
	var folder unsafe.Pointer
	if win(shell32, "SHCreateItemFromParsingName", ptr(wstr(start)), 0, ptr(&iidShellItem),
		uintptr(unsafe.Pointer(&folder))) == 0 {
		comCall(dlg, setFolder, uintptr(folder))
		comRelease(folder)
	}
	if r := comCall(dlg, show, 0); r != 0 { // HRESULT_FROM_WIN32(ERROR_CANCELLED) when cancelled
		return "", false
	}
	var item unsafe.Pointer
	if r := comCall(dlg, getResult, uintptr(unsafe.Pointer(&item))); int32(r) < 0 {
		return "", false
	}
	defer comRelease(item)
	var path *uint16
	if r := comCall(item, shellItemGetDisplayName, 0x80058000 /*SIGDN_FILESYSPATH*/, uintptr(unsafe.Pointer(&path))); int32(r) < 0 {
		return "", false
	}
	defer win(ole32, "CoTaskMemFree", uintptr(unsafe.Pointer(path)))
	return syscall.UTF16ToString(unsafe.Slice(path, wcslen(path))), true
}

func wcslen(p *uint16) int {
	n := 0
	for s := unsafe.Pointer(p); *(*uint16)(unsafe.Add(s, 2*n)) != 0; n++ {
	}
	return n
}

func (u *ui) addTray(modify bool) {
	if u.trayIcon != 0 {
		win(user32, "DestroyIcon", u.trayIcon)
	}
	u.trayIcon = u.makeTrayIcon()
	var nid struct {
		size        uint32
		hwnd        uintptr
		id, flags   uint32
		callback    uint32
		icon        uintptr
		tip         [128]uint16
		state, mask uint32
		info        [256]uint16
		version     uint32
		infoTitle   [64]uint16
		infoFlags   uint32
		guid        [16]byte
		balloonIcon uintptr
	}
	nid.size = uint32(unsafe.Sizeof(nid))
	nid.hwnd, nid.id, nid.callback, nid.icon = u.hwnd, 1, wmTray, u.trayIcon
	nid.flags = 0x1 | 0x2 | 0x4 | 0x80 // MESSAGE | ICON | TIP | SHOWTIP
	tip, _ := syscall.UTF16FromString(u.trayTip())
	copy(nid.tip[:127], tip)
	op := uintptr(0) // NIM_ADD
	if modify {
		op = 1
	}
	if win(shell32, "Shell_NotifyIconW", op, ptr(&nid)) == 0 && modify {
		win(shell32, "Shell_NotifyIconW", 0, ptr(&nid))
	}
	nid.version = 4
	win(shell32, "Shell_NotifyIconW", 4, ptr(&nid)) // NIM_SETVERSION
}

// makeTrayIcon draws the Fluent "phone with arrow" glyph in the taskbar's text colour.
func (u *ui) makeTrayIcon() uintptr {
	size := int(win(user32, "GetSystemMetrics", 49)) // SM_CXSMICON
	if size <= 0 {
		size = 16
	}
	light, ok := regDword(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "SystemUsesLightTheme")
	col := uint32(0xFFFFFFFF)
	if ok && light == 1 {
		col = 0xFF1A1A1A
	}
	c := trayCanvas(size, col)
	defer c.free()
	return c.hicon()
}

// trayCanvas: this glyph fills ~85% of its em box, so 15 px makes it ~13 px of the 16, like other
// tray icons; the layout box is much bigger than the icon, or GDI+ clips the glyph.
func trayCanvas(size int, col uint32) *canvas {
	c := newCanvas(size, size, float32(size)/16)
	c.clear()
	f := newFont([]string{"FluentSystemIcons-Filled"}, iconColl, 15, 0)
	defer gp("GdipDeleteFont", f)
	c.icon(icPhoneArrow, true, f, 15, rectF{-12, -12, 40, 40}, col)
	return c
}

func (u *ui) menu() {
	m := win(user32, "CreatePopupMenu")
	win(user32, "AppendMenuW", m, 0, 1, ptr(wstr("Open")))
	win(user32, "AppendMenuW", m, 0, 2, ptr(wstr("Settings")))
	win(user32, "AppendMenuW", m, 0, 3, ptr(wstr("Open received files")))
	win(user32, "AppendMenuW", m, 0x800, 0, 0) // separator
	win(user32, "AppendMenuW", m, 0, 4, ptr(wstr("Quit PhoneDrop")))
	var pt struct{ x, y int32 }
	win(user32, "GetCursorPos", ptr(&pt))
	win(user32, "SetForegroundWindow", u.hwnd)
	cmd := win(user32, "TrackPopupMenu", m, 0x0100|0x0020|0x0002, uintptr(pt.x), uintptr(pt.y), 0, u.hwnd, 0) // RETURNCMD|BOTTOMALIGN|RIGHTBUTTON
	win(user32, "DestroyMenu", m)
	win(user32, "PostMessageW", u.hwnd, 0, 0, 0)
	switch cmd {
	case 1:
		u.show("main")
	case 2:
		u.show("settings")
	case 3:
		os.MkdirAll(u.set.saveDir(), 0o755)
		shellOpen(u.set.saveDir())
	case 4:
		u.quit()
	}
}

func (u *ui) confirmQuit() bool { return time.Since(u.quitAt) < 3*time.Second }

func (u *ui) quit() {
	nid := make([]byte, 976) // sizeof(NOTIFYICONDATAW); only size, hwnd and id matter for NIM_DELETE
	*(*uint32)(unsafe.Pointer(&nid[0])) = 976
	*(*uintptr)(unsafe.Pointer(&nid[8])) = u.hwnd
	*(*uint32)(unsafe.Pointer(&nid[16])) = 1
	win(shell32, "Shell_NotifyIconW", 2, uintptr(unsafe.Pointer(&nid[0])))
	os.Exit(0)
}

func lparamXY(l uintptr, scale float32) (float32, float32) {
	return float32(int16(l&0xFFFF)) / scale, float32(int16(l>>16&0xFFFF)) / scale
}

func uiProc(hwnd, m, wParam, lParam uintptr) uintptr {
	u := theUI
	switch m {
	case wmTray:
		switch lParam & 0xFFFF {
		case 0x400, 0x401: // NIN_SELECT, NIN_KEYSELECT
			if u.visible && !u.hiding {
				u.hide()
			} else if time.Since(u.hiddenAt) > 300*time.Millisecond {
				u.show("main") // (a click that just closed it via deactivation doesn't reopen it)
			}
		case 0x7B: // WM_CONTEXTMENU
			u.menu()
		}
		return 0
	case wmRefresh:
		u.addTray(true) // the tooltip follows the status
		if u.visible && !u.hiding {
			u.paint()
		}
		return 0
	case 0x0113: // WM_TIMER
		switch wParam {
		case tAnim:
			el := time.Since(u.animStart).Seconds()
			if u.hiding && el >= 0.12 {
				win(user32, "KillTimer", hwnd, tAnim)
				win(user32, "ShowWindow", hwnd, 0)
				u.visible, u.hiding = false, false
				return 0
			}
			if !u.hiding && el >= 0.22 {
				win(user32, "KillTimer", hwnd, tAnim)
			}
			u.place(u.cv)
		case tFlash:
			win(user32, "KillTimer", hwnd, tFlash)
			u.paint()
		}
		return 0
	case 0x0006: // WM_ACTIVATE
		if wParam&0xFFFF == 0 {
			u.hide()
		}
		return 0
	case 0x0200: // WM_MOUSEMOVE
		if !u.tracking {
			tme := struct {
				size, flags uint32
				hwnd        uintptr
				hover       uint32
			}{flags: 2, hwnd: hwnd}
			tme.size = uint32(unsafe.Sizeof(tme))
			win(user32, "TrackMouseEvent", ptr(&tme))
			u.tracking = true
		}
		x, y := lparamXY(lParam, u.scale)
		if h := u.hitAt(x, y); h != u.hover {
			u.hover = h
			u.paint()
		}
		return 0
	case 0x02A3: // WM_MOUSELEAVE
		u.tracking = false
		if u.hover != "" {
			u.hover = ""
			u.paint()
		}
		return 0
	case 0x0201: // WM_LBUTTONDOWN
		x, y := lparamXY(lParam, u.scale)
		u.press = u.hitAt(x, y)
		u.paint()
		return 0
	case 0x0202: // WM_LBUTTONUP
		x, y := lparamXY(lParam, u.scale)
		p := u.press
		u.press = ""
		if p != "" && p == u.hitAt(x, y) {
			u.click(p)
		} else {
			u.paint()
		}
		return 0
	case 0x0100, 0x0104: // WM_KEYDOWN, WM_SYSKEYDOWN
		if wParam == 0x1B { // Esc
			if u.view == "settings" {
				u.click("btn:back")
			} else {
				u.hide()
			}
		}
		return 0
	case 0x0102, 0x0106: // WM_CHAR, WM_SYSCHAR: swallow (no "ding" for Alt+key)
		return 0
	case 0x001A: // WM_SETTINGCHANGE: light/dark or accent changed
		u.th = loadTheme()
		u.addTray(true)
		u.paint()
	}
	if u.taskbarCreated != 0 && uint32(m) == u.taskbarCreated { // Explorer restarted
		u.addTray(false)
		return 0
	}
	r, _, _ := defWindow.Call(hwnd, m, wParam, lParam)
	return r
}

// runUI owns the tray icon and panel on its own OS thread.
func runUI(u *ui) {
	runtime.LockOSThread()
	win(ole32, "CoInitializeEx", 0, 0x2) // COINIT_APARTMENTTHREADED, for the folder picker
	gdipInit()
	inst, _, _ := getModule.Call(0)
	proc := syscall.NewCallback(uiProc)
	cursor := win(user32, "LoadCursorW", 0, 32512) // IDC_ARROW
	wc := wndClassEx{wndProc: proc, instance: inst, cursor: cursor, className: wstr("PhoneDropPanel")}
	wc.size = uint32(unsafe.Sizeof(wc))
	registerCls.Call(ptr(&wc))
	const exLayered, exTool, exTopmost = 0x80000, 0x80, 0x8
	const wsPopup = 0x80000000
	u.hwnd, _, _ = createWindow.Call(exLayered|exTool|exTopmost, ptr(wstr("PhoneDropPanel")), ptr(wstr(appName)),
		wsPopup, 0, 0, 1, 1, 0, 0, inst, 0)
	u.taskbarCreated = uint32(win(user32, "RegisterWindowMessageW", ptr(wstr("TaskbarCreated"))))
	u.addTray(false)

	var m msg
	for {
		if r, _, _ := getMessage.Call(ptr(&m), 0, 0, 0); int32(r) <= 0 {
			return
		}
		win(user32, "TranslateMessage", ptr(&m))
		dispatchMsg.Call(ptr(&m))
	}
}
