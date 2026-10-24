package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testHost = "testpc.local"

// prodCAIPRanges: the production list, captured before init adds loopback for the tests (package
// variables are initialised before init runs).
var prodCAIPRanges = slices.Clone(caIPRanges)

func init() { caIPRanges = append(slices.Clone(caIPRanges), "127.0.0.0/8") }

func TestProductionCARanges(t *testing.T) {
	want := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fc00::/7", "fe80::/10"}
	if !slices.Equal(prodCAIPRanges, want) {
		t.Fatalf("production ranges %v", prodCAIPRanges)
	}
	ca, key, err := newCA(testHost, time.Now(), prodCAIPRanges)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range ca.PermittedIPRanges {
		got = append(got, n.String())
	}
	if !slices.Equal(got, want) || !ca.PermittedDNSDomainsCritical {
		t.Fatalf("CA permits %v", got)
	}
	// The leaf only gets the addresses the CA may certify, so loopback is left out (and the
	// leaf still verifies for the rest), and a leaf that holds it anyway is refused.
	m := testTLSManager(t, t.TempDir())
	m.ca, m.caKey = ca, key
	m.setAddrs([]netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("192.168.18.20"),
		netip.MustParseAddr("100.64.0.1"), netip.MustParseAddr("fe80::1")})
	if err := m.issueLocked(); err != nil {
		t.Fatal(err)
	}
	leaf := m.leaf.Load().Leaf
	var ips []string
	for _, ip := range leaf.IPAddresses {
		ips = append(ips, ip.String())
	}
	if !slices.Equal(ips, []string{"192.168.18.20", "fe80::1"}) {
		t.Fatalf("leaf IPs %v", ips)
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	for _, name := range []string{testHost, "192.168.18.20"} {
		if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: name}); err != nil {
			t.Errorf("verify %s: %v", name, err)
		}
	}
	tmpl := &x509.Certificate{SerialNumber: randSerial(), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	lk, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &lk.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	rogue, _ := x509.ParseCertificate(der)
	if _, err := rogue.Verify(x509.VerifyOptions{Roots: roots, DNSName: "127.0.0.1"}); !verifyErrIsConstraint(err) {
		t.Fatalf("production CA certified loopback: %v", err)
	}
}

// testTLSManager: a manager on a temp dir with fake DPAPI, not yet enabled.
func testTLSManager(t *testing.T, dir string) *tlsManager {
	m := newTLSManager()
	m.dir, m.protect, m.unprotect, m.host = dir, xorProtect, xorProtect, testHost
	return m
}

func (m *tlsManager) caCert() *x509.Certificate {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ca
}

// newTLSTestServer: a test server on 127.0.0.1 with TLS on and one paired phone.
func newTLSTestServer(t *testing.T, hooks appHooks) (*testServer, *phoneClient) {
	ts := newTestServer(t, hooks)
	ts.tlsm.dir, ts.tlsm.protect, ts.tlsm.unprotect, ts.tlsm.host = t.TempDir(), xorProtect, xorProtect, testHost
	if err := ts.setTLS(true); err != nil {
		t.Fatal(err)
	}
	c := &phoneClient{ts: ts, id: "0123456789abcdef", secret: secretA}
	if err := ts.phones.add(c.id, c.secret, "P"); err != nil {
		t.Fatal(err)
	}
	return ts, c
}

func httpsClient(ca *x509.Certificate, serverName string) *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: serverName}}}
}

func httpsBase(ts *testServer, ip string) string { return "https://" + ip + ":" + itoa(ts.port) }

