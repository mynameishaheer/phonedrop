// Trusted networks: Windows marks most new Wi-Fi networks Public, and PhoneDrop doesn't listen on
// Public networks. Rather than asking people to change Windows settings, the panel offers
// "Use here", which trusts that one network (by its Windows network id, so a café with the same
// Wi-Fi name isn't trusted with it). The Windows firewall blocks inbound connections on every
// Public network, so the first time, PhoneDrop asks once (UAC) to add a rule allowing itself on
// the Public profile. PhoneDrop itself still listens only on the networks you trusted.
package main

import (
	"encoding/base64"
	"log"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	wmFirewall       = 0x8006 // the elevated firewall helper finished; wParam 1 = rule added
	firewallRuleName = "PhoneDrop (trusted Public networks)"
)

// trustedIDs: the network ids (guid strings) the watcher may listen on although Windows calls
// them Public. Written by the UI thread, read by the watcher.
var trustedIDs atomic.Pointer[map[string]bool]

func setTrusted(list []trustedNet) {
	m := map[string]bool{}
	for _, t := range list {
		m[t.ID] = true
	}
	trustedIDs.Store(&m)
}

func isTrusted(id guid) bool {
	m := trustedIDs.Load()
	return m != nil && (*m)[id.String()]
}

func (u *ui) isTrustedNet(id string) bool {
	for _, t := range u.set.Trusted {
		if t.ID == id {
			return true
		}
	}
	return false
}

// trustNetwork: "Use here" on a Public network. The firewall rule comes first, so Windows doesn't
// also show its own firewall prompt when PhoneDrop starts listening.
func (u *ui) trustNetwork(n network) {
	id := n.id.String()
	if !u.isTrustedNet(id) {
		u.set.Trusted = append(u.set.Trusted, trustedNet{ID: id, Name: n.name})
		u.set.save()
	}
	if u.set.PublicFirewall {
		u.applyTrusted()
		return
	}
	u.flashMsg("Allow PhoneDrop through the firewall to finish")
	u.paint()
	go func() {
		runtime.LockOSThread() // ShellExecuteEx wants COM initialised on its thread
		win(ole32, "CoInitializeEx", 0, 0x2|0x4 /*APARTMENTTHREADED | DISABLE_OLE1DDE*/)
		ok := addFirewallRule()
		win(ole32, "CoUninitialize")
		w := uintptr(0)
		if ok {
			w = 1
		}
		postMsg.Call(u.hwndA.Load(), wmFirewall, w, 0)
	}()
}

// firewallDone runs on the UI thread when the elevated helper exits.
func (u *ui) firewallDone(ok bool) {
	if ok {
		u.set.PublicFirewall = true
		u.set.save()
		u.flashMsg("Ready on this network")
	} else {
		u.flashMsg("Firewall permission is needed to receive here")
	}
	u.applyTrusted()
}

func (u *ui) untrust(id string) {
	var keep []trustedNet
	for _, t := range u.set.Trusted {
		if t.ID != id {
			keep = append(keep, t)
		}
	}
	u.set.Trusted = keep
	u.set.save()
	u.applyTrusted()
	u.relayout()
}

// applyTrusted hands the list to the watcher and makes it look again at once.
func (u *ui) applyTrusted() {
	setTrusted(u.set.Trusted)
	select {
	case u.recheck <- struct{}{}:
	default:
	}
	if u.visible && !u.hiding {
		u.paint()
	}
}

// addFirewallRule runs an elevated, hidden PowerShell that adds an inbound rule for this exe on
// the Public profile. It blocks until the helper exits (call it off the UI thread) and reports
// whether the rule was added; declining the UAC prompt returns false.
func addFirewallRule() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "Remove-NetFirewallRule -DisplayName " + q(firewallRuleName) + " -ErrorAction SilentlyContinue; " +
		"New-NetFirewallRule -DisplayName " + q(firewallRuleName) + " -Program " + q(exe) +
		" -Direction Inbound -Action Allow -Protocol TCP -Profile Public" +
		" -Description 'PhoneDrop listens only on the Public networks you chose' -ErrorAction Stop | Out-Null"
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, 2*len(u16))
	for i, c := range u16 {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	params := "-NoProfile -NonInteractive -WindowStyle Hidden -EncodedCommand " + base64.StdEncoding.EncodeToString(b)

	sei := struct {
		size, mask       uint32
		hwnd             uintptr
		verb, file       *uint16
		params, dir      *uint16
		show             int32
		instApp, idList  uintptr
		class            *uint16
		hkeyClass        uintptr
		hotkey           uint32
		iconOrMon, proce uintptr
	}{mask: 0x40 /*SEE_MASK_NOCLOSEPROCESS*/, verb: wstr("runas"), file: wstr("powershell.exe"),
		params: wstr(params), show: 0 /*SW_HIDE*/}
	sei.size = uint32(unsafe.Sizeof(sei))
	if r, _, _ := shell32.NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&sei))); r == 0 || sei.proce == 0 {
		log.Printf("firewall: elevation declined or failed")
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(sei.proce))
	syscall.WaitForSingleObject(syscall.Handle(sei.proce), syscall.INFINITE)
	var code uint32
	if err := syscall.GetExitCodeProcess(syscall.Handle(sei.proce), &code); err != nil || code != 0 {
		log.Printf("firewall: helper exited with %d", code)
		return false
	}
	return true
}

// ---------- drawing ----------

func (u *ui) trustedHeight() float32 {
	if len(u.set.Trusted) == 0 {
		return 0
	}
	return labelH + float32(len(u.set.Trusted))*48
}

// drawTrusted: the settings page's list of trusted networks, each with a remove button.
func (u *ui) drawTrusted(c *canvas, x, y, w float32) float32 {
	if len(u.set.Trusted) == 0 {
		return y
	}
	t := u.th
	c.text("Trusted networks", fontSmallSemi, rectF{x + 2, y + 10, w, labelH - 10}, t.text2, 0, 1)
	y += labelH
	u.drawCard(c, rectF{x, y, w, float32(len(u.set.Trusted)) * 48}, t.card)
	for i, n := range u.set.Trusted {
		r := rectF{x, y + float32(i)*48, w, 48}
		if i > 0 {
			c.fill(rectF{x + 1, r.Y, w - 2, 1}, 0, t.divider)
		}
		c.icon(icWifi, false, fontIconSm, 16, rectF{r.X + 10, r.Y, 32, r.H}, t.text2)
		c.text(n.Name, fontText, rectF{r.X + 46, r.Y, r.W - 46 - 52, r.H}, t.text, 0, 1)
		id := "untrust:" + n.ID
		b := rectF{r.X + r.W - 12 - 32, r.Y + 8, 32, 32}
		if u.hover == id {
			c.fill(b, 4, alpha(t.text, 0.08))
		}
		c.icon(icDismiss, false, fontIconSm, 14, b, t.text2)
		u.addHit(b, id)
	}
	return y + float32(len(u.set.Trusted))*48
}
