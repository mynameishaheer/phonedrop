// Pairing in the panel: the QR code view, the "Allow this phone?" prompt and the paired-phones
// list. The server asks for a decision through confirmPairing (an HTTP goroutine blocks there);
// the UI thread shows the prompt and answers on the request's channel.
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	wmConfirm     = 0x8003 // a pairing request is waiting for Allow/Deny
	wmConfirmGone = 0x8004 // it timed out or the phone gave up
	wmReceived    = 0x8005 // items arrived (drained from ui.inbox)

	tTick = 3 // pairing countdown, only while the QR code is on screen

	phoneRowH = 56
	addRowH   = 44
	qrBox     = 232
	docsURL   = "https://github.com/mynameishaheer/phonedrop/blob/main/docs/SHORTCUTS.md"
)

// backend: what the panel needs from the server.
type backend interface {
	startPairing() (qrJSON string, expires time.Time, err error)
	cancelPairing()
	phoneList() []phoneView
	removePhone(id string)
}

type phoneView struct {
	ID, Name         string
	Paired, LastSeen time.Time
}

type confirmReq struct {
	name  string
	reply chan bool
}

// srvBackend adapts the server and its phone store to the panel.
type srvBackend struct{ srv *server }

func (b srvBackend) startPairing() (string, time.Time, error) {
	qr, exp := b.srv.startPairing(pairingInfo{})
	return qr, exp, nil
}

func (b srvBackend) cancelPairing() { b.srv.cancelPairing() }

func (b srvBackend) phoneList() []phoneView {
	var out []phoneView
	for _, p := range b.srv.phones.list() {
		out = append(out, phoneView{ID: p.ID, Name: p.Name, Paired: p.Paired, LastSeen: p.LastSeen})
	}
	return out
}

func (b srvBackend) removePhone(id string) {
	if err := b.srv.phones.remove(id); err != nil {
		log.Printf("remove phone %s: %v", id, err)
	}
}

// ---------- appHooks, called from server goroutines ----------

func (u *ui) confirmPairing(ctx context.Context, name string) bool {
	req := &confirmReq{name: name, reply: make(chan bool, 1)}
	u.mu.Lock()
	u.pending = req
	u.mu.Unlock()
	postMsg.Call(u.hwnd, wmConfirm, 0, 0)
	select {
	case ok := <-req.reply:
		return ok
	case <-ctx.Done():
		u.mu.Lock()
		if u.pending == req {
			u.pending = nil
		}
		u.mu.Unlock()
		postMsg.Call(u.hwnd, wmConfirmGone, 0, 0)
		return false
	}
}

func (u *ui) phonesChanged() { postMsg.Call(u.hwnd, wmRefresh, 0, 0) }

// ---------- UI thread ----------

func (u *ui) beginPairing() {
	qrJSON, exp, err := u.app.startPairing()
	if err == nil {
		u.qr, err = qrEncode([]byte(qrJSON))
	}
	if err != nil {
		u.flashMsg("Couldn't start pairing")
		u.paint()
		return
	}
	u.pairExpires = exp
	u.view, u.hover = "pair", ""
	u.relayout()
	win(user32, "SetTimer", u.hwnd, tTick, 1000, 0)
}

// endPairing leaves the QR view; the code stops working at once.
func (u *ui) endPairing(msg string) {
	win(user32, "KillTimer", u.hwnd, tTick)
	if u.view == "pair" {
		u.app.cancelPairing()
		u.qr = nil
		u.view, u.hover = "main", ""
		if msg != "" {
			u.flashMsg(msg)
		}
		if u.visible && !u.hiding {
			u.relayout()
		}
	}
}

func (u *ui) tick() {
	if u.view != "pair" {
		win(user32, "KillTimer", u.hwnd, tTick)
		return
	}
	if !time.Now().Before(u.pairExpires) {
		u.endPairing("The pairing code expired")
		return
	}
	u.paint()
}

// askConfirm shows the Allow/Deny prompt for the waiting request, opening the panel if needed.
func (u *ui) askConfirm() {
	u.mu.Lock()
	req := u.pending
	u.mu.Unlock()
	if req == nil {
		return
	}
	win(user32, "KillTimer", u.hwnd, tTick) // the QR code has been used
	u.confirm, u.qr = req, nil
	if u.visible && !u.hiding {
		u.view, u.hover = "confirm", ""
		u.relayout()
		win(user32, "SetForegroundWindow", u.hwnd)
	} else {
		u.show("confirm")
	}
}

