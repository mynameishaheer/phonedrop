package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// appHooks: everything the server needs from the rest of the app, so it can be tested on Linux
// with a fake. Hooks are called from request goroutines, never the UI thread.
type appHooks interface {
	// confirmPairing asks the user to allow a new phone; it blocks until they answer or ctx
	// ends (50 s, the phone hanging up, or the pairing being cancelled), and then returns false.
	confirmPairing(ctx context.Context, phoneName string) bool
	// received is called after an item from a phone has been saved (file) or read (text/url).
	received(it receivedItem)
	// phonesChanged is called when a phone is paired, so the UI can redraw.
	phonesChanged()
	// sent is called after an outbox item reached the phone: a text or link once listed, a file
	// once its whole body has been sent (outbox.go). The item is already out of the outbox, so
	// the UI can record history and refresh its count.
	sent(it outboxItem, phoneName string)
}

// receivedItem: one item from a phone.
type receivedItem struct {
	Kind string // "image", "video", "file", "text" or "url"
	Name string // saved file name (files)
	Path string // full path of the saved file (files)
	Text string // the text or link (text, url)
	Size int64
	From string // the phone's name
	Time time.Time
}

// noHooks denies every pairing and ignores events: the default until the UI is wired up.
type noHooks struct{}

func (noHooks) confirmPairing(context.Context, string) bool { return false }
func (noHooks) received(receivedItem)                       {}
func (noHooks) phonesChanged()                              {}
func (noHooks) sent(outboxItem, string)                     {}

// server: the phone-facing HTTP listener. It binds only to the laptop's addresses on networks
// Windows calls Private (the network watcher hands it that list), refuses peers that aren't on a
// private range, and turns away IPs that keep failing. Accept blocks, so idle costs no CPU.
type server struct {
	port           int
	gate           *gate
	http           *http.Server
	onErr          func(error) // listener failures (e.g. port in use), for the UI
	hooks          appHooks
	phones         *phoneStore
	auth           *verifier
	now            func() time.Time
	laptop         string // computer name, as the phone shows it
	confirmTimeout time.Duration
	out            outboxRef   // laptop → phone items (outbox.go); set with setOutbox
	items          itemState   // upload settings and limits (items.go)
	tlsm           *tlsManager // optional TLS: CA, leaf, profile window (tls.go)

	mu        sync.Mutex
	listeners map[netip.Addr]net.Listener

	pairMu     sync.Mutex
	pending    *pendingPair  // the open pairing window, if any
	confirming *confirmation // the pairing confirmation in flight, if any
}

func newServer(port int, hooks appHooks, phones *phoneStore, onErr func(error)) *server {
	if hooks == nil {
		hooks = noHooks{}
	}
	if phones == nil {
		phones = loadPhones("", nil, nil)
	}
	s := &server{port: port, gate: newGate(), onErr: onErr, hooks: hooks, phones: phones, now: time.Now,
		laptop: laptopName(), confirmTimeout: confirmTimeout, listeners: map[netip.Addr]net.Listener{}}
	s.tlsm = newTLSManager()
	s.auth = newVerifier(func() time.Time { return s.now() })
	s.http = &http.Server{
		Handler:           http.HandlerFunc(s.handle),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	return s
}

// bind makes the listeners match addrs: new addresses start listening, gone ones stop.
// It returns the addresses actually listening.
func (s *server) bind(addrs []netip.Addr) []netip.Addr {
	want := map[netip.Addr]bool{}
	for _, a := range addrs {
		want[a] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for a, l := range s.listeners {
		if !want[a] {
			l.Close()
			delete(s.listeners, a)
		}
	}
	var errs []error
	for a := range want {
		if _, ok := s.listeners[a]; ok {
			continue
		}
		l, err := net.Listen("tcp", net.JoinHostPort(a.String(), strconv.Itoa(s.port)))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// The gate drops a peer before the sniffer reads a byte from it.
		sl := newSniffListener(&gatedListener{Listener: l, gate: s.gate}, s.tlsm)
		s.listeners[a] = sl
		go s.http.Serve(sl)
	}
	if len(errs) > 0 && s.onErr != nil {
		s.onErr(errors.Join(errs...))
	}
	listening := s.listeningLocked()
	s.tlsm.setAddrs(listening) // a new address set gets a new leaf certificate
	return listening
}

// listening returns the addresses currently listening.
func (s *server) listening() []netip.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listeningLocked()
}

func (s *server) listeningLocked() []netip.Addr {
	out := make([]netip.Addr, 0, len(s.listeners))
	for a := range s.listeners {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

// handle routes a request. Anything not signed by a paired phone (or, for /v1/pair, by the
// pending pairing) gets deny, whatever the path; only then does the path or method matter.
func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/v1/pair" {
		s.handlePair(w, r)
		return
	}
	a, ok := s.auth.verify(r, s.phones.secretFor)
	if !ok {
		s.unauthorized(w, r)
		return
	}
	s.phones.touch(a.id)
	if !s.httpsOnly(w, r, a.id) {
		return
	}
	switch p := r.URL.Path; {
	case p == "/v1/ping":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		if err := checkBodyLen(r, a.body, maxPairBody); err != nil {
			bodyError(w, err)
			return
		}
		if _, err := readBody(r.Body, a.body, maxPairBody); err != nil {
			bodyError(w, err)
			return
		}
		reply(w, http.StatusOK, struct {
			OK      bool   `json:"ok"`
			Laptop  string `json:"laptop"`
			Version string `json:"version"`
			TLS     bool   `json:"tls"`
		}{true, s.laptop, version, r.TLS != nil})
	case p == "/v1/items":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		ph, ok := s.phones.get(a.id)
		if !ok { // removed while the request was being checked
			s.unauthorized(w, r)
			return
		}
		s.handleItem(w, r, a, ph.Name)
	case p == "/v1/outbox" || strings.HasPrefix(p, "/v1/outbox/"): // milestone 4
		s.handleOutbox(w, r, a)
	case p == caProfilePath:
		s.handleProfile(w, r, a)
	default:
		w.Header().Set("Connection", "close")
		replyErr(w, http.StatusNotFound, "not found")
	}
}

// unauthorized: a failed check counts towards the IP's penalty.
func (s *server) unauthorized(w http.ResponseWriter, r *http.Request) {
	s.gate.fail(peerIP(r.RemoteAddr))
	deny(w)
}

// deny: one answer for every unauthenticated request, so a prober learns nothing. The connection
// is closed rather than reading an unwanted body.
func deny(w http.ResponseWriter) {
	w.Header().Set("Connection", "close")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintln(w, "unauthorized")
}

// reply sends an authenticated JSON response (§4).
func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func replyErr(w http.ResponseWriter, code int, msg string) {
	reply(w, code, struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}{false, msg})
}

