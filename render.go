package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	xansi "github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

// trailingPaddingRe strips trailing spaces and ANSI CSI sequences (e.g.
// \x1b[0m) from the end of a string. Glamour pads code block lines with
// spaces followed by a reset, so TrimRight alone is not enough.
var trailingPaddingRe = regexp.MustCompile(`(\x1b\[[0-9;]*m| )*$`)

// ansiSGRRe matches any ANSI SGR escape sequence (\x1b[...m).
var ansiSGRRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 {
		return 80
	}
	return w
}

// padWithBackground applies a solid background to a single rendered line.
// Chroma emits per-token \x1b[0m resets that clear any background set at line
// start, so we replace every reset with reset+background-restore, then wrap
// the whole line in bg-start … padding … final-reset.
func padWithBackground(line string, width int) string {
	vw := xansi.StringWidth(line)
	// Re-inject the background after any SGR sequence that performs a full
	// reset. Chroma often emits combined sequences like \x1b[0;38;5;147m
	// (reset + colour in one shot) which the old exact \x1b[0m replacement
	// missed, leaving the background unset for the rest of that token.
	line = ansiSGRRe.ReplaceAllStringFunc(line, func(seq string) string {
		params := ansiSGRRe.FindStringSubmatch(seq)[1]
		for _, p := range strings.Split(params, ";") {
			if p == "0" || p == "" { // "0" = explicit reset, "" = \x1b[m bare reset
				return seq + codeBlockBgANSI
			}
		}
		return seq
	})
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
		glamour.WithStylesFromJSONBytes(glamourTheme),
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

	var sb strings.Builder
	sb.WriteString("\n")
	for _, line := range contentLines {
		// Strip trailing spaces and ANSI resets glamour appends to fill the
		// word-wrap width (e.g. "content     \x1b[0m"). TrimRight alone misses
		// this because the string ends with the escape sequence, not a space.
		line = trailingPaddingRe.ReplaceAllString(line, "")
		sb.WriteString(padWithBackground(line, termWidth()) + "\n")
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
		glamour.WithStylesFromJSONBytes(glamourTheme),
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
