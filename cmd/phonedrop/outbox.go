package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// Laptop → phone: the outbox (PROTOCOL.md §5, GET /v1/outbox and /v1/outbox/{id}). Things to
// send wait here until the phone's Receive shortcut collects them.

const (
	outboxTTL     = 24 * time.Hour
	outboxMaxText = 1 << 20 // text longer than this is cut
)

// outboxItem: one thing waiting for the phone.
type outboxItem struct {
	ID    string    `json:"id"`
	Kind  string    `json:"kind"` // image, video, file, text, url
	Name  string    `json:"name,omitempty"`
	Path  string    `json:"path,omitempty"`  // files: the source file (addFile) or our copy (addData)
	Owned bool      `json:"owned,omitempty"` // we made the copy at Path and must delete it
	Text  string    `json:"text,omitempty"`
	Size  int64     `json:"size,omitempty"`
	Added time.Time `json:"added"`
}

func (it outboxItem) isFile() bool { return it.Kind != "text" && it.Kind != "url" }

// outbox is the persisted list of pending items. Files added with addFile are only referenced;
// data (clipboard images) is copied into dir as <id>.bin. Nothing runs on a timer: expired items
// are dropped whenever the outbox is next touched.
type outbox struct {
	dir string
	now func() time.Time

	mu    sync.Mutex
	items []outboxItem
}

// newOutbox loads the outbox in dir (default: outbox\ in the data dir), dropping expired items,
// items whose source file has vanished, and stray files nothing refers to.
func newOutbox(dir string, now func() time.Time) *outbox {
	if dir == "" {
		dir = filepath.Join(dataDir(), "outbox")
	}
	if now == nil {
		now = time.Now
	}
	o := &outbox{dir: dir, now: now}
	var f struct {
		Items []outboxItem `json:"items"`
	}
	if b, err := os.ReadFile(o.indexPath()); err == nil {
		if err := json.Unmarshal(b, &f); err != nil {
			log.Printf("outbox: index unreadable, starting empty")
			f.Items = nil
		}
	}
	keep := map[string]bool{}
	changed := false
	t := now()
	for _, it := range f.Items {
		if !validOutboxID(it.ID) || keep[it.ID] || o.expired(it, t) {
			o.discard(it)
			changed = true
			continue
		}
		if it.Owned {
			it.Path = o.dataPath(it.ID) // never trust a path from the index for deletion
		}
		if it.isFile() && !isRegular(it.Path) {
			o.discard(it)
			changed = true
			continue
		}
		keep[it.ID] = true
		o.items = append(o.items, it)
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			n := e.Name()
			if n == "index.json" || (strings.HasSuffix(n, ".bin") && keep[strings.TrimSuffix(n, ".bin")]) {
				continue
			}
			os.Remove(filepath.Join(dir, n)) // orphan copy or temporary file
		}
	}
	if changed {
		o.saveLocked()
	}
	return o
}

