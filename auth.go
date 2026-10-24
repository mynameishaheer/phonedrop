package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Request signing, PROTOCOL.md §3. Shortcuts has no HMAC, so the signature is a nested SHA-256
// over the secret (as hex text) and the request's identifying parts.

const (
	maxSkew        = 90 * time.Second
	nonceTTL       = 5 * time.Minute
	maxNoncesPhone = 4096 // per phone per nonceTTL; a Shortcut can't get anywhere near this
)

var (
	errBodyHash = errors.New("body hash mismatch")
	errTooLarge = errors.New("too large")
)

// signRequest is what the phone computes (and what the Shortcuts guide documents):
//
//	inner = hex(SHA256(secret "|" METHOD "|" PATH "|" TIME "|" NONCE "|" BODY))
//	sig   = hex(SHA256(secret "|" inner))
//
// body is the X-PD-Body header value: the body's hex SHA-256, or "-" for no body.
func signRequest(secret, method, path, time, nonce, body string) string {
	inner := sha256.Sum256([]byte(secret + "|" + method + "|" + path + "|" + time + "|" + nonce + "|" + body))
	return hex.EncodeToString(outerSig(secret, hex.EncodeToString(inner[:])))
}

func outerSig(secret, inner string) []byte {
	s := sha256.Sum256([]byte(secret + "|" + inner))
	return s[:]
}

// authResult: who a verified request is from, and what its body has to hash to ("-" or
// lowercase hex).
type authResult struct {
	id   string
	body string
}

// verifier checks signed headers and remembers nonces. Only requests with a valid signature
// are recorded, so nobody without a secret can grow the nonce cache.
type verifier struct {
	now func() time.Time

	mu        sync.Mutex
	seen      map[string]map[string]time.Time // phone id → nonce → forget after
	lastSweep time.Time
}

func newVerifier(now func() time.Time) *verifier {
	return &verifier{now: now, seen: map[string]map[string]time.Time{}}
}

// dummySecret: signatures for unknown phone ids are still computed (against this), so the
// response time doesn't say whether an id exists.
const dummySecret = "0000000000000000000000000000000000000000000000000000000000000000"

// verify applies rules 1–4 of PROTOCOL.md §3. secretFor maps a phone id to its secret. Body
// hashing (rule 5) happens later, while the body streams: see verifyBody and saveBody.
func (v *verifier) verify(r *http.Request, secretFor func(id string) (string, bool)) (authResult, bool) {
	id, ok1 := header(r, "X-PD-Phone", 64)
	ts, ok2 := header(r, "X-PD-Time", 64)
	nonce, ok3 := header(r, "X-PD-Nonce", 128)
	body, ok4 := header(r, "X-PD-Body", 64)
	sigHex, ok5 := header(r, "X-PD-Sig", 64)
	if !(ok1 && ok2 && ok3 && ok4 && ok5) || !isHex(id) || !validNonce(nonce) ||
		!(body == "-" || len(body) == 64 && isHex(body)) {
		return authResult{}, false
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != sha256.Size {
		return authResult{}, false
	}
	now := v.now()
	t, ok := parseTime(ts)
	if !ok || t.Sub(now).Abs() > maxSkew {
		return authResult{}, false
	}
	secret, known := secretFor(id)
	if !known {
		secret = dummySecret
	}
	// inner may have been hex-encoded in either case; check both, always both.
	in := sha256.Sum256([]byte(secret + "|" + r.Method + "|" + r.URL.Path + "|" + ts + "|" + nonce + "|" + body))
	lower := hex.EncodeToString(in[:])
	m := subtle.ConstantTimeCompare(sig, outerSig(secret, lower)) |
		subtle.ConstantTimeCompare(sig, outerSig(secret, strings.ToUpper(lower)))
	if m != 1 || !known || !v.useNonce(id, nonce, now) {
		return authResult{}, false
	}
	return authResult{id: id, body: strings.ToLower(body)}, true
}

// header returns a header that is present exactly once and not too long.
func header(r *http.Request, name string, max int) (string, bool) {
	vs := r.Header.Values(name)
	if len(vs) != 1 || vs[0] == "" || len(vs[0]) > max {
		return "", false
	}
	return vs[0], true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
			return false
		}
	}
	return true
}

func validNonce(s string) bool {
	if len(s) < 8 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

// useNonce records a nonce, refusing one already seen for this phone within nonceTTL. Expired
// entries are swept lazily (at most once a minute, or when a phone's set is full); a full set
// refuses rather than forgets, since forgetting would allow a replay.
func (v *verifier) useNonce(id, nonce string, now time.Time) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if d := now.Sub(v.lastSweep); d > time.Minute || d < 0 {
		for pid, m := range v.seen {
			sweepNonces(m, now)
			if len(m) == 0 {
				delete(v.seen, pid)
			}
		}
		v.lastSweep = now
	}
	m := v.seen[id]
	if m == nil {
		m = map[string]time.Time{}
		v.seen[id] = m
	}
	if until, ok := m[nonce]; ok && now.Before(until) {
		return false
	}
	if len(m) >= maxNoncesPhone {
		if sweepNonces(m, now); len(m) >= maxNoncesPhone {
			return false
		}
	}
	m[nonce] = now.Add(nonceTTL)
	return true
}

func sweepNonces(m map[string]time.Time, now time.Time) {
	for n, until := range m {
		if !now.Before(until) {
			delete(m, n)
		}
	}
}

