# Release Notes

> Rendered by **mdtopdf** — GitHub styling, straight to A4.

## Highlights

1. Single static binary, no `node_modules`
2. GFM tables, task lists, footnotes and strikethrough
3. Syntax highlighting for ~200 languages via chroma

## Configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--light` | bool | `true` | GitHub light theme |
| `--dark` | bool | `false` | GitHub dark theme |
| `-o` | string | `<input>.pdf` | Output path |

## Code

```go
func renderMarkdown(src []byte, style string) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(highlighting.WithStyle(style)),
		),
	)
	var buf bytes.Buffer
	return buf.Bytes(), md.Convert(src, &buf)
}
```

```sh
mdtopdf --dark notes.md -o ~/Desktop/notes.pdf
```

## Checklist

- [x] Markdown parsed with goldmark
- [x] Styled with GitHub's stylesheet
- [ ] ~~Ship a browser with it~~ never

Inline `code`, a [link](https://github.com/artschekoff/mdtopdf), *emphasis*,
and a horizontal rule:

---

Everything above is real output — this page *is* `testdata/sample.md`.
