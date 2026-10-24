// Laptop → phone on the Windows side. Things to send go into the outbox, and the phone collects
// them with its Receive shortcut. They arrive from Explorer's Send to menu (a second PhoneDrop
// process hands its file arguments to the running one with WM_COPYDATA), the panel's Files
// button, or the clipboard (panel button or Ctrl+Alt+P).
package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

const (
	copyDataFiles = 0x5044 // WM_COPYDATA tag: NUL-separated file paths
	hotkeySend    = 1      // RegisterHotKey id
	hkMods        = modCtrl | modAlt
	hkKey         = 'P'

	modAlt, modCtrl, modShift, modWin = 1, 2, 4, 8
	modNoRepeat                       = 0x4000
)

// sender: what the panel needs from the outbox.
type sender interface {
	addFile(path string) error
	addText(text string)
	addData(name string, data []byte) error
	count() int
	clear()
}

// forwardToRunning hands files to an already running PhoneDrop; it reports whether one took them.
func forwardToRunning(files []string) bool {
	hwnd := win(user32, "FindWindowW", ptr(wstr("PhoneDropPanel")), 0)
	if hwnd == 0 {
		return false
	}
	var buf []uint16
	for _, f := range files {
		if abs, err := filepath.Abs(f); err == nil {
			f = abs
		}
		u, _ := syscall.UTF16FromString(f)
		buf = append(buf, u...) // keeps each string's NUL
	}
	if len(buf) == 0 {
		return true
	}
	cds := struct {
		data uintptr
		size uint32
		ptr  *uint16
	}{copyDataFiles, uint32(len(buf) * 2), &buf[0]}
	r := win(user32, "SendMessageW", hwnd, 0x004A /*WM_COPYDATA*/, 0, ptr(&cds))
	return r != 0
}

// copyDataFilesFrom decodes a WM_COPYDATA from forwardToRunning.
func copyDataFilesFrom(lParam uintptr) []string {
	type copyData struct {
		data uintptr
		size uint32
		ptr  *uint16
	}
	cds := *(**copyData)(unsafe.Pointer(&lParam)) // lParam points at the sender's COPYDATASTRUCT
	if cds.data != copyDataFiles || cds.ptr == nil || cds.size < 2 || cds.size > 1<<20 {
		return nil
	}
	all := unsafe.Slice(cds.ptr, cds.size/2)
	var out []string
	start := 0
	for i, c := range all {
		if c == 0 {
			if i > start {
				out = append(out, syscall.UTF16ToString(all[start:i]))
			}
			start = i + 1
		}
	}
	return out
}

// queueFiles adds files (not folders) to the outbox and says what happened.
func (u *ui) queueFiles(files []string) {
	if u.out == nil {
		return
	}
	n, skipped := 0, 0
	for _, f := range files {
		st, err := os.Stat(f)
		if err != nil || st.IsDir() {
			skipped++
			continue
		}
		if err := u.out.addFile(f); err != nil {
			log.Printf("outbox: %v", err)
			skipped++
			continue
		}
		n++
	}
	switch {
	case n == 1:
		u.notifyQueued(filepath.Base(files[0]))
	case n > 1:
		u.notifyQueued(strconv.Itoa(n) + " files")
	case skipped > 0:
		u.flashMsg("Folders can't be sent, only files")
	}
	if u.visible {
		u.paint()
	}
}

// notifyQueued confirms with a toast, since Send to and the hotkey have no other feedback.
func (u *ui) notifyQueued(what string) {
	u.showToast(toastItem{kind: "queued", title: what + " ready for your phone",
		text: "Run “Receive from PC” on your iPhone"})
}

