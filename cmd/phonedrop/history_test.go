package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHistoryOrderCapPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "history.json")
	h := loadHistory(path)
	if len(h.recent(10)) != 0 {
		t.Fatal("new history not empty")
	}
	base := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		h.add(historyEntry{Kind: "file", Name: fmt.Sprintf("f%d", i), Time: base.Add(time.Duration(i) * time.Second), Dir: "in"})
	}
	got := h.recent(0)
	if len(got) != historyMax {
		t.Fatalf("len = %d, want %d", len(got), historyMax)
	}
	if got[0].Name != "f59" || got[49].Name != "f10" {
		t.Errorf("order: %s ... %s", got[0].Name, got[49].Name)
	}
	if r := h.recent(3); len(r) != 3 || r[2].Name != "f57" {
		t.Errorf("recent(3) = %v", r)
	}

	// recent returns a copy.
	got[0].Name = "mutated"
	if h.recent(1)[0].Name != "f59" {
		t.Error("recent aliases internal slice")
	}

	// Persistence round trip, file mode and no temp leftovers.
	h2 := loadHistory(path)
	g2 := h2.recent(0)
	if len(g2) != historyMax || g2[0].Name != "f59" || !g2[0].Time.Equal(base.Add(59*time.Second)) || g2[0].Dir != "in" {
		t.Errorf("reload mismatch: %+v", g2[0])
	}
	if fi, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if fi.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("mode %v", fi.Mode())
	}
	ents, _ := os.ReadDir(filepath.Dir(path))
	if len(ents) != 1 {
		t.Errorf("leftover files: %v", ents)
	}

	h2.remove(0)
	h2.remove(-1)
	h2.remove(999)
	if r := loadHistory(path).recent(0); len(r) != historyMax-1 || r[0].Name != "f58" {
		t.Errorf("after remove: %d %s", len(r), r[0].Name)
	}
	h2.clear()
	if len(loadHistory(path).recent(0)) != 0 {
		t.Error("clear did not persist")
	}
}

func TestHistoryTextTruncated(t *testing.T) {
	h := loadHistory(filepath.Join(t.TempDir(), "h.json"))
	h.add(historyEntry{Kind: "text", Text: strings.Repeat("é✓", 3000)})
	h.add(historyEntry{Kind: "text", Text: "short"})
	r := h.recent(0)
	if n := len([]rune(r[1].Text)); n != historyMaxText {
		t.Errorf("stored %d runes", n)
	}
	if r[0].Text != "short" {
		t.Errorf("short text changed: %q", r[0].Text)
	}
}

func TestHistoryCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	for _, content := range []string{"{not json", "", `{"a":1}`, "\x00\x01"} {
		os.WriteFile(path, []byte(content), 0o600)
		h := loadHistory(path)
		if len(h.recent(0)) != 0 {
			t.Fatalf("corrupt %q not empty", content)
		}
		h.add(historyEntry{Kind: "url", Text: "https://example.com"})
		if len(loadHistory(path).recent(0)) != 1 {
			t.Errorf("could not recover after corrupt %q", content)
		}
	}
}

func TestHistoryConcurrent(t *testing.T) {
	h := loadHistory(filepath.Join(t.TempDir(), "h.json"))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.add(historyEntry{Kind: "text", Text: "x"})
			h.recent(5)
		}()
	}
	wg.Wait()
	if n := len(h.recent(0)); n != 20 {
		t.Errorf("len = %d", n)
	}
}
