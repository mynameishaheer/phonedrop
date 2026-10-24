// Minimal GDI+ (flat API) drawing for the tray panel and toasts: anti-aliased shapes and text into a
// premultiplied-alpha DIB that UpdateLayeredWindow puts on screen. Float arguments are passed as
// their bit patterns; Go's Windows syscall stub copies the first four arguments into XMM0-3 too,
// so the x64 calling convention picks them up.
package main

import (
	_ "embed"
	"math"
	"syscall"
	"unsafe"
)

var (
	gdiplus   = syscall.NewLazyDLL("gdiplus.dll")
	gdi32     = syscall.NewLazyDLL("gdi32.dll")
	gdipProcs = map[string]*syscall.LazyProc{}

	createCompatDC = gdi32.NewProc("CreateCompatibleDC")
	createDIB      = gdi32.NewProc("CreateDIBSection")
	selectObject   = gdi32.NewProc("SelectObject")
	deleteObject   = gdi32.NewProc("DeleteObject")
	deleteDC       = gdi32.NewProc("DeleteDC")
)

// Fluent UI System Icons (MIT, github.com/microsoft/fluentui-system-icons), 20 px glyphs subset with
// pyftsubset (~6 KB each). Code points come from the font's JSON map.
//
//go:embed icons-regular.ttf
var iconsRegular []byte

//go:embed icons-filled.ttf
var iconsFilled []byte

// Icon code points: {regular, filled}.
var (
	icPhone         = [2]rune{0xF5E0, 0xF5EA}
	icPhoneArrow    = [2]rune{0xE956, 0xE968}
	icPhoneAdd      = [2]rune{0xE954, 0xE966}
	icSend          = [2]rune{0xF699, 0xF6A2}
	icDownload      = [2]rune{0xF150, 0xF150}
	icLink          = [2]rune{0xF4E4, 0xF4EE}
	icImage         = [2]rune{0xF488, 0xF48F}
	icDocument      = [2]rune{0xF378, 0xF378}
	icClipboardText = [2]rune{0xE369, 0xE37E}
	icQrCode        = [2]rune{0xE9EC, 0xE9F4}
	icShieldCheck   = [2]rune{0xEAC8, 0xEAD1}
	icShieldError   = [2]rune{0xF6C2, 0xF6CB}
	icSettings      = [2]rune{0xF6A9, 0xF6B2}
	icDismiss       = [2]rune{0xF369, 0xF369}
	icBack          = [2]rune{0xF15B, 0xF15B}
	icOpen          = [2]rune{0xF582, 0xF58C}
	icAdd           = [2]rune{0xF109, 0xF109}
	icDelete        = [2]rune{0xF34C, 0xF34C}
	icFolderOpen    = [2]rune{0xF42E, 0xF432}
	icWifi          = [2]rune{0xF8AC, 0xF8C4}
	icWifiWarning   = [2]rune{0xEE5C, 0xEE5C}
	icWifiOff       = [2]rune{0xEE59, 0xEE59}
	icPower         = [2]rune{0xF60E, 0xF618}
	icCheckmark     = [2]rune{0xF294, 0xF294}
	icCopy          = [2]rune{0xF32B, 0xF32B}
	icHistory       = [2]rune{0xF47E, 0xF485}
	icVideo         = [2]rune{0xF84C, 0xF864}
	icWarning       = [2]rune{0xF869, 0xF881}
	icInfo          = [2]rune{0xF4A3, 0xF4AA}
	icLock          = [2]rune{0xE78F, 0xE79D}
	icSync          = [2]rune{0xF190, 0xF190}
)

// gp calls a GDI+ flat API function. Pointer arguments must be converted inline at the call
// (uintptr(unsafe.Pointer(&x))) so the compiler keeps them alive for the call.
//
//go:uintptrescapes
func gp(name string, args ...uintptr) uintptr {
	p, ok := gdipProcs[name]
	if !ok {
		p = gdiplus.NewProc(name)
		gdipProcs[name] = p
	}
	r, _, _ := p.Call(args...)
	return r
}

func fl(v float32) uintptr { return uintptr(math.Float32bits(v)) }

