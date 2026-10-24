package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
)

// GET /v1/ca.mobileconfig (§7): the CA as an unsigned iOS configuration profile. It is the only
// unauthenticated endpoint, and it answers only while TLS is on and the profile window is open;
// otherwise it is the same 401 as everything else.

const caProfilePath = "/v1/ca.mobileconfig"

func (s *server) handleProfile(w http.ResponseWriter, r *http.Request) {
	der, host, ok := s.tlsm.profileCA(s.now())
	if !ok || r.Method != http.MethodGet {
		s.unauthorized(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("Content-Disposition", `attachment; filename="PhoneDrop.mobileconfig"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "close")
	w.Write(caProfile(der, host, s.laptop))
	log.Printf("tls: CA profile served")
}

// uuid4: a random RFC 4122 version 4 UUID, upper case as Apple writes them.
func uuid4() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// caProfile builds the plist: one com.apple.security.root payload holding the CA's DER.
func caProfile(caDER []byte, host, laptop string) []byte {
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
	str("PayloadUUID", uuid4())
	b.WriteString("\t<key>PayloadVersion</key>\n\t<integer>1</integer>\n\t</dict>\n\t</array>\n")
	str("PayloadDescription", "Lets this iPhone verify that it is really talking to PhoneDrop on "+laptop+
		" when sending over encrypted connections. The certificate can only vouch for "+host+
		" and local network addresses. After installing, turn on full trust for it in Settings > General > About > Certificate Trust Settings.")
	str("PayloadDisplayName", "PhoneDrop ("+laptop+")")
	str("PayloadIdentifier", "app.phonedrop.ca."+host)
	str("PayloadOrganization", "PhoneDrop")
	str("PayloadType", "Configuration")
	str("PayloadUUID", uuid4())
	b.WriteString("\t<key>PayloadVersion</key>\n\t<integer>1</integer>\n</dict>\n</plist>\n")
	return b.Bytes()
}
