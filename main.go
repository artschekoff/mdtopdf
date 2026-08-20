package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

//go:embed github-light.css github-dark.css
var themes embed.FS

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

// ponytail: one template, no layout engine.
var tmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><style>
{{.CSS}}
@page { margin: 0; size: A4; }
html, body { background-color: {{.Bg}}; }
.markdown-body {
  font-size: 14px;
  line-height: 1.6;
  padding: 20mm;
  margin: 0;
  min-height: 100vh;
  background-color: {{.Bg}};
}
.markdown-body pre { background-color: {{.PreBg}} !important; }
.markdown-body pre, .markdown-body code { tab-size: 4; -moz-tab-size: 4; }
</style></head><body class="markdown-body">{{.Body}}</body></html>`))

func main() {
	dark := flag.Bool("dark", false, "GitHub dark theme")
	flag.Bool("light", false, "GitHub light theme (default)")
	out := flag.String("o", "", "output pdf path (default: <input>.pdf in cwd)")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: mdtopdf [--light|--dark] [-o out.pdf] <file.md>")
		flag.PrintDefaults()
	}
	// ponytail: Go's flag pkg stops at the first positional; hoist flags first
	// so `mdtopdf file.md --dark` works like the old shell script did.
	flag.CommandLine.Parse(reorder(os.Args[1:]))

	if *showVer {
		fmt.Println("mdtopdf", version)
		return
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	input, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fail(err)
	}
	src, err := os.ReadFile(input)
	if err != nil {
		fail(err)
	}

	theme, bg, preBg, chromaStyle := "light", "#ffffff", "#f6f8fa", "github"
	if *dark {
		theme, bg, preBg, chromaStyle = "dark", "#0d1117", "#161b22", "github-dark"
	}
	css, err := themes.ReadFile("github-" + theme + ".css")
	if err != nil {
		fail(err)
	}

	body, err := renderMarkdown(src, chromaStyle)
	if err != nil {
		fail(err)
	}

	var html bytes.Buffer
	err = tmpl.Execute(&html, map[string]any{
		"CSS":   template.CSS(css),
		"Bg":    bg,
		"PreBg": preBg,
		"Body":  template.HTML(body),
	})
	if err != nil {
		fail(err)
	}

	pdf, err := renderPDF(html.String(), filepath.Dir(input))
	if err != nil {
		fail(err)
	}

	outPath := *out
	if outPath == "" {
		name := filepath.Base(input)
		outPath = strings.TrimSuffix(name, filepath.Ext(name)) + ".pdf"
	}
	outPath, err = filepath.Abs(outPath)
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(outPath, pdf, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("Created: %s (%s theme)\n", outPath, theme)
}

// reorder moves non-flag arguments to the end.
func reorder(args []string) []string {
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i+1:]...)
			return append(flags, rest...)
		case strings.HasPrefix(a, "-"):
			flags = append(flags, a)
			// -o takes a separate value unless written as -o=x
			if (a == "-o" || a == "--o") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		default:
			rest = append(rest, a)
		}
	}
	return append(flags, rest...)
}

func renderMarkdown(src []byte, style string) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(highlighting.WithStyle(style)),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderPDF prints html through headless Chrome. baseDir is where a temp file is
// written so relative image paths in the markdown still resolve.
func renderPDF(html, baseDir string) ([]byte, error) {
	f, err := os.CreateTemp(baseDir, ".mdtopdf-*.html")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(html); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Flag("allow-file-access-from-files", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pdf []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate("file://"+f.Name()),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var e error
			pdf, _, e = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).WithPaperHeight(11.69). // A4 inches
				WithMarginTop(0).WithMarginBottom(0).
				WithMarginLeft(0).WithMarginRight(0).
				Do(ctx)
			return e
		}),
	)
	if err != nil {
		return nil, err
	}
	return pdf, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "mdtopdf:", err)
	os.Exit(1)
}
