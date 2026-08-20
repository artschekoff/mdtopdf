package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf16"
)

// Regression guard for the page-break margin bug.
//
// Vertical space only repeats on every sheet if it lives on @page. Padding on
// .markdown-body is applied once to the whole flow, so the first page gets a
// top gutter, the last page gets a bottom one, and every interior page break
// lands ~4mm from the paper edge. Measured before the fix: page 1 started
// 27.5mm from the top, pages 2+ started 4.2mm from the top.
func TestVerticalSpaceLivesOnPageRule(t *testing.T) {
	html, err := buildHTML([]byte("/* css */"), "#ffffff", "#f6f8fa", []byte("<p>hi</p>"))
	if err != nil {
		t.Fatal(err)
	}

	atPage := regexp.MustCompile(`@page\s*{([^}]*)}`).FindStringSubmatch(html)
	if atPage == nil {
		t.Fatal("no @page rule in the print stylesheet")
	}
	margin := regexp.MustCompile(`margin:\s*([^;]+);`).FindStringSubmatch(atPage[1])
	if margin == nil {
		t.Fatal("@page has no margin declaration")
	}
	top := strings.Fields(margin[1])[0]
	if top == "0" || strings.HasPrefix(top, "0 ") {
		t.Errorf("@page vertical margin is %q; interior page breaks will sit flush "+
			"against the paper edge", margin[1])
	}

	body := regexp.MustCompile(`\.markdown-body\s*{([^}]*)}`).FindStringSubmatch(html)
	if body == nil {
		t.Fatal("no .markdown-body rule")
	}
	if pad := regexp.MustCompile(`padding:\s*([^;]+);`).FindStringSubmatch(body[1]); pad != nil {
		if f := strings.Fields(pad[1]); f[0] != "0" {
			t.Errorf("padding %q puts vertical space on the body box, which does not "+
				"repeat across pages; move it to @page", pad[1])
		}
	}
}

// Non-UTF-8 input used to sail through unnoticed: goldmark passes invalid
// sequences along, the HTML declares utf-8, and Chrome renders every bad
// sequence as U+FFFD — pages of ◆ instead of an error.
func TestDecodeInput(t *testing.T) {
	const want = "# Заголовок\n\nПривет"

	utf16le := append([]byte{0xFF, 0xFE}, enc16(want, false)...)
	utf16be := append([]byte{0xFE, 0xFF}, enc16(want, true)...)

	for _, tc := range []struct {
		name string
		in   []byte
	}{
		{"plain utf-8", []byte(want)},
		{"utf-8 with BOM", append([]byte{0xEF, 0xBB, 0xBF}, want...)},
		{"utf-16le with BOM", utf16le},
		{"utf-16be with BOM", utf16be},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeInput(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}

	for _, tc := range []struct {
		name string
		in   []byte
	}{
		{"cp1251", []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}},
		{"binary", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x92, 0x8B}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeInput(tc.in); err == nil {
				t.Error("want an error for undecodable input, got nil")
			}
		})
	}
}

func enc16(s string, bigEndian bool) []byte {
	var out []byte
	for _, r := range utf16.Encode([]rune(s)) {
		hi, lo := byte(r>>8), byte(r)
		if bigEndian {
			out = append(out, hi, lo)
		} else {
			out = append(out, lo, hi)
		}
	}
	return out
}

// A wrong hint is worse than no hint: it sends you off converting a JPEG from
// an encoding it never had.
func TestDecodeInputHint(t *testing.T) {
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
		bytes.Repeat([]byte{0x00, 0x8B, 0x00, 0x00, 0xC3, 0x1F}, 40)...)
	_, err := decodeInput(jpeg)
	if err == nil {
		t.Fatal("want error for binary input")
	}
	if strings.Contains(err.Error(), "UTF-16") {
		t.Errorf("binary input misdiagnosed as UTF-16: %v", err)
	}

	// BOM-less UTF-16LE is *valid UTF-8* — NUL is a legal byte — so it slips
	// past utf8.Valid and renders as NUL-riddled nonsense. Detect and transcode.
	const text = "# Heading\n\nsome text here\n"
	var u16 []byte
	for _, c := range []byte(text) {
		u16 = append(u16, c, 0x00)
	}
	got, err := decodeInput(u16)
	if err != nil {
		t.Fatalf("BOM-less UTF-16 should transcode, got error: %v", err)
	}
	if string(got) != text {
		t.Errorf("got %q, want %q", got, text)
	}
}
