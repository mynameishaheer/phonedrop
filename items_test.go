package main

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recHooks records what the server hands to the app.
type recHooks struct {
	*fakeHooks
	mu    sync.Mutex
	items []receivedItem
}

func (h *recHooks) received(it receivedItem) {
	h.mu.Lock()
	h.items = append(h.items, it)
	h.mu.Unlock()
}

func (h *recHooks) got() []receivedItem {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]receivedItem(nil), h.items...)
}

type itemEnv struct {
	ts    *testServer
	c     *phoneClient
	h     *recHooks
	dir   string // the save folder; it does not exist yet
	outer string // its parent
}

// newItemEnv pairs a phone through the real pairing flow and points the save folder at a fresh
// directory below t.TempDir().
func newItemEnv(t *testing.T) *itemEnv {
	t.Helper()
	h := &recHooks{fakeHooks: newFakeHooks()}
	ts := newTestServer(t, h)
	c := pairClient(t, ts)
	done := c.pairAsync(t, []byte("Test Phone"))
	<-h.asked
	h.answer <- true
	if r := <-done; r.code != 200 {
		t.Fatalf("pair: %+v", r)
	}
	<-h.changed
	outer := t.TempDir()
	e := &itemEnv{ts: ts, c: c, h: h, dir: filepath.Join(outer, "save"), outer: outer}
	ts.setConfig(serverConfig{SaveDir: e.dir, MaxSize: 1 << 30})
	return e
}

var nonceN atomic.Int64

// itemReq builds a signed POST /v1/items. The signature covers signed; the body actually sent is
// sent (the same unless a test wants a hash mismatch).
func (e *itemEnv) itemReq(signed, sent []byte, hdr map[string]string) *http.Request {
	ts := e.ts.time().Format(time.RFC3339)
	r := signedReq("POST", "/v1/items", e.c.id, e.c.secret, ts, "item-nonce-"+itoa(int(nonceN.Add(1))), signed)
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	r.Body = io.NopCloser(bytes.NewReader(sent))
	r.ContentLength = int64(len(sent))
	return r
}

func sendErr(base string, r *http.Request) (resp, error) {
	req, _ := http.NewRequest(r.Method, base+r.URL.RequestURI(), r.Body)
	req.Header = r.Header
	req.ContentLength = r.ContentLength
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{res.StatusCode, strings.TrimSpace(string(b))}, nil
}

func (e *itemEnv) post(t *testing.T, body []byte, hdr map[string]string) resp {
	t.Helper()
	return e.postSigned(t, body, body, hdr)
}

func (e *itemEnv) postSigned(t *testing.T, signed, sent []byte, hdr map[string]string) resp {
	t.Helper()
	r, err := sendErr(e.ts.base, e.itemReq(signed, sent, hdr))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// files lists the save folder (nothing if it doesn't exist).
func (e *itemEnv) files(t *testing.T) []string {
	t.Helper()
	return ls(t, e.dir)
}

func ls(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var names []string
	for _, x := range es {
		names = append(names, x.Name())
	}
	return names
}

func (e *itemEnv) wantFiles(t *testing.T, want ...string) {
	t.Helper()
	got := e.files(t)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("save folder has %q, want %q", got, want)
	}
}

func randBytes(n int, seed int64) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(b)
	// Never accidentally look like text or a known format.
	b[0], b[1], b[2] = 0x00, 0x01, 0x02
	return b
}

