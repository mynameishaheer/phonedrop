package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"sync"
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
	s := newServer(0, nil, nil, func(err error) { t.Error(err) })
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
	s := newServer(0, nil, nil, nil)
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

// fakeHooks answers pairing confirmations from the test.
type fakeHooks struct {
	asked   chan string
	answer  chan bool
	changed chan struct{}
}

func newFakeHooks() *fakeHooks {
	return &fakeHooks{asked: make(chan string, 4), answer: make(chan bool, 4), changed: make(chan struct{}, 4)}
}

func (h *fakeHooks) confirmPairing(ctx context.Context, name string) bool {
	h.asked <- name
	select {
	case a := <-h.answer:
		return a
	case <-ctx.Done():
		return false
	}
}
func (h *fakeHooks) received(receivedItem) {}
func (h *fakeHooks) phonesChanged()        { h.changed <- struct{}{} }

// testServer: a server on 127.0.0.1 with a fake clock (shared by the phone client).
type testServer struct {
	*server
	base  string
	clock time.Time
	mu    sync.Mutex
}

func newTestServer(t *testing.T, hooks appHooks) *testServer {
	ts := &testServer{clock: time.Date(2026, 10, 24, 17, 15, 24, 0, time.UTC)}
	ts.server = newServer(freePort(t), hooks, loadPhones("", nil, nil), func(err error) { t.Error(err) })
	ts.now = ts.time
	ts.phones.now = ts.time
	ts.laptop = "TESTPC"
	ts.bind([]netip.Addr{netip.MustParseAddr("127.0.0.1")})
	t.Cleanup(func() { ts.bind(nil) })
	ts.base = "http://127.0.0.1:" + itoa(ts.port)
	return ts
}

func (ts *testServer) time() time.Time { ts.mu.Lock(); defer ts.mu.Unlock(); return ts.clock }
func (ts *testServer) advance(d time.Duration) {
	ts.mu.Lock()
	ts.clock = ts.clock.Add(d)
	ts.mu.Unlock()
}

// phoneClient signs requests the way the Shortcuts do.
type phoneClient struct {
	ts         *testServer
	id, secret string
	n          int
}

type resp struct {
	code int
	body string
}

func (c *phoneClient) do(t *testing.T, method, path string, body []byte) resp {
	t.Helper()
	c.n++
	ts := c.ts.time().In(time.FixedZone("", 5*3600)).Format(time.RFC3339)
	r := signedReq(method, path, c.id, c.secret, ts, fmt.Sprintf("nonce-%d-%d", c.n, time.Now().UnixNano()), body)
	return send(t, c.ts.base, r)
}

func send(t *testing.T, base string, r *http.Request) resp {
	t.Helper()
	req, _ := http.NewRequest(r.Method, base+r.URL.RequestURI(), r.Body)
	req.Header = r.Header
	req.ContentLength = r.ContentLength
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", r.Method, r.URL.Path, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{res.StatusCode, strings.TrimSpace(string(b))}
}

// pairClient opens a pairing window and returns a client holding the QR code's id and secret.
func pairClient(t *testing.T, ts *testServer) *phoneClient {
	t.Helper()
	qr, exp := ts.startPairing(pairingInfo{})
	if !exp.Equal(ts.time().Add(pairWindow)) {
		t.Fatalf("expires %v", exp)
	}
	var q qrPayload
	if err := json.Unmarshal([]byte(qr), &q); err != nil {
		t.Fatal(err)
	}
	return &phoneClient{ts: ts, id: q.ID, secret: q.Secret}
}

// pairAsync posts /v1/pair in the background.
func (c *phoneClient) pairAsync(t *testing.T, body []byte) chan resp {
	ch := make(chan resp, 1)
	go func() { ch <- c.do(t, "POST", "/v1/pair", body) }()
	return ch
}

func TestPairingQRCode(t *testing.T) {
	ts := newTestServer(t, newFakeHooks())
	qr, _ := ts.startPairing(pairingInfo{})
	var keys []string
	dec := json.NewDecoder(strings.NewReader(qr))
	dec.Token()
	for dec.More() {
		k, _ := dec.Token()
		keys = append(keys, k.(string))
		var v json.RawMessage
		dec.Decode(&v)
	}
	if got := strings.Join(keys, ","); got != "pd,name,host,ips,port,id,secret" {
		t.Fatalf("key order %s", got)
	}
	var q qrPayload
	json.Unmarshal([]byte(qr), &q)
	if q.PD != 1 || q.Name != "TESTPC" || q.Port != ts.port || len(q.IPs) != 1 || q.IPs[0] != "127.0.0.1" ||
		!strings.HasSuffix(q.Host, ".local") || q.Host != strings.ToLower(q.Host) ||
		!validPhoneID(q.ID) || !validSecret(q.Secret) {
		t.Fatalf("QR %s", qr)
	}
	qr2, _ := ts.startPairing(pairingInfo{Name: "N", Host: "h.local", IPs: []netip.Addr{}})
	if !strings.HasPrefix(qr2, `{"pd":1,"name":"N","host":"h.local","ips":[],"port":`) {
		t.Fatalf("QR with overrides %s", qr2)
	}
	if strings.Contains(qr2, q.Secret) || strings.Contains(qr2, q.ID) {
		t.Fatal("QR codes reuse credentials")
	}
}

