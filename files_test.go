package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSanitizeName(t *testing.T) {
	long := strings.Repeat("a", 300) + ".heic"
	longUni := strings.Repeat("😀", 200) + ".jpg"
	tests := []struct{ in, want string }{
		{"IMG_2041.HEIC", "IMG_2041.HEIC"},
		{"Shaheer’s photo ✓.heic", "Shaheer’s photo ✓.heic"},
		{"../../evil.exe", "evil.exe"},
		{`C:\Windows\x.dll`, "x.dll"},
		{`..\..\a\b.txt`, "b.txt"},
		{"/etc/passwd", "passwd"},
		{"..", ""},
		{".", ""},
		{"", ""},
		{"dir/", ""},
		{".jpg", ""},
		{"...", ""},
		{".bashrc", ""},
		{"..hidden.txt", "hidden.txt"},
		{"CON", "_CON"},
		{"con.txt", "_con.txt"},
		{"NUL .txt", "_NUL .txt"},
		{"Lpt1.jpg", "_Lpt1.jpg"},
		{"COM¹", "_COM¹"},
		{"com³.tar.gz", "_com³.tar.gz"},
		{"COM0.txt", "COM0.txt"},
		{"console.txt", "console.txt"},
		{"photo.jpg...", "photo.jpg"},
		{"photo. . ", "photo"},
		{"  spaced   out \t name .txt", "spaced out name .txt"},
		{"a\x00b\x1fc\x7fd.txt", "abcd.txt"},
		{`a<b>c:d"e|f?g*h.txt`, "abcdefgh.txt"},
		{"a\u202Egpj.exe", "agpj.exe"},
		{"bad\xffutf.txt", "badutf.txt"},
		{"a:stream.txt", "astream.txt"},
		{"???", ""},
	}
	for _, tc := range tests {
		if got := sanitizeName(tc.in); got != tc.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	got := sanitizeName(long)
	if utf16Len(got) != maxNameUnits || !strings.HasSuffix(got, ".heic") {
		t.Errorf("long ascii: len %d %q", utf16Len(got), got[len(got)-10:])
	}
	got = sanitizeName(longUni)
	if utf16Len(got) > maxNameUnits || !strings.HasSuffix(got, ".jpg") || !utf8.ValidString(got) {
		t.Errorf("long unicode: units %d", utf16Len(got))
	}
	if n := utf16Len(got); n != maxNameUnits {
		t.Errorf("long unicode should fill the budget exactly (surrogate pairs of 2): %d", n)
	}
	// Truncation must not leave a trailing space or dot before the extension.
	got = sanitizeName(strings.Repeat("a", 144) + " . " + "b" + ".txt")
	if strings.Contains(got, " .txt") || utf16Len(got) > maxNameUnits {
		t.Errorf("truncation left %q", got)
	}
	// Result is always a safe single name.
	for _, in := range []string{"x/y", `x\y`, "a:b", "CON", "con.", " con "} {
		g := sanitizeName(in)
		if strings.ContainsAny(g, `<>:"/\|?*`) || g != strings.TrimRight(g, ". ") {
			t.Errorf("unsafe result %q for %q", g, in)
		}
	}
}

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 3; i++ {
		p, f, err := uniquePath(dir, "a.txt")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		paths = append(paths, filepath.Base(p))
	}
	want := []string{"a.txt", "a (2).txt", "a (3).txt"}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("got %v, want %v", paths, want)
		}
	}
	p, f, err := uniquePath(dir, "noext")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	p2, f2, err := uniquePath(dir, "noext")
	if err != nil {
		t.Fatal(err)
	}
	f2.Close()
	if filepath.Base(p) != "noext" || filepath.Base(p2) != "noext (2)" {
		t.Errorf("no-ext: %q %q", p, p2)
	}
	if _, _, err := uniquePath(dir, "../x"); err == nil {
		t.Error("separator in name accepted")
	}
	if _, _, err := uniquePath(filepath.Join(dir, "missing"), "a.txt"); err == nil {
		t.Error("missing dir should fail, not loop")
	}
}

func TestUniquePathConcurrent(t *testing.T) {
	dir := t.TempDir()
	const n = 20
	res := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() {
			p, f, err := uniquePath(dir, "x.bin")
			if err != nil {
				res <- "ERR " + err.Error()
				return
			}
			f.Close()
			res <- p
		}()
	}
	seen := map[string]bool{}
	for i := 0; i < n; i++ {
		p := <-res
		if seen[p] || strings.HasPrefix(p, "ERR") {
			t.Fatalf("duplicate or error: %q", p)
		}
		seen[p] = true
	}
}