// via sends a signed request with a given client and base URL.
func (c *phoneClient) via(t *testing.T, hc *http.Client, base, method, path string, body []byte) resp {
	t.Helper()
	c.n++
	tsv := c.ts.time().Format(time.RFC3339)
	r := signedReq(method, path, c.id, c.secret, tsv, fmt.Sprintf("nonce-%d-%d", c.n, time.Now().UnixNano()), body)
	req, _ := http.NewRequest(r.Method, base+r.URL.RequestURI(), r.Body)
	req.Header = r.Header
	req.ContentLength = r.ContentLength
	res, err := hc.Do(req)
	if err != nil {
		t.Fatalf("%s %s%s: %v", method, base, path, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{res.StatusCode, strings.TrimSpace(string(b))}
}

func verifyErrIsConstraint(err error) bool {
	var ce x509.CertificateInvalidError
	return errors.As(err, &ce) && ce.Reason == x509.CANotAuthorizedForThisName
}

func TestCAAndLeaf(t *testing.T) {
	m := testTLSManager(t, t.TempDir())
	m.setAddrs([]netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("192.168.18.20")})
	if err := m.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	ca, leaf := m.caCert(), m.leaf.Load().Leaf
	if !ca.IsCA || !ca.MaxPathLenZero || ca.MaxPathLen != 0 || ca.KeyUsage != x509.KeyUsageCertSign|x509.KeyUsageCRLSign ||
		!slices.Equal(ca.PermittedDNSDomains, []string{testHost}) || !ca.PermittedDNSDomainsCritical ||
		ca.Subject.CommonName != "PhoneDrop CA (testpc.local)" || ca.SerialNumber.BitLen() > 129 ||
		ca.NotAfter.Sub(ca.NotBefore) < 9*365*24*time.Hour || ca.PublicKeyAlgorithm != x509.ECDSA ||
		!slices.Equal(ca.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}) ||
		!slices.Equal(ca.PermittedEmailAddresses, []string{"invalid"}) || !slices.Equal(ca.PermittedURIDomains, []string{"invalid"}) ||
		len(ca.ExcludedDNSDomains)+len(ca.ExcludedEmailAddresses)+len(ca.ExcludedURIDomains)+len(ca.ExcludedIPRanges) != 0 {
		t.Fatalf("CA %+v", ca)
	}
	critical := false
	for _, e := range ca.Extensions {
		if e.Id.String() == "2.5.29.30" {
			critical = e.Critical
		}
	}
	if !critical {
		t.Fatal("name constraints not critical")
	}
	if d := leaf.NotAfter.Sub(leaf.NotBefore); d != leafValidity+time.Hour || leaf.IsCA ||
		!slices.Equal(leaf.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}) ||
		leaf.KeyUsage != x509.KeyUsageDigitalSignature || !slices.Equal(leaf.DNSNames, []string{testHost}) {
		t.Fatalf("leaf %+v", leaf)
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	for _, name := range []string{testHost, "127.0.0.1", "192.168.18.20"} {
		if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: name}); err != nil {
			t.Errorf("verify %s: %v", name, err)
		}
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "10.0.0.1"}); err == nil {
		t.Error("leaf verified for an address it doesn't hold")
	}
	if fp := (&server{tlsm: m}).caFingerprint(); len(fp) != 64 || !isLowerHex(fp) {
		t.Errorf("fingerprint %q", fp)
	}
}

