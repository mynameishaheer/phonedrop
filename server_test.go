package main

import (
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestGateBansAfterRepeatedFailures(t *testing.T) {
	g := newGate()
	now := time.Unix(1_000_000, 0)
	g.now = func() time.Time { return now }
	ip := netip.MustParseAddr("192.168.1.50")
	for i := 0; i < g.maxFails; i++ {
		g.fail(ip)
	}
	if !g.allowed(ip) {
		t.Fatal("banned before exceeding maxFails")
	}
	g.fail(ip)
	if g.allowed(ip) {
		t.Fatal("not banned after exceeding maxFails")
	}
	if !g.allowed(netip.MustParseAddr("192.168.1.51")) {
		t.Fatal("another IP was banned too")
	}
	now = now.Add(g.penalty)
	if !g.allowed(ip) {
		t.Fatal("still banned after the penalty expired")
	}
}

func TestGateWindowResets(t *testing.T) {
	g := newGate()
	now := time.Unix(1_000_000, 0)
	g.now = func() time.Time { return now }
	ip := netip.MustParseAddr("10.0.0.7")
	for i := 0; i < 3*g.maxFails; i++ {
		g.fail(ip)
		now = now.Add(g.window / time.Duration(g.maxFails-1)) // slower than the limit
	}
	if !g.allowed(ip) {
		t.Fatal("slow failures got banned")
	}
}

func TestPrivatePeer(t *testing.T) {
	for addr, want := range map[string]bool{
		"192.168.18.5": true, "10.1.2.3": true, "172.16.0.9": true, "169.254.10.10": true,
		"fd12::1": true, "fe80::1": true, "127.0.0.1": true,
		"8.8.8.8": false, "100.64.0.1": false, "2001:4860::8888": false,
	} {
		if got := privatePeer(netip.MustParseAddr(addr)); got != want {
			t.Errorf("privatePeer(%s) = %v, want %v", addr, got, want)
		}
	}
	if privatePeer(netip.Addr{}) {
		t.Error("invalid address counted as private")
	}
}

func TestServerDeniesEverything(t *testing.T) {
	s := newServer(0, func(err error) { t.Error(err) })
	s.port = freePort(t)
	got := s.bind([]netip.Addr{netip.MustParseAddr("127.0.0.1")})
	if len(got) != 1 {
		t.Fatalf("bind returned %v", got)
	}
	base := "http://127.0.0.1:" + itoa(s.port)
	for _, req := range []struct{ method, path string }{{"GET", "/"}, {"POST", "/v1/items"}, {"GET", "/v1/outbox"}} {
		r, _ := http.NewRequest(req.method, base+req.path, strings.NewReader("hello"))
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 401 || strings.TrimSpace(string(body)) != "unauthorized" {
			t.Errorf("%s %s: %d %q", req.method, req.path, resp.StatusCode, body)
		}
	}
	// Unbinding closes the listener.
	if got := s.bind(nil); len(got) != 0 {
		t.Fatalf("still bound: %v", got)
	}
	if _, err := http.Get(base + "/"); err == nil {
		t.Fatal("request succeeded after unbinding")
	}
}

func TestServerTurnsAwayBannedPeer(t *testing.T) {
	s := newServer(0, nil)
	s.port = freePort(t)
	s.bind([]netip.Addr{netip.MustParseAddr("127.0.0.1")})
	defer s.bind(nil)
	base := "http://127.0.0.1:" + itoa(s.port) + "/"
	for i := 0; i <= s.gate.maxFails; i++ {
		resp, err := http.Get(base)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		resp.Body.Close()
	}
	if _, err := http.Get(base); err == nil {
		t.Fatal("banned peer still got an answer")
	}
}