func wstr(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

type rectF struct{ X, Y, W, H float32 }

func (r rectF) has(x, y float32) bool { return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H }

var (
	fontText, fontSemi, fontSmall, fontSmallSemi, fontTitle uintptr
	fontIcon, fontIconFilled, fontIconSm                    uintptr
	formats                                                 = map[[2]int]uintptr{}
	iconColl                                                uintptr
)

// gdipInit starts GDI+ and loads the fonts: Segoe UI Variable (Windows 11), else Segoe UI.
func gdipInit() {
	in := struct {
		version    uint32
		debugCb    uintptr
		suppressBg int32
		suppressEx int32
	}{version: 1}
	var token uintptr
	gp("GdiplusStartup", uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&in)), 0)

	regular := []string{"Segoe UI Variable Text", "Segoe UI", "Tahoma"}
	semibold := []string{"Segoe UI Variable Text Semibold", "Segoe UI Semibold"}
	display := []string{"Segoe UI Variable Display Semibold", "Segoe UI Semibold"}
	fontText = newFont(regular, 0, 13, 0)
	fontSmall = newFont(regular, 0, 12, 0)
	fontSemi = newFont(semibold, 0, 13, 0)
	if fontSemi == 0 {
		fontSemi = newFont(regular, 0, 13, 1)
	}
	fontSmallSemi = newFont(semibold, 0, 12, 0)
	if fontSmallSemi == 0 {
		fontSmallSemi = newFont(regular, 0, 12, 1)
	}
	fontTitle = newFont(display, 0, 14, 0)
	if fontTitle == 0 {
		fontTitle = fontSemi
	}

	gp("GdipNewPrivateFontCollection", uintptr(unsafe.Pointer(&iconColl)))
	gp("GdipPrivateAddMemoryFont", iconColl, uintptr(unsafe.Pointer(&iconsRegular[0])), uintptr(len(iconsRegular)))
	gp("GdipPrivateAddMemoryFont", iconColl, uintptr(unsafe.Pointer(&iconsFilled[0])), uintptr(len(iconsFilled)))
	fontIcon = newFont([]string{"FluentSystemIcons-Regular"}, iconColl, 20, 0)
	fontIconFilled = newFont([]string{"FluentSystemIcons-Filled"}, iconColl, 20, 0)
	fontIconSm = newFont([]string{"FluentSystemIcons-Regular"}, iconColl, 16, 0)
}

func newFont(families []string, coll uintptr, size float32, style int) uintptr {
	for _, name := range families {
		var fam, f uintptr
		if gp("GdipCreateFontFamilyFromName", uintptr(unsafe.Pointer(wstr(name))), coll, uintptr(unsafe.Pointer(&fam))) != 0 {
			continue
		}
		gp("GdipCreateFont", fam, fl(size), uintptr(style), 2 /*UnitPixel*/, uintptr(unsafe.Pointer(&f)))
		gp("GdipDeleteFontFamily", fam)
		if f != 0 {
			return f
		}
	}
	return 0
}

// format: align/valign 0 near, 1 center, 2 far; single line, ellipsis when it doesn't fit.
func format(align, valign int) uintptr {
	k := [2]int{align, valign}
	if f, ok := formats[k]; ok {
		return f
	}
	var typo, f uintptr
	gp("GdipStringFormatGetGenericTypographic", uintptr(unsafe.Pointer(&typo)))
	gp("GdipCloneStringFormat", typo, uintptr(unsafe.Pointer(&f)))
	gp("GdipSetStringFormatFlags", f, 0x1000|0x0800) // NoWrap | MeasureTrailingSpaces
	gp("GdipSetStringFormatTrimming", f, 3)          // EllipsisCharacter
	gp("GdipSetStringFormatAlign", f, uintptr(align))
	gp("GdipSetStringFormatLineAlign", f, uintptr(valign))
	formats[k] = f
	return f
}

// canvas: a 32-bit premultiplied DIB section selected into a memory DC, with a GDI+ Graphics on
// top whose world transform applies the DPI scale, so drawing code works in 96-DPI units.
type canvas struct {
	w, h          int // pixels
	scale         float32
	dc, dib, bits uintptr
	oldBmp        uintptr // the DC's original bitmap, selected back before the DIB is deleted
	bmp, g        uintptr
}