func (u *ui) answer(ok bool) {
	req := u.confirm
	u.confirm = nil
	u.mu.Lock()
	if u.pending == req {
		u.pending = nil
	}
	u.mu.Unlock()
	if req != nil {
		req.reply <- ok
		if ok {
			u.flashMsg("Paired with " + req.name)
		}
	}
	u.view, u.hover = "main", ""
	u.relayout()
}

func (u *ui) confirmGone() {
	u.mu.Lock()
	stillWaiting := u.confirm != nil && u.pending == u.confirm
	u.mu.Unlock()
	if u.confirm == nil || stillWaiting {
		return
	}
	u.confirm = nil
	if u.view == "confirm" {
		u.view, u.hover = "main", ""
		u.flashMsg("The pairing request timed out")
		if u.visible && !u.hiding {
			u.relayout()
		}
	}
}

// remove: the first click on a phone's bin arms it, a second within 3 s removes the phone.
func (u *ui) remove(id string) {
	if u.removeID == id && time.Since(u.removeAt) < 3*time.Second {
		u.app.removePhone(id)
		u.removeID = ""
		u.flashMsg("Removed. It can't send anything now")
		u.relayout()
		return
	}
	u.removeID, u.removeAt = id, time.Now()
	win(user32, "SetTimer", u.hwnd, tFlash, 3100, 0)
	u.paint()
}

// ---------- drawing ----------

func (u *ui) phonesHeight() float32 {
	if len(u.phones) == 0 {
		return rowH
	}
	return float32(len(u.phones))*phoneRowH + addRowH
}

func (u *ui) pairHeight() float32 { return 48 + qrBox + 16 + 52 + 12 + 32 + pad }

func (u *ui) confirmHeight() float32 { return 28 + 56 + 16 + 24 + 22 + 28 + 32 + 20 }

func ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case t.IsZero():
		return "never"
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	}
	return t.Format("2 Jan")
}

// drawPhones: the paired phones card in the main view, starting at y.
func (u *ui) drawPhones(c *canvas, x, y, w float32) {
	t := u.th
	if len(u.phones) == 0 {
		r := rectF{x, y, w, rowH}
		u.drawCard(c, r, t.card)
		c.circle(r.X+30, r.Y+rowH/2, 16, t.chip)
		c.icon(icPhoneAdd, false, fontIcon, 20, rectF{r.X + 14, r.Y + rowH/2 - 16, 32, 32}, t.text)
		bw := c.measure("Pair", fontText) + 32
		c.text("No phones paired yet", fontText, rectF{r.X + 58, r.Y + 13, r.W - 58 - bw - 24, 20}, t.text, 0, 1)
		c.text("Scan a QR code with your iPhone", fontSmall, rectF{r.X + 58, r.Y + 33, r.W - 58 - bw - 24, 18}, t.text2, 0, 1)
		u.accentButton(c, rectF{r.X + r.W - 14 - bw, r.Y + (rowH-32)/2, bw, 32}, "Pair", "btn:pair")
		return
	}
	u.drawCard(c, rectF{x, y, w, u.phonesHeight()}, t.card)
	now := time.Now()
	for i, p := range u.phones {
		r := rectF{x, y + float32(i)*phoneRowH, w, phoneRowH}
		rowID, rmID := "row:"+p.ID, "rm:"+p.ID
		if i > 0 {
			c.fill(rectF{x + 1, r.Y, w - 2, 1}, 0, t.divider)
		}
		c.circle(r.X+30, r.Y+phoneRowH/2, 16, t.chip)
		c.icon(icPhone, false, fontIcon, 20, rectF{r.X + 14, r.Y + phoneRowH/2 - 16, 32, 32}, t.text)
		c.text(p.Name, fontText, rectF{r.X + 58, r.Y + 9, r.W - 58 - 96, 20}, t.text, 0, 1)
		c.text("Active "+ago(p.LastSeen, now)+" · paired "+p.Paired.Format("2 Jan"), fontSmall,
			rectF{r.X + 58, r.Y + 29, r.W - 58 - 96, 18}, t.text2, 0, 1)
		u.addHit(r, rowID)
		armed := u.removeID == p.ID && time.Since(u.removeAt) < 3*time.Second
		switch {
		case armed:
			b := rectF{r.X + r.W - 12 - 84, r.Y + (phoneRowH-30)/2, 84, 30}
			c.fill(b, 4, alpha(t.danger, 0.18))
			c.text("Remove", fontText, b, t.danger, 1, 1)
			u.addHit(b, rmID)
		case u.hover == rowID || u.hover == rmID:
			b := rectF{r.X + r.W - 12 - 32, r.Y + (phoneRowH-32)/2, 32, 32}
			if u.hover == rmID {
				c.fill(b, 4, alpha(t.text, 0.08))
			}
			c.icon(icDelete, false, fontIconSm, 16, b, t.text2)
			u.addHit(b, rmID)
		}
	}
	r := rectF{x, y + float32(len(u.phones))*phoneRowH, w, addRowH}
	c.fill(rectF{x + 1, r.Y, w - 2, 1}, 0, t.divider)
	if u.hover == "btn:pair" {
		c.fill(rectF{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}, 5, alpha(t.text, 0.04))
	}
	c.icon(icAdd, false, fontIconSm, 16, rectF{r.X + 14, r.Y, 32, r.H}, t.accent)
	c.text("Pair another phone", fontText, rectF{r.X + 58, r.Y, r.W - 70, r.H}, t.text, 0, 1)
	u.addHit(r, "btn:pair")
}