func TestItemFileKeepsName(t *testing.T) {
	e := newItemEnv(t)
	body := []byte("not really a jpeg")
	r := e.post(t, body, map[string]string{"X-PD-Name": "Holiday%20Photo%20%E2%9C%93.jpg"})
	if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"Holiday Photo ✓.jpg"}` {
		t.Fatalf("post: %+v", r)
	}
	e.wantFiles(t, "Holiday Photo ✓.jpg")
	got, _ := os.ReadFile(filepath.Join(e.dir, "Holiday Photo ✓.jpg"))
	if !bytes.Equal(got, body) {
		t.Fatalf("content %q", got)
	}
	it := e.h.got()
	if len(it) != 1 || it[0].Kind != "image" || it[0].Name != "Holiday Photo ✓.jpg" ||
		it[0].Path != filepath.Join(e.dir, "Holiday Photo ✓.jpg") || it[0].Size != int64(len(body)) ||
		it[0].From != "Test Phone" || !it[0].Time.Equal(e.ts.time()) {
		t.Fatalf("hook %+v", it)
	}
	if fi, err := os.Stat(e.dir); err != nil || !fi.IsDir() {
		t.Fatal("save folder wasn't created")
	}
	if fi, _ := os.Stat(filepath.Join(e.dir, "Holiday Photo ✓.jpg")); fi.Mode().Perm() != 0o644 {
		t.Fatalf("mode %v", fi.Mode())
	}
}

func TestItemKinds(t *testing.T) {
	e := newItemEnv(t)
	for name, want := range map[string]string{"a.mov": "video", "b.pdf": "file", "c.PNG": "image"} {
		if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": name}); r.code != 200 {
			t.Fatalf("%s: %+v", name, r)
		}
		var kinds []string
		for _, it := range e.h.got() {
			if it.Name == name {
				kinds = append(kinds, it.Kind)
			}
		}
		if len(kinds) != 1 || kinds[0] != want {
			t.Errorf("%s: kind %v, want %s", name, kinds, want)
		}
	}
}

func TestItemSniffsMissingExtension(t *testing.T) {
	e := newItemEnv(t)
	heic := append([]byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic"), make([]byte, 2000)...)
	r := e.post(t, heic, map[string]string{"X-PD-Name": "IMG_2041"})
	if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"IMG_2041.heic"}` {
		t.Fatalf("post: %+v", r)
	}
	e.wantFiles(t, "IMG_2041.heic")
	if it := e.h.got(); len(it) != 1 || it[0].Kind != "image" || it[0].Name != "IMG_2041.heic" {
		t.Fatalf("hook %+v", it)
	}
	// Unrecognised content keeps the bare name, and an explicit kind=file skips the text probe.
	r = e.post(t, []byte("plain words"), map[string]string{"X-PD-Name": "README", "X-PD-Kind": "file"})
	if r.code != 200 || !strings.HasSuffix(r.body, `"saved":"README"}`) {
		t.Fatalf("kind=file: %+v", r)
	}
	e.wantFiles(t, "IMG_2041.heic", "README")
}