func TestPairingAllowJSONName(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	c := pairClient(t, ts)
	done := c.pairAsync(t, []byte(`{"name": "  Shaheer's\t‮iPhone\u0007  "}`))
	if name := <-h.asked; name != "Shaheer's iPhone" {
		t.Fatalf("asked about %q", name)
	}
	h.answer <- true
	if r := <-done; r.code != 200 || r.body != `{"ok":true,"laptop":"TESTPC"}` {
		t.Fatalf("pair: %+v", r)
	}
	<-h.changed
	p, ok := ts.phones.get(c.id)
	if !ok || p.Name != "Shaheer's iPhone" || !p.Paired.Equal(ts.time()) {
		t.Fatalf("stored %+v %v", p, ok)
	}
	if sec, _ := ts.phones.secretFor(c.id); sec != c.secret {
		t.Fatal("stored secret differs from the QR code's")
	}
	// The phone can now make signed requests.
	ts.advance(time.Minute)
	if r := c.do(t, "GET", "/v1/ping", nil); r.code != 200 ||
		r.body != `{"ok":true,"laptop":"TESTPC","version":"`+version+`","tls":false}` {
		t.Fatalf("ping: %+v", r)
	}
	if p, _ := ts.phones.get(c.id); !p.LastSeen.Equal(ts.time()) {
		t.Fatal("ping didn't touch LastSeen")
	}
	// The QR code is spent.
	if r := c.do(t, "POST", "/v1/pair", []byte("again")); r.code != 401 || r.body != "unauthorized" {
		t.Fatalf("reused QR: %+v", r)
	}
}

func TestPairingRawTextName(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	for body, want := range map[string]string{
		"Shaheer's \"best\"   iPhone\n": `Shaheer's "best" iPhone`,
		"":                              "iPhone",
		strings.Repeat("é", 100):        strings.Repeat("é", 64),
		`{"name": ""}`:                  "iPhone",
	} {
		c := pairClient(t, ts)
		var b []byte
		if body != "" {
			b = []byte(body)
		}
		done := c.pairAsync(t, b)
		if name := <-h.asked; name != want {
			t.Errorf("body %q: asked about %q", body, name)
		}
		h.answer <- true
		if r := <-done; r.code != 200 {
			t.Fatalf("pair: %+v", r)
		}
		<-h.changed
	}
	c := pairClient(t, ts)
	if r := c.do(t, "POST", "/v1/pair", []byte(`{"name": broken`)); r.code != 400 {
		t.Fatalf("broken JSON: %+v", r)
	}
	c = pairClient(t, ts)
	if r := c.do(t, "POST", "/v1/pair", bytes.Repeat([]byte("x"), maxPairBody+1)); r.code != 413 {
		t.Fatalf("big body: %+v", r)
	}
}

func TestPairingDenied(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	c := pairClient(t, ts)
	done := c.pairAsync(t, []byte("Phone"))
	<-h.asked
	h.answer <- false
	if r := <-done; r.code != 403 || r.body != `{"ok":false,"error":"denied"}` {
		t.Fatalf("pair: %+v", r)
	}
	if len(ts.phones.list()) != 0 {
		t.Fatal("denied phone stored")
	}
	if r := c.do(t, "POST", "/v1/pair", []byte("Phone")); r.code != 401 {
		t.Fatalf("retry after deny: %+v", r)
	}
	if r := c.do(t, "GET", "/v1/ping", nil); r.code != 401 {
		t.Fatalf("denied phone pinged: %+v", r)
	}
}

func TestPairingTimesOut(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	ts.confirmTimeout = 50 * time.Millisecond
	c := pairClient(t, ts)
	if r := c.do(t, "POST", "/v1/pair", []byte("Phone")); r.code != 403 {
		t.Fatalf("pair: %+v", r)
	}
	<-h.asked
	if len(ts.phones.list()) != 0 {
		t.Fatal("phone stored without an answer")
	}
}

func TestPairingWindowExpires(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	c := pairClient(t, ts)
	ts.advance(pairWindow)
	if r := c.do(t, "POST", "/v1/pair", []byte("Phone")); r.code != 401 {
		t.Fatalf("pair after the window: %+v", r)
	}
	select {
	case <-h.asked:
		t.Fatal("user asked about an expired pairing")
	default:
	}
}

func TestPairingCancel(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	c := pairClient(t, ts)
	ts.cancelPairing()
	if r := c.do(t, "POST", "/v1/pair", []byte("Phone")); r.code != 401 {
		t.Fatalf("pair after cancel: %+v", r)
	}
	// Cancelling while the user is being asked answers "denied".
	c = pairClient(t, ts)
	done := c.pairAsync(t, []byte("Phone"))
	<-h.asked
	ts.cancelPairing()
	if r := <-done; r.code != 403 {
		t.Fatalf("pair cancelled mid-confirmation: %+v", r)
	}
	// A new QR code also replaces the old one.
	old := pairClient(t, ts)
	pairClient(t, ts)
	if r := old.do(t, "POST", "/v1/pair", []byte("Phone")); r.code != 401 {
		t.Fatalf("replaced QR: %+v", r)
	}
	if len(ts.phones.list()) != 0 {
		t.Fatal("phone stored")
	}
}