// methodNotAllowed, notYet and bodyError leave the body unread (or half read), so they close
// the connection rather than have net/http drain it.
func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	w.Header().Set("Connection", "close")
	replyErr(w, http.StatusMethodNotAllowed, "method not allowed")
}

func notYet(w http.ResponseWriter) {
	w.Header().Set("Connection", "close")
	replyErr(w, http.StatusNotImplemented, "not yet")
}

func bodyError(w http.ResponseWriter, err error) {
	w.Header().Set("Connection", "close")
	switch {
	case errors.Is(err, errTooLarge):
		replyErr(w, http.StatusRequestEntityTooLarge, "too large")
	case errors.Is(err, errBodyHash):
		replyErr(w, http.StatusBadRequest, "body hash mismatch")
	default:
		replyErr(w, http.StatusBadRequest, "incomplete body")
	}
}

func peerIP(remote string) netip.Addr {
	ap, err := netip.ParseAddrPort(remote)
	if err != nil {
		return netip.Addr{}
	}
	return ap.Addr().Unmap()
}

// privatePeer: LAN ranges only (RFC 1918, link-local, IPv6 ULA). Anything else never reaches
// the handler, even if a router forwards the port.
func privatePeer(a netip.Addr) bool {
	return a.IsValid() && (a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsLoopback())
}

// gatedListener drops connections from non-LAN peers and from IPs in the penalty box before
// any HTTP parsing happens.
type gatedListener struct {
	net.Listener
	gate *gate
}

func (l *gatedListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		ip := peerIP(c.RemoteAddr().String())
		if privatePeer(ip) && l.gate.allowed(ip) {
			return c, nil
		}
		c.Close()
	}
}

// gate counts failed requests per IP. More than maxFails within a window puts the IP in the
// penalty box: its connections are closed on accept until the box expires. No timers run; state
// is checked lazily when that IP shows up again.
type gate struct {
	mu       sync.Mutex
	now      func() time.Time
	maxFails int
	window   time.Duration
	penalty  time.Duration
	peers    map[netip.Addr]*peerState
}

type peerState struct {
	fails       int
	windowStart time.Time
	bannedUntil time.Time
}

func newGate() *gate {
	return &gate{now: time.Now, maxFails: 10, window: time.Minute, penalty: 5 * time.Minute,
		peers: map[netip.Addr]*peerState{}}
}

func (g *gate) allowed(ip netip.Addr) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	p := g.peers[ip]
	return p == nil || !g.now().Before(p.bannedUntil)
}

func (g *gate) fail(ip netip.Addr) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	p := g.peers[ip]
	if p == nil {
		if len(g.peers) >= 4096 { // a LAN has far fewer hosts; forget everyone rather than grow
			g.peers = map[netip.Addr]*peerState{}
		}
		p = &peerState{windowStart: now}
		g.peers[ip] = p
	}
	if now.Sub(p.windowStart) > g.window {
		p.fails, p.windowStart = 0, now
	}
	p.fails++
	if p.fails > g.maxFails {
		p.bannedUntil = now.Add(g.penalty)
		p.fails, p.windowStart = 0, now
	}
}
