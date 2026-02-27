package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

const (
	headerHeight = 2
	footerHeight = 2
)

// style is a custom glamour theme that simulates heading size hierarchy
// through the 5 terminal levers: background, spacing, bold, color brightness,
// and full-width decorators — since terminals cannot change font size.
var style = []byte(`{
  "document": {
    "block_prefix": "\n",
    "block_suffix": "\n",
    "color": "252",
    "margin": 2
  },
  "block_quote": {
    "indent": 1,
    "indent_token": "│ ",
    "color": "246",
    "italic": true
  },
  "paragraph": {},
  "list": { "level_indent": 2 },
  "heading": { "block_suffix": "\n" },
  "h1": {
    "block_prefix": "\n",
    "block_suffix": "\n\n",
    "prefix": " ", "suffix": " ",
    "color": "231", "background_color": "63", "bold": true
  },
  "h2": {
    "block_prefix": "\n",
    "prefix": "▌ ",
    "color": "75", "bold": true,
    "block_suffix": "\n"
  },
  "h3": {
    "prefix": "  ◆ ",
    "color": "85", "bold": true
  },
  "h4": {
    "prefix": "    ◇ ",
    "color": "183", "bold": true
  },
  "h5": {
    "prefix": "      · ",
    "color": "183"
  },
  "h6": {
    "prefix": "        ‣ ",
    "color": "243"
  },
  "text": {},
  "strikethrough": { "crossed_out": true },
  "emph": { "italic": true },
  "strong": { "bold": true },
  "hr": {
    "color": "240",
    "format": "\n──────────────────────────────────────────\n\n"
  },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "link": { "format": " " },
  "link_text": { "format": " " },
  "image": { "color": "212", "underline": true },
  "image_text": { "color": "243", "format": "Image: {{.text}} →" },
  "code": {
    "color": "147", "background_color": "236"
  },
  "code_block": {
    "color": "244",
    "margin": 2,
    "chroma": {
      "text":                  { "color": "#C4C4C4" },
      "error":                 { "color": "#F1F1F1", "background_color": "#F05B5B" },
      "comment":               { "color": "#676767" },
      "comment_preproc":       { "color": "#FF875F" },
      "keyword":               { "color": "#00AAFF" },
      "keyword_reserved":      { "color": "#FF5FD2" },
      "keyword_namespace":     { "color": "#FF5F87" },
      "keyword_type":          { "color": "#6E6ED8" },
      "operator":              { "color": "#EF8080" },
      "punctuation":           { "color": "#E8E8A8" },
      "name":                  { "color": "#C4C4C4" },
      "name_builtin":          { "color": "#FF8EC7" },
      "name_tag":              { "color": "#B083EA" },
      "name_attribute":        { "color": "#7A7AE6" },
      "name_class":            { "color": "#F1F1F1", "underline": true, "bold": true },
      "name_constant":         {},
      "name_decorator":        { "color": "#FFFF87" },
      "name_exception":        {},
      "name_function":         { "color": "#00D787" },
      "literal_number":        { "color": "#6EEFC0" },
      "literal_string":        { "color": "#C69669" },
      "literal_string_escape": { "color": "#AFFFD7" },
      "generic_deleted":       { "color": "#FD5B5B" },
      "generic_emph":          { "italic": true },
      "generic_inserted":      { "color": "#00D787" },
      "generic_strong":        { "bold": true },
      "generic_subheading":    { "color": "#777777" },
      "background":            {}
    }
  },
  "table": {},
  "definition_list": {},
  "definition_term": {},
  "definition_description": { "block_prefix": "\n🠶 " },
  "html_block": {},
  "html_span": {}
}`)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)

type model struct {
	viewport viewport.Model
	content  string
	filename string
	ready    bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "esc", "ctrl+c":
			return m, tea.Quit
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "l":
			m.viewport.LineDown(15)
		case "p":
			m.viewport.LineUp(15)
		}

	case tea.WindowSizeMsg:
		verticalSpace := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalSpace)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalSpace
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "\n  Rendering..."
	}

	title := titleStyle.Render(filepath.Base(m.filename))
	divider := dividerStyle.Render(repeatStr("─", m.viewport.Width))

	scrollPct := int(m.viewport.ScrollPercent() * 100)
	footer := footerStyle.Render(fmt.Sprintf(
		"  ↑/↓ j/k  •  l/p page up/down  •  f/b scroll full page  •  g/G top/bottom  •  q/esc quit  •  %d%%",
		scrollPct,
	))

	return fmt.Sprintf("%s\n%s\n%s\n%s", title, divider, m.viewport.View(), footer)
}

func repeatStr(s string, n int) string {
	result := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		result = append(result, s...)
	}
	return string(result)
}

// linkRe matches inline markdown links: [text](url)
var linkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// trailingPaddingRe strips trailing spaces and ANSI CSI sequences (e.g.
// \x1b[0m) from the end of a string. Glamour pads code block lines with
// spaces followed by a reset, so TrimRight alone is not enough.
var trailingPaddingRe = regexp.MustCompile(`(\x1b\[[0-9;]*m| )*$`)