func newCanvas(w, h int, scale float32) *canvas {
	c := &canvas{w: w, h: h, scale: scale}
	c.dc, _, _ = createCompatDC.Call(0)
	bmi := struct {
		size                 uint32
		width, height        int32
		planes, bitCount     uint16
		compression, sizeImg uint32
		xppm, yppm           int32
		clrUsed, clrImp      uint32
		colors               uint32
	}{width: int32(w), height: -int32(h), planes: 1, bitCount: 32}
	bmi.size = 40
	c.dib, _, _ = createDIB.Call(c.dc, uintptr(unsafe.Pointer(&bmi)), 0, uintptr(unsafe.Pointer(&c.bits)), 0, 0)
	c.oldBmp, _, _ = selectObject.Call(c.dc, c.dib)
	gp("GdipCreateBitmapFromScan0", uintptr(w), uintptr(h), uintptr(w*4), 0x000E200B /*PARGB*/, c.bits, uintptr(unsafe.Pointer(&c.bmp)))
	gp("GdipGetImageGraphicsContext", c.bmp, uintptr(unsafe.Pointer(&c.g)))
	gp("GdipSetSmoothingMode", c.g, 4)      // AntiAlias
	gp("GdipSetPixelOffsetMode", c.g, 2)    // HighQuality
	gp("GdipSetTextRenderingHint", c.g, 4)  // AntiAlias (grayscale; ClearType needs an opaque target)
	gp("GdipSetCompositingQuality", c.g, 4) // AssumeLinear: plain sRGB blending (HighQuality blends gamma-corrected,
	// which made every translucent shade far lighter on real Windows; Wine ignores it)
	gp("GdipScaleWorldTransform", c.g, fl(scale), fl(scale), 0)
	return c
}

func (c *canvas) free() {
	gp("GdipDeleteGraphics", c.g)
	gp("GdipDisposeImage", c.bmp)
	if c.oldBmp != 0 { // a bitmap still selected into a DC can't be deleted
		selectObject.Call(c.dc, c.oldBmp)
	}
	deleteObject.Call(c.dib)
	deleteDC.Call(c.dc)
}

func (c *canvas) clear() { gp("GdipGraphicsClear", c.g, 0) }

func withBrush(col uint32, f func(b uintptr)) {
	var b uintptr
	gp("GdipCreateSolidFill", uintptr(col), uintptr(unsafe.Pointer(&b)))
	f(b)
	gp("GdipDeleteBrush", b)
}

func roundPath(r rectF, rad float32) uintptr {
	var p uintptr
	gp("GdipCreatePath", 0, uintptr(unsafe.Pointer(&p)))
	if rad <= 0 {
		gp("GdipAddPathRectangle", p, fl(r.X), fl(r.Y), fl(r.W), fl(r.H))
		return p
	}
	d := rad * 2
	gp("GdipAddPathArc", p, fl(r.X), fl(r.Y), fl(d), fl(d), fl(180), fl(90))
	gp("GdipAddPathArc", p, fl(r.X+r.W-d), fl(r.Y), fl(d), fl(d), fl(270), fl(90))
	gp("GdipAddPathArc", p, fl(r.X+r.W-d), fl(r.Y+r.H-d), fl(d), fl(d), fl(0), fl(90))
	gp("GdipAddPathArc", p, fl(r.X), fl(r.Y+r.H-d), fl(d), fl(d), fl(90), fl(90))
	gp("GdipClosePathFigure", p)
	return p
}

func (c *canvas) fill(r rectF, rad float32, col uint32) {
	if col>>24 == 0 {
		return
	}
	p := roundPath(r, rad)
	withBrush(col, func(b uintptr) { gp("GdipFillPath", c.g, b, p) })
	gp("GdipDeletePath", p)
}