func (u *ui) drawPair(c *canvas, p rectF) {
	t := u.th
	x, w := p.X+pad, float32(panelW-2*pad)
	u.iconButton(c, rectF{x - 4, p.Y + 8, 36, 32}, icBack, "btn:back")
	c.text("Pair a phone", fontTitle, rectF{x + 40, p.Y + 8, w - 40, 32}, t.text, 0, 1)
	y := p.Y + 48

	// The code, black on white in both themes (scanners want contrast), drawn on whole device
	// pixels so modules stay crisp.
	box := rectF{p.X + (panelW-qrBox)/2, y, qrBox, qrBox}
	c.fill(box, 8, 0xFFFFFFFF)
	if !t.dark {
		c.stroke(box, 8, 1, t.cardLine)
	}
	if n := len(u.qr); n > 0 {
		modules := n + 8 // quiet zone of 4 on each side
		px := float32(int(float32(qrBox-8)*u.scale) / modules)
		if px < 1 {
			px = 1
		}
		m := px / u.scale
		size := m * float32(modules)
		ox := float32(int((box.X+(qrBox-size)/2+4*m)*u.scale)) / u.scale
		oy := float32(int((box.Y+(qrBox-size)/2+4*m)*u.scale)) / u.scale
		var rs []rectF
		for r, row := range u.qr {
			for col, dark := range row {
				if dark {
					rs = append(rs, rectF{ox + float32(col)*m, oy + float32(r)*m, m, m})
				}
			}
		}
		c.rects(rs, 0xFF000000)
	}
	y += qrBox + 16

	step := func(n, text string, yy float32) {
		c.circle(x+12, yy+10, 10, t.chip)
		c.text(n, fontSmallSemi, rectF{x + 2, yy, 20, 20}, t.text, 1, 1)
		c.text(text, fontText, rectF{x + 32, yy, w - 32, 20}, t.text, 0, 1)
	}
	step("1", "On your iPhone, run PhoneDrop Setup", y)
	step("2", "Scan this code, then click Allow here", y+28)
	y += 52 + 12

	left := time.Until(u.pairExpires).Round(time.Second)
	if left < 0 {
		left = 0
	}
	c.icon(icLock, false, fontIconSm, 14, rectF{x, y, 18, 32}, t.text2)
	c.text(fmt.Sprintf("Single use · expires in %d:%02d", int(left.Minutes()), int(left.Seconds())%60),
		fontSmall, rectF{x + 22, y, w - 150, 32}, t.text2, 0, 1)
	bw := c.measure("Get the shortcut", fontSmall) + 20
	lr := rectF{x + w - bw, y, bw, 32}
	if u.hover == "btn:docs" {
		c.fill(lr, 4, alpha(t.text, 0.06))
	}
	c.text("Get the shortcut", fontSmall, lr, t.accent, 1, 1)
	u.addHit(lr, "btn:docs")
}

func (u *ui) drawConfirm(c *canvas, p rectF) {
	t := u.th
	x, w := p.X+pad, float32(panelW-2*pad)
	name := "a phone"
	if u.confirm != nil {
		name = u.confirm.name
	}
	y := p.Y + 28
	cx := p.X + panelW/2
	c.circle(cx, y+28, 28, t.accent)
	c.icon(icPhone, true, fontIcon, 24, rectF{cx - 28, y, 56, 56}, t.onText)
	y += 56 + 16
	c.text("Allow “"+name+"”?", fontTitle, rectF{x, y, w, 24}, t.text, 1, 1)
	y += 24
	c.text("It will be able to send photos, files and text here", fontSmall, rectF{x, y, w, 22}, t.text2, 1, 1)
	y += 22 + 28
	bw := (w - 8) / 2
	u.button(c, rectF{x, y, bw, 32}, "Deny", "btn:deny")
	u.accentButton(c, rectF{x + bw + 8, y, bw, 32}, "Allow", "btn:allow")
}
