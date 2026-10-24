package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Pairing (PROTOCOL.md §2, §5): the panel shows a QR code holding a fresh phone id and secret;
// the phone proves it read the code by signing POST /v1/pair with them, and the user confirms.
// The pending pairing is single use and lives at most pairWindow.

const (
	pairWindow     = 2 * time.Minute
	confirmTimeout = 50 * time.Second
	maxPairBody    = 4 << 10
	defaultPhone   = "iPhone"
)

// pairingInfo overrides what the QR code advertises; zero fields get the defaults (the
// computer name, <hostname>.local, and the addresses currently listening).
type pairingInfo struct {
	Name string
	Host string
	IPs  []netip.Addr
}

type pendingPair struct {
	id, secret string
	laptop     string
	expires    time.Time
}

// confirmation: the one pairing confirmation that may be in flight.
type confirmation struct{ cancel context.CancelFunc }

// qrPayload: field names and order are the wire contract (§2).
type qrPayload struct {
	PD     int      `json:"pd"`
	Name   string   `json:"name"`
	Host   string   `json:"host"`
	IPs    []string `json:"ips"`
	Port   int      `json:"port"`
	Base   string   `json:"base"` // "<host>:<port>", so a shortcut never has to format the port number
	ID     string   `json:"id"`
	Secret string   `json:"secret"`
	TLS    bool     `json:"tls,omitempty"` // only when TLS is on (§7)
	CA     string   `json:"ca,omitempty"`  // hex SHA-256 of the CA profile, with tls
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b) // never fails (Go 1.24)
	return hex.EncodeToString(b)
}

// laptopName: the computer name as Windows shows it (first label only).
func laptopName() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "PC"
	}
	h, _, _ = strings.Cut(h, ".")
	return h
}

// mdnsHost: the name phones use for this PC, <hostname>.local in lower case (the QR code's host
// and the name the TLS certificates are for).
func mdnsHost() string { return strings.ToLower(laptopName()) + ".local" }

// startPairing opens a pairing window and returns the QR code's JSON and when the window
// closes. It replaces any earlier pending pairing and aborts a confirmation in flight.
func (s *server) startPairing(info pairingInfo) (qrJSON string, expires time.Time) {
	if info.Name == "" {
		info.Name = s.laptop
	}
	if info.Host == "" {
		info.Host = mdnsHost()
	}
	if info.IPs == nil {
		info.IPs = s.listening()
	}
	ips := make([]string, 0, len(info.IPs))
	for _, a := range info.IPs {
		ips = append(ips, a.String())
	}
	p := &pendingPair{id: randHex(8), secret: randHex(32), laptop: info.Name, expires: s.now().Add(pairWindow)}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	q := qrPayload{PD: 1, Name: info.Name, Host: info.Host, IPs: ips, Port: s.port,
		Base: info.Host + ":" + strconv.Itoa(s.port), ID: p.id, Secret: p.secret}
	if h := s.caProfileHash(); h != "" {
		q.TLS, q.CA = true, h
	}
	enc.Encode(q)

	s.pairMu.Lock()
	s.pending = p
	if s.confirming != nil {
		s.confirming.cancel()
	}
	s.pairMu.Unlock()
	log.Printf("pairing window open")
	return strings.TrimSpace(buf.String()), p.expires
}

// cancelPairing closes the pairing window. A confirmation in flight is aborted (the phone is
// told "denied"), so the UI should call this when the user closes the pairing view.
func (s *server) cancelPairing() {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	s.pending = nil
	if s.confirming != nil {
		s.confirming.cancel()
	}
}

// pendingPairing returns the open pairing, or nil once it has expired.
func (s *server) pendingPairing() *pendingPair {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	if s.pending != nil && !s.now().Before(s.pending.expires) {
		s.pending = nil
	}
	return s.pending
}

var errBusy = errors.New("busy")

