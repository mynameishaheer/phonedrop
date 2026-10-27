package main

import "errors"

// QR code encoder (ISO/IEC 18004): byte mode, error correction level M, versions 1-40.
// Used to show the pairing payload on screen. Written from the spec, no dependencies.

// Level M block layout per version (index v-1): EC codewords per block and number of blocks.
// Data codewords follow from the module count (qrDataCodewords).
var qrECLen = [40]int{10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26,
	26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28}
var qrNumBlocks = [40]int{1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16,
	17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49}

// qrEncode returns the QR code for data as a square matrix of modules (true = dark),
// without the quiet zone. Byte mode, error correction level M, smallest version that fits
// (versions 1-40), mask chosen by the standard penalty rules.
func qrEncode(data []byte) ([][]bool, error) {
	ver := 0
	for v := 1; v <= 40; v++ {
		if 4+qrCountBits(v)+8*len(data) <= 8*qrDataCodewords(v) {
			ver = v
			break
		}
	}
	if ver == 0 {
		return nil, errors.New("qr: data too long for version 40-M")
	}

	// Bit stream: mode 0100, character count, bytes, terminator, pad bytes 0xEC 0x11...
	var bs qrBits
	bs.put(0b0100, 4)
	bs.put(len(data), qrCountBits(ver))
	for _, b := range data {
		bs.put(int(b), 8)
	}
	capBits := 8 * qrDataCodewords(ver)
	bs.put(0, min(4, capBits-bs.n)) // the partial last byte is already zero-padded
	for pad := byte(0xEC); len(bs.b) < capBits/8; pad ^= 0xEC ^ 0x11 {
		bs.b = append(bs.b, pad)
	}

	g := newQRGrid(ver)
	g.drawFunctionPatterns()
	g.placeData(qrInterleave(bs.b, ver))

	// Try all 8 masks and keep the lowest penalty. Masking is an XOR, so it undoes itself.
	best, bestScore := 0, -1
	for mask := 0; mask < 8; mask++ {
		g.applyMask(mask)
		g.drawFormat(mask)
		if s := g.penalty(); bestScore < 0 || s < bestScore {
			best, bestScore = mask, s
		}
		g.applyMask(mask)
	}
	g.applyMask(best)
	g.drawFormat(best)
	return g.m, nil
}

func qrCountBits(ver int) int {
	if ver < 10 {
		return 8
	}
	return 16
}

// qrRawCodewords is the number of codewords (data + EC) that fit in version ver.
func qrRawCodewords(ver int) int {
	n := (16*ver+128)*ver + 64 // all modules except finders, timing, format and dark module
	if ver >= 2 {
		a := ver/7 + 2 // alignment patterns per side
		n -= (25*a-10)*a - 55
		if ver >= 7 {
			n -= 36 // version information blocks
		}
	}
	return n / 8
}

func qrDataCodewords(ver int) int {
	return qrRawCodewords(ver) - qrECLen[ver-1]*qrNumBlocks[ver-1]
}

// qrBits is a big-endian bit buffer.
type qrBits struct {
	b []byte
	n int // bits written
}

func (q *qrBits) put(val, bits int) {
	for i := bits - 1; i >= 0; i-- {
		if q.n%8 == 0 {
			q.b = append(q.b, 0)
		}
		q.b[len(q.b)-1] |= byte(val>>i&1) << (7 - q.n%8)
		q.n++
	}
}

// GF(256) with the QR polynomial x^8+x^4+x^3+x^2+1 (0x11D); exp is doubled to skip a modulo.
var qrExp, qrLog = func() (e [510]byte, l [256]int) {
	x := 1
	for i := 0; i < 255; i++ {
		e[i], e[i+255], l[x] = byte(x), byte(x), i
		if x <<= 1; x >= 256 {
			x ^= 0x11D
		}
	}
	return
}()

func qrMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return qrExp[qrLog[a]+qrLog[b]]
}

// qrECCodewords returns the n Reed-Solomon check bytes of data: the remainder of
// data*x^n divided by the generator (x-1)(x-2)(x-4)... of degree n.
func qrECCodewords(data []byte, n int) []byte {
	gen := make([]byte, n) // generator coefficients without the leading 1, highest power first
	gen[n-1] = 1
	root := byte(1)
	for i := 0; i < n; i++ {
		for j := range gen {
			gen[j] = qrMul(gen[j], root)
			if j+1 < n {
				gen[j] ^= gen[j+1]
			}
		}
		root = qrMul(root, 2)
	}
	rem := make([]byte, n)
	for _, d := range data {
		f := d ^ rem[0]
		copy(rem, rem[1:])
		rem[n-1] = 0
		for i, c := range gen {
			rem[i] ^= qrMul(c, f)
		}
	}
	return rem
}