func validOutboxID(id string) bool {
	if len(id) != 16 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func isRegular(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

func (o *outbox) indexPath() string { return filepath.Join(o.dir, "index.json") }

func (o *outbox) dataPath(id string) string { return filepath.Join(o.dir, id+".bin") }

func (o *outbox) expired(it outboxItem, now time.Time) bool {
	return !now.Before(it.Added.Add(outboxTTL))
}

// discard deletes the copy we own, if any.
func (o *outbox) discard(it outboxItem) {
	if it.Owned && validOutboxID(it.ID) {
		os.Remove(o.dataPath(it.ID))
	}
}

// sweepLocked drops expired items.
func (o *outbox) sweepLocked() {
	t := o.now()
	kept := o.items[:0]
	changed := false
	for _, it := range o.items {
		if o.expired(it, t) {
			o.discard(it)
			changed = true
			continue
		}
		kept = append(kept, it)
	}
	o.items = kept
	if changed {
		o.saveLocked()
	}
}

// saveLocked writes the index via temp file + rename, so a crash never leaves half a file.
func (o *outbox) saveLocked() {
	if err := o.writeIndex(); err != nil {
		log.Printf("outbox: save failed: %v", err) // never log item contents
	}
}

func (o *outbox) writeIndex() error {
	items := o.items
	if items == nil {
		items = []outboxItem{}
	}
	b, err := json.Marshal(struct {
		Items []outboxItem `json:"items"`
	}{items})
	if err != nil {
		return err
	}
	return o.writeAtomic(o.indexPath(), "index-*.tmp", func(w io.Writer) error {
		_, err := w.Write(b)
		return err
	})
}

func (o *outbox) writeAtomic(final, pattern string, fill func(io.Writer) error) error {
	if err := os.MkdirAll(o.dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(o.dir, pattern)
	if err != nil {
		return err
	}
	err = fill(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(f.Name(), 0o600)
	}
	if err == nil {
		err = os.Rename(f.Name(), final)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

func (o *outbox) add(it outboxItem) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	it.ID = randHex(8)
	it.Added = o.now()
	o.items = append(o.items, it)
	o.saveLocked()
}

// addFile queues a file by reference; it is read when the phone downloads it, so changes made
// before then are sent too.
func (o *outbox) addFile(path string) error {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", filepath.Base(path))
	}
	name := filepath.Base(path)
	o.add(outboxItem{Kind: mediaKind(name), Name: name, Path: path, Size: fi.Size()})
	return nil
}

// addText queues text (or a link). Blank text is ignored.
func (o *outbox) addText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if len(text) > outboxMaxText {
		text = text[:outboxMaxText]
		for len(text) > 0 && !utf8.ValidString(text) { // don't cut a character in half
			text = text[:len(text)-1]
		}
		text = strings.TrimSpace(text)
	}
	kind := "text"
	if isLink(text) {
		kind = "url"
	}
	o.add(outboxItem{Kind: kind, Text: text, Size: int64(len(text))})
}

// addData queues bytes that have no file of their own (a clipboard image) under name; the
// outbox keeps its own copy and deletes it once delivered.
func (o *outbox) addData(name string, data []byte) error {
	name = sanitizeName(name)
	if name == "" {
		name = "clipboard"
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	id := randHex(8)
	path := o.dataPath(id)
	err := o.writeAtomic(path, "data-*.tmp", func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
	if err != nil {
		return err
	}
	o.items = append(o.items, outboxItem{ID: id, Kind: mediaKind(name), Name: name, Path: path,
		Owned: true, Size: int64(len(data)), Added: o.now()})
	o.saveLocked()
	return nil
}

// list returns the pending items, oldest first.
func (o *outbox) list() []outboxItem {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	return append([]outboxItem(nil), o.items...)
}

func (o *outbox) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	return len(o.items)
}

// get returns a pending item by id.
func (o *outbox) get(id string) (outboxItem, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	for _, it := range o.items {
		if it.ID == id {
			return it, true
		}
	}
	return outboxItem{}, false
}

// remove drops an item (deleting our copy) and reports whether it was there.
func (o *outbox) remove(id string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	for i, it := range o.items {
		if it.ID == id {
			o.discard(it)
			o.items = append(o.items[:i:i], o.items[i+1:]...)
			o.saveLocked()
			return true
		}
	}
	return false
}

func (o *outbox) clear() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, it := range o.items {
		o.discard(it)
	}
	o.items = nil
	o.saveLocked()
}

// take returns every pending item, oldest first, and removes the text and link ones: they are
// delivered by being listed. Taking them in one step means two overlapping requests never both
// deliver the same text.
func (o *outbox) take() (all, texts []outboxItem) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweepLocked()
	all = append(all, o.items...)
	kept := make([]outboxItem, 0, len(o.items))
	for _, it := range o.items {
		if it.isFile() {
			kept = append(kept, it)
		} else {
			texts = append(texts, it)
		}
	}
	if len(texts) > 0 {
		o.items = kept
		o.saveLocked()
	}
	return all, texts
}

// ---------- HTTP ----------

// setOutbox attaches the outbox that /v1/outbox serves; until then the endpoints answer 501.
func (s *server) setOutbox(o *outbox) { s.out.p.Store(o) }

// outboxRef: the server's pointer to its outbox, safe to set while requests run.
type outboxRef struct{ p atomic.Pointer[outbox] }

// handleOutbox serves GET /v1/outbox and GET /v1/outbox/{id} for an authenticated phone.
func (s *server) handleOutbox(w http.ResponseWriter, r *http.Request, a authResult) {
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
	o := s.out.p.Load()
	if o == nil {
		notYet(w)
		return
	}
	ph, ok := s.phones.get(a.id)
	if !ok { // removed while the request was being checked
		s.unauthorized(w, r)
		return
	}
	if r.URL.Path == "/v1/outbox" {
		s.outboxList(w, o, ph.Name)
		return
	}
	s.outboxGet(w, o, strings.TrimPrefix(r.URL.Path, "/v1/outbox/"), ph.Name)
}

type outboxEntry struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
	Size *int64 `json:"size,omitempty"`
	Path string `json:"path,omitempty"`
	Text string `json:"text,omitempty"`
}