func TestPairingBadSignatureKeepsWindowOpen(t *testing.T) {
	h := newFakeHooks()
	ts := newTestServer(t, h)
	c := pairClient(t, ts)
	wrong := &phoneClient{ts: ts, id: c.id, secret: strings.Repeat("0", 64)}
	if r := wrong.do(t, "POST", "/v1/pair", []byte("Evil")); r.code != 401 || r.body != "unauthorized" {
		t.Fatalf("forged pair: %+v", r)
	}
	done := c.pairAsync(t, []byte("Phone"))
	<-h.asked
	h.answer <- true
	if r := <-done; r.code != 200 {
		t.Fatalf("real pair: %+v", r)
	}
}

func TestClaimIsSingleFlight(t *testing.T) {
	ts := newTestServer(t, newFakeHooks())
	ts.startPairing(pairingInfo{})
	p := ts.pendingPairing()
	ts.confirming = &confirmation{cancel: func() {}}
	if err := ts.claim(p, &confirmation{}); !errors.Is(err, errBusy) {
		t.Fatalf("claim during another confirmation: %v", err)
	}
	if ts.pendingPairing() != nil {
		t.Fatal("busy attempt didn't burn the pairing")
	}
	if err := ts.claim(p, &confirmation{}); err == nil || errors.Is(err, errBusy) {
		t.Fatalf("second claim: %v", err)
	}
}

func TestAuthenticatedRoutes(t *testing.T) {
	ts := newTestServer(t, newFakeHooks())
	c := &phoneClient{ts: ts, id: "0123456789abcdef", secret: secretA}
	if err := ts.phones.add(c.id, c.secret, "P"); err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		method, path string
		body         []byte
		code         int
	}{
		{"GET", "/v1/ping", nil, 200},
		{"POST", "/v1/ping", nil, 405},
		{"GET", "/v1/ping", []byte("x"), 200},   // a body with a correct hash is tolerated
		{"POST", "/v1/items", []byte("x"), 200}, // text: nothing is written
		{"GET", "/v1/items", nil, 405},
		{"GET", "/v1/outbox", nil, 501},
		{"GET", "/v1/outbox/a1b2c3d4", nil, 501},
		{"DELETE", "/v1/outbox", nil, 405},
		{"GET", "/v1/nothing", nil, 404},
		{"GET", "/v1/pair", nil, 401}, // paired phones can't use the pairing endpoint
	} {
		r := c.do(t, x.method, x.path, x.body)
		if r.code != x.code {
			t.Errorf("%s %s: %+v", x.method, x.path, r)
		}
		if x.code != 401 && !strings.HasPrefix(r.body, `{"ok":`) {
			t.Errorf("%s %s: not JSON: %s", x.method, x.path, r.body)
		}
	}
	// "-" with a body is a body error, not a silent success.
	ts2 := c.ts.time().Format(time.RFC3339)
	r := signedReq("GET", "/v1/ping", c.id, c.secret, ts2, "dash-with-body", nil)
	r.Body = io.NopCloser(strings.NewReader("sneaky"))
	r.ContentLength = 6
	if got := send(t, ts.base, r); got.code != 400 || !strings.Contains(got.body, "body hash mismatch") {
		t.Fatalf("- with body: %+v", got)
	}
	// Wrong body hash.
	r = signedReq("GET", "/v1/ping", c.id, c.secret, ts2, "wrong-hash-1", []byte("one"))
	r.Body = io.NopCloser(strings.NewReader("two"))
	if got := send(t, ts.base, r); got.code != 400 {
		t.Fatalf("wrong body: %+v", got)
	}
	// Replay of a good request.
	r = signedReq("GET", "/v1/ping", c.id, c.secret, ts2, "replayed-nonce", nil)
	if got := send(t, ts.base, r); got.code != 200 {
		t.Fatalf("first: %+v", got)
	}
	r = signedReq("GET", "/v1/ping", c.id, c.secret, ts2, "replayed-nonce", nil)
	if got := send(t, ts.base, r); got.code != 401 {
		t.Fatalf("replay: %+v", got)
	}
	// Removing the phone revokes it at once.
	ts.phones.remove(c.id)
	if got := c.do(t, "GET", "/v1/ping", nil); got.code != 401 {
		t.Fatalf("removed phone: %+v", got)
	}
}

func TestAuthenticatedRequestsDontCountAsFailures(t *testing.T) {
	ts := newTestServer(t, newFakeHooks())
	c := &phoneClient{ts: ts, id: "0123456789abcdef", secret: secretA}
	ts.phones.add(c.id, c.secret, "P")
	for i := 0; i < 3*ts.gate.maxFails; i++ {
		if r := c.do(t, "GET", "/v1/ping", nil); r.code != 200 {
			t.Fatalf("ping %d: %+v", i, r)
		}
	}
}
