package main

import (
	"fmt"
	"time"
)

// previewMore renders the pairing views and toasts (see preview in main_windows.go).
func previewMore(dir string) {
	now := time.Now()
	phones := []phoneView{
		{ID: "a1", Name: "Shaheer’s iPhone", Paired: now.AddDate(0, 0, -3), LastSeen: now.Add(-5 * time.Minute)},
		{ID: "b2", Name: "iPad", Paired: now.AddDate(0, -1, 0), LastSeen: now.AddDate(0, 0, -6)},
	}
	qrJSON := `{"pd":1,"name":"DESKTOP-AB12CD","host":"desktop-ab12cd.local","ips":["192.168.1.20"],` +
		`"port":47474,"id":"9f2c1a7e5b3d4c60","secret":"` + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" + `"}`
	qr, _ := qrEncode([]byte(qrJSON))
	for _, dark := range []bool{true, false} {
		name := map[bool]string{true: "dark", false: "light"}[dark]
		render := func(file string, u *ui) {
			c := u.ensureCanvas(&u.cv, panelW, u.height())
			u.render(c)
			c.savePNG(fmt.Sprintf("%s/%s-%s.png", dir, file, name))
		}
		base := func(view string) *ui {
			u := &ui{view: view, set: defaultSettings(), th: themeFor(dark), scale: 1.5, phones: phones}
			u.netKnown = true
			return u
		}
		u := base("main")
		u.hover = "row:a1"
		render("phones", u)
		u = base("main")
		u.removeID, u.removeAt, u.flash, u.flashUntil = "b2", now, "Paired with Shaheer’s iPhone", now.Add(time.Hour)
		render("phones-remove", u)
		u = base("main")
		u.out = fakeOut(3)
		u.recent = []historyEntry{
			{Kind: "image", Name: "IMG_2041.HEIC", Path: `C:\x\IMG_2041.HEIC`, From: "Shaheer’s iPhone", Time: now.Add(-2 * time.Minute), Dir: "in"},
			{Kind: "url", Text: "https://github.com/mynameishaheer/phonedrop", From: "Shaheer’s iPhone", Time: now.Add(-time.Hour), Dir: "in"},
			{Kind: "file", Name: "Report.pdf", From: "Shaheer’s iPhone", Time: now.Add(-26 * time.Hour), Dir: "out"},
		}
		u.hover = "rec:1"
		render("full", u)
		u = base("settings")
		u.set.Encrypt = true
		u.set.Trusted = []trustedNet{{ID: "{X}", Name: "Shaheer Home 5G"}}
		render("settings-encrypt", u)
		u = base("pair")
		u.qr, u.pairExpires = qr, now.Add(102*time.Second)
		render("pair", u)
		for _, scale := range []float32{1, 1.25} { // the QR modules must land on whole pixels
			u.scale = scale // (ensureCanvas replaces the canvas when the scale changes)
			c := u.ensureCanvas(&u.cv, panelW, u.height())
			u.render(c)
			c.savePNG(fmt.Sprintf("%s/pair-%s-%.0f.png", dir, name, scale*100))
		}
		u = base("confirm")
		u.confirm = &confirmReq{name: "Shaheer’s iPhone"}
		u.hover = "btn:allow"
		render("confirm", u)

		for i, items := range [][]toastItem{
			{{kind: "image", title: "IMG_2041.HEIC", from: "Shaheer’s iPhone", path: `C:\x\IMG_2041.HEIC`, size: 2_400_000}},
			{{kind: "url", title: "Link", from: "Shaheer’s iPhone", text: "https://github.com/mynameishaheer/phonedrop"}},
			{{kind: "text", from: "Shaheer’s iPhone", text: "Meet at 7 by the north entrance\nbring the keys", copied: true}},
			{{kind: "file", title: "setup.exe", from: "iPad", path: `C:\x\setup.exe`, size: 52_000_000}},
			{{kind: "image", title: "a.jpg", from: "iPad", path: `C:\x\a.jpg`}, {kind: "video", title: "IMG_2050.MOV", from: "iPad", path: `C:\x\b.mov`}, {kind: "image", title: "c.png", from: "iPad", path: `C:\x\c.png`}},
		} {
			u := base("main")
			u.toast.items = items
			if i == 0 {
				u.toast.hover = "t:open"
			}
			c := u.ensureCanvas(&u.toast.cv, toastW, u.toastHeight())
			u.drawToast(c)
			c.savePNG(fmt.Sprintf("%s/toast%d-%s.png", dir, i, name))
		}
	}
}

type fakeOut int

func (f fakeOut) addFile(string) error         { return nil }
func (f fakeOut) addText(string)               {}
func (f fakeOut) addData(string, []byte) error { return nil }
func (f fakeOut) count() int                   { return int(f) }
func (f fakeOut) clear()                       {}
