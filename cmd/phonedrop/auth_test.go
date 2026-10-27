package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

// The known-answer vector, also published in the Shortcuts guide.
const (
	kaSecret = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	kaTime   = "2026-10-24T22:15:24+05:00"
	kaNonce  = "n0nce-7f3a9c21"
	kaBody   = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" // SHA-256 of "hello"
	kaInner  = "f795c82b4e578020b2393c09a9ca484d8e006738bf732af151ef43a93dff74ad"
	kaSig    = "936b011316a67ab2656939ab738b1786fa40ca55380faeec5111d0693056caaf"
	kaSigUp  = "f14fe1f8e6dab993f61ab6229ca63ceaeb81a26f86a206d2b77149e51fc0c449" // inner in upper case
	kaID     = "9f2c1a7e5b3d4c60"
)

func TestSignRequestVector(t *testing.T) {
	inner := sha256.Sum256([]byte(kaSecret + "|POST|/v1/items|" + kaTime + "|" + kaNonce + "|" + kaBody))
	if got := hex.EncodeToString(inner[:]); got != kaInner {
		t.Fatalf("inner = %s", got)
	}
	if got := signRequest(kaSecret, "POST", "/v1/items", kaTime, kaNonce, kaBody); got != kaSig {
		t.Fatalf("sig = %s", got)
	}
	if got := hex.EncodeToString(outerSig(kaSecret, strings.ToUpper(kaInner))); got != kaSigUp {
		t.Fatalf("upper-case sig = %s", got)
	}
}

