package main

import (
	"regexp"
	"strings"
	"testing"
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
