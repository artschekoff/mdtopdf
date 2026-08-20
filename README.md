# mdtopdf

Markdown → PDF with GitHub styling. Single Go binary, no `node_modules`.

Renders Markdown with [goldmark](https://github.com/yuin/goldmark) (GFM: tables,
task lists, strikethrough, autolinks) plus [chroma](https://github.com/alecthomas/chroma)
syntax highlighting, wraps it in the embedded GitHub stylesheet, and prints it
through your locally installed Chrome via [chromedp](https://github.com/chromedp/chromedp).

## Install

```sh
make install          # -> ~/.local/bin/mdtopdf
make install PREFIX=/usr/local/bin
```

Requires Go 1.22+ to build and Google Chrome (or Chromium) at runtime.

## Usage

```sh
mdtopdf notes.md                  # -> ./notes.pdf, light theme
mdtopdf --dark notes.md           # GitHub dark theme
mdtopdf notes.md -o /tmp/out.pdf  # explicit output path
mdtopdf --version
```

| Flag | Description |
|---|---|
| `--light` | GitHub light theme (default) |
| `--dark` | GitHub dark theme |
| `-o PATH` | Output path (default: `<input>.pdf` in the current directory) |
| `--version` | Print version and exit |

Flags may appear before or after the file name.

Output is A4 with zero page margins; the 20mm inner padding comes from the
stylesheet, so backgrounds bleed to the page edge in dark mode. Relative image
paths resolve against the Markdown file's directory.

## Development

```sh
make build      # -> bin/mdtopdf
make test       # renders testdata/sample.md and checks the PDF header
make validate   # fmt + vet + test
make build-all  # darwin/linux, amd64/arm64
make release    # prompt for bump, tag, push, pack, gh release
```

## Credits

The bundled `github-light.css` / `github-dark.css` are GitHub's Markdown
stylesheet, as distributed by [github-markdown-css](https://github.com/sindresorhus/github-markdown-css) (MIT).

## License

MIT
