package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// POST /v1/items: phone → laptop (PROTOCOL.md §5).

const (
	defaultMaxSize         = 16 << 30 // largest accepted item unless the settings say otherwise
	maxUploadsPhone        = 4        // uploads in flight per phone; a Shortcut sends a handful at once
	maxNameHeader          = 1024     // X-PD-Name, as sent (percent-encoded)
	sniffLen               = 512
	partPattern            = ".phonedrop-*.part" // temporary files, hidden by the leading dot
	partStale              = time.Hour           // a .part file this old is a crashed upload
	partPrefix, partSuffix = ".phonedrop-", ".part"
)

// serverConfig: the settings the upload endpoint reads. The UI thread replaces the whole value
// with setConfig when settings change; requests take one snapshot at their start.
type serverConfig struct {
	SaveDir string // empty: the Downloads folder
	MaxSize int64  // largest item in bytes; ≤ 0: 16 GiB
}

// itemState: the upload endpoint's state, so server.go only has to hold one field. The zero
// value is ready to use.
type itemState struct {
	cfg atomic.Pointer[serverConfig]

	mu       sync.Mutex
	inflight map[string]int  // phone id → uploads in progress
	swept    map[string]bool // save dirs already cleaned of stale .part files
}

// setConfig applies new settings; it is safe to call from any goroutine while uploads run.
func (s *server) setConfig(c serverConfig) {
	c = normalizeConfig(c)
	s.items.cfg.Store(&c)
}

func (s *server) config() serverConfig {
	if c := s.items.cfg.Load(); c != nil {
		return *c
	}
	return normalizeConfig(serverConfig{})
}

func normalizeConfig(c serverConfig) serverConfig {
	if c.SaveDir == "" {
		c.SaveDir = defaultSettings().saveDir()
	}
	if c.MaxSize <= 0 {
		c.MaxSize = defaultMaxSize
	}
	return c
}

func (st *itemState) acquire(id string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.inflight == nil {
		st.inflight = map[string]int{}
	}
	if st.inflight[id] >= maxUploadsPhone {
		return false
	}
	st.inflight[id]++
	return true
}

func (st *itemState) release(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.inflight[id]--; st.inflight[id] <= 0 {
		delete(st.inflight, id)
	}
}

// firstSweep reports whether dir hasn't been cleaned of stale .part files yet (once per
// directory per process, so no timer is needed).
func (st *itemState) firstSweep(dir string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.swept == nil {
		st.swept = map[string]bool{}
	}
	if st.swept[dir] {
		return false
	}
	st.swept[dir] = true
	return true
}

// cleanStaleParts removes .phonedrop-*.part files older than partStale: what a crash or power
// cut in the middle of an upload leaves behind.
func cleanStaleParts(dir string, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasPrefix(n, partPrefix) || !strings.HasSuffix(n, partSuffix) {
			continue
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > partStale {
			os.Remove(filepath.Join(dir, n))
		}
	}
}