// claim burns pending pairing p (single use) and registers the confirmation, atomically, so a
// cancel can never slip between the two. It fails if p is no longer the open pairing; it burns
// p but returns errBusy if another confirmation is still in flight.
func (s *server) claim(p *pendingPair, c *confirmation) error {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	if p == nil || s.pending != p || !s.now().Before(p.expires) {
		return errors.New("no pairing")
	}
	s.pending = nil
	if s.confirming != nil {
		return errBusy
	}
	s.confirming = c
	return nil
}

func (s *server) release(c *confirmation) {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	if s.confirming == c {
		s.confirming = nil
	}
}

func (s *server) handlePair(w http.ResponseWriter, r *http.Request) {
	p := s.pendingPairing()
	a, ok := s.auth.verify(r, func(id string) (string, bool) {
		if p != nil && id == p.id {
			return p.secret, true
		}
		return "", false
	})
	if !ok {
		s.unauthorized(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.confirmTimeout)
	defer cancel()
	c := &confirmation{cancel: cancel}
	switch err := s.claim(p, c); {
	case errors.Is(err, errBusy):
		replyErr(w, http.StatusForbidden, "denied")
		return
	case err != nil: // cancelled or replaced while the request was being checked
		s.unauthorized(w, r)
		return
	}
	defer s.release(c)

	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := checkBodyLen(r, a.body, maxPairBody); err != nil {
		bodyError(w, err)
		return
	}
	// 4 KB can't take long; don't let a trickled body hold the one confirmation slot.
	http.NewResponseController(w).SetReadDeadline(time.Now().Add(15 * time.Second))
	body, err := readBody(r.Body, a.body, maxPairBody)
	if err != nil {
		bodyError(w, err)
		return
	}
	name, ok := pairName(body)
	if !ok {
		replyErr(w, http.StatusBadRequest, "bad name")
		return
	}
	if !s.confirm(ctx, name) {
		log.Printf("pairing: phone %s not allowed", a.id)
		replyErr(w, http.StatusForbidden, "denied")
		return
	}
	if err := s.phones.add(a.id, p.secret, name); err != nil {
		log.Printf("pairing: saving phone %s: %v", a.id, err)
		replyErr(w, http.StatusInternalServerError, "could not save the pairing")
		return
	}
	log.Printf("pairing: phone %s paired", a.id)
	s.hooks.phonesChanged()
	reply(w, http.StatusOK, struct {
		OK     bool   `json:"ok"`
		Laptop string `json:"laptop"`
	}{true, p.laptop})
}

// confirm asks the user, giving up when ctx ends even if the hook doesn't.
func (s *server) confirm(ctx context.Context, name string) bool {
	if ctx.Err() != nil {
		return false
	}
	res := make(chan bool, 1)
	go func() { res <- s.hooks.confirmPairing(ctx, name) }()
	select {
	case ok := <-res:
		return ok && ctx.Err() == nil
	case <-ctx.Done():
		return false
	}
}

// pairName reads the phone's name from the pair body: JSON {"name": ...} if the body starts
// with '{', otherwise the body itself as text (easier to build in Shortcuts). Empty: "iPhone".
func pairName(body []byte) (string, bool) {
	t := bytes.TrimSpace(body)
	if bytes.HasPrefix(t, []byte("{")) {
		var v struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(t, &v) != nil {
			return "", false
		}
		return sanitizePhoneName(v.Name), true
	}
	return sanitizePhoneName(string(t)), true
}

// sanitizePhoneName makes a phone-supplied name safe to show: valid UTF-8, no control or format
// characters (which includes bidi overrides that could disguise the name), whitespace runs
// collapsed, at most 64 runes.
func sanitizePhoneName(s string) string {
	var b strings.Builder
	n, space := 0, false
	for _, r := range strings.ToValidUTF8(s, "") {
		switch {
		case unicode.IsSpace(r):
			space = b.Len() > 0
			continue
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r), r == utf8.RuneError,
			unicode.Is(unicode.Co, r), unicode.Is(unicode.Cs, r):
			continue
		}
		if space {
			if n++; n > 64 {
				break
			}
			b.WriteByte(' ')
			space = false
		}
		if n++; n > 64 {
			break
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return defaultPhone
	}
	return out
}