var (
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
	inlineBoldRe = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	inlineEmRe   = regexp.MustCompile(`\*([^*]+)\*`)
)

var (
	inlineCodeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("147")).Background(lipgloss.Color("236"))
	inlineBoldStyle = lipgloss.NewStyle().Bold(true)
	inlineEmStyle   = lipgloss.NewStyle().Italic(true)
)

// processInline renders inline markdown (code spans, bold, italic, links)
// inside a plain string, used for table cell content.
func processInline(text string) string {
	// Links
	text = linkRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := linkRe.FindStringSubmatch(match)
		return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", sub[2], linkTextStyle.Render(sub[1]))
	})
	// Bold before italic to avoid consuming ** as two *
	text = inlineBoldRe.ReplaceAllStringFunc(text, func(match string) string {
		return inlineBoldStyle.Render(inlineBoldRe.FindStringSubmatch(match)[1])
	})
	text = inlineEmRe.ReplaceAllStringFunc(text, func(match string) string {
		return inlineEmStyle.Render(inlineEmRe.FindStringSubmatch(match)[1])
	})
	// Inline code
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		return inlineCodeStyle.Render(inlineCodeRe.FindStringSubmatch(match)[1])
	})
	return text
}

var linkTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("87")).
	Bold(true).
	Underline(true)

type linkRef struct{ text, url string }

// extractLinks replaces [text](url) with a placeholder and returns the
// mapping so we can restore OSC 8 hyperlinks after glamour renders.
func extractLinks(content string) (string, []linkRef) {
	var links []linkRef
	result := linkRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := linkRe.FindStringSubmatch(match)
		idx := len(links)
		links = append(links, linkRef{text: sub[1], url: sub[2]})
		return fmt.Sprintf("MDLINK%04d", idx)
	})
	return result, links
}

// restoreLinks substitutes placeholders back with OSC 8 hyperlink sequences.
// Format: ESC ] 8 ;; url ST  text  ESC ] 8 ;; ST
func restoreLinks(rendered string, links []linkRef) string {
	for i, l := range links {
		placeholder := fmt.Sprintf("MDLINK%04d", i)
		osc8 := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
			l.url, linkTextStyle.Render(l.text))
		rendered = strings.ReplaceAll(rendered, placeholder, osc8)
	}
	return rendered
}

// isTableRow returns true if line looks like a markdown table row (| ... |).
func isTableRow(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "|") && strings.HasSuffix(t, "|") && len(t) > 1
}

// isSeparatorRow returns true if line is a markdown table separator (|---|---|).
func isSeparatorRow(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "|") || !strings.HasSuffix(t, "|") {
		return false
	}
	for _, c := range strings.Trim(t, "|") {
		if c != '-' && c != ':' && c != '|' && c != ' ' {
			return false
		}
	}
	return true
}

// parseRow splits a markdown table row into trimmed, inline-rendered cell strings.
func parseRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = processInline(strings.TrimSpace(p))
	}
	return cells
}