// sendClipboard queues whatever is on the clipboard: files, an image, or text.
func (u *ui) sendClipboard() {
	if u.out == nil {
		return
	}
	for i := 0; win(user32, "OpenClipboard", u.hwnd) == 0; i++ {
		if i == 10 {
			u.flashMsg("The clipboard is busy; try again")
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	var files []string
	var text string
	var png []byte
	switch {
	case win(user32, "IsClipboardFormatAvailable", 15 /*CF_HDROP*/) != 0:
		h := win(user32, "GetClipboardData", 15)
		n := int(win(shell32, "DragQueryFileW", h, 0xFFFFFFFF, 0, 0))
		for i := 0; i < n && i < 500; i++ {
			l := win(shell32, "DragQueryFileW", h, uintptr(i), 0, 0)
			buf := make([]uint16, l+1)
			win(shell32, "DragQueryFileW", h, uintptr(i), ptr(&buf[0]), l+1)
			files = append(files, syscall.UTF16ToString(buf))
		}
	case win(user32, "IsClipboardFormatAvailable", 2 /*CF_BITMAP*/) != 0:
		png = bitmapPNG(win(user32, "GetClipboardData", 2))
	case win(user32, "IsClipboardFormatAvailable", 13 /*CF_UNICODETEXT*/) != 0:
		text = clipboardText()
	}
	win(user32, "CloseClipboard")
	switch {
	case len(files) > 0:
		u.queueFiles(files)
	case png != nil:
		name := "Clipboard " + time.Now().Format("2006-01-02 15.04.05") + ".png"
		if err := u.out.addData(name, png); err != nil {
			log.Printf("outbox: %v", err)
			u.flashMsg("Couldn't keep the image")
			return
		}
		u.notifyQueued("An image")
	case text != "":
		u.out.addText(text)
		u.notifyQueued("Your clipboard text")
	default:
		u.flashMsg("Nothing to send on the clipboard")
	}
	if u.visible {
		u.paint()
	}
}

// clipboardText reads CF_UNICODETEXT (clipboard already open).
func clipboardText() string {
	h := win(user32, "GetClipboardData", 13)
	if h == 0 {
		return ""
	}
	p := win(kernel32, "GlobalLock", h)
	if p == 0 {
		return ""
	}
	defer win(kernel32, "GlobalUnlock", h)
	n := int(win(kernel32, "GlobalSize", h)) / 2
	buf := make([]uint16, n)
	win(kernel32, "RtlMoveMemory", ptr(&buf[0]), p, uintptr(n*2))
	return syscall.UTF16ToString(buf)
}

// bitmapPNG encodes an HBITMAP as PNG with GDI+, through a temp file.
func bitmapPNG(hbm uintptr) []byte {
	if hbm == 0 {
		return nil
	}
	var bmp uintptr
	if gp("GdipCreateBitmapFromHBITMAP", hbm, 0, ptr(&bmp)) != 0 || bmp == 0 {
		return nil
	}
	defer gp("GdipDisposeImage", bmp)
	f, err := os.CreateTemp("", "phonedrop-*.png")
	if err != nil {
		return nil
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)
	if gp("GdipSaveImageToFile", bmp, ptr(wstr(path)), ptr(&pngEncoder), 0) != 0 {
		return nil
	}
	b, _ := os.ReadFile(path)
	return b
}

var pngEncoder = struct {
	d1     uint32
	d2, d3 uint16
	d4     [8]byte
}{0x557cf406, 0x1a04, 0x11d3, [8]byte{0x9a, 0x73, 0x00, 0x00, 0xf8, 0x1e, 0xf3, 0x2e}}

// pickFiles shows the standard open dialog with multi-select.
func pickFiles() []string {
	clsid := guid{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iid := guid{0xD57C7288, 0xD4AD, 0x4768, [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	var dlg unsafe.Pointer
	if r := win(ole32, "CoCreateInstance", ptr(&clsid), 0, 1, ptr(&iid), uintptr(unsafe.Pointer(&dlg))); int32(r) < 0 {
		return nil
	}
	defer comRelease(dlg)
	const (
		show, setOptions, getOptions, setTitle, getResults = 3, 9, 10, 17, 27
		arrayGetCount, arrayGetItemAt, itemGetDisplayName  = 7, 8, 5
	)
	var opts uint32
	comCall(dlg, getOptions, ptr(&opts))
	comCall(dlg, setOptions, uintptr(opts|0x200|0x40|0x1000)) // ALLOWMULTISELECT | FORCEFILESYSTEM | FILEMUSTEXIST
	comCall(dlg, setTitle, ptr(wstr("Send to your phone")))
	if comCall(dlg, show, 0) != 0 {
		return nil
	}
	var arr unsafe.Pointer
	if r := comCall(dlg, getResults, uintptr(unsafe.Pointer(&arr))); int32(r) < 0 {
		return nil
	}
	defer comRelease(arr)
	var n uint32
	comCall(arr, arrayGetCount, ptr(&n))
	var out []string
	for i := uint32(0); i < n; i++ {
		var item unsafe.Pointer
		if r := comCall(arr, arrayGetItemAt, uintptr(i), uintptr(unsafe.Pointer(&item))); int32(r) < 0 {
			continue
		}
		var path *uint16
		if r := comCall(item, itemGetDisplayName, 0x80058000, uintptr(unsafe.Pointer(&path))); int32(r) >= 0 {
			out = append(out, syscall.UTF16ToString(unsafe.Slice(path, wcslen(path))))
			win(ole32, "CoTaskMemFree", uintptr(unsafe.Pointer(path)))
		}
		comRelease(item)
	}
	return out
}

// registerHotkey (re)registers Ctrl+Alt+P when the setting is on.
func (u *ui) registerHotkey() {
	win(user32, "UnregisterHotKey", u.hwnd, hotkeySend)
	u.hotkeyErr = false
	if !u.set.Hotkey {
		return
	}
	if win(user32, "RegisterHotKey", u.hwnd, hotkeySend, hkMods|modNoRepeat, hkKey) == 0 {
		u.hotkeyErr = true
	}
}

const sendH = 100

func (u *ui) sendHeight() float32 {
	if len(u.phones) == 0 || u.out == nil {
		return 0
	}
	return labelH + sendH
}

// drawSend: the "To your phone" card in the main view, at y.
func (u *ui) drawSend(c *canvas, x, y, w float32) {
	if u.sendHeight() == 0 {
		return
	}
	t := u.th
	c.text("To your phone", fontSmallSemi, rectF{x + 2, y + 10, w, labelH - 10}, t.text2, 0, 1)
	y += labelH
	r := rectF{x, y, w, sendH}
	u.drawCard(c, r, t.card)
	n := u.out.count()
	chip, fg := t.chip, t.text
	title, sub := "Send files or the clipboard", "The phone collects them with Receive from PC"
	if u.set.Hotkey {
		sub = "Ctrl+Alt+P sends the clipboard from anywhere"
		if u.hotkeyErr {
			sub = "Ctrl+Alt+P is taken by another app"
		}
	}
	if n > 0 {
		chip, fg = t.accent, t.onText
		title = strconv.Itoa(n) + " items waiting"
		if n == 1 {
			title = "1 item waiting"
		}
		sub = "Run “Receive from PC” on your iPhone"
	}
	c.circle(r.X+30, r.Y+30, 16, chip)
	c.icon(icSend, n > 0, fontIcon, 20, rectF{r.X + 14, r.Y + 14, 32, 32}, fg)
	c.text(title, fontText, rectF{r.X + 58, r.Y + 11, r.W - 70, 20}, t.text, 0, 1)
	c.text(sub, fontSmall, rectF{r.X + 58, r.Y + 31, r.W - 70, 18}, t.text2, 0, 1)
	by := r.Y + sendH - 12 - 32
	bx := r.X + 58
	bw := c.measure("Clipboard", fontText) + 28
	u.button(c, rectF{bx, by, bw, 32}, "Files…", "btn:sendfiles")
	u.button(c, rectF{bx + bw + 8, by, bw, 32}, "Clipboard", "btn:sendclip")
	if n > 0 {
		cw := c.measure("Clear", fontText) + 28
		u.button(c, rectF{r.X + r.W - 14 - cw, by, cw, 32}, "Clear", "btn:clearout")
	}
}