// qrInterleave splits the data codewords into blocks, appends each block's EC codewords and
// interleaves them: first all blocks' 1st data byte, then 2nd, ..., then the EC bytes.
// The first blocks are one data byte shorter than the last ones.
func qrInterleave(data []byte, ver int) []byte {
	nb, ecl, raw := qrNumBlocks[ver-1], qrECLen[ver-1], qrRawCodewords(ver)
	short := nb - raw%nb // number of short blocks
	shortData := raw/nb - ecl
	var dbs, ebs [][]byte
	for i, k := 0, 0; i < nb; i++ {
		n := shortData
		if i >= short {
			n++
		}
		dbs = append(dbs, data[k:k+n])
		ebs = append(ebs, qrECCodewords(data[k:k+n], ecl))
		k += n
	}
	out := make([]byte, 0, raw)
	for i := 0; i <= shortData; i++ {
		for _, d := range dbs {
			if i < len(d) { // the extra byte exists only in long blocks
				out = append(out, d[i])
			}
		}
	}
	for i := 0; i < ecl; i++ {
		for _, e := range ebs {
			out = append(out, e[i])
		}
	}
	return out
}

// qrGrid is the module matrix under construction; fn marks function modules (not data).
type qrGrid struct {
	ver, n int
	m, fn  [][]bool
}

func newQRGrid(ver int) *qrGrid {
	n := 4*ver + 17
	g := &qrGrid{ver: ver, n: n, m: make([][]bool, n), fn: make([][]bool, n)}
	for i := range g.m {
		g.m[i], g.fn[i] = make([]bool, n), make([]bool, n)
	}
	return g
}

// set colours a function module; x is the column, y the row.
func (g *qrGrid) set(x, y int, dark bool) { g.m[y][x], g.fn[y][x] = dark, true }

func (g *qrGrid) drawFunctionPatterns() {
	n := g.n
	for i := 0; i < n; i++ { // timing patterns
		g.set(6, i, i%2 == 0)
		g.set(i, 6, i%2 == 0)
	}
	// Finders with their light separator (ring at distance 4, clipped to the symbol).
	for _, c := range [][2]int{{3, 3}, {n - 4, 3}, {3, n - 4}} {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := c[0]+dx, c[1]+dy
				if x < 0 || y < 0 || x >= n || y >= n {
					continue
				}
				d := max(abs(dx), abs(dy))
				g.set(x, y, d != 2 && d != 4)
			}
		}
	}
	pos := qrAlignPositions(g.ver)
	for i, cy := range pos {
		for j, cx := range pos {
			last := len(pos) - 1
			if (i == 0 && j == 0) || (i == 0 && j == last) || (i == last && j == 0) {
				continue // would overlap a finder
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					g.set(cx+dx, cy+dy, max(abs(dx), abs(dy)) != 1)
				}
			}
		}
	}
	g.drawFormat(0) // reserve the format area; redrawn once the mask is known
	if g.ver >= 7 {
		// 6-bit version + 12-bit BCH(18,6) remainder, written twice (bottom-left, top-right).
		bits := qrBCH(g.ver, 12, 0x1F25)
		for i := 0; i < 18; i++ {
			a, b := n-11+i%3, i/3
			g.set(a, b, bits>>i&1 != 0)
			g.set(b, a, bits>>i&1 != 0)
		}
	}
}

// qrBCH appends the bch-bit remainder of val*x^bch modulo poly to val.
func qrBCH(val, bch, poly int) int {
	rem := val
	for i := 0; i < bch; i++ {
		rem = rem<<1 ^ (rem>>(bch-1))*poly
	}
	return val<<bch | rem
}

// qrAlignPositions returns the row/column coordinates of the alignment pattern centres:
// 6, then evenly spaced (step rounded up to even) back from the bottom-right corner.
func qrAlignPositions(ver int) []int {
	if ver == 1 {
		return nil
	}
	cnt := ver/7 + 2
	step := (ver*4 + cnt*2 + 1) / (cnt*2 - 2) * 2
	if ver == 32 {
		step = 26 // the one version where the formula is off
	}
	pos := make([]int, cnt)
	pos[0] = 6
	for i := 1; i < cnt; i++ {
		pos[cnt-i] = 4*ver + 10 - (i-1)*step
	}
	return pos
}

