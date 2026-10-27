package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Optional TLS, PROTOCOL.md §7. A private CA, name-constrained to this PC's .local name and the
// private IP ranges, issues an in-memory leaf for the addresses currently bound. A paired phone
// downloads the CA as a configuration profile (mobileconfig.go) and checks it against the hash
// in its QR code. HTTPS shares the plain-HTTP port: the
// first byte of a connection tells them apart (sniffListener).

const (
	caValidity   = 10 * 365 * 24 * time.Hour
	leafValidity = 397 * 24 * time.Hour
	leafRenew    = 30 * 24 * time.Hour // re-issue a leaf (or replace the CA) this close to expiry
	sniffTimeout = 10 * time.Second
)

// caIPRanges: the addresses a new CA may certify besides the host name. A variable only so the
// tests can add loopback; production CAs permit exactly these.
var caIPRanges = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16",
	"fc00::/7", "fe80::/10"}

// caNoName is the only email domain and URI host a CA may certify: the reserved TLD .invalid
// (RFC 6761), which names nothing, so a stolen key can't issue S/MIME or URI certificates.
const caNoName = "invalid"

func tlsDir() string { return filepath.Join(dataDir(), "tls") }

// tlsManager owns the CA, the leaf and the profile window. The leaf is read lock-free by every
// handshake; everything else is under mu.
type tlsManager struct {
	dir                string // ca.crt, ca.key
	protect, unprotect func([]byte) ([]byte, error)
	host               string // the .local name; the CA is constrained to it
	now                func() time.Time
	sniffTimeout       time.Duration
	config             *tls.Config

	enabled atomic.Bool
	leaf    atomic.Pointer[tls.Certificate]

	mu    sync.Mutex
	ca    *x509.Certificate
	caKey *ecdsa.PrivateKey
	addrs []netip.Addr // bound addresses (sorted), the leaf's IP SANs
}

func newTLSManager() *tlsManager {
	m := &tlsManager{dir: tlsDir(), protect: dpapiProtect, unprotect: dpapiUnprotect, host: mdnsHost(),
		now: time.Now, sniffTimeout: sniffTimeout}
	m.config = &tls.Config{
		MinVersion:     tls.VersionTLS12,
		NextProtos:     []string{"http/1.1"},
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return m.certificate() },
	}
	return m
}

// setTLS turns encryption on (loading or creating the CA and issuing a leaf) or off. Turning it
// off never locks a phone out: plain HTTP is accepted again from every phone.
func (s *server) setTLS(enabled bool) error { return s.tlsm.setEnabled(enabled) }

func (s *server) tlsEnabled() bool { return s.tlsm.enabled.Load() }

// caFingerprint: hex SHA-256 of the CA certificate, for the UI to show (and the user to compare
// with the profile's details on the iPhone). Empty while TLS is off.
func (s *server) caFingerprint() string { return s.tlsm.fingerprint() }

// resetTLS deletes the CA so a new one is made (now, if TLS is on). Every phone has to install
// the new profile.
func (s *server) resetTLS() error { return s.tlsm.reset() }

func (m *tlsManager) setEnabled(on bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !on { // the CA key needn't stay in memory; turning TLS on again reloads it
		m.enabled.Store(false)
		m.ca, m.caKey = nil, nil
		m.leaf.Store(nil)
		log.Printf("tls: off")
		return nil
	}
	if err := m.loadOrCreateLocked(); err != nil {
		return err
	}
	if err := m.issueLocked(); err != nil {
		return err
	}
	m.enabled.Store(true)
	log.Printf("tls: on")
	return nil
}

func (m *tlsManager) fingerprint() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ca == nil {
		return ""
	}
	h := sha256.Sum256(m.ca.Raw)
	return hex.EncodeToString(h[:])
}