func TestItemDuplicateNames(t *testing.T) {
	e := newItemEnv(t)
	for i, want := range []string{"notes.txt", "notes (2).txt", "notes (3).txt"} {
		r := e.post(t, []byte{byte('a' + i)}, map[string]string{"X-PD-Name": "notes.txt"})
		if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"`+want+`"}` {
			t.Fatalf("%d: %+v", i, r)
		}
	}
	e.wantFiles(t, "notes (2).txt", "notes (3).txt", "notes.txt")
	if b, _ := os.ReadFile(filepath.Join(e.dir, "notes (2).txt")); string(b) != "b" {
		t.Fatalf("second copy has %q", b)
	}
}

func TestItemText(t *testing.T) {
	e := newItemEnv(t)
	r := e.post(t, []byte("hello"), map[string]string{"X-PD-Name": "Text"})
	if r.code != 200 || r.body != `{"ok":true,"kind":"text"}` {
		t.Fatalf("post: %+v", r)
	}
	e.wantFiles(t) // nothing on disk, not even the folder's contents
	it := e.h.got()
	if len(it) != 1 || it[0].Kind != "text" || it[0].Text != "hello" || it[0].Size != 5 ||
		it[0].From != "Test Phone" || it[0].Name != "" || it[0].Path != "" {
		t.Fatalf("hook %+v", it)
	}
	// No name at all, and an explicit kind against a name with an extension.
	if r := e.post(t, []byte("héllo\n"), nil); r.code != 200 || r.body != `{"ok":true,"kind":"text"}` {
		t.Fatalf("no name: %+v", r)
	}
	if r := e.post(t, []byte("from a file"), map[string]string{"X-PD-Name": "x.txt", "X-PD-Kind": "text"}); r.code != 200 ||
		r.body != `{"ok":true,"kind":"text"}` {
		t.Fatalf("kind=text: %+v", r)
	}
	e.wantFiles(t)
	if n := len(e.h.got()); n != 3 {
		t.Fatalf("%d hook calls", n)
	}
	// Empty text is refused rather than shown.
	if r := e.post(t, []byte("  \n"), nil); r.code != 400 || !strings.Contains(r.body, "empty") {
		t.Fatalf("empty: %+v", r)
	}
	if n := len(e.h.got()); n != 3 {
		t.Fatalf("empty text reached the app")
	}
}

func TestItemURL(t *testing.T) {
	e := newItemEnv(t)
	r := e.post(t, []byte("https://example.com/a?b=c\n"), map[string]string{"X-PD-Name": "URL"})
	if r.code != 200 || r.body != `{"ok":true,"kind":"url"}` {
		t.Fatalf("post: %+v", r)
	}
	it := e.h.got()
	if len(it) != 1 || it[0].Kind != "url" || it[0].Text != "https://example.com/a?b=c" {
		t.Fatalf("hook %+v", it)
	}
	e.wantFiles(t)
}

func TestItemUnknownLength(t *testing.T) {
	e := newItemEnv(t)
	send := func(body []byte, hdr map[string]string) resp {
		t.Helper()
		r := e.itemReq(body, body, hdr)
		r.ContentLength = -1 // chunked
		got, err := sendErr(e.ts.base, r)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if r := send([]byte("chunked text"), nil); r.body != `{"ok":true,"kind":"text"}` {
		t.Fatalf("text: %+v", r)
	}
	big := append([]byte("%PDF-1.7\n"), randBytes(maxTextBody+5000, 7)...)
	if r := send(big, nil); r.code != 200 || !strings.HasSuffix(r.body, `.pdf"}`) {
		t.Fatalf("big, no name: %+v", r)
	}
	if r := send(big, map[string]string{"X-PD-Name": "big.dat"}); r.code != 200 || !strings.HasSuffix(r.body, `"saved":"big.dat"}`) {
		t.Fatalf("big, named: %+v", r)
	}
	if got, _ := os.ReadFile(filepath.Join(e.dir, "big.dat")); !bytes.Equal(got, big) {
		t.Fatal("content differs")
	}
}