// renderCompactTable renders markdown table lines into a compact lipgloss table
// with auto-sized columns (no forced full-terminal-width stretching).
func renderCompactTable(lines []string) string {
	if len(lines) < 2 {
		return strings.Join(lines, "\n")
	}

	headers := parseRow(lines[0])
	// lines[1] is the separator row — skip it

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)

	t := ltable.New().
		Headers(headers...).
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(false).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == ltable.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	for _, line := range lines[2:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		t.Row(parseRow(line)...)
	}

	// Indent 2 spaces to match glamour's document margin.
	var sb strings.Builder
	sb.WriteString("\n")
	for _, l := range strings.Split(t.String(), "\n") {
		sb.WriteString("  " + l + "\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

// extractTables removes markdown tables from content, replaces each with a
// placeholder, and returns the pre-rendered compact versions.
func extractTables(content string) (string, []string) {
	var tables []string
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		if isTableRow(lines[i]) && i+1 < len(lines) && isSeparatorRow(lines[i+1]) {
			tableLines := []string{lines[i], lines[i+1]}
			i += 2
			for i < len(lines) && isTableRow(lines[i]) {
				tableLines = append(tableLines, lines[i])
				i++
			}
			idx := len(tables)
			tables = append(tables, renderCompactTable(tableLines))
			result = append(result, fmt.Sprintf("MDTABLE%04d", idx))
			i-- // the outer loop will i++ past this
		} else {
			result = append(result, lines[i])
		}
	}
	return strings.Join(result, "\n"), tables
}

// restoreTables substitutes MDTABLE placeholders with their pre-rendered tables.
func restoreTables(rendered string, tables []string) string {
	for i, t := range tables {
		rendered = strings.ReplaceAll(rendered, fmt.Sprintf("MDTABLE%04d", i), t)
	}
	return rendered
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 {
		return 80
	}
	return w
}

const (
	codeBlockBg     = "235"
	codeBlockBgANSI = "\x1b[48;5;235m"
	ansiReset       = "\x1b[0m"
)

// padWithBackground applies a solid background to a single rendered line.
// Chroma emits per-token \x1b[0m resets that clear any background set at line
// start, so we replace every reset with reset+background-restore, then wrap
// the whole line in bg-start … padding … final-reset.
func padWithBackground(line string, width int) string {
	vw := xansi.StringWidth(line)
	line = strings.ReplaceAll(line, ansiReset, ansiReset+codeBlockBgANSI)
	padding := ""
	if vw < width {
		padding = strings.Repeat(" ", width-vw)
	}
	return codeBlockBgANSI + line + padding + ansiReset
}

// renderCodeBlock renders a single fenced code block with glamour for syntax
// highlighting, then pads every line to `width` with a background colour,
// producing a solid panel that visually separates code from regular text.
func renderCodeBlock(lang, code string, width int) (string, error) {
	src := "```" + lang + "\n" + code + "\n```\n"
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	rendered, err := r.Render(src)
	if err != nil {
		return "", err
	}

	// Collect non-empty lines from the glamour output, strip surrounding blank
	// lines that glamour adds, then wrap every content line with the background.
	rawLines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")

	// Trim leading/trailing blank lines produced by glamour's block margins.
	start, end := 0, len(rawLines)-1
	for start <= end && strings.TrimSpace(rawLines[start]) == "" {
		start++
	}
	for end >= start && strings.TrimSpace(rawLines[end]) == "" {
		end--
	}
	contentLines := rawLines[start : end+1]

	// Determine panel width from the original source lines before glamour
	// renders them. Glamour indents code content by document.margin(2) +
	// code_block.margin(2) = 4 spaces, so we add that back.
	const glamourIndent = 4
	panelWidth := 0
	for _, codeLine := range strings.Split(code, "\n") {
		if w := xansi.StringWidth(codeLine) + glamourIndent; w > panelWidth {
			panelWidth = w
		}
	}

	var sb strings.Builder
	sb.WriteString("\n")
	for _, line := range contentLines {
		// Strip trailing spaces and ANSI resets glamour appends to fill the
		// word-wrap width (e.g. "content     \x1b[0m"). TrimRight alone misses
		// this because the string ends with the escape sequence, not a space.
		line = trailingPaddingRe.ReplaceAllString(line, "")
		sb.WriteString(padWithBackground(line, panelWidth) + "\n")
	}
	sb.WriteString("\n")
	return sb.String(), nil
}

// extractCodeBlocks removes fenced code blocks from content, renders each one
// immediately (with background padding), and returns the pre-rendered blocks.
func extractCodeBlocks(content string, width int) (string, []string) {
	var blocks []string
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	var fence []string
	inFence := false
	lang := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inFence && strings.HasPrefix(trimmed, "```") {
			inFence = true
			lang = strings.TrimPrefix(trimmed, "```")
			if lang == "" {
				lang = "go"
			}
			fence = append(fence, line)
		} else if inFence && strings.HasPrefix(trimmed, "```") {
			fence = append(fence, line)

			code := strings.Join(fence[1:len(fence)-1], "\n")
			rendered, err := renderCodeBlock(lang, code, width)
			if err != nil {
				rendered = strings.Join(fence, "\n") + "\n"
			}

			idx := len(blocks)
			blocks = append(blocks, rendered)
			result = append(result, fmt.Sprintf("MDFENCE%04d", idx))
			fence = nil
			inFence = false
			lang = ""
		} else if inFence {
			fence = append(fence, line)
		} else {
			result = append(result, line)
		}
	}
	// Unclosed fence — pass through untouched.
	result = append(result, fence...)

	return strings.Join(result, "\n"), blocks
}

func restoreCodeBlocks(content string, blocks []string) string {
	for i, b := range blocks {
		content = strings.ReplaceAll(content, fmt.Sprintf("MDFENCE%04d", i), b)
	}
	return content
}

func renderMarkdown(content string, width int) (string, error) {
	// Pre-render code blocks with background padding before anything else
	// touches the content. extractTables and extractLinks never see them.
	content, codeBlocks := extractCodeBlocks(content, width)
	content, tables := extractTables(content)
	processed, links := extractLinks(content)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}

	rendered, err := renderer.Render(processed)
	if err != nil {
		return "", err
	}

	rendered = restoreLinks(rendered, links)
	rendered = restoreTables(rendered, tables)
	rendered = restoreCodeBlocks(rendered, codeBlocks)
	return rendered, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: md <file.md>")
		os.Exit(1)
	}

	filename := os.Args[1]

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Render at actual terminal width so word-wrap matches the viewport.
	// Subtract 4 to account for the document margin (2 each side).
	width := termWidth() - 4

	rendered, err := renderMarkdown(string(data), width)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering markdown: %v\n", err)
		os.Exit(1)
	}

	m := model{
		content:  rendered,
		filename: filename,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