func (m *tlsManager) reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range []string{"ca.key", "ca.crt"} {
		if err := os.Remove(filepath.Join(m.dir, f)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	m.ca, m.caKey = nil, nil
	m.leaf.Store(nil)
	log.Printf("tls: CA deleted")
	if !m.enabled.Load() {
		return nil
	}
	if err := m.loadOrCreateLocked(); err != nil {
		m.enabled.Store(false)
		return err
	}
	if err := m.issueLocked(); err != nil {
		m.enabled.Store(false)
		return err
	}
	return nil
}

// setAddrs records the bound addresses; a changed set gets a new leaf at once if TLS is on.
func (m *tlsManager) setAddrs(addrs []netip.Addr) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if slices.Equal(addrs, m.addrs) {
		return
	}
	m.addrs = slices.Clone(addrs)
	if m.enabled.Load() && m.ca != nil {
		if err := m.issueLocked(); err != nil {
			log.Printf("tls: issuing certificate: %v", err)
		}
	}
}

// certificate is GetCertificate: the current leaf, re-issued first if it is close to expiry.
func (m *tlsManager) certificate() (*tls.Certificate, error) {
	if c := m.leaf.Load(); c != nil && m.fresh(c) {
		return c, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.leaf.Load(); c != nil && m.fresh(c) {
		return c, nil
	}
	if m.ca == nil {
		return nil, errors.New("tls: no CA")
	}
	if err := m.issueLocked(); err != nil {
		log.Printf("tls: issuing certificate: %v", err)
		return nil, err
	}
	return m.leaf.Load(), nil
}

func (m *tlsManager) fresh(c *tls.Certificate) bool {
	return m.now().Before(c.Leaf.NotAfter.Add(-leafRenew))
}

// profileCA: the CA's DER and host, while TLS is on.
func (m *tlsManager) profileCA() (der []byte, host string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled.Load() || m.ca == nil {
		return nil, "", false
	}
	return m.ca.Raw, caHost(m.ca), true
}

// caHost: the one DNS name a PhoneDrop CA is constrained to.
func caHost(ca *x509.Certificate) string {
	if len(ca.PermittedDNSDomains) != 1 {
		return ""
	}
	return ca.PermittedDNSDomains[0]
}

// loadOrCreateLocked makes sure m.ca is a usable CA for m.host: the one in memory, the stored
// one, or a new one (saved before use, so it survives a restart).
func (m *tlsManager) loadOrCreateLocked() error {
	if m.ca != nil && m.usable(m.ca) {
		return nil
	}
	ca, key, err := m.loadCA()
	switch {
	case err == nil && caHost(ca) != m.host:
		log.Printf("tls: computer renamed; making a new CA (phones must install the new profile)")
	case err == nil && !caConstrained(ca):
		log.Printf("tls: CA lacks the server-only and email/URI constraints; making a new one (phones must install the new profile)")
	case err == nil && !m.usable(ca):
		log.Printf("tls: CA expiring; making a new one (phones must install the new profile)")
	case err == nil:
		m.ca, m.caKey = ca, key
		m.leaf.Store(nil)
		return nil
	case errors.Is(err, os.ErrNotExist):
	default:
		log.Printf("tls: stored CA unusable (%v); making a new one", err)
	}
	ca, key, err = newCA(m.host, m.now(), caIPRanges)
	if err != nil {
		return err
	}
	if err := m.saveCA(ca, key); err != nil {
		return err
	}
	m.ca, m.caKey = ca, key
	m.leaf.Store(nil)
	log.Printf("tls: new CA created")
	return nil
}

func (m *tlsManager) usable(ca *x509.Certificate) bool {
	return caHost(ca) == m.host && caConstrained(ca) && m.now().Before(ca.NotAfter.Add(-leafRenew))
}

// caConstrained: the CA is limited to server certificates and permits no real email address or
// URI. CAs made by earlier versions lack this and are replaced.
func caConstrained(ca *x509.Certificate) bool {
	return slices.Equal(ca.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}) &&
		slices.Equal(ca.PermittedEmailAddresses, []string{caNoName}) &&
		slices.Equal(ca.PermittedURIDomains, []string{caNoName})
}

