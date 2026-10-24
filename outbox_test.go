package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var outboxT0 = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

// clockAt: a clock tests can move.
type clockAt struct{ t time.Time }

func (c *clockAt) now() time.Time          { return c.t }
func (c *clockAt) advance(d time.Duration) { c.t = c.t.Add(d) }

func writeTemp(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOutboxPersistence(t *testing.T) {
	dir, src := filepath.Join(t.TempDir(), "outbox"), t.TempDir()
	clk := &clockAt{outboxT0}
	o := newOutbox(dir, clk.now)
	if o.count() != 0 {
		t.Fatal("not empty")
	}
	f := writeTemp(t, src, "Screenshot.PNG", []byte("png-bytes"))
	if err := o.addFile(f); err != nil {
		t.Fatal(err)
	}
	o.addText("  hello \n")
	o.addText("   ")
	o.addText("\thttps://example.com/a?b=1 ")
	if err := o.addData("clip.jpg", []byte("jpeg")); err != nil {
		t.Fatal(err)
	}
	if err := o.addFile(src); err == nil {
		t.Fatal("directory accepted")
	}
	if err := o.addFile(filepath.Join(src, "nope.txt")); err == nil {
		t.Fatal("missing file accepted")
	}
	want := o.list()
	if len(want) != 4 || o.count() != 4 {
		t.Fatalf("items %+v", want)
	}
	kinds := ""
	for _, it := range want {
		kinds += it.Kind + ","
		if !validOutboxID(it.ID) {
			t.Errorf("bad id %q", it.ID)
		}
	}
	if kinds != "image,text,url,image," {
		t.Fatalf("kinds %s", kinds)
	}
	if want[0].Name != "Screenshot.PNG" || want[0].Path != f || want[0].Owned || want[0].Size != 9 {
		t.Errorf("file item %+v", want[0])
	}
	if want[1].Text != "hello" || want[2].Text != "https://example.com/a?b=1" {
		t.Errorf("text items %+v %+v", want[1], want[2])
	}
	d := want[3]
	if !d.Owned || d.Name != "clip.jpg" || d.Size != 4 || d.Path != filepath.Join(dir, d.ID+".bin") {
		t.Errorf("data item %+v", d)
	}
	if b, err := os.ReadFile(d.Path); err != nil || string(b) != "jpeg" {
		t.Errorf("copy %q %v", b, err)
	}
	if fi, _ := os.Stat(d.Path); runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		t.Errorf("copy mode %v", fi.Mode())
	}

	// Strays are swept on load; the rest survives.
	writeTemp(t, dir, "deadbeefdeadbeef.bin", []byte("orphan"))
	writeTemp(t, dir, "index-123.tmp", []byte("{"))
	o2 := newOutbox(dir, clk.now)
	got := o2.list()
	if len(got) != 4 {
		t.Fatalf("reloaded %+v", got)
	}
	for i := range got {
		if !got[i].Added.Equal(want[i].Added) {
			got[i].Added = want[i].Added // monotonic/location noise only
		}
		if got[i] != want[i] {
			t.Errorf("item %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if names := strings.Join(ls(t, dir), " "); names != d.ID+".bin index.json" {
		t.Errorf("outbox dir has %q", names)
	}
}

func TestOutboxExpiry(t *testing.T) {
	dir, src := t.TempDir(), t.TempDir()
	clk := &clockAt{outboxT0}
	o := newOutbox(dir, clk.now)
	o.addText("old")
	o.addData("old.png", []byte("x"))
	oldBin := o.list()[1].Path
	clk.advance(outboxTTL - time.Second)
	o.addFile(writeTemp(t, src, "new.txt", []byte("n")))
	if o.count() != 3 {
		t.Fatalf("count %d", o.count())
	}
	clk.advance(time.Second) // the first two are now 24 h old
	if l := o.list(); len(l) != 1 || l[0].Name != "new.txt" {
		t.Fatalf("after expiry: %+v", l)
	}
	if _, err := os.Stat(oldBin); !os.IsNotExist(err) {
		t.Fatal("expired copy kept")
	}
	if o2 := newOutbox(dir, clk.now); o2.count() != 1 {
		t.Fatal("expiry not persisted")
	}
	// Expired items are also dropped at load time.
	clk.advance(outboxTTL)
	if o3 := newOutbox(dir, clk.now); o3.count() != 0 {
		t.Fatal("expired item loaded")
	}
	if got := ls(t, dir); strings.Join(got, " ") != "index.json" {
		t.Fatalf("dir %q", got)
	}
}

func TestOutboxVanishedSourceDroppedOnLoad(t *testing.T) {
	dir, src := t.TempDir(), t.TempDir()
	o := newOutbox(dir, nil)
	a := writeTemp(t, src, "a.txt", []byte("a"))
	b := writeTemp(t, src, "b.txt", []byte("b"))
	o.addFile(a)
	o.addFile(b)
	o.addData("c.png", []byte("c"))
	o.addText("t")
	os.Remove(a)
	os.Remove(o.list()[2].Path) // our own copy went missing
	o2 := newOutbox(dir, nil)
	l := o2.list()
	if len(l) != 2 || l[0].Name != "b.txt" || l[1].Kind != "text" {
		t.Fatalf("after load: %+v", l)
	}
}

func TestOutboxTextCapAndKinds(t *testing.T) {
	o := newOutbox(t.TempDir(), nil)
	o.addText(strings.Repeat("é", outboxMaxText)) // 2 MB
	it := o.list()[0]
	if len(it.Text) > outboxMaxText || !strings.HasPrefix(it.Text, "é") || it.Kind != "text" {
		t.Fatalf("len %d kind %s", len(it.Text), it.Kind)
	}
	o.clear()
	o.addText("see https://example.com") // not only a link
	o.addText("ftp://example.com")
	o.addText("HTTP://Example.com")
	var ks []string
	for _, it := range o.list() {
		ks = append(ks, it.Kind)
	}
	if strings.Join(ks, ",") != "text,text,url" {
		t.Fatalf("kinds %v", ks)
	}
}

func TestOutboxRemoveAndClearDeleteCopies(t *testing.T) {
	dir := t.TempDir()
	o := newOutbox(dir, nil)
	src := writeTemp(t, t.TempDir(), "keep.txt", []byte("k"))
	o.addFile(src)
	o.addData("a.png", []byte("a"))
	o.addData("b.png", []byte("b"))
	l := o.list()
	if !o.remove(l[1].ID) || o.remove(l[1].ID) {
		t.Fatal("remove result")
	}
	if _, err := os.Stat(l[1].Path); !os.IsNotExist(err) {
		t.Fatal("copy kept after remove")
	}
	o.clear()
	if o.count() != 0 || len(ls(t, dir)) != 1 { // just index.json
		t.Fatalf("after clear: %v", ls(t, dir))
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal("clear deleted the user's own file")
	}
}

func TestOutboxIgnoresTamperedIndex(t *testing.T) {
	dir := t.TempDir()
	victim := writeTemp(t, t.TempDir(), "victim.txt", []byte("v"))
	idx := fmt.Sprintf(`{"items":[{"id":"0123456789abcdef","kind":"file","name":"x","path":%q,"owned":true,"added":%q},`+
		`{"id":"../../x","kind":"text","text":"t","added":%q}]}`, victim, outboxT0.Format(time.RFC3339), outboxT0.Format(time.RFC3339))
	writeTemp(t, dir, "index.json", []byte(idx))
	clk := &clockAt{outboxT0}
	o := newOutbox(dir, clk.now)
	if o.count() != 0 { // the owned copy is dir/<id>.bin, which doesn't exist
		t.Fatalf("items %+v", o.list())
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatal("deleted a file outside the outbox")
	}
}

// ---------- HTTP ----------

type outboxEnv struct {
	*itemEnv
	o   *outbox
	clk *clockAt
	src string
}

func newOutboxEnv(t *testing.T) *outboxEnv {
	t.Helper()
	e := &outboxEnv{itemEnv: newItemEnv(t), src: t.TempDir()}
	e.clk = &clockAt{outboxT0}
	e.o = newOutbox(filepath.Join(t.TempDir(), "outbox"), e.clk.now)
	e.ts.setOutbox(e.o)
	return e
}

// get sends a signed GET and returns the open response.
func (e *outboxEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	ts := e.ts.time().Format(time.RFC3339)
	r := signedReq("GET", path, e.c.id, e.c.secret, ts, "ob-nonce-"+itoa(int(nonceN.Add(1))), nil)
	req, _ := http.NewRequest("GET", e.ts.base+path, nil)
	req.Header = r.Header
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (e *outboxEnv) getBody(t *testing.T, path string) (int, string) {
	t.Helper()
	res := e.get(t, path)
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, strings.TrimSpace(string(b))
}

func TestOutboxEndpointsNeedOutbox(t *testing.T) {
	e := newItemEnv(t)
	if r := e.c.do(t, "GET", "/v1/outbox", nil); r.code != 501 {
		t.Fatalf("no outbox: %+v", r)
	}
	if r := e.c.do(t, "GET", "/v1/outbox/abc", nil); r.code != 501 {
		t.Fatalf("no outbox: %+v", r)
	}
	if r := e.c.do(t, "POST", "/v1/outbox", []byte("x")); r.code != 405 {
		t.Fatalf("POST: %+v", r)
	}
}

func TestOutboxListShapeAndTextDelivery(t *testing.T) {
	e := newOutboxEnv(t)
	if code, body := e.getBody(t, "/v1/outbox"); code != 200 || body != `{"ok":true,"items":[]}` {
		t.Fatalf("empty: %d %s", code, body)
	}
	e.o.addFile(writeTemp(t, e.src, "Shot.png", []byte("123")))
	e.o.addText("hello <&> \"world\"")
	e.o.addText("https://example.com/x")
	e.o.addData("clip.mov", []byte("1234567"))
	l := e.o.list()
	want := fmt.Sprintf(`{"ok":true,"items":[`+
		`{"id":%q,"kind":"image","name":"Shot.png","size":3,"path":"/v1/outbox/%s"},`+
		`{"id":%q,"kind":"text","text":"hello \u003c\u0026\u003e \"world\""},`+
		`{"id":%q,"kind":"url","text":"https://example.com/x"},`+
		`{"id":%q,"kind":"video","name":"clip.mov","size":7,"path":"/v1/outbox/%s"}]}`,
		l[0].ID, l[0].ID, l[1].ID, l[2].ID, l[3].ID, l[3].ID)
	res := e.get(t, "/v1/outbox")
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/json" || strings.TrimSpace(string(b)) != want {
		t.Fatalf("list: %d %s\nwant %s", res.StatusCode, b, want)
	}
	var parsed struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil || parsed.Items[1]["text"] != `hello <&> "world"` {
		t.Fatalf("parse: %v %v", err, parsed)
	}
	// Texts were delivered by listing and are gone; files stay.
	if l2 := e.o.list(); len(l2) != 2 || l2[0].ID != l[0].ID || l2[1].ID != l[3].ID {
		t.Fatalf("after listing: %+v", l2)
	}
	sent := e.h.sentLog()
	if len(sent) != 2 || sent[0].item.Text != `hello <&> "world"` || sent[1].item.Kind != "url" ||
		sent[0].phone != "Test Phone" || sent[1].phone != "Test Phone" {
		t.Fatalf("sent hook: %+v", sent)
	}
	// Listing again shows the files only, with no new hook calls.
	_, body := e.getBody(t, "/v1/outbox")
	if strings.Contains(body, "text") || len(e.h.sentLog()) != 2 {
		t.Fatalf("second list: %s", body)
	}
	// Text ids are not downloadable.
	if code, body := e.getBody(t, "/v1/outbox/"+l[1].ID); code != 404 || body != `{"ok":false,"error":"not found"}` {
		t.Fatalf("text id: %d %s", code, body)
	}
}

func TestOutboxListDropsVanishedFiles(t *testing.T) {
	e := newOutboxEnv(t)
	a := writeTemp(t, e.src, "a.txt", []byte("a"))
	e.o.addFile(a)
	e.o.addFile(writeTemp(t, e.src, "b.txt", []byte("b")))
	os.Remove(a)
	code, body := e.getBody(t, "/v1/outbox")
	if code != 200 || strings.Contains(body, "a.txt") || !strings.Contains(body, "b.txt") {
		t.Fatalf("%d %s", code, body)
	}
	if e.o.count() != 1 {
		t.Fatal("vanished item kept")
	}
	if len(e.h.sentLog()) != 0 {
		t.Fatal("sent hook for a vanished file")
	}
}

func TestOutboxDownload(t *testing.T) {
	e := newOutboxEnv(t)
	data := randBytes(3<<20+17, 7)
	name := "café ✓ \"q\".heic"
	src := writeTemp(t, e.src, name, data)
	e.o.addFile(src)
	e.o.addData("scan.PDF", []byte("%PDF-1.4"))
	l := e.o.list()

	res := e.get(t, "/v1/outbox/"+l[0].ID)
	got, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil || res.StatusCode != 200 || !bytes.Equal(got, data) {
		t.Fatalf("download: %d %v (%d bytes)", res.StatusCode, err, len(got))
	}
	h := res.Header
	if h.Get("Content-Type") != "image/heic" || h.Get("Content-Length") != itoa(len(data)) {
		t.Errorf("headers %v", h)
	}
	wantCD := `attachment; filename*=UTF-8''caf%C3%A9%20%E2%9C%93%20%22q%22.heic; filename="caf_ _ q.heic"`
	if h.Get("Content-Disposition") != wantCD {
		t.Errorf("Content-Disposition %q, want %q", h.Get("Content-Disposition"), wantCD)
	}
	if h.Get("Content-Range") != "" || h.Get("Accept-Ranges") != "" {
		t.Errorf("range headers %v", h)
	}
	// Delivered: gone from the outbox, the user's own file untouched, hook called once.
	if _, ok := e.o.get(l[0].ID); ok {
		t.Fatal("item still queued after a full download")
	}
	if b, err := os.ReadFile(src); err != nil || !bytes.Equal(b, data) {
		t.Fatal("source file damaged")
	}
	if s := e.h.sentLog(); len(s) != 1 || s[0].item.ID != l[0].ID || s[0].phone != "Test Phone" || s[0].item.Name != name {
		t.Fatalf("sent hook: %+v", s)
	}
	if code, _ := e.getBody(t, "/v1/outbox/"+l[0].ID); code != 404 {
		t.Fatalf("second download: %d", code)
	}

	// An owned copy is deleted once delivered; and types come from the fixed table.
	res = e.get(t, "/v1/outbox/"+l[1].ID)
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(b) != "%PDF-1.4" || res.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("pdf: %q %v", b, res.Header)
	}
	if _, err := os.Stat(l[1].Path); !os.IsNotExist(err) {
		t.Fatal("owned copy kept after delivery")
	}
	if e.o.count() != 0 {
		t.Fatal("outbox not empty")
	}
}

func TestContentTypeFor(t *testing.T) {
	for name, want := range map[string]string{
		"a.heic": "image/heic", "a.HEIF": "image/heif", "a.mov": "video/quicktime", "a.mp4": "video/mp4",
		"a.png": "image/png", "a.jpg": "image/jpeg", "a.gif": "image/gif", "a.webp": "image/webp",
		"a.pdf": "application/pdf", "noext": "application/octet-stream", "a.zzqq9": "application/octet-stream",
	} {
		if got := contentTypeFor(name); got != want {
			t.Errorf("%s: %s, want %s", name, got, want)
		}
	}
}

func TestOutboxDownloadMissingSource(t *testing.T) {
	e := newOutboxEnv(t)
	p := writeTemp(t, e.src, "gone.bin", []byte("x"))
	e.o.addFile(p)
	id := e.o.list()[0].ID
	os.Remove(p)
	if code, body := e.getBody(t, "/v1/outbox/"+id); code != 404 || body != `{"ok":false,"error":"not found"}` {
		t.Fatalf("%d %s", code, body)
	}
	if e.o.count() != 0 {
		t.Fatal("vanished item kept")
	}
}

func TestOutboxInterruptedDownloadKeepsItem(t *testing.T) {
	e := newOutboxEnv(t)
	big := make([]byte, 96<<20) // far more than the socket buffers hold
	if err := e.o.addData("big.mp4", big); err != nil {
		t.Fatal(err)
	}
	it := e.o.list()[0]
	res := e.get(t, "/v1/outbox/"+it.ID)
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if _, err := io.ReadFull(res.Body, make([]byte, 1<<20)); err != nil {
		t.Fatal(err)
	}
	res.Body.Close() // the phone gives up
	http.DefaultClient.CloseIdleConnections()
	time.Sleep(300 * time.Millisecond) // let the handler notice
	if _, ok := e.o.get(it.ID); !ok {
		t.Fatal("item removed after an interrupted download")
	}
	if _, err := os.Stat(it.Path); err != nil {
		t.Fatal("copy deleted after an interrupted download")
	}
	if len(e.h.sentLog()) != 0 {
		t.Fatal("sent hook after an interrupted download")
	}
	// A retry that completes delivers it.
	res = e.get(t, "/v1/outbox/"+it.ID)
	n, err := io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if err != nil || n != int64(len(big)) {
		t.Fatalf("retry: %d %v", n, err)
	}
	if _, ok := e.o.get(it.ID); ok {
		t.Fatal("item kept after a full download")
	}
	if _, err := os.Stat(it.Path); !os.IsNotExist(err) {
		t.Fatal("copy kept after delivery")
	}
	if len(e.h.sentLog()) != 1 {
		t.Fatalf("sent hook %+v", e.h.sentLog())
	}
}

func TestOutboxExpiredNotListed(t *testing.T) {
	e := newOutboxEnv(t)
	e.o.addText("t")
	e.o.addData("a.png", []byte("a"))
	id := e.o.list()[1].ID
	e.clk.advance(outboxTTL)
	if code, body := e.getBody(t, "/v1/outbox"); code != 200 || body != `{"ok":true,"items":[]}` {
		t.Fatalf("%d %s", code, body)
	}
	if code, _ := e.getBody(t, "/v1/outbox/"+id); code != 404 {
		t.Fatalf("expired download: %d", code)
	}
	if len(e.h.sentLog()) != 0 {
		t.Fatal("expired text reported as sent")
	}
}

func TestOutboxNotFoundAndMethods(t *testing.T) {
	e := newOutboxEnv(t)
	for _, p := range []string{"/v1/outbox/nope", "/v1/outbox/", "/v1/outbox/a/b", "/v1/outbox/0123456789abcdef"} {
		if code, body := e.getBody(t, p); code != 404 || body != `{"ok":false,"error":"not found"}` {
			t.Errorf("%s: %d %s", p, code, body)
		}
	}
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		if r := e.c.do(t, m, "/v1/outbox", nil); r.code != 405 {
			t.Errorf("%s /v1/outbox: %+v", m, r)
		}
		if r := e.c.do(t, m, "/v1/outbox/abc", nil); r.code != 405 {
			t.Errorf("%s /v1/outbox/abc: %+v", m, r)
		}
	}
}