func TestCANameConstraints(t *testing.T) {
	m := testTLSManager(t, t.TempDir())
	if err := m.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	ca := m.caCert()
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	rogueWith := func(tmpl *x509.Certificate) *x509.Certificate {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		tmpl.SerialNumber, tmpl.Subject = randSerial(), pkix.Name{CommonName: "rogue"}
		tmpl.NotBefore, tmpl.NotAfter = time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
		if tmpl.ExtKeyUsage == nil {
			tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, m.caKey)
		if err != nil {
			t.Fatal(err)
		}
		c, _ := x509.ParseCertificate(der)
		return c
	}
	rogue := func(dns []string, ips []net.IP) *x509.Certificate {
		return rogueWith(&x509.Certificate{DNSNames: dns, IPAddresses: ips})
	}
	for _, x := range []struct {
		name string
		dns  []string
		ips  []net.IP
	}{
		{"example.com", []string{"example.com"}, nil},
		{"8.8.8.8", nil, []net.IP{net.ParseIP("8.8.8.8")}},
		{"2001:4860::8888", nil, []net.IP{net.ParseIP("2001:4860::8888")}},
		{testHost, []string{testHost, "example.com"}, nil}, // one bad name spoils the certificate
		{"testpc.local.example.com", []string{"testpc.local.example.com"}, nil},
		{"eviltestpc.local", []string{"eviltestpc.local"}, nil},
	} {
		_, err := rogue(x.dns, x.ips).Verify(x509.VerifyOptions{Roots: roots, DNSName: x.name})
		if !verifyErrIsConstraint(err) {
			t.Errorf("rogue leaf %v %v: err = %v, want a name-constraint error", x.dns, x.ips, err)
		}
	}
	// Email and URI names: the CA permits only the reserved .invalid, so S/MIME and URI
	// certificates fail even when the leaf also carries this PC's name.
	uri, _ := url.Parse("https://example.com/x")
	for what, tmpl := range map[string]*x509.Certificate{
		"email":             {DNSNames: []string{testHost}, EmailAddresses: []string{"victim@example.com"}},
		"email only":        {EmailAddresses: []string{"victim@example.com"}, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection}},
		"URI":               {DNSNames: []string{testHost}, URIs: []*url.URL{uri}},
		"email in .invalid": {EmailAddresses: []string{"a@x.invalid"}, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection}},
	} {
		c := rogueWith(tmpl)
		opts := x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}
		if len(tmpl.DNSNames) > 0 {
			opts.DNSName = testHost
		}
		_, err := c.Verify(opts)
		if what == "email in .invalid" {
			if err != nil {
				t.Errorf("%s: %v (the constraint isn't encoded as expected)", what, err)
			}
			continue
		}
		if !verifyErrIsConstraint(err) {
			t.Errorf("rogue %s leaf: err = %v, want a name-constraint error", what, err)
		}
	}
	// Server certificates only: the CA's EKU stops a client-auth or code-signing leaf.
	for _, eku := range []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageEmailProtection, x509.ExtKeyUsageCodeSigning} {
		c := rogueWith(&x509.Certificate{DNSNames: []string{testHost}, ExtKeyUsage: []x509.ExtKeyUsage{eku}})
		var ie x509.CertificateInvalidError
		if _, err := c.Verify(x509.VerifyOptions{Roots: roots, DNSName: testHost, KeyUsages: []x509.ExtKeyUsage{eku}}); !errors.As(err, &ie) || ie.Reason != x509.IncompatibleUsage {
			t.Errorf("EKU %v: err = %v, want incompatible usage", eku, err)
		}
	}
	// A subdomain of the host is inside the constraint (harmless: it is still this PC's name).
	if _, err := rogue([]string{"sub." + testHost}, nil).Verify(x509.VerifyOptions{Roots: roots, DNSName: "sub." + testHost}); err != nil {
		t.Errorf("subdomain: %v", err)
	}
}