// stroke draws a border of width w inside r.
func (c *canvas) stroke(r rectF, rad, w float32, col uint32) {
	if col>>24 == 0 {
		return
	}
	h := w / 2
	p := roundPath(rectF{r.X + h, r.Y + h, r.W - w, r.H - w}, rad-h)
	var pen uintptr
	gp("GdipCreatePen1", uintptr(col), fl(w), 0 /*UnitWorld*/, uintptr(unsafe.Pointer(&pen)))
	gp("GdipDrawPath", c.g, pen, p)
	gp("GdipDeletePen", pen)
	gp("GdipDeletePath", p)
}

// rects fills many axis-aligned rectangles in one call, without anti-aliasing (QR modules).
func (c *canvas) rects(rs []rectF, col uint32) {
	if len(rs) == 0 {
		return
	}
	gp("GdipSetSmoothingMode", c.g, 3) // None: neighbours must not leave seams
	withBrush(col, func(b uintptr) { gp("GdipFillRectangles", c.g, b, uintptr(unsafe.Pointer(&rs[0])), uintptr(len(rs))) })
	gp("GdipSetSmoothingMode", c.g, 4)
}

func (c *canvas) circle(cx, cy, rad float32, col uint32) {
	withBrush(col, func(b uintptr) { gp("GdipFillEllipse", c.g, b, fl(cx-rad), fl(cy-rad), fl(2*rad), fl(2*rad)) })
}

func (c *canvas) text(s string, font uintptr, r rectF, col uint32, align, valign int) {
	if s == "" || font == 0 {
		return
	}
	u, _ := syscall.UTF16FromString(s)
	withBrush(col, func(b uintptr) {
		gp("GdipDrawString", c.g, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), font, uintptr(unsafe.Pointer(&r)), format(align, valign), b)
	})
}

func (c *canvas) measure(s string, font uintptr) float32 {
	if s == "" {
		return 0
	}
	u, _ := syscall.UTF16FromString(s)
	layout, out := rectF{0, 0, 2000, 200}, rectF{}
	gp("GdipMeasureString", c.g, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), font,
		uintptr(unsafe.Pointer(&layout)), format(0, 0), uintptr(unsafe.Pointer(&out)), 0, 0)
	return out.W
}

// icon draws a Fluent glyph centred in r. The font's Windows line metrics are a little taller
// than its em box, so a centred line sits ~4% low; nudge it back up.
func (c *canvas) icon(ic [2]rune, filled bool, font uintptr, size float32, r rectF, col uint32) {
	if filled && font == fontIcon {
		font = fontIconFilled
	}
	r.Y -= size * 0.04
	c.text(string(ic[btoi(filled)]), font, r, col, 1, 1)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// savePNG writes the canvas to a file (preview mode only).
func (c *canvas) savePNG(path string) {
	clsid := struct {
		d1     uint32
		d2, d3 uint16
		d4     [8]byte
	}{0x557cf406, 0x1a04, 0x11d3, [8]byte{0x9a, 0x73, 0x00, 0x00, 0xf8, 0x1e, 0xf3, 0x2e}}
	gp("GdipSaveImageToFile", c.bmp, uintptr(unsafe.Pointer(wstr(path))), uintptr(unsafe.Pointer(&clsid)), 0)
}

func (c *canvas) hicon() uintptr {
	var h uintptr
	gp("GdipCreateHICONFromBitmap", c.bmp, uintptr(unsafe.Pointer(&h)))
	return h
}

// Colour helpers (0xAARRGGBB).
func alpha(col uint32, a float32) uint32 {
	return uint32(float32(col>>24)*a)<<24 | col&0xFFFFFF
}

func mix(a, b uint32, t float32) uint32 {
	ch := func(s uint) uint32 {
		x, y := float32(a>>s&0xFF), float32(b>>s&0xFF)
		return uint32(x+(y-x)*t+0.5) << s
	}
	return ch(24) | ch(16) | ch(8) | ch(0)
}

func hexColor(s string) (uint32, bool) {
	if len(s) != 7 || s[0] != '#' {
		return 0, false
	}
	var v uint32
	for _, ch := range s[1:] {
		v <<= 4
		switch {
		case ch >= '0' && ch <= '9':
			v |= uint32(ch - '0')
		case ch >= 'a' && ch <= 'f':
			v |= uint32(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			v |= uint32(ch-'A') + 10
		default:
			return 0, false
		}
	}
	return 0xFF000000 | v, true
}
