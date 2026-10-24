package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"sync"
	"time"
)

// server: the phone-facing HTTP listener. It binds only to the laptop's addresses on networks
// Windows calls Private (the network watcher hands it that list), refuses peers that aren't on a
// private range, and turns away IPs that keep failing. Accept blocks, so idle costs no CPU.
//
// Milestone 1: no phone is paired yet, so every request gets the same generic 401.
type server struct {
	port  int
	gate  *gate
	http  *http.Server
	onErr func(error) // listener failures (e.g. port in use), for the UI

	mu        sync.Mutex
	listeners map[netip.Addr]net.Listener
}

func newServer(port int, onErr func(error)) *server {
	s := &server{port: port, gate: newGate(), onErr: onErr, listeners: map[netip.Addr]net.Listener{}}
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
		gl := &gatedListener{Listener: l, gate: s.gate}
		s.listeners[a] = gl
		go s.http.Serve(gl)
	}
	if len(errs) > 0 && s.onErr != nil {
		s.onErr(errors.Join(errs...))
	}
	out := make([]netip.Addr, 0, len(s.listeners))
	for a := range s.listeners {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	ip := peerIP(r.RemoteAddr)
	// Nothing is authenticated yet; later milestones verify the signed headers here first.
	s.gate.fail(ip)
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
