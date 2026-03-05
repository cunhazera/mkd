package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
)

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
				return tableHeaderStyle
			}
			return tableCellStyle
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