// signedHeaders builds a request the way a phone would.
func signedReq(method, path, id, secret, ts, nonce string, body []byte) *http.Request {
	bh := "-"
	if body != nil {
		h := sha256.Sum256(body)
		bh = hex.EncodeToString(h[:])
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("X-PD-Phone", id)
	r.Header.Set("X-PD-Time", ts)
	r.Header.Set("X-PD-Nonce", nonce)
	r.Header.Set("X-PD-Body", bh)
	r.Header.Set("X-PD-Sig", signRequest(secret, method, path, ts, nonce, bh))
	return r
}

func kaVerifier() (*verifier, func(string) (string, bool)) {
	now, _ := time.Parse(time.RFC3339, kaTime)
	v := newVerifier(func() time.Time { return now })
	return v, func(id string) (string, bool) { return kaSecret, id == kaID }
}

func TestVerifyKnownAnswer(t *testing.T) {
	v, lookup := kaVerifier()
	r := httptest.NewRequest("POST", "/v1/items?x=1", strings.NewReader("hello"))
	for k, val := range map[string]string{"X-PD-Phone": kaID, "X-PD-Time": kaTime, "X-PD-Nonce": kaNonce,
		"X-PD-Body": strings.ToUpper(kaBody), "X-PD-Sig": kaSig} {
		r.Header.Set(k, val)
	}
	// X-PD-Body is signed exactly as sent, so the upper-case header needs its own signature.
	r.Header.Set("X-PD-Sig", signRequest(kaSecret, "POST", "/v1/items", kaTime, kaNonce, strings.ToUpper(kaBody)))
	a, ok := v.verify(r, lookup)
	if !ok || a.id != kaID || a.body != kaBody {
		t.Fatalf("verify = %+v, %v", a, ok)
	}
}

func TestVerifyAcceptsHexCaseVariants(t *testing.T) {
	for i, sig := range []string{kaSig, strings.ToUpper(kaSig), kaSigUp, strings.ToUpper(kaSigUp)} {
		v, lookup := kaVerifier()
		r := httptest.NewRequest("POST", "/v1/items", strings.NewReader("hello"))
		for k, val := range map[string]string{"X-PD-Phone": kaID, "X-PD-Time": kaTime, "X-PD-Nonce": kaNonce,
			"X-PD-Body": kaBody, "X-PD-Sig": sig} {
			r.Header.Set(k, val)
		}
		if _, ok := v.verify(r, lookup); !ok {
			t.Errorf("variant %d rejected", i)
		}
	}
}

func TestVerifyRejects(t *testing.T) {
	good := func() *http.Request {
		return signedReq("POST", "/v1/items", kaID, kaSecret, kaTime, kaNonce, []byte("hello"))
	}
	cases := map[string]func(r *http.Request){
		"method":        func(r *http.Request) { r.Method = "PUT" },
		"path":          func(r *http.Request) { r.URL.Path = "/v1/outbox" },
		"body hash":     func(r *http.Request) { r.Header.Set("X-PD-Body", strings.Repeat("0", 64)) },
		"sig":           func(r *http.Request) { r.Header.Set("X-PD-Sig", strings.Repeat("a", 64)) },
		"sig short":     func(r *http.Request) { r.Header.Set("X-PD-Sig", kaSig[:62]) },
		"sig not hex":   func(r *http.Request) { r.Header.Set("X-PD-Sig", "zz"+kaSig[2:]) },
		"unknown phone": func(r *http.Request) { r.Header.Set("X-PD-Phone", "0000000000000000") },
		"no phone":      func(r *http.Request) { r.Header.Del("X-PD-Phone") },
		"two nonces":    func(r *http.Request) { r.Header.Add("X-PD-Nonce", "another-nonce") },
		"short nonce":   func(r *http.Request) { r.Header.Set("X-PD-Nonce", "abc") },
		"bad nonce":     func(r *http.Request) { r.Header.Set("X-PD-Nonce", "nonce with spaces") },
		"long nonce":    func(r *http.Request) { r.Header.Set("X-PD-Nonce", strings.Repeat("n", 129)) },
		"bad body hdr":  func(r *http.Request) { r.Header.Set("X-PD-Body", "nope") },
		"bad time":      func(r *http.Request) { r.Header.Set("X-PD-Time", "yesterday") },
		"other secret": func(r *http.Request) {
			r.Header.Set("X-PD-Sig", signRequest(strings.Repeat("b", 64), "POST", "/v1/items", kaTime, kaNonce, kaBody))
		},
	}
	for name, tamper := range cases {
		v, lookup := kaVerifier()
		r := good()
		tamper(r)
		if _, ok := v.verify(r, lookup); ok {
			t.Errorf("%s: accepted", name)
		}
	}
	v, lookup := kaVerifier()
	if _, ok := v.verify(good(), lookup); !ok {
		t.Fatal("untampered request rejected")
	}
}

func TestVerifyClockSkew(t *testing.T) {
	base, _ := time.Parse(time.RFC3339, kaTime)
	for _, c := range []struct {
		off time.Duration
		ok  bool
	}{{0, true}, {89 * time.Second, true}, {-90 * time.Second, true}, {91 * time.Second, false}, {-5 * time.Minute, false}} {
		v, lookup := kaVerifier()
		ts := base.Add(c.off).Format(time.RFC3339)
		r := signedReq("GET", "/v1/ping", kaID, kaSecret, ts, kaNonce, nil)
		if _, ok := v.verify(r, lookup); ok != c.ok {
			t.Errorf("offset %v: ok=%v", c.off, ok)
		}
	}
}

func TestVerifyRejectsReplay(t *testing.T) {
	v, lookup := kaVerifier()
	now, _ := time.Parse(time.RFC3339, kaTime)
	v.now = func() time.Time { return now }
	r := func() *http.Request { return signedReq("GET", "/v1/ping", kaID, kaSecret, kaTime, kaNonce, nil) }
	if _, ok := v.verify(r(), lookup); !ok {
		t.Fatal("first use rejected")
	}
	if _, ok := v.verify(r(), lookup); ok {
		t.Fatal("replay accepted")
	}
	// A failed signature doesn't burn (or record) a nonce.
	bad := signedReq("GET", "/v1/ping", kaID, kaSecret, kaTime, "fresh-nonce-1", nil)
	bad.Header.Set("X-PD-Sig", strings.Repeat("0", 64))
	v.verify(bad, lookup)
	if _, ok := v.verify(signedReq("GET", "/v1/ping", kaID, kaSecret, kaTime, "fresh-nonce-1", nil), lookup); !ok {
		t.Fatal("nonce burned by an unauthenticated request")
	}
	// Even after the nonce has been forgotten, the replay is too old to pass the time check.
	now = now.Add(nonceTTL + time.Second)
	if _, ok := v.verify(r(), lookup); ok {
		t.Fatal("stale replay accepted")
	}
	fresh := signedReq("GET", "/v1/ping", kaID, kaSecret, now.Format(time.RFC3339), "fresh-nonce-2", nil)
	if _, ok := v.verify(fresh, lookup); !ok {
		t.Fatal("fresh request rejected")
	}
	if len(v.seen[kaID]) != 1 {
		t.Fatalf("expired nonces not swept: %d kept", len(v.seen[kaID]))
	}
}

func TestNonceCacheIsBounded(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	v := newVerifier(func() time.Time { return now })
	for i := 0; i < maxNoncesPhone; i++ {
		if !v.useNonce("p", "n"+itoa(i), now) {
			t.Fatalf("nonce %d refused", i)
		}
	}
	if v.useNonce("p", "one-too-many", now) {
		t.Fatal("full cache accepted another nonce")
	}
	if !v.useNonce("q", "other-phone", now) {
		t.Fatal("one phone's full cache blocked another")
	}
	now = now.Add(nonceTTL)
	if !v.useNonce("p", "after-expiry", now) {
		t.Fatal("cache didn't free expired nonces")
	}
	if n := len(v.seen["p"]); n != 1 {
		t.Fatalf("%d nonces kept", n)
	}
}

func TestParseTime(t *testing.T) {
	want := time.Date(2026, 10, 24, 17, 15, 24, 0, time.UTC)
	for _, s := range []string{
		"2026-10-24T22:15:24+05:00", "2026-10-24T17:15:24Z", "2026-10-24T17:15:24z",
		"2026-10-24T22:15:24.000+05:00", "2026-10-24T22:15:24+0500", "2026-10-24T12:15:24-05:00",
		"2026-10-24 22:15:24+05:00", " 2026-10-24T17:15:24Z ",
		"1792862124", "1,792,862,124", "1.792.862.124", "1 792 862 124", "1 792 862 124",
		"1'792'862'124", "1792862124.0", "1792862124,0",
	} {
		got, ok := parseTime(s)
		if !ok || !got.Equal(want) {
			t.Errorf("parseTime(%q) = %v, %v", s, got, ok)
		}
	}
	frac := want.Add(250 * time.Millisecond)
	for _, s := range []string{"2026-10-24T17:15:24.25Z", "1792862124.25", "1792862124,25",
		"1,792,862,124.25", "1.792.862.124,25", "2026-10-24T22:15:24,25+05:00"} {
		got, ok := parseTime(s)
		if !ok || !got.Equal(frac) {
			t.Errorf("parseTime(%q) = %v, %v", s, got, ok)
		}
	}
	for _, s := range []string{"", "now", "2026-10-24T22:15:24", "2026-10-24", "17928621x4", "-1792862124",
		"1,79,2862,124", "1,792.862,124.5", "1792862124.", "99999999999999999999", "1e9", "0x6ADC8A6C"} {
		if got, ok := parseTime(s); ok {
			t.Errorf("parseTime(%q) accepted: %v", s, got)
		}
	}
}

func hashOf(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestVerifyBody(t *testing.T) {
	body := []byte("the quick brown fox")
	got, err := readBody(bytes.NewReader(body), strings.ToUpper(hashOf(body)), 100)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("good body: %q, %v", got, err)
	}
	if _, err := readBody(bytes.NewReader(body), hashOf([]byte("other")), 100); !errors.Is(err, errBodyHash) {
		t.Fatalf("wrong hash: %v", err)
	}
	if _, err := readBody(bytes.NewReader(body), hashOf(body), 10); !errors.Is(err, errTooLarge) {
		t.Fatalf("too large: %v", err)
	}
	if b, err := readBody(bytes.NewReader(nil), "-", 10); err != nil || len(b) != 0 {
		t.Fatalf("empty with -: %q, %v", b, err)
	}
	if _, err := readBody(bytes.NewReader([]byte("x")), "-", 10); !errors.Is(err, errBodyHash) {
		t.Fatalf("body with -: %v", err)
	}
	// One byte at a time, the check still happens at EOF.
	if _, err := io.ReadAll(verifyBody(iotest.OneByteReader(bytes.NewReader(body)), hashOf(body[:5]))); !errors.Is(err, errBodyHash) {
		t.Fatalf("slow reader: %v", err)
	}
	// A transport error is passed through, not mistaken for a good body.
	_, err = io.ReadAll(verifyBody(iotest.ErrReader(io.ErrUnexpectedEOF), hashOf(nil)))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("transport error: %v", err)
	}
}