// permittedIP reports whether the CA's name constraints allow ip.
func permittedIP(ca *x509.Certificate, ip net.IP) bool {
	for _, n := range ca.PermittedIPRanges {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (m *tlsManager) loadCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	crtPEM, err := os.ReadFile(filepath.Join(m.dir, "ca.crt"))
	if err != nil {
		return nil, nil, err
	}
	keyB64, err := os.ReadFile(filepath.Join(m.dir, "ca.key"))
	if err != nil {
		return nil, nil, err
	}
	blk, _ := pem.Decode(crtPEM)
	if blk == nil || blk.Type != "CERTIFICATE" {
		return nil, nil, errors.New("ca.crt is not a PEM certificate")
	}
	ca, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		return nil, nil, err
	}
	enc, err := base64.StdEncoding.DecodeString(string(keyB64))
	if err != nil {
		return nil, nil, errors.New("ca.key is not base64")
	}
	der, err := m.unprotect(enc)
	if err != nil {
		return nil, nil, errors.New("can't unprotect ca.key: " + err.Error())
	}
	k, err := x509.ParsePKCS8PrivateKey(der)
	clear(der)
	if err != nil {
		return nil, nil, errors.New("ca.key is malformed")
	}
	key, ok := k.(*ecdsa.PrivateKey)
	if !ok || !key.PublicKey.Equal(ca.PublicKey) || !ca.IsCA {
		return nil, nil, errors.New("ca.key doesn't match ca.crt")
	}
	return ca, key, nil
}

// saveCA writes the key (only ever as base64(protect(PKCS#8))) and then the certificate.
func (m *tlsManager) saveCA(ca *x509.Certificate, key *ecdsa.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	enc, err := m.protect(der)
	clear(der)
	if err != nil {
		return errors.New("protecting CA key: " + err.Error())
	}
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(m.dir, "ca.key"), []byte(base64.StdEncoding.EncodeToString(enc))); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(m.dir, "ca.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw}))
}

// writeAtomic: temp file (created 0600) + rename, so a crash never leaves half a file.
func writeAtomic(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

func randSerial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return n.Add(n, big.NewInt(1)) // never zero
}

