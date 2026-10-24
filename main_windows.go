// PhoneDrop: AirDrop-style transfers between an iPhone (Shortcuts, no app) and this PC over the
// local network. One hidden exe with a tray icon; it listens only on Private networks and only
// paired phones can send. Stdlib only; build: build.sh.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/netip"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	previewDir := flag.String("preview", "", "render the panel to PNGs in this folder and exit")
	flag.Parse()
	if *previewDir != "" {
		preview(*previewDir)
		return
	}
	// One instance per sign-in; starting it again (Start menu, Run key) is harmless. Explorer's
	// Send to starts a second one with file arguments, which it hands to the running one.
	files := flag.Args()
	if _, _, err := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(wstr(`Local\PhoneDrop`)))); err == syscall.Errno(183) {
		for i := 0; i < 20 && len(files) > 0 && !forwardToRunning(files); i++ {
			time.Sleep(250 * time.Millisecond) // it may still be starting
		}
		return
	}
	user32.NewProc("SetProcessDpiAwarenessContext").Call(^uintptr(3)) // PER_MONITOR_AWARE_V2 (-4)
	openLog()

	u := &ui{view: "main", set: loadSettings(), th: loadTheme(), scale: 1, recheck: make(chan struct{}, 1),
		startFiles: files}
	theUI = u
	log.Printf("%s starting, port %d", appName, u.set.Port)
	phones := loadPhones(phonesPath(), dpapiProtect, dpapiUnprotect)
	log.Printf("%d paired phone(s)", len(phones.list()))
	srv := newServer(u.set.Port, u, phones, func(err error) {
		log.Printf("listen: %v", err)
		u.setListenErr(err)
	})
	srv.setConfig(serverConfig{SaveDir: u.set.saveDir()})
	if u.set.Encrypt {
		if err := srv.setTLS(true); err != nil {
			log.Printf("encryption: %v", err)
			u.set.Encrypt = false // not saved: it's retried next start, but the settings page mustn't claim it
		}
	}
	u.onEncrypt = srv.setTLS
	u.onSettings = func(st settings) { srv.setConfig(serverConfig{SaveDir: st.saveDir()}) }
	u.app = srvBackend{srv}
	out := newOutbox("", time.Now)
	srv.setOutbox(out)
	u.out = out
	u.hist = loadHistory("")
	setTrusted(u.set.Trusted)
	go watchNetworks(func(st netState) {
		for _, n := range st.networks {
			log.Printf("network %q category=%d addrs=%v", n.name, n.category, n.addrs)
		}
		if st.err != nil {
			log.Printf("network list: %v", st.err)
		}
		listening := srv.bind(st.listen)
		log.Printf("listening on %v", listening)
		u.setNetwork(st, listening)
	}, u.recheck)
	runUI(u)
}

// preview renders the panel states to PNGs without any windows (for checking layout under Wine;
// colours and fonts only look right on real Windows).
func preview(dir string) {
	gdipInit()
	fmt.Println("fonts (0 = missing):", fontText, fontSemi, fontSmall, fontSmallSemi, fontTitle, fontIcon, fontIconFilled)
	for _, size := range []int{16, 24, 32} {
		c := trayCanvas(size, 0xFFFFFFFF)
		c.savePNG(fmt.Sprintf("%s/tray-%d.png", dir, size))
		c.free()
	}
	appIcon(dir + `\app-icon.png`)
	previewMore(dir)

	home := network{name: "HomeNet", category: catPrivate, addrs: []netip.Addr{netip.MustParseAddr("192.168.18.20")}}
	cafe := network{name: "Café Wi-Fi", category: catPublic, addrs: []netip.Addr{netip.MustParseAddr("10.20.0.7")}}
	states := map[string]snap{
		"ready":   {known: true, net: netState{networks: []network{home}}, listening: home.addrs},
		"public":  {known: true, net: netState{networks: []network{cafe}}},
		"offline": {known: true},
		"porterr": {known: true, net: netState{networks: []network{home}}, listenErr: "Only one usage of each socket address"},
	}
	for _, dark := range []bool{true, false} {
		name := map[bool]string{true: "dark", false: "light"}[dark]
		for state, s := range states {
			u := &ui{view: "main", set: defaultSettings(), th: themeFor(dark), scale: 1.5}
			u.net, u.netKnown, u.listening, u.listenErr = s.net, s.known, s.listening, s.listenErr
			if state == "ready" {
				u.hover = "btn:pair"
			}
			c := u.ensureCanvas(&u.cv, panelW, u.height())
			u.render(c)
			c.savePNG(fmt.Sprintf("%s/main-%s-%s.png", dir, state, name))
		}
		u := &ui{view: "settings", set: defaultSettings(), th: themeFor(dark), scale: 1.5, startAtLogin: true}
		u.hover = "sw:autocopy"
		c := u.ensureCanvas(&u.cv, panelW, u.height())
		u.render(c)
		c.savePNG(fmt.Sprintf("%s/settings-%s.png", dir, name))
		u.view, u.quitAt, u.hover = "main", time.Now(), ""
		c = u.ensureCanvas(&u.cv, panelW, u.height())
		u.render(c)
		c.savePNG(fmt.Sprintf("%s/quit-%s.png", dir, name))
	}
}

// appIcon renders the exe/shortcut icon (winres/icon.png, linked in by go-winres): the white
// tray glyph on a Windows-blue rounded square.
func appIcon(path string) {
	c := newCanvas(256, 256, 16)
	c.clear()
	c.fill(rectF{0.5, 0.5, 15, 15}, 3.4, 0xFF0067C0)
	f := newFont([]string{"FluentSystemIcons-Filled"}, iconColl, 10.5, 0)
	defer gp("GdipDeleteFont", f)
	c.icon(icPhoneArrow, true, f, 10.5, rectF{-12, -12, 40, 40}, 0xFFFFFFFF)
	c.savePNG(path)
	c.free()
}