// handleItem receives one item. Text and links go to the app; everything else is streamed to a
// hidden temporary file in the save folder and renamed into place once the hash has checked out.
func (s *server) handleItem(w http.ResponseWriter, r *http.Request, a authResult, from string) {
	cfg := s.config()
	if err := checkBodyLen(r, a.body, cfg.MaxSize); err != nil {
		bodyError(w, err)
		return
	}
	if !s.items.acquire(a.id) {
		w.Header().Set("Connection", "close")
		replyErr(w, http.StatusTooManyRequests, "busy")
		return
	}
	defer s.items.release(a.id)

	name, ok := itemName(r)
	if !ok {
		w.Header().Set("Connection", "close")
		replyErr(w, http.StatusBadRequest, "bad name")
		return
	}
	kindH := strings.ToLower(strings.TrimSpace(r.Header.Get("X-PD-Kind")))
	track := &trackReader{r: r.Body}
	var src io.Reader = track
	verified := false // src is already checked against X-PD-Body

	// A body that might be text is read ahead (at most maxTextBody+1 bytes). If it turns out
	// to be bigger, the bytes already read are put back in front and it streams to disk as a file.
	small := r.ContentLength < 0 || r.ContentLength <= maxTextBody
	if small && kindH != "file" && (kindH == "text" || kindH == "url" || !hasExt(name)) {
		head, err := io.ReadAll(io.LimitReader(track, maxTextBody+1))
		if err != nil {
			bodyError(w, err)
			return
		}
		src = io.MultiReader(bytes.NewReader(head), track)
		if len(head) <= maxTextBody { // the whole body
			body, err := readBody(src, a.body, maxTextBody)
			if err != nil {
				bodyError(w, err)
				return
			}
			if int64(len(body)) > cfg.MaxSize {
				bodyError(w, errTooLarge)
				return
			}
			switch k := inferKind(name, kindH, body, int64(len(body)), true); k {
			case "text", "url":
				text := strings.ToValidUTF8(string(body), "")
				if k == "url" {
					text = strings.TrimSpace(text)
				}
				if strings.TrimSpace(text) == "" {
					replyErr(w, http.StatusBadRequest, "empty")
					return
				}
				s.hooks.received(receivedItem{Kind: k, Text: text, Size: int64(len(body)), From: from, Time: s.now()})
				reply(w, http.StatusOK, struct {
					OK   bool   `json:"ok"`
					Kind string `json:"kind"`
				}{true, k})
				return
			}
			src, verified = bytes.NewReader(body), true
		}
	}

	if err := os.MkdirAll(cfg.SaveDir, 0o755); err != nil {
		logItemErr(a.id, "creating the save folder", err)
		replyErr(w, http.StatusInternalServerError, "could not save")
		return
	}
	if s.items.firstSweep(cfg.SaveDir) {
		go cleanStaleParts(cfg.SaveDir, s.now())
	}
	var (
		tmp  string
		size int64
		err  error
	)
	if verified {
		tmp, size, err = saveStream(src, cfg.SaveDir, cfg.MaxSize)
	} else {
		tmp, size, err = saveBody(src, a.body, cfg.SaveDir, cfg.MaxSize)
	}
	if err != nil {
		if errors.Is(err, errTooLarge) || errors.Is(err, errBodyHash) {
			bodyError(w, err)
		} else if track.err != nil { // the phone went away mid-upload
			bodyError(w, track.err)
		} else {
			logItemErr(a.id, "writing the file", err)
			replyErr(w, http.StatusInternalServerError, "could not save")
		}
		return
	}
	final, err := s.place(tmp, cfg.SaveDir, name)
	if err != nil {
		logItemErr(a.id, "placing the file", err)
		replyErr(w, http.StatusInternalServerError, "could not save")
		return
	}
	if err := markOfTheWeb(final); err != nil {
		logItemErr(a.id, "marking the file as downloaded", err)
	}
	base := filepath.Base(final)
	s.hooks.received(receivedItem{Kind: mediaKind(base), Name: base, Path: final, Size: size, From: from, Time: s.now()})
	reply(w, http.StatusOK, struct {
		OK    bool   `json:"ok"`
		Kind  string `json:"kind"`
		Saved string `json:"saved"`
	}{true, "file", base})
}

// place gives the finished temporary file its final name in dir and returns the path. A name
// without an extension gets one from the content, and a missing name becomes "Received ...".
// On failure the temporary file is gone.
func (s *server) place(tmp, dir, name string) (string, error) {
	head := readHead(tmp, sniffLen)
	ext := ""
	if !hasExt(name) {
		ext = sniffExt(head)
	}
	if name == "" {
		name = defaultName(s.now(), ext)
	} else if ext != "" {
		name = limitName(name + ext)
	}
	final, f, err := uniquePath(dir, name)
	if err != nil {
		os.Remove(tmp)
		return "", err
	}
	f.Close() // Windows can't rename over an open file
	os.Chmod(tmp, 0o644)
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		os.Remove(final)
		return "", err
	}
	return final, nil
}

// readHead returns up to n bytes from the start of the file at path.
func readHead(path string, n int) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	b := make([]byte, n)
	m, _ := io.ReadFull(f, b)
	return b[:m]
}

// itemName reads X-PD-Name: percent-decoded (the raw value if it isn't valid encoding) and
// sanitized; "" when absent or nothing usable is left. ok is false for a header that is
// repeated or too long.
func itemName(r *http.Request) (name string, ok bool) {
	vs := r.Header.Values("X-PD-Name")
	switch {
	case len(vs) == 0:
		return "", true
	case len(vs) > 1 || len(vs[0]) > maxNameHeader:
		return "", false
	}
	raw := vs[0]
	if dec, err := url.PathUnescape(raw); err == nil {
		raw = dec
	}
	return sanitizeName(raw), true
}

// hasExt reports whether a sanitized name ends in something that looks like a file extension
// (the same test inferKind uses).
func hasExt(name string) bool {
	ext := filepath.Ext(name)
	return len(ext) > 1 && len(ext) <= 10 && isAlnum(ext[1:])
}

// logItemErr logs a failure under the phone's id. Path errors are cut down to the underlying
// error, because the path would carry the file's name.
func logItemErr(id, what string, err error) {
	var pe *fs.PathError
	var le *os.LinkError
	switch {
	case errors.As(err, &pe):
		err = pe.Err
	case errors.As(err, &le):
		err = le.Err
	}
	log.Printf("items: phone %s: %s: %v", id, what, err)
}

// trackReader remembers the first real read error of the request body, to tell a phone that
// hung up mid-upload from a disk problem.
type trackReader struct {
	r   io.Reader
	err error
}

func (t *trackReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if err != nil && err != io.EOF && t.err == nil {
		t.err = err
	}
	return n, err
}