func TestCheckBodyLen(t *testing.T) {
	r := httptest.NewRequest("GET", "/v1/ping", strings.NewReader("x"))
	if err := checkBodyLen(r, "-", 100); !errors.Is(err, errBodyHash) {
		t.Fatalf("- with Content-Length 1: %v", err)
	}
	r = httptest.NewRequest("POST", "/v1/items", strings.NewReader(strings.Repeat("x", 101)))
	if err := checkBodyLen(r, hashOf(nil), 100); !errors.Is(err, errTooLarge) {
		t.Fatalf("over max: %v", err)
	}
	r = httptest.NewRequest("GET", "/v1/ping", nil)
	if err := checkBodyLen(r, "-", 100); err != nil {
		t.Fatalf("empty: %v", err)
	}
}

func TestSaveBody(t *testing.T) {
	dir := t.TempDir()
	body := bytes.Repeat([]byte("0123456789"), 10_000)
	path, n, err := saveBody(bytes.NewReader(body), hashOf(body), dir, 1<<20)
	if err != nil || n != int64(len(body)) {
		t.Fatalf("saveBody: %d, %v", n, err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, body) {
		t.Fatal("saved content differs")
	}
	os.Remove(path)

	tampered := append([]byte(nil), body...)
	tampered[len(tampered)-1] ^= 1
	for name, c := range map[string]struct {
		r    io.Reader
		want string
		max  int64
		err  error
	}{
		"mismatch":  {bytes.NewReader(tampered), hashOf(body), 1 << 20, errBodyHash},
		"too large": {bytes.NewReader(body), hashOf(body), 1000, errTooLarge},
		"dash":      {bytes.NewReader(body), "-", 1 << 20, errBodyHash},
		"truncated": {io.MultiReader(bytes.NewReader(body[:500]), iotest.ErrReader(io.ErrUnexpectedEOF)), hashOf(body), 1 << 20, io.ErrUnexpectedEOF},
	} {
		if _, _, err := saveBody(c.r, c.want, dir, c.max); !errors.Is(err, c.err) {
			t.Errorf("%s: err = %v", name, err)
		}
		if left, _ := os.ReadDir(dir); len(left) != 0 {
			t.Errorf("%s: temp file left behind: %v", name, left[0].Name())
		}
	}
}