func TestTLSAndPlainOnOnePort(t *testing.T) {
	ts, c := newTLSTestServer(t, newFakeHooks())
	ca := ts.tlsm.caCert()
	plain := &http.Client{Timeout: 5 * time.Second}
	if r := c.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 200 || !strings.Contains(r.body, `"tls":false`) {
		t.Fatalf("plain ping %+v", r)
	}
	byIP := httpsClient(ca, "")
	if r := c.via(t, byIP, httpsBase(ts, "127.0.0.1"), "GET", "/v1/ping", nil); r.code != 200 || !strings.Contains(r.body, `"tls":true`) {
		t.Fatalf("https ping %+v", r)
	}
	byName := httpsClient(ca, testHost)
	if r := c.via(t, byName, httpsBase(ts, "127.0.0.1"), "GET", "/v1/ping", []byte("x")); r.code != 200 || !strings.Contains(r.body, `"tls":true`) {
		t.Fatalf("https ping by name %+v", r)
	}
	res, err := byIP.Get(httpsBase(ts, "127.0.0.1") + "/v1/ping")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 401 {
		t.Fatalf("unsigned https: %d", res.StatusCode)
	}
	// A client that doesn't trust the CA fails the handshake.
	if _, err := (&http.Client{Timeout: 5 * time.Second}).Get(httpsBase(ts, "127.0.0.1") + "/v1/ping"); err == nil {
		t.Fatal("untrusted client connected")
	}
	// With TLS off, a TLS hello is just a bad HTTP request.
	ts.setTLS(false)
	byIP.CloseIdleConnections()
	if _, err := byIP.Get(httpsBase(ts, "127.0.0.1") + "/v1/ping"); err == nil {
		t.Fatal("https worked with TLS off")
	}
}

func TestSilentClientDoesNotBlockOthers(t *testing.T) {
	ts, c := newTLSTestServer(t, newFakeHooks())
	silent, err := net.Dial("tcp", "127.0.0.1:"+itoa(ts.port))
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	time.Sleep(50 * time.Millisecond) // let it be accepted and start sniffing
	quick := &http.Client{Timeout: 3 * time.Second}
	start := time.Now()
	if r := c.via(t, quick, ts.base, "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("plain ping %+v", r)
	}
	hc := httpsClient(ts.tlsm.caCert(), "")
	hc.Timeout = 3 * time.Second
	if r := c.via(t, hc, httpsBase(ts, "127.0.0.1"), "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("https ping %+v", r)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("requests took %v behind a silent client", d)
	}
}

func TestSilentClientIsDropped(t *testing.T) {
	ts := newTestServer(t, nil)
	ts.tlsm.dir, ts.tlsm.protect, ts.tlsm.unprotect = t.TempDir(), xorProtect, xorProtect
	ts.tlsm.sniffTimeout = 200 * time.Millisecond
	if err := ts.setTLS(true); err != nil {
		t.Fatal(err)
	}
	silent, err := net.Dial("tcp", "127.0.0.1:"+itoa(ts.port))
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	silent.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := silent.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("silent client read: %v, want EOF", err)
	}
}

func TestCAPersistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	m := testTLSManager(t, dir)
	if err := m.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	fp := m.fingerprint()
	keyDER, _ := x509.MarshalPKCS8PrivateKey(m.caKey)
	raw, err := os.ReadFile(filepath.Join(dir, "ca.key"))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		t.Fatalf("ca.key isn't base64: %v", err)
	}
	if bytes.Contains(raw, keyDER) || bytes.Contains(enc, keyDER) || bytes.Contains(raw, []byte(base64.StdEncoding.EncodeToString(keyDER))) ||
		bytes.Contains(raw, []byte("PRIVATE KEY")) {
		t.Fatal("CA key on disk in plain form")
	}
	if runtime.GOOS != "windows" {
		for _, f := range []string{"ca.key", "ca.crt"} {
			if fi, _ := os.Stat(filepath.Join(dir, f)); fi.Mode().Perm() != 0o600 {
				t.Fatalf("%s mode %v", f, fi.Mode().Perm())
			}
		}
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(left) > 0 {
		t.Fatalf("temp files left: %v", left)
	}

	// A restart loads the same CA.
	m2 := testTLSManager(t, dir)
	if err := m2.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	if m2.fingerprint() != fp {
		t.Fatal("reload made a new CA")
	}
	// Off drops the key from memory; on again reloads the same CA.
	m2.setEnabled(false)
	if m2.fingerprint() != "" || m2.caKey != nil || m2.leaf.Load() != nil {
		t.Fatal("CA kept in memory while off")
	}
	if m2.setEnabled(true); m2.fingerprint() != fp {
		t.Fatal("re-enabling made a new CA")
	}
	// Without the right unprotect the key can't be read: a new CA, not a crash.
	m3 := testTLSManager(t, dir)
	m3.unprotect = func(b []byte) ([]byte, error) { return nil, errors.New("dpapi says no") }
	if err := m3.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	if m3.fingerprint() == fp {
		t.Fatal("unreadable key was used")
	}
	// Renamed computer: a new CA, constrained to the new name.
	m4 := testTLSManager(t, dir)
	m4.host = "renamed.local"
	if err := m4.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	if fp4 := m4.fingerprint(); fp4 == fp || fp4 == m3.fingerprint() || caHost(m4.caCert()) != "renamed.local" ||
		!slices.Equal(m4.leaf.Load().Leaf.DNSNames, []string{"renamed.local"}) {
		t.Fatal("host change didn't make a new CA")
	}
	// Reset makes a new one at once while enabled.
	s := &server{tlsm: m4}
	before := s.caFingerprint()
	if err := s.resetTLS(); err != nil {
		t.Fatal(err)
	}
	if after := s.caFingerprint(); after == "" || after == before || !s.tlsEnabled() {
		t.Fatalf("reset: %q → %q", before, after)
	}
	m5 := testTLSManager(t, dir)
	m5.host = "renamed.local"
	m5.setEnabled(true)
	if m5.fingerprint() != s.caFingerprint() {
		t.Fatal("reset CA wasn't saved")
	}
}