func (s *server) outboxList(w http.ResponseWriter, o *outbox, phone string) {
	all, texts := o.take()
	entries := make([]outboxEntry, 0, len(all))
	for _, it := range all {
		if !it.isFile() {
			entries = append(entries, outboxEntry{ID: it.ID, Kind: it.Kind, Text: it.Text})
			continue
		}
		fi, err := os.Stat(it.Path)
		if err != nil || !fi.Mode().IsRegular() { // the source was moved or deleted
			o.remove(it.ID)
			continue
		}
		size := fi.Size()
		// The stored kind is mediaKind's (for the panel); the phone gets the Photos-safe one.
		entries = append(entries, outboxEntry{ID: it.ID, Kind: photosKind(it.Name), Name: it.Name, Size: &size,
			Path: "/v1/outbox/" + it.ID})
	}
	reply(w, http.StatusOK, struct {
		OK    bool          `json:"ok"`
		Items []outboxEntry `json:"items"`
	}{true, entries})
	for _, it := range texts {
		s.hooks.sent(it, phone)
	}
}

func (s *server) outboxGet(w http.ResponseWriter, o *outbox, id, phone string) {
	it, ok := o.get(id)
	if !ok || !it.isFile() {
		replyErr(w, http.StatusNotFound, "not found")
		return
	}
	f, err := os.Open(it.Path)
	var fi os.FileInfo
	if err == nil {
		fi, err = f.Stat()
	}
	if err == nil && !fi.Mode().IsRegular() {
		err = errors.New("not a regular file")
	}
	if err != nil {
		if f != nil {
			f.Close()
		}
		o.remove(id) // the source is gone
		replyErr(w, http.StatusNotFound, "not found")
		return
	}
	defer f.Close()
	size := fi.Size()
	h := w.Header()
	h.Set("Content-Type", contentTypeFor(it.Name))
	h.Set("Content-Length", strconv.FormatInt(size, 10))
	h.Set("Content-Disposition", contentDisposition(it.Name))
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	n, err := io.CopyN(w, f, size)
	if err == nil && n == size {
		// What's still buffered must reach the socket too; a phone that hung up in the last
		// segment shows up here.
		err = http.NewResponseController(w).Flush()
	}
	if err != nil || n != size {
		return // the item stays for another try
	}
	if o.remove(id) {
		s.hooks.sent(it, phone)
	}
}

// contentTypes covers what phones send around, for systems whose mime table is thin.
var contentTypes = map[string]string{
	".heic": "image/heic", ".heif": "image/heif", ".avif": "image/avif",
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif",
	".webp": "image/webp", ".bmp": "image/bmp", ".tif": "image/tiff", ".tiff": "image/tiff",
	".mov": "video/quicktime", ".mp4": "video/mp4", ".m4v": "video/x-m4v", ".webm": "video/webm",
	".3gp": "video/3gpp", ".pdf": "application/pdf", ".txt": "text/plain; charset=utf-8",
	".zip": "application/zip",
}

func contentTypeFor(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if t, ok := contentTypes[ext]; ok {
		return t
	}
	if ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return t
		}
	}
	return "application/octet-stream"
}

// contentDisposition: an attachment with the name as RFC 5987 UTF-8 plus a plain ASCII fallback.
func contentDisposition(name string) string {
	if name == "" {
		name = "file"
	}
	var fb strings.Builder
	for _, r := range name {
		switch {
		case r == '"' || r == '\\':
		case r < 0x20 || r == 0x7F:
			fb.WriteByte('_')
		case r > 0x7E:
			fb.WriteByte('_')
		default:
			fb.WriteRune(r)
		}
	}
	return `attachment; filename*=UTF-8''` + encodeRFC5987(name) + `; filename="` + fb.String() + `"`
}

func encodeRFC5987(s string) string {
	const attrChar = "!#$&+-.^_`|~"
	var b strings.Builder
	for _, c := range []byte(s) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte(attrChar, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
