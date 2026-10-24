// Toasts: a small card above the tray when something arrives ("IMG_2041.HEIC from iPhone ·
// Open · Show in folder"), drawn like the panel. It never takes focus, stays while hovered and
// slides away after a few seconds. Items arriving while one is up merge into "3 items from …".
// Timers run only while a toast is on screen.
package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	toastW    = 364
	toastLife = 6 * time.Second
	toastStay = 3 * time.Second // after the pointer leaves

	tToastAnim, tToastLife = 11, 12
)

// toastItem: what a toast shows. kind is file, image, video, text or url.
type toastItem struct {
	kind, title, from string
	path, text        string // file path, or the text / link
	size              int64
	copied            bool // text was put on the clipboard
}

type toastAction struct{ id, label string }

type toaster struct {
	hwnd         uintptr
	cv           *canvas
	items        []toastItem // merged while the toast is up; the last one is shown
	visible      bool
	hiding       bool
	animStart    time.Time
	x, y         int
	hover, press string
	tracking     bool
	hits         []hit
}

// isRisky: files Windows would run rather than open; the toast offers only "Show in folder".
func isRisky(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".exe", ".com", ".bat", ".cmd", ".ps1", ".psm1", ".vbs", ".vbe", ".js", ".jse", ".wsf", ".wsh",
		".msi", ".msp", ".scr", ".pif", ".lnk", ".url", ".hta", ".cpl", ".jar", ".reg", ".inf", ".appx",
		".msix", ".appinstaller", ".application", ".gadget", ".msc", ".sys", ".dll", ".iso", ".img", ".vhd", ".vhdx":
		return true
	}
	return false
}

func kindIcon(kind string) [2]rune {
	switch kind {
	case "image":
		return icImage
	case "video":
		return icVideo
	case "text":
		return icClipboardText
	case "url":
		return icLink
	}
	return icDocument
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// toastText: title, subtitle and buttons for the current toast.
func (t *toaster) content() (icon [2]rune, title, sub string, actions []toastAction) {
	it := t.items[len(t.items)-1]
	if n := len(t.items); n > 1 {
		files := 0
		for _, x := range t.items {
			if x.path != "" {
				files++
			}
		}
		title = fmt.Sprintf("%d items from %s", n, it.from)
		sub = "Latest: " + it.title
		if files > 0 {
			actions = append(actions, toastAction{"t:folder", "Show in folder"})
		}
		return kindIcon(it.kind), title, sub, actions
	}
	icon, title = kindIcon(it.kind), it.title
	switch it.kind {
	case "text":
		title = "Text from " + it.from
		sub = firstLine(it.text)
		if it.copied {
			title = "Copied text from " + it.from
		} else {
			actions = append(actions, toastAction{"t:copy", "Copy"})
		}
	case "url":
		title = "Link from " + it.from
		sub = it.text
		actions = append(actions, toastAction{"t:open", "Open link"}, toastAction{"t:copy", "Copy"})
	default:
		sub = "From " + it.from + " · " + humanSize(it.size)
		if !isRisky(it.path) {
			actions = append(actions, toastAction{"t:open", "Open"})
		}
		actions = append(actions, toastAction{"t:folder", "Show in folder"})
	}
	return
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i] + " …"
	}
	return s
}

func (u *ui) toastHeight() float32 {
	_, _, _, actions := u.toast.content()
	if len(actions) == 0 {
		return 72
	}
	return 72 + 44
}

