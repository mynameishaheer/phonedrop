package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestQRReedSolomon(t *testing.T) {
	// ISO/IEC 18004 Annex I example ("01234567", version 1-M): data codewords -> 10 EC codewords.
	data := []byte{0x10, 0x20, 0x0C, 0x56, 0x61, 0x80, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11}
	want := []byte{0xA5, 0x24, 0xD4, 0xC1, 0xED, 0x36, 0xC7, 0x87, 0x2C, 0x55}
	if got := qrECCodewords(data, 10); !bytes.Equal(got, want) {
		t.Errorf("01234567: got % X, want % X", got, want)
	}
	// "HELLO WORLD" at 1-M (byte-for-byte the well-known tutorial example).
	data = []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	want = []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}
	if got := qrECCodewords(data, 10); !bytes.Equal(got, want) {
		t.Errorf("HELLO WORLD: got %v, want %v", got, want)
	}
}

func TestQRTables(t *testing.T) {
	// Data codewords per version at level M, from the spec's capacity table.
	want := []int{16, 28, 44, 64, 86, 108, 124, 154, 182, 216, 254, 290, 334, 365, 415, 453, 507, 563, 627, 669,
		714, 782, 860, 914, 1000, 1062, 1128, 1193, 1267, 1373, 1455, 1541, 1631, 1725, 1812, 1914, 1992, 2102, 2216, 2334}
	for v, w := range want {
		if got := qrDataCodewords(v + 1); got != w {
			t.Errorf("version %d: %d data codewords, want %d", v+1, got, w)
		}
	}
	for v, pos := range map[int][]int{
		1: nil, 2: {6, 18}, 7: {6, 22, 38}, 14: {6, 26, 46, 66}, 21: {6, 28, 50, 72, 94},
		32: {6, 34, 60, 86, 112, 138}, 36: {6, 24, 50, 76, 102, 128, 154}, 40: {6, 30, 58, 86, 114, 142, 170},
	} {
		if got := qrAlignPositions(v); !slices.Equal(got, pos) {
			t.Errorf("version %d alignment: %v, want %v", v, got, pos)
		}
	}
}

func TestQRInfoBits(t *testing.T) {
	// Format info for level M, masks 0-7 (spec Table C.1), after the 0x5412 XOR.
	fmts := []int{0b101010000010010, 0b101000100100101, 0b101111001111100, 0b101101101001011,
		0b100010111111001, 0b100000011001110, 0b100111110010111, 0b100101010100000}
	for mask, w := range fmts {
		if got := qrBCH(mask, 10, 0x537) ^ 0x5412; got != w {
			t.Errorf("format mask %d: %015b, want %015b", mask, got, w)
		}
	}
	// Version info (spec Table D.1).
	for v, w := range map[int]int{7: 0b000111110010010100, 8: 0b001000010110111100, 40: 0b101000110001101001} {
		if got := qrBCH(v, 12, 0x1F25); got != w {
			t.Errorf("version %d: %018b, want %018b", v, got, w)
		}
	}
}

func TestQRStructure(t *testing.T) {
	for _, n := range []int{1, 20, 150, 400} { // versions 1, 2, 9-ish, 18-ish
		m, err := qrEncode(bytes.Repeat([]byte("x"), n))
		if err != nil {
			t.Fatal(err)
		}
		size := len(m)
		if (size-17)%4 != 0 {
			t.Fatalf("size %d is not 17+4v", size)
		}
		for i, row := range m {
			if len(row) != size {
				t.Fatalf("row %d has %d modules, want %d", i, len(row), size)
			}
		}
		// Finder patterns: dark 7x7 ring, light ring, dark 3x3 core, in three corners.
		for _, o := range [][2]int{{0, 0}, {size - 7, 0}, {0, size - 7}} {
			for dy := 0; dy < 7; dy++ {
				for dx := 0; dx < 7; dx++ {
					d := max(abs(dx-3), abs(dy-3))
					if m[o[1]+dy][o[0]+dx] != (d != 2) {
						t.Fatalf("size %d: bad finder module at %d,%d", size, o[0]+dx, o[1]+dy)
					}
				}
			}
		}
		for i := 8; i < size-8; i++ { // timing patterns
			if m[6][i] != (i%2 == 0) || m[i][6] != (i%2 == 0) {
				t.Fatalf("size %d: bad timing at %d", size, i)
			}
		}
		if !m[size-8][8] {
			t.Fatalf("size %d: dark module missing", size)
		}
		// The two format copies agree and carry a valid level-M mask.
		a, b := 0, 0
		for i := 0; i < 15; i++ {
			x1, y1 := 8, i // first copy as laid out by drawFormat
			switch {
			case i < 6:
			case i < 8:
				y1 = i + 1
			case i == 8:
				x1, y1 = 7, 8
			default:
				x1, y1 = 14-i, 8
			}
			x2, y2 := size-1-i, 8
			if i >= 8 {
				x2, y2 = 8, size-15+i
			}
			if m[y1][x1] {
				a |= 1 << i
			}
			if m[y2][x2] {
				b |= 1 << i
			}
		}
		if a != b {
			t.Fatalf("size %d: format copies differ: %015b vs %015b", size, a, b)
		}
		if mask := (a ^ 0x5412) >> 10; mask > 7 || qrBCH(mask, 10, 0x537)^0x5412 != a {
			t.Fatalf("size %d: format bits %015b are not a valid level-M code", size, a)
		}
	}
}

