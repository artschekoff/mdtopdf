package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The gutter has regressed three times, each time invisible to a test that only
// checked the PDF header. This renders a real multi-page document through Chrome
// and measures where text actually lands on every page.
func TestEveryPageKeepsItsGutter(t *testing.T) {
	var md strings.Builder
	md.WriteString("# Pagination\n\n")
	for i := 1; i <= 140; i++ {
		fmt.Fprintf(&md, "Paragraph %d: filler text long enough to push this document "+
			"across several page boundaries.\n\n", i)
	}

	body, err := renderMarkdown([]byte(md.String()), "github-dark")
	if err != nil {
		t.Fatal(err)
	}
	html, err := buildHTML(nil, "#0d1117", "#161b22", gutterHeight, body)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := renderPDF(html, t.TempDir())
	if err != nil {
		t.Skipf("cannot drive Chrome here: %v", err)
	}

	pages := firstBaselines(t, pdf)
	if len(pages) < 3 {
		t.Fatalf("wanted a multi-page document to test, got %d page(s)", len(pages))
	}
	const minGutter = 15.0 // mm; the configured gutter is 20mm plus block margins
	for i, topMM := range pages {
		if topMM < minGutter {
			t.Errorf("page %d starts %.1fmm from the top, want >= %.0fmm — "+
				"the gutter stopped repeating", i+1, topMM, minGutter)
		}
	}
}

var (
	reObj   = regexp.MustCompile(`(?s)(\d+)\s+0\s+obj(.*?)endobj`)
	rePage  = regexp.MustCompile(`/Type\s*/Page[^s]`)
	reCont  = regexp.MustCompile(`/Contents\s+(\d+)\s+0\s+R`)
	reFlip  = regexp.MustCompile(`([0-9.]+) 0 0 -[0-9.]+ 0 [0-9.]+ cm`)
	reMoves = regexp.MustCompile(`([0-9.]+) 0 0 [0-9.]+ 0 (-?[0-9.]+) cm|1 0 0 -1 [0-9.-]+ (-?[0-9.]+) Tm`)
)

// firstBaselines returns, per page, how far the topmost text baseline sits from
// the top edge in millimetres. Chrome nests two transforms: an outer flip that
// scales device units and inverts the y axis, and a per-page translate. Compose
// them rather than assuming a fixed page stride.
func firstBaselines(t *testing.T, pdf []byte) []float64 {
	t.Helper()
	objs := map[string][]byte{}
	for _, m := range reObj.FindAllSubmatch(pdf, -1) {
		objs[string(m[1])] = m[2]
	}

	var out []float64
	for _, m := range reObj.FindAllSubmatch(pdf, -1) {
		body := m[2]
		if !rePage.Match(body) {
			continue
		}
		c := reCont.FindSubmatch(body)
		if c == nil {
			continue
		}
		stream := inflate(objs[string(c[1])])
		if stream == nil {
			continue
		}

		flip := reFlip.FindSubmatch(stream)
		if flip == nil {
			continue
		}
		outer := mustFloat(t, string(flip[1]))

		var ty, inner float64
		top := -1.0
		for _, mm := range reMoves.FindAllSubmatch(stream, -1) {
			if len(mm[1]) > 0 { // a translate: remember this page's scale and offset
				inner = mustFloat(t, string(mm[1]))
				ty = mustFloat(t, string(mm[2]))
				continue
			}
			// a text matrix; recover its offset from the page top
			pt := outer * (inner*mustFloat(t, string(mm[3])) + ty)
			if top < 0 || pt < top {
				top = pt
			}
		}
		if top >= 0 {
			out = append(out, top*25.4/72.0)
		}
	}
	return out
}

func inflate(obj []byte) []byte {
	s := bytes.Index(obj, []byte("stream"))
	e := bytes.Index(obj, []byte("endstream"))
	if s < 0 || e < 0 {
		return nil
	}
	r, err := zlib.NewReader(bytes.NewReader(bytes.Trim(obj[s+6:e], "\r\n")))
	if err != nil {
		return nil
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	return b
}

func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("bad number %q: %v", s, err)
	}
	return f
}