func (u *ui) drawToast(c *canvas) {
	t, tt := u.th, &u.toast
	tt.hits = tt.hits[:0]
	c.clear()
	h := u.toastHeight()
	r := rectF{sh, sh, toastW, h}
	u.shadow(c, r, 0.36)
	c.fill(r, radius, t.bg)
	c.stroke(r, radius, 1, t.border)
	icon, title, sub, actions := tt.content()
	c.circle(r.X+34, r.Y+36, 20, t.accent)
	c.icon(icon, true, fontIcon, 20, rectF{r.X + 14, r.Y + 16, 40, 40}, t.onText)
	c.text(title, fontSemi, rectF{r.X + 66, r.Y + 16, r.W - 66 - 40, 20}, t.text, 0, 1)
	c.text(sub, fontSmall, rectF{r.X + 66, r.Y + 37, r.W - 66 - 20, 18}, t.text2, 0, 1)
	tt.hits = append(tt.hits, hit{r, "t:body"})
	if tt.hover != "" { // dismiss button while hovered
		d := rectF{r.X + r.W - 36, r.Y + 8, 28, 28}
		if tt.hover == "t:close" {
			c.fill(d, 4, alpha(t.text, 0.08))
		}
		c.icon(icDismiss, false, fontIconSm, 14, d, t.text2)
		tt.hits = append(tt.hits, hit{d, "t:close"})
	}
	if len(actions) > 0 {
		bw := (r.W - 24 - float32(len(actions)-1)*8) / float32(len(actions))
		for i, a := range actions {
			b := rectF{r.X + 12 + float32(i)*(bw+8), r.Y + 72, bw, 32}
			fill := alpha(t.text, 0.06)
			if !t.dark {
				fill = 0xFFFDFDFD
			}
			switch {
			case tt.press == a.id:
				fill = alpha(t.text, 0.03)
			case tt.hover == a.id:
				fill = alpha(t.text, 0.09)
			}
			c.fill(b, 4, fill)
			c.stroke(b, 4, 1, alpha(t.text, 0.09))
			c.text(a.label, fontText, b, t.text, 1, 1)
			tt.hits = append(tt.hits, hit{b, a.id})
		}
	}
}

// showToast adds an item: it opens a toast, or merges into the one on screen.
func (u *ui) showToast(it toastItem) {
	tt := &u.toast
	if !tt.visible || tt.hiding {
		tt.items = nil
	}
	tt.items = append(tt.items, it)
	if len(tt.items) > 50 {
		tt.items = tt.items[len(tt.items)-50:]
	}
	u.th = loadTheme()
	var mon uintptr
	if u.hwnd != 0 {
		r := u.trayRect()
		pt := uint64(uint32((r.left+r.right)/2)) | uint64(uint32((r.top+r.bottom)/2))<<32
		mon = win(user32, "MonitorFromPoint", uintptr(pt), 2)
	}
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
	scale := float32(dx) / 96
	if !u.visible {
		u.scale = scale
	}
	saved := u.scale
	u.scale = scale
	c := u.ensureCanvas(&tt.cv, toastW, u.toastHeight())
	u.drawToast(c)
	u.scale = saved
	m, sp := int(12*scale), int(sh*scale)
	tt.x = int(mi.work.right) - m - int(toastW*scale) - sp
	tt.y = int(mi.work.bottom) - m - int(u.toastHeight()*scale) - sp
	if !tt.visible || tt.hiding {
		tt.visible, tt.hiding, tt.animStart = true, false, time.Now()
		present(tt.hwnd, c, tt.x, tt.y, 0)
		win(user32, "ShowWindow", tt.hwnd, 4) // SW_SHOWNOACTIVATE
		win(user32, "SetTimer", tt.hwnd, tToastAnim, 10, 0)
	} else {
		present(tt.hwnd, c, tt.x, tt.y, 1)
	}
	if tt.hover == "" {
		win(user32, "SetTimer", tt.hwnd, tToastLife, uintptr(toastLife.Milliseconds()), 0)
	}
}

func (u *ui) placeToast() {
	tt := &u.toast
	el := time.Since(tt.animStart).Seconds()
	var a float32
	off := 0
	if tt.hiding {
		a = 1 - easeOut(el/0.15)
		if el >= 0.15 {
			win(user32, "KillTimer", tt.hwnd, tToastAnim)
			win(user32, "ShowWindow", tt.hwnd, 0)
			tt.visible, tt.hiding, tt.items, tt.hover = false, false, nil, ""
			return
		}
	} else {
		a = min(1, float32(el/0.12))
		off = int((1 - easeOut(el/0.25)) * 24 * u.scale)
		if el >= 0.25 {
			win(user32, "KillTimer", tt.hwnd, tToastAnim)
		}
	}
	present(tt.hwnd, tt.cv, tt.x+off, tt.y, a)
}

func (u *ui) hideToast() {
	tt := &u.toast
	if !tt.visible || tt.hiding {
		return
	}
	win(user32, "KillTimer", tt.hwnd, tToastLife)
	tt.hiding, tt.animStart = true, time.Now()
	win(user32, "SetTimer", tt.hwnd, tToastAnim, 10, 0)
}

func (u *ui) repaintToast() {
	tt := &u.toast
	if !tt.visible || tt.hiding || len(tt.items) == 0 {
		return
	}
	saved := u.scale
	u.scale = tt.cv.scale
	c := u.ensureCanvas(&tt.cv, toastW, u.toastHeight())
	u.drawToast(c)
	u.scale = saved
	present(tt.hwnd, c, tt.x, tt.y, 1)
}