func TestQRCapacity(t *testing.T) {
	for _, c := range []struct{ n, ver int }{
		{0, 1}, {1, 1}, {14, 1}, {15, 2}, {26, 2}, {27, 3}, {106, 6}, {107, 7}, {180, 9}, {181, 10},
		{213, 10}, {214, 11}, {287, 12}, {288, 13}, {2331, 40},
	} {
		m, err := qrEncode(make([]byte, c.n))
		if err != nil {
			t.Fatalf("%d bytes: %v", c.n, err)
		}
		if got := (len(m) - 17) / 4; got != c.ver {
			t.Errorf("%d bytes: version %d, want %d", c.n, got, c.ver)
		}
	}
	if _, err := qrEncode(make([]byte, 2332)); err == nil {
		t.Error("2332 bytes should not fit")
	}
}

// qrDecode is a minimal reader for our own symbols: it unmasks, un-zigzags, checks every
// block's Reed-Solomon codewords and returns the byte-mode payload.
func qrDecode(t *testing.T, m [][]bool) []byte {
	t.Helper()
	ver := (len(m) - 17) / 4
	g := newQRGrid(ver)
	g.drawFunctionPatterns()
	mask := 0
	for i := 0; i < 15; i++ { // first format copy
		x, y := 8, i
		switch {
		case i >= 9:
			x, y = 14-i, 8
		case i == 8:
			x, y = 7, 8
		case i >= 6:
			y = i + 1
		}
		if m[y][x] {
			mask |= 1 << i
		}
	}
	mask = (mask ^ 0x5412) >> 10
	for y := range m {
		copy(g.m[y], m[y])
	}
	g.applyMask(mask)
	var cw []byte
	bit := 0
	for right := g.n - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for v := 0; v < g.n; v++ {
			for j := 0; j < 2; j++ {
				x, y := right-j, v
				if (right+1)&2 == 0 {
					y = g.n - 1 - v
				}
				if !g.fn[y][x] && bit < qrRawCodewords(ver)*8 {
					if bit%8 == 0 {
						cw = append(cw, 0)
					}
					if g.m[y][x] {
						cw[bit/8] |= 0x80 >> (bit % 8)
					}
					bit++
				}
			}
		}
	}
	// De-interleave into blocks and verify EC.
	nb, ecl, raw := qrNumBlocks[ver-1], qrECLen[ver-1], qrRawCodewords(ver)
	short := nb - raw%nb
	shortData := raw/nb - ecl
	blocks := make([][]byte, nb)
	k := 0
	for i := 0; i <= shortData; i++ {
		for j := range blocks {
			if i < shortData || j >= short {
				blocks[j] = append(blocks[j], cw[k])
				k++
			}
		}
	}
	var data []byte
	for j := range blocks {
		ec := make([]byte, ecl) // EC codewords are interleaved too
		for i := range ec {
			ec[i] = cw[k+i*nb+j]
		}
		if want := qrECCodewords(blocks[j], ecl); !bytes.Equal(want, ec) {
			t.Fatalf("version %d block %d: EC mismatch", ver, j)
		}
		data = append(data, blocks[j]...)
	}
	// Parse: mode 0100, count, bytes.
	var bs qrBits
	bs.b = data
	rd := func(n int) (v int) {
		for ; n > 0; n-- {
			v = v<<1 | int(data[bs.n/8]>>(7-bs.n%8)&1)
			bs.n++
		}
		return
	}
	if mode := rd(4); mode != 4 {
		t.Fatalf("version %d: mode %04b", ver, mode)
	}
	out := make([]byte, rd(qrCountBits(ver)))
	for i := range out {
		out[i] = byte(rd(8))
	}
	// The last payload nibble shares a byte with the 4-bit terminator; pad bytes 0xEC 0x11
	// alternate after it up to the capacity.
	if end := bs.n / 8; end < len(data) {
		if data[end]&0x0F != 0 {
			t.Fatalf("version %d: bad terminator", ver)
		}
		for i, pad := end+1, byte(0xEC); i < len(data); i, pad = i+1, pad^(0xEC^0x11) {
			if data[i] != pad {
				t.Fatalf("version %d: bad pad byte %02X at %d", ver, data[i], i)
			}
		}
	}
	return out
}

func TestQRRoundTrip(t *testing.T) {
	json := `{"pd":1,"name":"DESKTOP-AB12CD","host":"desktop-ab12cd.local","ips":["192.168.18.20"],"port":47474,` +
		`"id":"9f2c1a7e5b3d4c60","secret":"` + strings.Repeat("0123456789abcdef", 4) + `"}`
	cases := [][]byte{nil, []byte("A"), []byte("Shaheer’s iPhone ✓"), []byte(json), bytes.Repeat([]byte{0xFF, 0}, 140)}
	for v := 1; v <= 40; v++ { // exactly full symbol of every version
		n := (8*qrDataCodewords(v) - 4 - qrCountBits(v)) / 8
		cases = append(cases, bytes.Repeat([]byte("QR!"), n)[:n])
	}
	for _, d := range cases {
		m, err := qrEncode(d)
		if err != nil {
			t.Fatal(err)
		}
		if got := qrDecode(t, m); !bytes.Equal(got, d) {
			t.Errorf("%d bytes: round trip mismatch", len(d))
		}
	}
}
