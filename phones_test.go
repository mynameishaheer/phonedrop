package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// xorProtect stands in for DPAPI: reversible, and nothing like the plaintext.
func xorProtect(b []byte) ([]byte, error) {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[i] ^ 0xA5
	}
	return out, nil
}

const (
	secretA = "1111111111111111111111111111111111111111111111111111111111111111"
	secretB = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
)

func TestPhoneStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "phones.json")
	s := loadPhones(path, xorProtect, xorProtect)
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	if err := s.add("00000000000000b2", secretB, "Second"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(-time.Hour) // paired earlier, so listed first
	if err := s.add("00000000000000a1", secretA, "First"); err != nil {
		t.Fatal(err)
	}
	if err := s.add("short", secretA, "x"); err == nil {
		t.Fatal("bad id accepted")
	}
	s.setTLS("00000000000000b2")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range []string{secretA, secretB} {
		for _, plain := range []string{sec, base64.StdEncoding.EncodeToString([]byte(sec)), strings.ToUpper(sec)} {
			if bytes.Contains(raw, []byte(plain)) {
				t.Fatalf("secret on disk in plain form: %s", raw)
			}
		}
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v", fi.Mode().Perm())
		}
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.tmp")); len(left) > 0 {
		t.Fatalf("temp files left: %v", left)
	}

	s2 := loadPhones(path, xorProtect, xorProtect)
	l := s2.list()
	if len(l) != 2 || l[0].Name != "First" || l[1].Name != "Second" || !l[1].TLS || l[0].TLS {
		t.Fatalf("list after reload: %+v", l)
	}
	if sec, ok := s2.secretFor("00000000000000a1"); !ok || sec != secretA {
		t.Fatal("secret didn't survive the round trip")
	}
	if _, ok := s2.get("00000000000000b2"); !ok {
		t.Fatal("get failed")
	}
	if err := s2.remove("00000000000000a1"); err != nil {
		t.Fatal(err)
	}
	s3 := loadPhones(path, xorProtect, xorProtect)
	if _, ok := s3.secretFor("00000000000000a1"); ok || len(s3.list()) != 1 {
		t.Fatal("removed phone came back")
	}
}

func TestPhoneStoreTouchIsThrottled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phones.json")
	s := loadPhones(path, xorProtect, xorProtect)
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	s.add("00000000000000a1", secretA, "A")
	seenOnDisk := func() time.Time {
		p, _ := loadPhones(path, xorProtect, xorProtect).get("00000000000000a1")
		return p.LastSeen
	}
	now = now.Add(10 * time.Second)
	s.touch("00000000000000a1")
	if p, _ := s.get("00000000000000a1"); !p.LastSeen.Equal(now) {
		t.Fatal("touch didn't update memory")
	}
	if seenOnDisk().Equal(now) {
		t.Fatal("touch wrote to disk within a minute")
	}
	now = now.Add(time.Minute)
	s.touch("00000000000000a1")
	if !seenOnDisk().Equal(now) {
		t.Fatal("touch never persisted")
	}
	s.touch("unknown") // no panic
}

func TestPhoneStoreSkipsBadEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phones.json")
	s := loadPhones(path, xorProtect, xorProtect)
	s.add("00000000000000a1", secretA, "Good")
	raw, _ := os.ReadFile(path)
	enc := func(s string) string { b, _ := xorProtect([]byte(s)); return base64.StdEncoding.EncodeToString(b) }
	bad := `{"id":"00000000000000b2","name":"Not base64","secret_protected":"!!!"},
		{"id":"00000000000000c3","name":"DPAPI says no","secret_protected":"` + base64.StdEncoding.EncodeToString([]byte("locked")) + `"},
		{"id":"00000000000000d4","name":"Short secret","secret_protected":"` + enc("abc") + `"},
		{"id":"NOT-AN-ID","name":"Bad id","secret_protected":"` + enc(secretB) + `"},
		"just a string",`
	raw = bytes.Replace(raw, []byte(`"phones": [`), []byte(`"phones": [`+bad), 1)
	os.WriteFile(path, raw, 0o600)

	unprotect := func(b []byte) ([]byte, error) {
		if string(b) == "locked" {
			return nil, errors.New("the key is gone")
		}
		return xorProtect(b)
	}
	l := loadPhones(path, xorProtect, unprotect).list()
	if len(l) != 1 || l[0].Name != "Good" {
		t.Fatalf("got %+v", l)
	}

	os.WriteFile(path, []byte("{not json"), 0o600)
	if l := loadPhones(path, xorProtect, xorProtect).list(); len(l) != 0 {
		t.Fatalf("corrupt file gave %+v", l)
	}
	if l := loadPhones(filepath.Join(t.TempDir(), "missing.json"), xorProtect, xorProtect).list(); len(l) != 0 {
		t.Fatal("missing file gave phones")
	}
}

func TestPhoneStoreAddFailsCleanly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phones.json")
	fail := func([]byte) ([]byte, error) { return nil, errors.New("no DPAPI") }
	s := loadPhones(path, fail, fail)
	if err := s.add("00000000000000a1", secretA, "A"); err == nil {
		t.Fatal("add succeeded without protection")
	}
	if len(s.list()) != 0 {
		t.Fatal("unsaved phone kept")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("file written without protection")
	}
}
