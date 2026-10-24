// Command favicon draws the 竜熊 favicon (white glyphs on an indigo rounded
// tile) as favicon.svg, favicon.ico (16, 32, 48), and a full-bleed
// apple-touch-icon.png. The glyphs are converted to paths, so the SVG does
// not depend on installed fonts. Layout: a 100x100 box, 8 units of padding,
// two 42-unit cells, glyphs at 1.02x the cell size centered on their em box.
//
// To regenerate, from this directory (colors are the constants below):
//
//	curl -s -A curl "https://fonts.googleapis.com/css2?family=Noto+Sans+JP:wght@700&text=%E7%AB%9C%E7%86%8A" \
//	  | grep -oE 'https://[^)]+' | xargs curl -s -o /tmp/noto-sans-jp-700.ttf
//	go run . /tmp/noto-sans-jp-700.ttf ../../static
//
// Noto Sans JP is under the SIL Open Font License; only rendered outlines
// end up in the repo.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

const (
	tileHex  = "#4b2a9a"
	glyphHex = "#ffffff"
	pad      = 8.0
	cell     = (100 - 2*pad) / 2
	fontSize = cell * 1.02
)

var (
	tileRGBA  = color.RGBA{0x4b, 0x2a, 0x9a, 0xff}
	glyphRGBA = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// op is one path command in icon units (0..100, y down).
type op struct {
	kind byte // M, L, Q, C, Z
	pts  []float64
}

func glyph(f *sfnt.Font, r rune, cx, cy float64) []op {
	var b sfnt.Buffer
	upem := fixed.I(int(f.UnitsPerEm()))
	idx, err := f.GlyphIndex(&b, r)
	if err != nil || idx == 0 {
		log.Fatalf("no glyph for %q", r)
	}
	segs, err := f.LoadGlyph(&b, idx, upem, nil)
	if err != nil {
		log.Fatal(err)
	}
	m, err := f.Metrics(&b, upem, font.HintingNone)
	if err != nil {
		log.Fatal(err)
	}
	adv, err := f.GlyphAdvance(&b, idx, upem, font.HintingNone)
	if err != nil {
		log.Fatal(err)
	}
	fu := func(v fixed.Int26_6) float64 { return float64(v) / 64 }
	// SVG's "central" baseline sits midway between ascent and descent.
	central := (fu(m.Ascent) - fu(m.Descent)) / 2
	s := fontSize / float64(f.UnitsPerEm())
	pt := func(p fixed.Point26_6) (float64, float64) {
		return cx + (fu(p.X)-fu(adv)/2)*s, cy + (fu(p.Y)+central)*s
	}

	var ops []op
	for _, seg := range segs {
		var o op
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			if len(ops) > 0 {
				ops = append(ops, op{kind: 'Z'})
			}
			o.kind = 'M'
		case sfnt.SegmentOpLineTo:
			o.kind = 'L'
		case sfnt.SegmentOpQuadTo:
			o.kind = 'Q'
		case sfnt.SegmentOpCubeTo:
			o.kind = 'C'
		}
		n := map[byte]int{'M': 1, 'L': 1, 'Q': 2, 'C': 3}[o.kind]
		for i := range n {
			x, y := pt(seg.Args[i])
			o.pts = append(o.pts, x, y)
		}
		ops = append(ops, o)
	}
	return append(ops, op{kind: 'Z'})
}

// roundRect returns a rounded rectangle path; r = 0 gives square corners.
func roundRect(r float64) []op {
	if r == 0 {
		return []op{{'M', []float64{0, 0}}, {'L', []float64{100, 0}}, {'L', []float64{100, 100}}, {'L', []float64{0, 100}}, {'Z', nil}}
	}
	k := r * 0.5523 // cubic approximation of a quarter circle
	return []op{
		{'M', []float64{r, 0}}, {'L', []float64{100 - r, 0}},
		{'C', []float64{100 - r + k, 0, 100, r - k, 100, r}}, {'L', []float64{100, 100 - r}},
		{'C', []float64{100, 100 - r + k, 100 - r + k, 100, 100 - r, 100}}, {'L', []float64{r, 100}},
		{'C', []float64{r - k, 100, 0, 100 - r + k, 0, 100 - r}}, {'L', []float64{0, r}},
		{'C', []float64{0, r - k, r - k, 0, r, 0}}, {'Z', nil},
	}
}

func pathData(ops []op) string {
	var sb strings.Builder
	for _, o := range ops {
		sb.WriteByte(o.kind)
		for i, v := range o.pts {
			if i > 0 {
				sb.WriteByte(' ')
			}
			fmt.Fprintf(&sb, "%.2f", v)
		}
	}
	return strings.ReplaceAll(sb.String(), ".00", "")
}

func rasterize(ops []op, size int, c color.Color, dst *image.RGBA) {
	r := vector.NewRasterizer(size, size)
	k := float32(size) / 100
	for _, o := range ops {
		p := make([]float32, len(o.pts))
		for i, v := range o.pts {
			p[i] = float32(v) * k
		}
		switch o.kind {
		case 'M':
			r.MoveTo(p[0], p[1])
		case 'L':
			r.LineTo(p[0], p[1])
		case 'Q':
			r.QuadTo(p[0], p[1], p[2], p[3])
		case 'C':
			r.CubeTo(p[0], p[1], p[2], p[3], p[4], p[5])
		case 'Z':
			r.ClosePath()
		}
	}
	r.Draw(dst, dst.Bounds(), image.NewUniform(c), image.Point{})
}

func render(tile, glyphs []op, size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	rasterize(tile, size, tileRGBA, img)
	rasterize(glyphs, size, glyphRGBA, img)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

// ico packs PNG images into an .ico file, which modern readers accept.
func ico(sizes []int, pngs [][]byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, [3]uint16{0, 1, uint16(len(pngs))})
	offset := 6 + 16*len(pngs)
	for i, data := range pngs {
		buf.Write([]byte{byte(sizes[i]), byte(sizes[i]), 0, 0})
		binary.Write(&buf, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&buf, binary.LittleEndian, [2]uint32{uint32(len(data)), uint32(offset)})
		offset += len(data)
	}
	for _, data := range pngs {
		buf.Write(data)
	}
	return buf.Bytes()
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: favgen FONT.ttf OUTDIR")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		log.Fatal(err)
	}
	out := os.Args[2]
	glyphs := append(glyph(f, '竜', pad+cell/2, 50), glyph(f, '熊', pad+1.5*cell, 50)...)
	tile := roundRect(22)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><title>竜熊</title>`+
		`<path fill="%s" d="%s"/><path fill="%s" d="%s"/></svg>`+"\n",
		tileHex, pathData(tile), glyphHex, pathData(glyphs))
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(out, name), b, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-22s %6d bytes\n", name, len(b))
	}
	write("favicon.svg", []byte(svg))

	sizes := []int{16, 32, 48}
	var pngs [][]byte
	for _, s := range sizes {
		pngs = append(pngs, render(tile, glyphs, s))
	}
	write("favicon.ico", ico(sizes, pngs))
	// iOS masks its own rounded corners, so the touch icon is full-bleed.
	write("apple-touch-icon.png", render(roundRect(0), glyphs, 180))
}
