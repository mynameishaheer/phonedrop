// Received items on the UI side: each one is recorded in the history, text goes on the clipboard
// when auto-copy is on, a toast pops up, and the panel's Recent list shows the last few.
package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	recentShown = 3
	recentRowH  = 48
)

// received (appHooks) runs on a request goroutine; the UI thread picks the item up.
func (u *ui) received(it receivedItem) {
	u.mu.Lock()
	u.inbox = append(u.inbox, it)
	u.mu.Unlock()
	postMsg.Call(u.hwnd, wmReceived, 0, 0)
}

func (u *ui) drainInbox() {
	u.mu.Lock()
	items := u.inbox
	u.inbox = nil
	u.mu.Unlock()
	for _, it := range items {
		t := toastItem{kind: it.Kind, title: it.Name, from: it.From, path: it.Path, text: it.Text, size: it.Size}
		if (it.Kind == "text" || it.Kind == "url") && u.set.AutoCopy {
			t.copied = setClipboardText(u.hwnd, it.Text)
		}
		if u.hist != nil {
			u.hist.add(historyEntry{Kind: it.Kind, Name: it.Name, Path: it.Path, Text: it.Text, Size: it.Size,
				From: it.From, Time: it.Time, Dir: "in"})
		}
		u.showToast(t)
	}
	if len(items) > 0 && u.visible && !u.hiding && u.view == "main" {
		u.refreshRecent()
		u.relayout()
	}
}

func (u *ui) refreshRecent() {
	if u.hist != nil {
		u.recent = u.hist.recent(recentShown)
	}
}

func (u *ui) recentHeight() float32 {
	if len(u.recent) == 0 {
		return 0
	}
	return labelH + float32(len(u.recent))*recentRowH
}

func entryTitle(e historyEntry) string {
	switch e.Kind {
	case "text":
		return firstLine(e.Text)
	case "url":
		return e.Text
	}
	return e.Name
}

// drawRecent: the last few transfers, at y.
func (u *ui) drawRecent(c *canvas, x, y, w float32) {
	if len(u.recent) == 0 {
		return
	}
	t := u.th
	c.text("Recent", fontSmallSemi, rectF{x + 2, y + 10, w, labelH - 10}, t.text2, 0, 1)
	y += labelH
	u.drawCard(c, rectF{x, y, w, float32(len(u.recent)) * recentRowH}, t.card)
	now := time.Now()
	for i, e := range u.recent {
		r := rectF{x, y + float32(i)*recentRowH, w, recentRowH}
		id, fid := "rec:"+strconv.Itoa(i), "recf:"+strconv.Itoa(i)
		if i > 0 {
			c.fill(rectF{x + 1, r.Y, w - 2, 1}, 0, t.divider)
		}
		if u.hover == id || u.hover == fid {
			c.fill(rectF{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}, 5, alpha(t.text, 0.03))
		}
		ic := kindIcon(e.Kind)
		if e.Dir == "out" {
			ic = icSend
		}
		c.icon(ic, false, fontIcon, 20, rectF{r.X + 10, r.Y, 36, r.H}, t.text2)
		dir := "From " + e.From
		if e.Dir == "out" {
			dir = "Sent to phone"
		}
		gone := e.Path != "" && !exists(e.Path)
		sub := dir + " · " + ago(e.Time, now)
		if gone {
			sub = "Moved or deleted · " + ago(e.Time, now)
		}
		right := float32(16)
		if e.Path != "" && !gone && (u.hover == id || u.hover == fid) {
			b := rectF{r.X + r.W - 12 - 32, r.Y + (recentRowH-32)/2, 32, 32}
			if u.hover == fid {
				c.fill(b, 4, alpha(t.text, 0.08))
			}
			c.icon(icFolderOpen, false, fontIconSm, 16, b, t.text2)
			u.addHit(r, id)
			u.addHit(b, fid)
			right = 52
		} else {
			u.addHit(r, id)
		}
		c.text(entryTitle(e), fontText, rectF{r.X + 52, r.Y + 6, r.W - 52 - right, 20}, t.text, 0, 1)
		c.text(sub, fontSmall, rectF{r.X + 52, r.Y + 26, r.W - 52 - right, 16}, t.text2, 0, 1)
	}
}

// openRecent: files open (risky ones only show in Explorer), links open, text is copied.
func (u *ui) openRecent(id string) {
	folder := strings.HasPrefix(id, "recf:")
	i, err := strconv.Atoi(id[strings.IndexByte(id, ':')+1:])
	if err != nil || i < 0 || i >= len(u.recent) {
		return
	}
	e := u.recent[i]
	switch {
	case e.Kind == "text":
		if setClipboardText(u.hwnd, e.Text) {
			u.flashMsg("Copied to the clipboard")
		}
		u.paint()
		return
	case e.Kind == "url":
		if isLink(e.Text) {
			shellOpen(e.Text)
		}
	case e.Path == "" || !exists(e.Path):
		return
	case folder || isRisky(e.Path):
		showInFolder(e.Path)
	default:
		shellOpen(e.Path)
	}
	u.hide()
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
