package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
)

// GET /v1/ca.mobileconfig (§7): the CA as an unsigned iOS configuration profile, for a paired
// phone's signed request. The profile is byte-for-byte deterministic for a CA and laptop name,
// so the QR code can carry its SHA-256 ("ca") and the phone can check what it downloaded: a
// profile swapped on the way (pairing and this download may be plain HTTP) doesn't match.

const caProfilePath = "/v1/ca.mobileconfig"

// handleProfile serves the profile. Exempt from the HTTPS-only rule (httpsOnly), so a phone can
// fetch a new CA's profile after a reset.
func (s *server) handleProfile(w http.ResponseWriter, r *http.Request, a authResult) {
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
	prof, ok := s.caProfile()
	if !ok {
		replyErr(w, http.StatusNotFound, "encryption is off")
		return
	}
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("Content-Disposition", `attachment; filename="PhoneDrop.mobileconfig"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(prof)
	log.Printf("tls: CA profile sent to phone %s", a.id)
}

// caProfile: the current profile, if TLS is on.
func (s *server) caProfile() ([]byte, bool) {
	der, host, ok := s.tlsm.profileCA()
	if !ok {
		return nil, false
	}
	return caProfile(der, host, s.laptop), true
}

// caProfileHash: hex SHA-256 of the exact profile bytes (the QR code's "ca"); "" if TLS is off.
func (s *server) caProfileHash() string {
	prof, ok := s.caProfile()
	if !ok {
		return ""
	}
	h := sha256.Sum256(prof)
	return hex.EncodeToString(h[:])
}

// uuidFrom formats 16 bytes as a version 4 style UUID, upper case as Apple writes them.
func uuidFrom(b []byte) string {
	var u [16]byte
	copy(u[:], b)
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return fmt.Sprintf("%X-%X-%X-%X-%X", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// caProfile builds the plist: one com.apple.security.root payload holding the CA's DER. Nothing
// in it is random: both UUIDs come from the CA's hash.
func caProfile(caDER []byte, host, laptop string) []byte {
	sum := sha256.Sum256(caDER)
	var b bytes.Buffer
	str := func(key, val string) {
		b.WriteString("\t<key>" + key + "</key>\n\t<string>")
		xml.EscapeText(&b, []byte(val))
		b.WriteString("</string>\n")
	}
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
	<dict>
`)
	str("PayloadCertificateFileName", "PhoneDrop CA.cer")
	b.WriteString("\t<key>PayloadContent</key>\n\t<data>" + base64.StdEncoding.EncodeToString(caDER) + "</data>\n")
	str("PayloadDescription", "The certificate authority PhoneDrop on "+laptop+" uses to identify itself.")
	str("PayloadDisplayName", "PhoneDrop CA ("+host+")")
	str("PayloadIdentifier", "app.phonedrop.ca."+host+".root")
	str("PayloadType", "com.apple.security.root")
	str("PayloadUUID", uuidFrom(sum[16:]))
	b.WriteString("\t<key>PayloadVersion</key>\n\t<integer>1</integer>\n\t</dict>\n\t</array>\n")
	str("PayloadDescription", "Lets this iPhone verify that it is really talking to PhoneDrop on "+laptop+
		" when sending over encrypted connections. The certificate can only vouch for "+host+
		" and local network addresses. After installing, turn on full trust for it in Settings > General > About > Certificate Trust Settings.")
	str("PayloadDisplayName", "PhoneDrop ("+laptop+")")
	str("PayloadIdentifier", "app.phonedrop.ca."+host)
	str("PayloadOrganization", "PhoneDrop")
	str("PayloadType", "Configuration")
	str("PayloadUUID", uuidFrom(sum[:16]))
	b.WriteString("\t<key>PayloadVersion</key>\n\t<integer>1</integer>\n</dict>\n</plist>\n")
	return b.Bytes()
}