// drawFormat writes the 15-bit format information (level M = 00, then the mask) twice, and
// the always-dark module.
func (g *qrGrid) drawFormat(mask int) {
	n := g.n
	bits := qrBCH(mask, 10, 0x537) ^ 0x5412 // level M is 00, so the 5 data bits are just the mask
	bit := func(i int) bool { return bits>>i&1 != 0 }
	for i := 0; i < 6; i++ {
		g.set(8, i, bit(i))
	}
	g.set(8, 7, bit(6))
	g.set(8, 8, bit(7))
	g.set(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		g.set(14-i, 8, bit(i))
	}
	for i := 0; i < 8; i++ {
		g.set(n-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		g.set(8, n-15+i, bit(i))
	}
	g.set(8, n-8, true)
}

// placeData writes the codewords MSB first in the zigzag order: two-column strips from the
// right edge, alternately going up and down, skipping function modules (and the timing column).
// Leftover remainder bits stay light.
func (g *qrGrid) placeData(cw []byte) {
	i := 0
	for right := g.n - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for v := 0; v < g.n; v++ {
			for j := 0; j < 2; j++ {
				x, y := right-j, v
				if (right+1)&2 == 0 { // this strip runs upward
					y = g.n - 1 - v
				}
				if !g.fn[y][x] && i < len(cw)*8 {
					g.m[y][x] = cw[i/8]>>(7-i%8)&1 != 0
					i++
				}
			}
		}
	}
}

func (g *qrGrid) applyMask(mask int) {
	for y := 0; y < g.n; y++ {
		for x := 0; x < g.n; x++ {
			var flip bool
			switch mask {
			case 0:
				flip = (x+y)%2 == 0
			case 1:
				flip = y%2 == 0
			case 2:
				flip = x%3 == 0
			case 3:
				flip = (x+y)%3 == 0
			case 4:
				flip = (x/3+y/2)%2 == 0
			case 5:
				flip = x*y%2+x*y%3 == 0
			case 6:
				flip = (x*y%2+x*y%3)%2 == 0
			case 7:
				flip = ((x+y)%2+x*y%3)%2 == 0
			}
			if flip && !g.fn[y][x] {
				g.m[y][x] = !g.m[y][x]
			}
		}
	}
}

// penalty scores the current matrix with the four spec rules (lower is better).
func (g *qrGrid) penalty() int {
	n, p, dark := g.n, 0, 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if g.m[y][x] {
				dark++
			}
			// Rule 2: 2x2 blocks of one colour.
			if x+1 < n && y+1 < n && g.m[y][x] == g.m[y][x+1] && g.m[y][x] == g.m[y+1][x] && g.m[y][x] == g.m[y+1][x+1] {
				p += 3
			}
		}
	}
	line := make([]bool, n)
	for t := 0; t < 2*n; t++ { // every row, then every column
		for i := range line {
			if t < n {
				line[i] = g.m[t][i]
			} else {
				line[i] = g.m[i][t-n]
			}
		}
		p += qrRuns(line)
	}
	// Rule 4: deviation of the dark share from 50%, in steps of 5%.
	total := n * n
	p += 10 * ((abs(dark*20-total*10)+total-1)/total - 1)
	return p
}

// qrRuns applies rules 1 and 3 to one row or column: 3+extra for each run of 5 or more equal
// modules, and 40 for each 1:1:3:1:1 finder-like pattern with 4 light modules on a side
// (the light quiet zone counts).
func qrRuns(line []bool) int {
	p, run := 0, 0
	for i, c := range line {
		if i > 0 && c == line[i-1] {
			run++
			if run == 5 {
				p += 3
			} else if run > 5 {
				p++
			}
		} else {
			run = 1
		}
	}
	w := 0 // last 11 modules of the line padded with 4 light modules at each end
	for i := 0; i < len(line)+8; i++ {
		w <<= 1
		if j := i - 4; j >= 0 && j < len(line) && line[j] {
			w |= 1
		}
		w &= 0x7FF
		if i >= 10 && (w == 0b10111010000 || w == 0b00001011101) {
			p += 40
		}
	}
	return p
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