// A CA from before the EKU and email/URI constraints is replaced on load.
func TestOldCAIsReplaced(t *testing.T) {
	dir := t.TempDir()
	m := testTLSManager(t, dir)
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	_, lan, _ := net.ParseCIDR("192.168.0.0/16")
	tmpl := &x509.Certificate{SerialNumber: randSerial(), Subject: pkix.Name{CommonName: "old"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(caValidity), IsCA: true,
		BasicConstraintsValid: true, MaxPathLenZero: true, KeyUsage: x509.KeyUsageCertSign,
		PermittedDNSDomainsCritical: true, PermittedDNSDomains: []string{testHost}, PermittedIPRanges: []*net.IPNet{lan}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	old, _ := x509.ParseCertificate(der)
	if err := m.saveCA(old, key); err != nil {
		t.Fatal(err)
	}
	if caConstrained(old) {
		t.Fatal("old CA counted as constrained")
	}
	if err := m.setEnabled(true); err != nil {
		t.Fatal(err)
	}
	if ca := m.caCert(); bytes.Equal(ca.Raw, old.Raw) || !caConstrained(ca) {
		t.Fatal("old CA kept")
	}
	m2 := testTLSManager(t, dir) // and the replacement was saved
	if m2.setEnabled(true); m2.fingerprint() != m.fingerprint() {
		t.Fatal("replacement CA not saved")
	}
}

func TestRebindReissuesLeaf(t *testing.T) {
	ts, c := newTLSTestServer(t, newFakeHooks())
	lo1, lo2 := netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("127.0.0.2")
	ips := func() []string {
		var out []string
		for _, ip := range ts.tlsm.leaf.Load().Leaf.IPAddresses {
			out = append(out, ip.String())
		}
		return out
	}
	if got := ips(); !slices.Equal(got, []string{"127.0.0.1"}) {
		t.Fatalf("leaf IPs %v", got)
	}
	first := ts.tlsm.leaf.Load()
	ts.bind([]netip.Addr{lo1, lo2})
	if got := ips(); !slices.Equal(got, []string{"127.0.0.1", "127.0.0.2"}) {
		t.Fatalf("leaf IPs after adding: %v", got)
	}
	ts.bind([]netip.Addr{lo2})
	if got := ips(); !slices.Equal(got, []string{"127.0.0.2"}) {
		t.Fatalf("leaf IPs after removing: %v", got)
	}
	if ts.tlsm.leaf.Load() == first {
		t.Fatal("leaf not re-issued")
	}
	same := ts.tlsm.leaf.Load()
	ts.bind([]netip.Addr{lo2})
	if ts.tlsm.leaf.Load() != same {
		t.Fatal("leaf re-issued for an unchanged address set")
	}
	// No listener restart: the running listener serves the new certificate.
	if r := c.via(t, httpsClient(ts.tlsm.caCert(), ""), httpsBase(ts, "127.0.0.2"), "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("ping on 127.0.0.2: %+v", r)
	}
}

// plist parses the subset of XML plists caProfile writes.
func plist(t *testing.T, b []byte) map[string]any {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(b))
	var value func(se xml.StartElement) any
	value = func(se xml.StartElement) any {
		switch se.Name.Local {
		case "dict", "array":
			m, a, key := map[string]any{}, []any{}, ""
			for {
				tok, err := d.Token()
				if err != nil {
					t.Fatal(err)
				}
				switch tk := tok.(type) {
				case xml.StartElement:
					if tk.Name.Local == "key" {
						d.DecodeElement(&key, &tk)
						continue
					}
					v := value(tk)
					if se.Name.Local == "dict" {
						if _, dup := m[key]; dup {
							t.Fatalf("duplicate key %s", key)
						}
						m[key] = v
					} else {
						a = append(a, v)
					}
				case xml.EndElement:
					if se.Name.Local == "dict" {
						return m
					}
					return a
				}
			}
		case "true", "false":
			d.Skip()
			return se.Name.Local == "true"
		}
		var s string
		if err := d.DecodeElement(&s, &se); err != nil {
			t.Fatal(err)
		}
		switch se.Name.Local {
		case "data":
			b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
			if err != nil {
				t.Fatal(err)
			}
			return b
		case "integer":
			n, _ := strconv.Atoi(s)
			return n
		}
		return s
	}
	for {
		tok, err := d.Token()
		if err != nil {
			t.Fatal(err)
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local != "plist" {
			return value(se).(map[string]any)
		}
	}
}

// fetchProfile downloads the CA profile with a signed request, keeping the exact bytes.
func fetchProfile(t *testing.T, c *phoneClient, hc *http.Client, base, method string) (int, http.Header, []byte) {
	t.Helper()
	c.n++
	tsv := c.ts.time().Format(time.RFC3339)
	r := signedReq(method, caProfilePath, c.id, c.secret, tsv, fmt.Sprintf("prof-%d-%d", c.n, time.Now().UnixNano()), nil)
	req, _ := http.NewRequest(method, base+caProfilePath, nil)
	req.Header = r.Header
	res, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header, b
}

func TestProfile(t *testing.T) {
	ts := newTestServer(t, nil)
	ts.tlsm.dir, ts.tlsm.protect, ts.tlsm.unprotect, ts.tlsm.host = t.TempDir(), xorProtect, xorProtect, testHost
	ts.laptop = `Shaheer's <PC> & "Co"`
	c := &phoneClient{ts: ts, id: "0123456789abcdef", secret: secretA}
	ts.phones.add(c.id, c.secret, "P")
	plain := &http.Client{Timeout: 5 * time.Second}

	if code, _, b := fetchProfile(t, c, plain, ts.base, "GET"); code != 404 || strings.TrimSpace(string(b)) != `{"ok":false,"error":"encryption is off"}` {
		t.Fatalf("TLS off: %d %s", code, b)
	}
	if err := ts.setTLS(true); err != nil {
		t.Fatal(err)
	}
	// No unauthenticated path: unsigned, or signed by an unknown phone, is the usual 401.
	res, err := plain.Get(ts.base + caProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	stranger := &phoneClient{ts: ts, id: "fedcba9876543210", secret: secretB}
	if code, _, _ := fetchProfile(t, stranger, plain, ts.base, "GET"); res.StatusCode != 401 || code != 401 {
		t.Fatalf("unauthenticated: %d, unknown phone: %d", res.StatusCode, code)
	}
	if code, _, _ := fetchProfile(t, c, plain, ts.base, "POST"); code != 405 {
		t.Fatalf("POST: %d", code)
	}

	code, hdr, body := fetchProfile(t, c, plain, ts.base, "GET")
	if code != 200 || hdr.Get("Content-Type") != "application/x-apple-aspen-config" ||
		hdr.Get("Content-Disposition") != `attachment; filename="PhoneDrop.mobileconfig"` {
		t.Fatalf("profile: %d %v", code, hdr)
	}
	p := plist(t, body)
	uuid := regexp.MustCompile(`^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`)
	content, _ := p["PayloadContent"].([]any)
	if p["PayloadType"] != "Configuration" || p["PayloadVersion"] != 1 || p["PayloadIdentifier"] != "app.phonedrop.ca."+testHost ||
		p["PayloadDisplayName"] != `PhoneDrop (Shaheer's <PC> & "Co")` || !uuid.MatchString(p["PayloadUUID"].(string)) ||
		!strings.Contains(p["PayloadDescription"].(string), "verify") || len(content) != 1 {
		t.Fatalf("profile %v", p)
	}
	inner := content[0].(map[string]any)
	if inner["PayloadType"] != "com.apple.security.root" || inner["PayloadVersion"] != 1 ||
		!uuid.MatchString(inner["PayloadUUID"].(string)) || inner["PayloadUUID"] == p["PayloadUUID"] ||
		inner["PayloadIdentifier"] == p["PayloadIdentifier"] || inner["PayloadDisplayName"] == "" {
		t.Fatalf("payload %v", inner)
	}
	if der, _ := inner["PayloadContent"].([]byte); !bytes.Equal(der, ts.tlsm.caCert().Raw) {
		t.Fatal("profile doesn't hold the CA")
	}
	// Deterministic: the same CA gives the same bytes, and the hash the QR code carries.
	_, _, again := fetchProfile(t, c, plain, ts.base, "GET")
	if h := sha256.Sum256(body); !bytes.Equal(again, body) || hex.EncodeToString(h[:]) != ts.caProfileHash() {
		t.Fatal("profile isn't deterministic")
	}
	// A new CA gives a different profile, with different UUIDs.
	ts.resetTLS()
	_, _, fresh := fetchProfile(t, c, plain, ts.base, "GET")
	if p2 := plist(t, fresh); bytes.Equal(fresh, body) || p2["PayloadUUID"] == p["PayloadUUID"] {
		t.Fatal("reset CA gave the same profile")
	}
	// Exempt from HTTPS-only: a phone that has used HTTPS can still fetch it over plain HTTP.
	ts.phones.setTLS(c.id)
	if r := c.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 403 {
		t.Fatalf("plain ping from an HTTPS phone: %+v", r)
	}
	if code, _, _ := fetchProfile(t, c, plain, ts.base, "GET"); code != 200 {
		t.Fatalf("plain profile from an HTTPS phone: %d", code)
	}
	if code, _, _ := fetchProfile(t, c, httpsClient(ts.tlsm.caCert(), ""), httpsBase(ts, "127.0.0.1"), "GET"); code != 200 {
		t.Fatalf("profile over HTTPS: %d", code)
	}
	ts.setTLS(false)
	if code, _, _ := fetchProfile(t, c, plain, ts.base, "GET"); code != 404 {
		t.Fatalf("after turning TLS off: %d", code)
	}
}

func TestQRCarriesProfileHash(t *testing.T) {
	ts, c := newTLSTestServer(t, newFakeHooks())
	qr, _ := ts.startPairing(pairingInfo{})
	var keys []string
	dec := json.NewDecoder(strings.NewReader(qr))
	dec.Token()
	for dec.More() {
		k, _ := dec.Token()
		keys = append(keys, k.(string))
		var v json.RawMessage
		dec.Decode(&v)
	}
	if got := strings.Join(keys, ","); got != "pd,name,host,ips,port,base,id,secret,tls,ca" {
		t.Fatalf("key order %s", got)
	}
	var q qrPayload
	json.Unmarshal([]byte(qr), &q)
	_, _, prof := fetchProfile(t, c, &http.Client{Timeout: 5 * time.Second}, ts.base, "GET")
	if h := sha256.Sum256(prof); !q.TLS || q.CA != hex.EncodeToString(h[:]) {
		t.Fatalf("QR tls=%v ca=%s doesn't match the profile", q.TLS, q.CA)
	}
	ts.setTLS(false)
	if qr, _ := ts.startPairing(pairingInfo{}); strings.Contains(qr, `"tls"`) || strings.Contains(qr, `"ca"`) {
		t.Fatalf("TLS off but QR has tls fields: %s", qr)
	}
}

func TestHTTPSOnlyPerPhone(t *testing.T) {
	ts, c := newTLSTestServer(t, newFakeHooks())
	other := &phoneClient{ts: ts, id: "fedcba9876543210", secret: secretB}
	ts.phones.add(other.id, other.secret, "Other")
	plain := &http.Client{Timeout: 5 * time.Second}
	hc := httpsClient(ts.tlsm.caCert(), "")
	if r := c.via(t, hc, httpsBase(ts, "127.0.0.1"), "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("https ping %+v", r)
	}
	if p, _ := ts.phones.get(c.id); !p.TLS {
		t.Fatal("phone not marked after HTTPS")
	}
	if r := c.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 403 || r.body != `{"ok":false,"error":"use https"}` {
		t.Fatalf("plain after https: %+v", r)
	}
	if r := c.via(t, plain, ts.base, "POST", "/v1/items", []byte("hi")); r.code != 403 {
		t.Fatalf("plain item after https: %+v", r)
	}
	if r := other.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("a phone that never used https: %+v", r)
	}
	if err := ts.setTLS(false); err != nil {
		t.Fatal(err)
	}
	if r := c.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("plain with TLS off: %+v", r)
	}
	if err := ts.setTLS(true); err != nil {
		t.Fatal(err)
	}
	if r := c.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 403 {
		t.Fatalf("plain with TLS back on: %+v", r)
	}
}

func TestTLSListenerNoGoroutineLeak(t *testing.T) {
	settle := func(max int) int {
		n := runtime.NumGoroutine()
		for i := 0; i < 100 && n > max; i++ {
			time.Sleep(20 * time.Millisecond)
			n = runtime.NumGoroutine()
		}
		return n
	}
	before := settle(0)
	ts, c := newTLSTestServer(t, newFakeHooks())
	plain := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{}}
	hc := httpsClient(ts.tlsm.caCert(), "")
	if r := c.via(t, plain, ts.base, "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("plain %+v", r)
	}
	if r := c.via(t, hc, httpsBase(ts, "127.0.0.1"), "GET", "/v1/ping", nil); r.code != 200 {
		t.Fatalf("https %+v", r)
	}
	silent, err := net.Dial("tcp", "127.0.0.1:"+itoa(ts.port))
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	sl := func() *sniffListener {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		return ts.listeners[netip.MustParseAddr("127.0.0.1")].(*sniffListener)
	}()
	for i := 0; i < 100; i++ {
		sl.mu.Lock()
		n := len(sl.sniffing)
		sl.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ts.bind(nil)
	silent.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := silent.Read(make([]byte, 1)); err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("silent connection not closed by unbinding: %v", err)
	}
	plain.CloseIdleConnections()
	hc.CloseIdleConnections()
	if after := settle(before); after > before {
		buf := make([]byte, 1<<20)
		t.Fatalf("goroutines: %d before, %d after\n%s", before, after, buf[:runtime.Stack(buf, true)])
	}
}
