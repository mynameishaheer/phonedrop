package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxNameUnits = 150     // UTF-16 code units, so full paths stay well under MAX_PATH-ish limits
	maxExtUnits  = 32      // a longer "extension" is garbage; treat it as part of the stem
	maxTextBody  = 1 << 20 // largest body that may be classified as text/url
)

// reservedNames are the Windows device names that cannot be file names (with or without an
// extension). The superscript digits are reserved too (COM¹, LPT²...).
var reservedNames = func() map[string]bool {
	m := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true, "CONIN$": true, "CONOUT$": true}
	for _, d := range "123456789¹²³" {
		m["COM"+string(d)] = true
		m["LPT"+string(d)] = true
	}
	return m
}()

// sanitizeName turns a phone-supplied (already percent-decoded) name into a single safe Windows
// file name. It returns "" when nothing usable remains, so the caller can substitute a default.
func sanitizeName(name string) string {
	name = strings.ToValidUTF8(name, "")
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	space := false
	for _, r := range name {
		switch {
		case unicode.IsSpace(r):
			space = true
			continue
		case r < 0x20 || r == 0x7F || unicode.IsControl(r), strings.ContainsRune(`<>:"/\|?*`, r),
			r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069: // bidi overrides spoof extensions
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	name = strings.Trim(b.String(), " ")
	// No hidden or dot-only names; ".jpg" alone has no stem, so it's rejected too.
	stripped := strings.TrimLeft(name, ".")
	if stripped != name && !strings.Contains(stripped, ".") {
		return ""
	}
	name = strings.TrimRight(stripped, ". ")
	if name == "" {
		return ""
	}
	// Windows matches the device name before the first dot, ignoring trailing spaces.
	stem, _, _ := strings.Cut(name, ".")
	if reservedNames[strings.ToUpper(strings.TrimRight(stem, " "))] {
		name = "_" + name
	}
	return limitName(name)
}

func utf16Len(s string) (n int) {
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

// limitName keeps name within maxNameUnits, cutting the stem (on a rune boundary) and keeping
// the extension.
func limitName(name string) string {
	if utf16Len(name) <= maxNameUnits {
		return name
	}
	ext := filepath.Ext(name)
	if utf16Len(ext) > maxExtUnits {
		ext = ""
	}
	stem := strings.TrimSuffix(name, ext)
	budget := maxNameUnits - utf16Len(ext)
	n, cut := 0, 0
	for i, r := range stem {
		w := 1
		if r > 0xFFFF {
			w = 2
		}
		if n+w > budget {
			break
		}
		n += w
		cut = i + utf8.RuneLen(r)
	}
	return strings.TrimRight(stem[:cut], ". ") + ext
}

// uniquePath exclusively creates dir/name, or "name (2).ext", "name (3).ext"... and returns the
// open file. O_EXCL makes it race-free; callers must pass a sanitized name.
func uniquePath(dir, name string) (string, *os.File, error) {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return "", nil, errors.New("invalid file name")
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i <= 9999; i++ {
		n := name
		if i > 1 {
			n = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		p := filepath.Join(dir, n)
		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			return p, f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, errors.New("too many files with the same name")
}

// magic is a fixed-offset signature; ISO BMFF and MPEG audio need logic, see sniffExt.
var magic = []struct {
	off  int
	sig  string
	ext  string
	also string // second signature at offset 8 (RIFF subtype), if any
}{
	{0, "\xFF\xD8\xFF", ".jpg", ""},
	{0, "\x89PNG\r\n\x1a\n", ".png", ""},
	{0, "GIF87a", ".gif", ""},
	{0, "GIF89a", ".gif", ""},
	{0, "RIFF", ".webp", "WEBP"},
	{0, "RIFF", ".wav", "WAVE"},
	{0, "%PDF-", ".pdf", ""},
	{0, "PK\x03\x04", ".zip", ""},
	{0, "\x1f\x8b", ".gz", ""},
	{0, "ID3", ".mp3", ""},
}

// ftypBrands maps the ISO BMFF major brand to an extension.
var ftypBrands = map[string]string{
	"heic": ".heic", "heix": ".heic", "hevc": ".heic", "heim": ".heic", "heis": ".heic",
	"mif1": ".heic", "msf1": ".heic",
	"avif": ".avif", "avis": ".avif",
	"qt  ": ".mov",
	"isom": ".mp4", "iso2": ".mp4", "mp41": ".mp4", "mp42": ".mp4", "avc1": ".mp4", "M4V ": ".mp4",
	"M4A ": ".m4a",
}

// sniffExt guesses a lowercase extension (with dot) from the first bytes of a file, or "".
func sniffExt(head []byte) string {
	for _, m := range magic {
		if len(head) >= m.off+len(m.sig) && string(head[m.off:m.off+len(m.sig)]) == m.sig {
			if m.also == "" || (len(head) >= 12 && string(head[8:12]) == m.also) {
				return m.ext
			}
		}
	}
	if len(head) >= 12 && bytes.Equal(head[4:8], []byte("ftyp")) {
		return ftypBrands[string(head[8:12])]
	}
	// MPEG audio frame sync: 11 set bits, valid version/layer/bitrate/sample rate (this also
	// rules out AAC ADTS, whose layer bits are 00).
	if len(head) >= 3 && head[0] == 0xFF && head[1]&0xE0 == 0xE0 &&
		head[1]>>3&3 != 1 && head[1]>>1&3 != 0 && head[2]>>4 != 0xF && head[2]>>2&3 != 3 {
		return ".mp3"
	}
	return ""
}

// isLink reports whether s is exactly one http(s) URL with a host and no whitespace.
func isLink(s string) bool {
	s = strings.TrimSpace(s)
	if strings.IndexFunc(s, unicode.IsSpace) >= 0 {
		return false
	}
	l := strings.ToLower(s)
	if !strings.HasPrefix(l, "http://") && !strings.HasPrefix(l, "https://") {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

// inferKind implements PROTOCOL.md §5: an explicit X-PD-Kind wins; a name with an extension is a
// file; a small valid-UTF-8 body (no NUL bytes) is a url or text; anything else is a file. head is the whole body
// when complete is true.
func inferKind(name, kindHeader string, head []byte, size int64, complete bool) string {
	switch k := strings.ToLower(strings.TrimSpace(kindHeader)); k {
	case "file", "text", "url":
		return k
	}
	if ext := filepath.Ext(sanitizeName(name)); len(ext) > 1 && len(ext) <= 10 && isAlnum(ext[1:]) {
		return "file"
	}
	if complete && size <= maxTextBody && utf8.Valid(head) && bytes.IndexByte(head, 0) < 0 { // NUL: binary that happens to be valid UTF-8
		if isLink(string(head)) {
			return "url"
		}
		return "text"
	}
	return "file"
}

func isAlnum(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return s != ""
}

// defaultName names an unnamed file "Received 2026-10-24 22.15.24.jpg"; pass time.Now() for
// local time. A missing extension becomes ".bin".
func defaultName(now time.Time, ext string) string {
	if ext == "" {
		ext = ".bin"
	} else if ext[0] != '.' {
		ext = "." + ext
	}
	return now.Format("Received 2006-01-02 15.04.05") + ext
}

// mediaKind classifies by extension: "image", "video" or "file". It picks icons and history
// kinds; what the phone is told (photosKind) is narrower.
func mediaKind(name string) string {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")) {
	case "jpg", "jpeg", "png", "gif", "webp", "heic", "heif", "avif", "bmp", "tif", "tiff":
		return "image"
	case "mov", "mp4", "m4v", "avi", "mkv", "webm", "3gp":
		return "video"
	}
	return "file"
}

// photosKind is the kind the outbox reports to the phone: "image" or "video" only for formats
// iOS Photos imports, since the Receive shortcut saves those to the photo library; anything else
// (an mkv, webm or avif) is a "file" and goes to Files instead.
func photosKind(name string) string {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")) {
	case "jpg", "jpeg", "png", "heic", "heif", "gif", "webp", "tif", "tiff", "bmp":
		return "image"
	case "mov", "mp4", "m4v", "3gp":
		return "video"
	}
	return "file"
}
