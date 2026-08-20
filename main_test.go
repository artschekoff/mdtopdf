package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf16"
)

// Regression guard for the page-break margin bug, round three.
//
// The invariant: blank space must repeat at the top and bottom of EVERY page,
// while the background still reaches the paper edge. Body padding cannot do it
// (applied once to the whole flow, so interior breaks sit flush against the
// edge) and @page margins cannot either (Chrome never paints that area, which
// leaves white bands in dark mode). The gutter therefore lives in thead/tfoot
// rows, which Chrome repeats on every page, with @page margin left at 0 so the
// canvas covers the full sheet.
func TestGutterRepeatsAndBackgroundBleeds(t *testing.T) {
	html, err := buildHTML([]byte("/* css */"), "#0d1117", "#161b22", "20mm", []byte("<p>hi</p>"))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"<thead>", "<tfoot>", `class="gutter"`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s: the gutter no longer repeats across pages", want)
		}
	}
	if strings.Count(html, `class="gutter"`) < 2 {
		t.Error("need a gutter in both thead and tfoot")
	}

	atPage := regexp.MustCompile(`@page\s*{([^}]*)}`).FindStringSubmatch(html)
	if atPage == nil {
		t.Fatal("no @page rule")
	}
	if m := regexp.MustCompile(`margin:\s*([^;]+);`).FindStringSubmatch(atPage[1]); m == nil || strings.Fields(m[1])[0] != "0" {
		t.Errorf("@page margin must stay 0 or Chrome leaves an unpainted band at the "+
			"sheet edge; got %v", atPage[1])
	}

	gut := regexp.MustCompile(`\.gutter\s*{[^}]*height:\s*([^;]+);`).FindStringSubmatch(html)
	if gut == nil || strings.HasPrefix(gut[1], "0") {
		t.Errorf("gutter height missing or zero: %v", gut)
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