func (u *ui) toastClick(id string) {
	tt := &u.toast
	if len(tt.items) == 0 {
		return
	}
	it := tt.items[len(tt.items)-1]
	if id == "t:body" { // the card itself does the first action
		if _, _, _, actions := tt.content(); len(actions) > 0 {
			id = actions[0].id
		} else {
			id = "t:close"
		}
	}
	switch id {
	case "t:open":
		if it.kind == "url" {
			if strings.HasPrefix(it.text, "https://") || strings.HasPrefix(it.text, "http://") {
				shellOpen(it.text)
			}
		} else if it.path != "" && !isRisky(it.path) {
			shellOpen(it.path)
		}
	case "t:folder":
		showInFolder(it.path)
	case "t:copy":
		setClipboardText(tt.hwnd, it.text)
	}
	u.hideToast()
}

// showInFolder opens Explorer with the file selected (or the save folder if it's gone).
func showInFolder(path string) {
	if path == "" {
		return
	}
	args := `/select,"` + path + `"`
	win(shell32, "ShellExecuteW", 0, ptr(wstr("open")), ptr(wstr("explorer.exe")), ptr(wstr(args)), 0, 1)
}

func toastProc(u *ui, hwnd, m, wParam, lParam uintptr) uintptr {
	tt := &u.toast
	switch m {
	case 0x0113: // WM_TIMER
		switch wParam {
		case tToastAnim:
			u.placeToast()
		case tToastLife:
			u.hideToast()
		}
		return 0
	case 0x0021: // WM_MOUSEACTIVATE: clicks must not steal focus
		return 3 // MA_NOACTIVATE
	case 0x0200: // WM_MOUSEMOVE
		if !tt.tracking {
			tme := struct {
				size, flags uint32
				hwnd        uintptr
				hover       uint32
			}{flags: 2, hwnd: hwnd}
			tme.size = uint32(unsafe.Sizeof(tme))
			win(user32, "TrackMouseEvent", ptr(&tme))
			tt.tracking = true
			win(user32, "KillTimer", hwnd, tToastLife) // stays while hovered
		}
		x, y := lparamXY(lParam, tt.cv.scale)
		h := ""
		for i := len(tt.hits) - 1; i >= 0; i-- {
			if tt.hits[i].r.has(x, y) {
				h = tt.hits[i].id
				break
			}
		}
		if h == "" {
			h = "t:body"
		}
		if h != tt.hover {
			tt.hover = h
			u.repaintToast()
		}
		return 0
	case 0x02A3: // WM_MOUSELEAVE
		tt.tracking, tt.hover, tt.press = false, "", ""
		u.repaintToast()
		if tt.visible && !tt.hiding {
			win(user32, "SetTimer", hwnd, tToastLife, uintptr(toastStay.Milliseconds()), 0)
		}
		return 0
	case 0x0201: // WM_LBUTTONDOWN
		tt.press = tt.hover
		u.repaintToast()
		return 0
	case 0x0202: // WM_LBUTTONUP
		p := tt.press
		tt.press = ""
		if p != "" && p == tt.hover {
			if p == "t:close" {
				u.hideToast()
			} else {
				u.toastClick(p)
			}
		} else {
			u.repaintToast()
		}
		return 0
	}
	r, _, _ := defWindow.Call(hwnd, m, wParam, lParam)
	return r
}

// setClipboardText puts text on the clipboard as CF_UNICODETEXT (UI thread).
func setClipboardText(hwnd uintptr, s string) bool {
	u16, err := syscall.UTF16FromString(s)
	if err != nil {
		return false
	}
	for i := 0; win(user32, "OpenClipboard", hwnd) == 0; i++ { // another app may hold it briefly
		if i == 10 {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	defer win(user32, "CloseClipboard")
	win(user32, "EmptyClipboard")
	h := win(kernel32, "GlobalAlloc", 0x0002 /*GMEM_MOVEABLE*/, uintptr(len(u16)*2))
	if h == 0 {
		return false
	}
	p := win(kernel32, "GlobalLock", h)
	win(kernel32, "RtlMoveMemory", p, uintptr(unsafe.Pointer(&u16[0])), uintptr(len(u16)*2))
	win(kernel32, "GlobalUnlock", h)
	if win(user32, "SetClipboardData", 13 /*CF_UNICODETEXT*/, h) == 0 {
		win(kernel32, "GlobalFree", h)
		return false
	}
	return true
}
