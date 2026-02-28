# mkd

A terminal Markdown viewer with syntax highlighting, clickable links, and a clean reading experience — built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

```
mkd README.mkd
```

---

## Features

- **Rich Markdown rendering** — headings, bold, italic, strikethrough, blockquotes, lists, task lists, and horizontal rules all rendered with proper visual hierarchy
- **Syntax highlighted code blocks** — fenced code blocks rendered with full syntax highlighting via [Chroma](https://github.com/alecthomas/chroma), with a sensible default when no language is specified
- **Inline code styling** — `code spans` rendered in a distinct lavender colour with a dark background
- **Compact tables** — tables sized to their content instead of being stretched to fill the terminal width; inline code, bold and links inside cells are fully rendered
- **Clickable links** — links open in the browser on click using [OSC 8](https://github.com/nicm/latency/blob/master/osc8.mkd) terminal hyperlinks (supported by iTerm2, Kitty, GNOME Terminal, Windows Terminal, and others)
- **Scrollable viewport** — the entire document fits in a scrollable pane that adapts to any terminal size
- **Native mouse selection** — click and drag to select and copy any text with your terminal's native copy shortcut

---

## Install

**Requirements:** Go 1.21+

```bash
git clone https://github.com/gabrielcunha/mkd
cd mkd
go install .
```

The binary is installed to `~/go/bin/mkd`. Make sure `~/go/bin` is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Or build locally without installing:

```bash
go build -o mkd .
./mkd README.mkd
```

---

## Usage

```
mkd <file.mkd>
```

```bash
mkd README.mkd
mkd ~/notes/todo.mkd
mkd docs/api-reference.mkd
```

---

## Keyboard shortcuts

| Key | Action |
|---|---|
| `↑` / `↓` | Scroll one line up / down |
| `j` / `k` | Scroll one line down / up (vim-style) |
| `l` / `p` | Scroll 15 lines down / up |
| `f` / `b` | Scroll one full page down / up |
| `g` | Go to top |
| `G` | Go to bottom |
| `q` / `esc` / `ctrl+c` | Quit |

---

## Built with

| Library | Role |
|---|---|
| [Bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework and event loop |
| [Bubbles](https://github.com/charmbracelet/bubbles) | Scrollable viewport component |
| [Glamour](https://github.com/charmbracelet/glamour) | Markdown to ANSI renderer |
| [Lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and layout |
| [Chroma](https://github.com/alecthomas/chroma) | Syntax highlighting (via Glamour) |

---

### Code block
```
func repeatStr(s string, n int) string {
	result := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		result = append(result, s...)
	}
	return string(result)
}
```

## License

MIT