func TestSniffExt(t *testing.T) {
	pad := func(b string) []byte { return append([]byte(b), make([]byte, 512)...) }
	ftyp := func(brand string) []byte { return pad("\x00\x00\x00\x18ftyp" + brand) }
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"jpg", pad("\xFF\xD8\xFF\xE0"), ".jpg"},
		{"png", pad("\x89PNG\r\n\x1a\n"), ".png"},
		{"gif87", pad("GIF87a"), ".gif"},
		{"gif89", pad("GIF89a"), ".gif"},
		{"webp", pad("RIFF\x00\x00\x00\x00WEBPVP8 "), ".webp"},
		{"wav", pad("RIFF\x00\x00\x00\x00WAVEfmt "), ".wav"},
		{"riff other", pad("RIFF\x00\x00\x00\x00AVI "), ""},
		{"heic", ftyp("heic"), ".heic"},
		{"heix", ftyp("heix"), ".heic"},
		{"hevc", ftyp("hevc"), ".heic"},
		{"heim", ftyp("heim"), ".heic"},
		{"heis", ftyp("heis"), ".heic"},
		{"mif1", ftyp("mif1"), ".heic"},
		{"msf1", ftyp("msf1"), ".heic"},
		{"avif", ftyp("avif"), ".avif"},
		{"mov", ftyp("qt  "), ".mov"},
		{"isom", ftyp("isom"), ".mp4"},
		{"iso2", ftyp("iso2"), ".mp4"},
		{"mp41", ftyp("mp41"), ".mp4"},
		{"mp42", ftyp("mp42"), ".mp4"},
		{"avc1", ftyp("avc1"), ".mp4"},
		{"m4v", ftyp("M4V "), ".mp4"},
		{"m4a", ftyp("M4A "), ".m4a"},
		{"unknown brand", ftyp("zzzz"), ""},
		{"pdf", pad("%PDF-1.7"), ".pdf"},
		{"zip", pad("PK\x03\x04"), ".zip"},
		{"gzip", pad("\x1f\x8b\x08"), ".gz"},
		{"mp3 id3", pad("ID3\x04"), ".mp3"},
		{"mp3 frame", pad("\xFF\xFB\x90\x00"), ".mp3"},
		{"aac adts", pad("\xFF\xF1\x50\x80"), ""},
		{"text", []byte("hello world, just text"), ""},
		{"empty", nil, ""},
		{"short", []byte{0xFF}, ""},
	}
	for _, tc := range tests {
		if got := sniffExt(tc.in); got != tc.want {
			t.Errorf("%s: sniffExt = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestInferKind(t *testing.T) {
	heic := append([]byte("\x00\x00\x00\x18ftypheic"), make([]byte, 100)...)
	tests := []struct {
		name, kind string
		head       []byte
		size       int64
		complete   bool
		want       string
	}{
		{"Text", "", []byte("hello"), 5, true, "text"},
		{"Text", "", []byte("https://example.com/a?b=c"), 25, true, "url"},
		{"Text", "", []byte("  http://example.com \n"), 22, true, "url"},
		{"Text", "", []byte("https://a.com and more"), 22, true, "text"},
		{"", "", []byte("hello"), 5, true, "text"},
		{"Text", "", []byte("ftp://example.com"), 17, true, "text"},
		{"Text", "", []byte("https://"), 8, true, "text"},
		{"Text", "", []byte("https:///path"), 13, true, "text"},
		{"IMG_2041", "", heic, int64(len(heic)), false, "file"},
		{"IMG_2041", "", heic, int64(len(heic)), true, "file"},
		{"notes.txt", "", []byte("hello"), 5, true, "file"},
		{"../notes.txt", "", []byte("hello"), 5, true, "file"},
		{"Text.", "", []byte("hello"), 5, true, "text"},
		{".jpg", "", []byte("hello"), 5, true, "text"},
		{"v1.verylongextension", "", []byte("x"), 1, true, "text"},
		{"a.b-c", "", []byte("x"), 1, true, "text"},
		{"", "", []byte{0xff, 0xfe, 0xfd}, 3, true, "file"},
		{"", "", []byte("hello"), 5, false, "file"},
		{"", "", []byte("hello"), 2 << 20, true, "file"},
		{"", "", []byte("hello"), 1 << 20, true, "text"},
		{"notes.txt", "text", []byte("hello"), 5, true, "text"},
		{"Text", "FILE", []byte("hello"), 5, true, "file"},
		{"Text", " Url ", []byte("hello"), 5, true, "url"},
		{"Text", "bogus", []byte("hello"), 5, true, "text"},
	}
	for _, tc := range tests {
		if got := inferKind(tc.name, tc.kind, tc.head, tc.size, tc.complete); got != tc.want {
			t.Errorf("inferKind(%q,%q,%q,%d,%v) = %q, want %q", tc.name, tc.kind, tc.head, tc.size, tc.complete, got, tc.want)
		}
	}
}

func TestIsLink(t *testing.T) {
	for s, want := range map[string]bool{
		"https://example.com":      true,
		"HTTP://Example.com/x?y=1": true,
		"http://192.168.1.2:8080/": true,
		"https://example.com/a b":  false,
		"example.com":              false,
		"javascript:alert(1)":      false,
		"http://":                  false,
		"":                         false,
		"https://a.com\nhttps://b": false,
	} {
		if got := isLink(s); got != want {
			t.Errorf("isLink(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestDefaultName(t *testing.T) {
	ts := time.Date(2026, 10, 24, 22, 15, 24, 0, time.Local)
	for ext, want := range map[string]string{
		".jpg": "Received 2026-10-24 22.15.24.jpg",
		"jpg":  "Received 2026-10-24 22.15.24.jpg",
		"":     "Received 2026-10-24 22.15.24.bin",
	} {
		if got := defaultName(ts, ext); got != want {
			t.Errorf("defaultName(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestMediaKind(t *testing.T) {
	for name, want := range map[string]string{
		"a.JPG": "image", "a.jpeg": "image", "a.png": "image", "a.gif": "image", "a.webp": "image",
		"a.heic": "image", "a.heif": "image", "a.avif": "image", "a.bmp": "image", "a.tif": "image", "a.tiff": "image",
		"a.mov": "video", "a.MP4": "video", "a.m4v": "video", "a.avi": "video", "a.mkv": "video",
		"a.webm": "video", "a.3gp": "video",
		"a.pdf": "file", "a": "file", "": "file", "a.m4a": "file",
	} {
		if got := mediaKind(name); got != want {
			t.Errorf("mediaKind(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestMarkOfTheWeb(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := markOfTheWeb(p); err != nil { // no-op off Windows; must not error
		t.Fatal(err)
	}
}