// newCA: self-signed, path length 0, server certificates only, and critically name-constrained,
// so even a stolen key can only vouch for this PC's name and LAN addresses.
func newCA(host string, now time.Time, ipRanges []string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	var ranges []*net.IPNet
	for _, r := range ipRanges {
		_, n, err := net.ParseCIDR(r)
		if err != nil {
			return nil, nil, err
		}
		ranges = append(ranges, n)
	}
	tmpl := &x509.Certificate{
		SerialNumber:                randSerial(),
		Subject:                     pkix.Name{CommonName: "PhoneDrop CA (" + host + ")", Organization: []string{"PhoneDrop"}},
		NotBefore:                   now.Add(-time.Hour),
		NotAfter:                    now.Add(caValidity),
		IsCA:                        true,
		BasicConstraintsValid:       true,
		MaxPathLen:                  0,
		MaxPathLenZero:              true,
		KeyUsage:                    x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:                 []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		PermittedDNSDomainsCritical: true,
		PermittedDNSDomains:         []string{host},
		PermittedIPRanges:           ranges,
		PermittedEmailAddresses:     []string{caNoName},
		PermittedURIDomains:         []string{caNoName},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	ca, err := x509.ParseCertificate(der)
	return ca, key, err
}

// issueLocked makes a new leaf for m.host and m.addrs and publishes it to new handshakes.
func (m *tlsManager) issueLocked() error {
	if m.ca == nil {
		return errors.New("tls: no CA")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	now := m.now()
	notAfter := now.Add(leafValidity)
	if m.ca.NotAfter.Before(notAfter) {
		notAfter = m.ca.NotAfter
	}
	ips := make([]net.IP, 0, len(m.addrs))
	for _, a := range m.addrs {
		// Only what the CA may certify: one address outside its ranges would make clients
		// reject the whole leaf. (The network watcher filters too; this is the backstop.)
		if ip := net.IP(a.Unmap().AsSlice()); permittedIP(m.ca, ip) {
			ips = append(ips, ip)
		}
	}
	tmpl := &x509.Certificate{
		SerialNumber:          randSerial(),
		Subject:               pkix.Name{CommonName: m.host},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{m.host},
		IPAddresses:           ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, m.ca, &key.PublicKey, m.caKey)
	if err != nil {
		return err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return err
	}
	m.leaf.Store(&tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf})
	return nil
}

// httpsOnly applies §7 after a request is authenticated: an HTTPS request marks the phone, and a
// marked phone's plain-HTTP requests are refused while TLS is on, except the CA profile (after a
// reset the phone can't do HTTPS until it has the new one). It reports whether to go on.
func (s *server) httpsOnly(w http.ResponseWriter, r *http.Request, id string) bool {
	p, ok := s.phones.get(id)
	if !ok {
		return true
	}
	if r.TLS != nil {
		if !p.TLS {
			s.phones.setTLS(id)
		}
		return true
	}
	if p.TLS && s.tlsEnabled() && r.URL.Path != caProfilePath {
		w.Header().Set("Connection", "close")
		replyErr(w, http.StatusForbidden, "use https")
		return false
	}
	return true
}

// sniffListener tells HTTPS from plain HTTP on one port by the first byte (0x16 starts a TLS
// handshake). Its own goroutine accepts (through the gate, so dropped peers are never read
// from); each new connection is read in a goroutine of its own, so a client that connects and
// sends nothing can't hold up anyone else. With TLS off, connections are passed on unread.
type sniffListener struct {
	net.Listener // the gated listener
	tls          *tlsManager
	gate         *gate // a peer that connects and sends nothing counts as a failure; may be nil
	conns        chan net.Conn
	errs         chan error
	done         chan struct{}

	mu       sync.Mutex
	closed   bool
	sniffing map[net.Conn]struct{}
}

func newSniffListener(l net.Listener, m *tlsManager) *sniffListener {
	sl := &sniffListener{Listener: l, tls: m, conns: make(chan net.Conn), errs: make(chan error),
		done: make(chan struct{}), sniffing: map[net.Conn]struct{}{}}
	if gl, ok := l.(*gatedListener); ok {
		sl.gate = gl.gate
	}
	go sl.loop()
	return sl
}

func (l *sniffListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case err := <-l.errs:
		return nil, err
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// Close stops the accept loop and closes connections still being sniffed.
func (l *sniffListener) Close() error {
	l.mu.Lock()
	if !l.closed {
		l.closed = true
		close(l.done)
		for c := range l.sniffing {
			c.Close()
		}
	}
	l.mu.Unlock()
	return l.Listener.Close()
}

func (l *sniffListener) loop() {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			// Handing the error over waits for net/http's next Accept, which keeps its backoff
			// for temporary errors.
			select {
			case l.errs <- err:
			case <-l.done:
				return
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		if !l.tls.enabled.Load() {
			l.deliver(c)
			continue
		}
		if !l.track(c) {
			c.Close()
			return
		}
		go l.sniff(c)
	}
}

func (l *sniffListener) deliver(c net.Conn) {
	select {
	case l.conns <- c:
	case <-l.done:
		c.Close()
	}
}

func (l *sniffListener) track(c net.Conn) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	l.sniffing[c] = struct{}{}
	return true
}

func (l *sniffListener) sniff(c net.Conn) {
	var b [1]byte
	c.SetReadDeadline(time.Now().Add(l.tls.sniffTimeout))
	_, err := io.ReadFull(c, b[:])
	c.SetReadDeadline(time.Time{})
	l.mu.Lock()
	delete(l.sniffing, c)
	l.mu.Unlock()
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() && l.gate != nil {
			l.gate.fail(peerIP(c.RemoteAddr().String()))
		}
		c.Close()
		return
	}
	pc := &peekedConn{Conn: c, first: b[0], has: true}
	if b[0] == 0x16 { // TLS record type: handshake
		l.deliver(tls.Server(pc, l.tls.config)) // net/http fills r.TLS only for a *tls.Conn
		return
	}
	l.deliver(pc)
}

// peekedConn gives back the byte the sniffer read before the rest of the connection.
type peekedConn struct {
	net.Conn
	first byte
	has   bool
}

func (c *peekedConn) Read(p []byte) (int, error) {
	if c.has && len(p) > 0 {
		p[0], c.has = c.first, false
		return 1, nil
	}
	return c.Conn.Read(p)
}

// ReadFrom keeps net/http's sendfile path for large plain-HTTP responses.
func (c *peekedConn) ReadFrom(r io.Reader) (int64, error) { return io.Copy(c.Conn, r) }

// CloseWrite lets net/http half-close before closing, so a response sent while the request
// body is still arriving isn't lost to a reset.
func (c *peekedConn) CloseWrite() error { return closeWrite(c.Conn) }