func TestOutboxUnauthenticated(t *testing.T) {
	e := newOutboxEnv(t)
	e.o.addText("secret")
	e.o.addData("a.png", []byte("a"))
	id := e.o.list()[1].ID
	for _, p := range []string{"/v1/outbox", "/v1/outbox/" + id} {
		res, err := http.Get(e.ts.base + p)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 401 || strings.TrimSpace(string(b)) != "unauthorized" {
			t.Errorf("%s: %d %q", p, res.StatusCode, b)
		}
		// Signed by someone who isn't paired.
		bad := &phoneClient{ts: e.ts, id: strings.Repeat("a", 16), secret: strings.Repeat("b", 64), n: 100}
		if r := bad.do(t, "GET", p, nil); r.code != 401 || r.body != "unauthorized" {
			t.Errorf("unpaired %s: %+v", p, r)
		}
	}
	if e.o.count() != 2 || len(e.h.sentLog()) != 0 {
		t.Fatal("unauthenticated requests changed the outbox")
	}
}

func TestContentDispositionFallback(t *testing.T) {
	for name, want := range map[string]string{
		"a.png":       `attachment; filename*=UTF-8''a.png; filename="a.png"`,
		"my file.pdf": `attachment; filename*=UTF-8''my%20file.pdf; filename="my file.pdf"`,
		"日本.jpg":      `attachment; filename*=UTF-8''%E6%97%A5%E6%9C%AC.jpg; filename="__.jpg"`,
		`a"b\c.txt`:   `attachment; filename*=UTF-8''a%22b%5Cc.txt; filename="abc.txt"`,
		"a;b,c'd.txt": `attachment; filename*=UTF-8''a%3Bb%2Cc%27d.txt; filename="a;b,c'd.txt"`,
	} {
		if got := contentDisposition(name); got != want {
			t.Errorf("%q: %s, want %s", name, got, want)
		}
	}
}
