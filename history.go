package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	historyMax     = 50
	historyMaxText = 2000 // runes of text/url content kept per entry
)

// historyEntry is one transfer shown in the panel.
type historyEntry struct {
	Kind string    `json:"kind"`           // file, image, video, text, url
	Name string    `json:"name,omitempty"` // display name
	Path string    `json:"path,omitempty"` // saved file, files only (may have been deleted since)
	Text string    `json:"text,omitempty"` // text/url content
	Size int64     `json:"size,omitempty"`
	From string    `json:"from,omitempty"` // phone name
	Time time.Time `json:"time"`
	Dir  string    `json:"dir"` // "in" phone to laptop, "out" laptop to phone
}

// history is the recent-transfers list, newest first, persisted as JSON on every change.
type history struct {
	mu      sync.Mutex
	path    string
	entries []historyEntry
}

// loadHistory reads path (default: history.json in the data dir). A missing or corrupt file
// starts empty.
func loadHistory(path string) *history {
	if path == "" {
		path = filepath.Join(dataDir(), "history.json")
	}
	h := &history{path: path}
	if b, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(b, &h.entries) != nil {
			h.entries = nil
		}
		if len(h.entries) > historyMax {
			h.entries = h.entries[:historyMax]
		}
	}
	return h
}

// add puts e at the front, trimming stored text and the list length.
func (h *history) add(e historyEntry) {
	if r := []rune(e.Text); len(r) > historyMaxText {
		e.Text = string(r[:historyMaxText])
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append([]historyEntry{e}, h.entries...)
	if len(h.entries) > historyMax {
		h.entries = h.entries[:historyMax]
	}
	h.saveLocked()
}

// recent returns a copy of the newest n entries (all if n <= 0 or n is larger).
func (h *history) recent(n int) []historyEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n <= 0 || n > len(h.entries) {
		n = len(h.entries)
	}
	return append([]historyEntry(nil), h.entries[:n]...)
}

// remove deletes entry i (an index into recent); out-of-range is ignored.
func (h *history) remove(i int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if i < 0 || i >= len(h.entries) {
		return
	}
	h.entries = append(h.entries[:i:i], h.entries[i+1:]...)
	h.saveLocked()
}

func (h *history) clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = nil
	h.saveLocked()
}

// saveLocked writes via temp file + rename so a crash never leaves a half-written file.
func (h *history) saveLocked() {
	if err := h.write(); err != nil {
		log.Printf("history: save failed: %v", err) // never log entry contents
	}
}

func (h *history) write() error {
	es := h.entries
	if es == nil {
		es = []historyEntry{}
	}
	b, err := json.Marshal(es)
	if err != nil {
		return err
	}
	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "history-*.tmp")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(f.Name(), 0o600)
	}
	if err == nil {
		err = os.Rename(f.Name(), h.path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}