func TestItemLargeUnnamedSplitsHeadAndTail(t *testing.T) {
	e := newItemEnv(t)
	png := append([]byte("\x89PNG\r\n\x1a\n"), randBytes(2<<20, 1)...)
	r := e.post(t, png, nil)
	want := "Received 2026-10-24 17.15.24.png"
	if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"`+want+`"}` {
		t.Fatalf("post: %+v", r)
	}
	e.wantFiles(t, want)
	if got, _ := os.ReadFile(filepath.Join(e.dir, want)); !bytes.Equal(got, png) {
		t.Fatal("content differs (head/tail seam?)")
	}
	if it := e.h.got(); len(it) != 1 || it[0].Kind != "image" || it[0].Size != int64(len(png)) {
		t.Fatalf("hook %+v", it)
	}
	// Unrecognisable content: .bin. Same second, so the second file gets a (2).
	rnd := randBytes(2<<20, 2)
	r = e.post(t, rnd, nil)
	if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"Received 2026-10-24 17.15.24.bin"}` {
		t.Fatalf("bin: %+v", r)
	}
	// A body just over the text limit that is plain UTF-8 still becomes a file.
	txt := bytes.Repeat([]byte("a"), maxTextBody+1)
	r = e.post(t, txt, nil)
	if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"Received 2026-10-24 17.15.24 (2).bin"}` {
		t.Fatalf("just over: %+v", r)
	}
}

func TestItemHashMismatchLeavesNothing(t *testing.T) {
	e := newItemEnv(t)
	flip := func(b []byte, i int) []byte {
		c := append([]byte(nil), b...)
		c[i] ^= 0xFF
		return c
	}
	big := randBytes(2<<20, 3)
	small := []byte("a small file body")
	text := []byte("hello")
	for _, c := range []struct {
		name         string
		signed, sent []byte
		hdr          map[string]string
	}{
		{"big unnamed, flipped in the head", big, flip(big, 10), nil},
		{"big unnamed, flipped in the tail", big, flip(big, 1<<20+100), nil},
		{"big unnamed, flipped at the seam", big, flip(big, maxTextBody), nil},
		{"big named", big, flip(big, 1<<20+100), map[string]string{"X-PD-Name": "big.bin"}},
		{"small named", small, flip(small, 3), map[string]string{"X-PD-Name": "s.txt"}},
		{"small unnamed file", append([]byte{0}, small...), flip(append([]byte{0}, small...), 4), map[string]string{"X-PD-Kind": "file"}},
		{"text", text, []byte("jello"), nil},
	} {
		r := e.postSigned(t, c.signed, c.sent, c.hdr)
		if r.code != 400 || r.body != `{"ok":false,"error":"body hash mismatch"}` {
			t.Errorf("%s: %+v", c.name, r)
		}
		if names := e.files(t); len(names) != 0 {
			t.Fatalf("%s: left %q", c.name, names)
		}
	}
	if n := len(e.h.got()); n != 0 {
		t.Fatalf("%d items reached the app", n)
	}
	// "-" with a body.
	r := e.itemReq(nil, []byte("sneaky"), map[string]string{"X-PD-Name": "n.txt"})
	got, err := sendErr(e.ts.base, r)
	if err != nil || got.code != 400 || !strings.Contains(got.body, "body hash mismatch") {
		t.Fatalf("- with body: %+v %v", got, err)
	}
	// ... also when chunked.
	r = e.itemReq(nil, []byte("sneaky"), nil)
	r.ContentLength = -1
	got, err = sendErr(e.ts.base, r)
	if err != nil || got.code != 400 || !strings.Contains(got.body, "body hash mismatch") {
		t.Fatalf("- with chunked body: %+v %v", got, err)
	}
	e.wantFiles(t)
}

func TestItemTooLarge(t *testing.T) {
	e := newItemEnv(t)
	e.ts.setConfig(serverConfig{SaveDir: e.dir, MaxSize: 1000})
	body := bytes.Repeat([]byte("z"), 2000)
	if r := e.post(t, body, map[string]string{"X-PD-Name": "z.bin"}); r.code != 413 || r.body != `{"ok":false,"error":"too large"}` {
		t.Fatalf("declared: %+v", r)
	}
	// Unknown length: stops once the limit is passed.
	r := e.itemReq(body, body, map[string]string{"X-PD-Name": "z.bin"})
	r.ContentLength = -1
	if got, err := sendErr(e.ts.base, r); err != nil || got.code != 413 {
		t.Fatalf("chunked: %+v %v", got, err)
	}
	// Text over the limit too.
	r = e.itemReq(body, body, nil)
	r.ContentLength = -1
	if got, err := sendErr(e.ts.base, r); err != nil || got.code != 413 {
		t.Fatalf("chunked text: %+v %v", got, err)
	}
	e.wantFiles(t)
	// Exactly at the limit is fine.
	if r := e.post(t, body[:1000], map[string]string{"X-PD-Name": "z.bin"}); r.code != 200 {
		t.Fatalf("at the limit: %+v", r)
	}
	// The limit follows setConfig.
	e.ts.setConfig(serverConfig{SaveDir: e.dir, MaxSize: 5000})
	if r := e.post(t, body, map[string]string{"X-PD-Name": "z.bin"}); r.code != 200 {
		t.Fatalf("after raising the limit: %+v", r)
	}
}

func TestItemConfigDefaults(t *testing.T) {
	var s server
	if c := s.config(); c.MaxSize != defaultMaxSize || c.SaveDir != defaultSettings().saveDir() {
		t.Fatalf("zero config: %+v", c)
	}
	s.setConfig(serverConfig{SaveDir: "x", MaxSize: -1})
	if c := s.config(); c.MaxSize != defaultMaxSize || c.SaveDir != "x" {
		t.Fatalf("negative size: %+v", c)
	}
	s.setConfig(serverConfig{MaxSize: 7})
	if c := s.config(); c.MaxSize != 7 || c.SaveDir != defaultSettings().saveDir() {
		t.Fatalf("empty dir: %+v", c)
	}
}

func TestItemNames(t *testing.T) {
	e := newItemEnv(t)
	r := e.post(t, []byte("MZ"), map[string]string{"X-PD-Name": "..%2F..%2Fevil.exe"})
	if r.code != 200 || r.body != `{"ok":true,"kind":"file","saved":"evil.exe"}` {
		t.Fatalf("traversal: %+v", r)
	}
	e.wantFiles(t, "evil.exe")
	if got := ls(t, e.outer); strings.Join(got, "|") != "save" {
		t.Fatalf("something escaped the save folder: %q", got)
	}
	// Not valid percent-encoding: the raw value is used.
	if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": "100%.txt"}); r.code != 200 ||
		r.body != `{"ok":true,"kind":"file","saved":"100%.txt"}` {
		t.Fatalf("raw fallback: %+v", r)
	}
	// Device names and backslashes.
	if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": "C%3A%5Cdir%5CCON.txt"}); r.code != 200 ||
		r.body != `{"ok":true,"kind":"file","saved":"_CON.txt"}` {
		t.Fatalf("device name: %+v", r)
	}
	// Nothing usable left: a default name.
	if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": "%2E%2E%2F", "X-PD-Kind": "file"}); r.code != 200 ||
		r.body != `{"ok":true,"kind":"file","saved":"Received 2026-10-24 17.15.24.bin"}` {
		t.Fatalf("empty name: %+v", r)
	}
	// Header cap.
	if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": strings.Repeat("a", maxNameHeader+1) + ".txt"}); r.code != 400 ||
		r.body != `{"ok":false,"error":"bad name"}` {
		t.Fatalf("long name: %+v", r)
	}
	if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": strings.Repeat("a", maxNameHeader-4) + ".txt"}); r.code != 200 {
		t.Fatalf("name at the cap: %+v", r)
	}
	for _, n := range e.files(t) {
		if len(n) > 200 {
			t.Fatalf("saved name is %d bytes", len(n))
		}
	}
}

func TestItemUnknownPhone(t *testing.T) {
	e := newItemEnv(t)
	stranger := &phoneClient{ts: e.ts, id: "fedcba9876543210", secret: strings.Repeat("a", 64)}
	if r := stranger.do(t, "POST", "/v1/items", []byte("hi")); r.code != 401 || r.body != "unauthorized" {
		t.Fatalf("stranger: %+v", r)
	}
	// Revoked phone: same answer.
	e.ts.phones.remove(e.c.id)
	if r := e.post(t, []byte("hi"), nil); r.code != 401 || r.body != "unauthorized" {
		t.Fatalf("revoked: %+v", r)
	}
	e.wantFiles(t)
	if len(e.h.got()) != 0 {
		t.Fatal("unauthenticated item reached the app")
	}
}

func TestItemConcurrencyLimit(t *testing.T) {
	e := newItemEnv(t)
	release := make(chan struct{})
	var dones []chan resp
	for i := 0; i < maxUploadsPhone; i++ {
		data := []byte(strings.Repeat(string(rune('a'+i)), 100))
		r := e.itemReq(data, nil, map[string]string{"X-PD-Name": "slow" + itoa(i) + ".bin"})
		pr, pw := io.Pipe()
		r.Body, r.ContentLength = pr, int64(len(data))
		done := make(chan resp, 1)
		dones = append(dones, done)
		go func() {
			got, err := sendErr(e.ts.base, r)
			if err != nil {
				got = resp{-1, err.Error()}
			}
			done <- got
		}()
		// Hold the upload open after the first 10 bytes until release is closed.
		go func() {
			pw.Write(data[:10])
			<-release
			pw.Write(data[10:])
			pw.Close()
		}()
	}
	inflight := func() int {
		e.ts.items.mu.Lock()
		defer e.ts.items.mu.Unlock()
		return e.ts.items.inflight[e.c.id]
	}
	waitFor(t, func() bool { return inflight() == maxUploadsPhone })
	if r := e.post(t, []byte("fifth"), map[string]string{"X-PD-Name": "fifth.bin"}); r.code != 429 || r.body != `{"ok":false,"error":"busy"}` {
		t.Fatalf("fifth: %+v", r)
	}
	if n := inflight(); n != maxUploadsPhone {
		t.Fatalf("a refused request changed the count to %d", n)
	}
	close(release)
	for i, d := range dones {
		if r := <-d; r.code != 200 {
			t.Fatalf("upload %d: %+v", i, r)
		}
	}
	waitFor(t, func() bool { return inflight() == 0 })
	if r := e.post(t, []byte("sixth"), map[string]string{"X-PD-Name": "sixth.bin"}); r.code != 200 {
		t.Fatalf("after the others finished: %+v", r)
	}
	if got := e.files(t); len(got) != maxUploadsPhone+1 {
		t.Fatalf("files: %q", got)
	}
}

func TestItemDroppedUploadLeavesNothing(t *testing.T) {
	e := newItemEnv(t)
	data := randBytes(3<<20, 9)
	r := e.itemReq(data, nil, map[string]string{"X-PD-Name": "cut.bin"})
	pr, pw := io.Pipe()
	r.Body, r.ContentLength = pr, int64(len(data))
	errc := make(chan error, 1)
	go func() { _, err := sendErr(e.ts.base, r); errc <- err }()
	pw.Write(data[:1<<20])
	waitFor(t, func() bool { return len(e.files(t)) == 1 }) // the .part file exists
	pw.CloseWithError(io.ErrClosedPipe)
	<-errc
	waitFor(t, func() bool { return len(e.files(t)) == 0 })
	waitFor(t, func() bool {
		e.ts.items.mu.Lock()
		defer e.ts.items.mu.Unlock()
		return len(e.ts.items.inflight) == 0
	})
	if len(e.h.got()) != 0 {
		t.Fatal("a cut-off upload reached the app")
	}
}

func TestCleanStaleParts(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 24, 17, 0, 0, 0, time.UTC)
	mk := func(name string, age time.Duration) {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, now.Add(-age), now.Add(-age))
	}
	mk(".phonedrop-old1.part", 2*time.Hour)
	mk(".phonedrop-fresh.part", 5*time.Minute)
	mk(".phonedrop-edge.part", 59*time.Minute)
	mk("holiday.part", 48*time.Hour)         // not ours
	mk(".phonedrop-notes.txt", 48*time.Hour) // not ours
	mk("photo.jpg", 48*time.Hour)            // a user's file
	os.Mkdir(filepath.Join(dir, ".phonedrop-dir.part"), 0o755)
	cleanStaleParts(dir, now)
	want := ".phonedrop-dir.part|.phonedrop-edge.part|.phonedrop-fresh.part|.phonedrop-notes.txt|holiday.part|photo.jpg"
	if got := strings.Join(ls(t, dir), "|"); got != want {
		t.Fatalf("left %s", got)
	}
	cleanStaleParts(filepath.Join(dir, "missing"), now) // no folder: nothing happens
}

func TestItemSweepsStalePartsOnce(t *testing.T) {
	e := newItemEnv(t)
	if err := os.MkdirAll(e.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(e.dir, ".phonedrop-crashed.part")
	os.WriteFile(stale, []byte("junk"), 0o644)
	old := e.ts.time().Add(-3 * time.Hour)
	os.Chtimes(stale, old, old)
	if r := e.post(t, []byte("x"), map[string]string{"X-PD-Name": "a.txt"}); r.code != 200 {
		t.Fatalf("post: %+v", r)
	}
	waitFor(t, func() bool { return strings.Join(e.files(t), "|") == "a.txt" })
	// Later uploads don't sweep again.
	os.WriteFile(stale, []byte("junk"), 0o644)
	os.Chtimes(stale, old, old)
	e.post(t, []byte("y"), map[string]string{"X-PD-Name": "b.txt"})
	time.Sleep(50 * time.Millisecond)
	if _, err := os.Stat(stale); err != nil {
		t.Fatal("swept a second time")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