// parseTime reads X-PD-Time: ISO 8601 with an offset or Z (fractional seconds optional), or
// Unix seconds, possibly fractional and possibly formatted by Shortcuts with the phone's
// locale ("1,792,862,105", "1.792.862.105,5", "1 792 862 105").
func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, "Tt") || strings.Count(s, "-") >= 2 {
		u := strings.ToUpper(s)
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05Z07",
			"2006-01-02 15:04:05Z07:00", "2006-01-02 15:04:05Z0700"} {
			if t, err := time.Parse(layout, u); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	}
	return parseUnix(s)
}

func parseUnix(s string) (time.Time, bool) {
	// Drop spaces and apostrophes used as group separators (including NBSP and narrow NBSP).
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', ' ', ' ', ' ', '\'', '’':
			return -1
		}
		return r
	}, s)
	if s == "" {
		return time.Time{}, false
	}
	// The decimal separator is the last '.' or ',' unless it is plainly a group separator: the
	// only separator kind present and either repeated or followed by exactly three digits
	// after a short (1–3 digit) leading group.
	intPart, frac, hasFrac := s, "", false
	if i := strings.LastIndexAny(s, ".,"); i >= 0 {
		sep := s[i]
		other := map[byte]string{'.': ",", ',': "."}[sep]
		grouping := !strings.Contains(s, other) &&
			(strings.Count(s, string(sep)) > 1 || len(s)-i-1 == 3 && i <= 3)
		if !grouping {
			intPart, frac, hasFrac = s[:i], s[i+1:], true
		}
	}
	digits, ok := ungroup(intPart)
	if !ok || hasFrac && !allDigits(frac) {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || sec > 1e11 { // year 5138: nonsense, and keeps time arithmetic far from overflow
		return time.Time{}, false
	}
	var nsec int64
	if hasFrac {
		f := (frac + "000000000")[:9]
		nsec, _ = strconv.ParseInt(f, 10, 64)
	}
	return time.Unix(sec, nsec), true
}

// ungroup strips thousands separators ('.' or ','), insisting the groups are well formed.
func ungroup(s string) (string, bool) {
	if allDigits(s) {
		return s, true
	}
	groups := strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == ',' })
	if len(groups) < 2 || strings.Count(s, ".")+strings.Count(s, ",") != len(groups)-1 ||
		strings.Contains(s, ".") && strings.Contains(s, ",") {
		return "", false
	}
	for i, g := range groups {
		if !allDigits(g) || i == 0 && len(g) > 3 || i > 0 && len(g) != 3 {
			return "", false
		}
	}
	return strings.Join(groups, ""), true
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// checkBodyLen rejects a body that can't be right before reading any of it: "-" with a
// declared length, or a declared length over max.
func checkBodyLen(r *http.Request, want string, max int64) error {
	if want == "-" && r.ContentLength > 0 {
		return errBodyHash
	}
	if r.ContentLength > max {
		return errTooLarge
	}
	return nil
}

// verifyBody wraps a request body so that reading it to the end checks it against X-PD-Body
// (want: "-" or hex, any case). At EOF it returns errBodyHash instead of io.EOF if the hash
// differs; with "-", the first byte is already a mismatch.
func verifyBody(body io.Reader, want string) io.Reader {
	hr := &hashReader{r: body, h: sha256.New()}
	if want != "-" {
		var err error
		if hr.want, err = hex.DecodeString(want); err != nil || len(hr.want) != sha256.Size {
			hr.err = errBodyHash
		}
	}
	return hr
}

type hashReader struct {
	r    io.Reader
	h    hash.Hash
	want []byte // nil: the body must be empty
	err  error  // sticky
}

func (hr *hashReader) Read(p []byte) (int, error) {
	if hr.err != nil {
		return 0, hr.err
	}
	n, err := hr.r.Read(p)
	if n > 0 && hr.want == nil {
		hr.err = errBodyHash
		return 0, hr.err
	}
	hr.h.Write(p[:n])
	if err == io.EOF && hr.want != nil && subtle.ConstantTimeCompare(hr.h.Sum(nil), hr.want) != 1 {
		err = errBodyHash
	}
	if err != nil {
		hr.err = err
	}
	return n, err
}

// readBody reads a small body completely, verified against want, refusing more than max bytes.
func readBody(body io.Reader, want string, max int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(verifyBody(body, want), max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errTooLarge
	}
	return b, nil
}

// saveBody streams a body into a new temporary file in dir and returns its path only if the
// whole body arrived, is at most max bytes and matches want. On any failure the temporary file
// is gone. The caller renames or removes the file.
func saveBody(body io.Reader, want, dir string, max int64) (path string, size int64, err error) {
	return saveStream(verifyBody(body, want), dir, max)
}

// saveStream is saveBody for a body that is already verified (or verifies itself as it is read).
func saveStream(body io.Reader, dir string, max int64) (path string, size int64, err error) {
	f, err := os.CreateTemp(dir, partPattern)
	if err != nil {
		return "", 0, err
	}
	hideFile(f.Name()) // the one place upload temp files are made; place unhides the result
	size, err = io.Copy(f, io.LimitReader(body, max+1))
	if err == nil && size > max {
		err = errTooLarge
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(f.Name())
		return "", 0, err
	}
	return f.Name(), size, nil
}
