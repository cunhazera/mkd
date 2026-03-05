package main

import (
	"strings"
	"unicode/utf8"

	xansi "github.com/charmbracelet/x/ansi"
)

// csiHasReset reports whether an SGR sequence contains a full reset ("0" or "").
func csiHasReset(seq string) bool {
	if len(seq) < 3 || seq[len(seq)-1] != 'm' {
		return false
	}
	for _, p := range strings.Split(seq[2:len(seq)-1], ";") {
		if p == "0" || p == "" {
			return true
		}
	}
	return false
}

// stripMarkdownQuery strips common markdown syntax from a search query so
// that e.g. "# Heading" and "`code`" match their rendered equivalents.
func stripMarkdownQuery(q string) string {
	q = strings.TrimLeft(q, "#")
	q = strings.Trim(q, "`")
	return strings.TrimSpace(q)
}

// findMatches returns the line indices (0-based) that contain query.
// It also tries the query with markdown syntax stripped, so searches like
// "# Heading" or "`code`" find their rendered equivalents.
func findMatches(content, query string) []int {
	if query == "" {
		return nil
	}
	alt := stripMarkdownQuery(query)

	lines := strings.Split(content, "\n")
	seen := make(map[int]bool)
	var result []int
	for i, line := range lines {
		plain := xansi.Strip(line)
		if strings.Contains(plain, query) || (alt != query && alt != "" && strings.Contains(plain, alt)) {
			if !seen[i] {
				seen[i] = true
				result = append(result, i)
			}
		}
	}
	return result
}

// highlightLine injects highlight ANSI codes around every occurrence of query
// in an ANSI-coded line. isCurrent uses a brighter colour for the active match.
func highlightLine(line, query string, isCurrent bool) string {
	plain := xansi.Strip(line)
	plainRunes := []rune(plain)
	queryRunes := []rune(query)
	qLen := len(queryRunes)

	if len(plainRunes) < qLen || !strings.Contains(plain, query) {
		return line
	}

	bg := matchHL
	if isCurrent {
		bg = currentHL
	}

	// Collect match spans as rune offsets in plain text.
	type span struct{ start, end int }
	var spans []span
	for i := 0; i <= len(plainRunes)-qLen; {
		match := true
		for j := 0; j < qLen; j++ {
			if plainRunes[i+j] != queryRunes[j] {
				match = false
				break
			}
		}
		if match {
			spans = append(spans, span{i, i + qLen})
			i += qLen
		} else {
			i++
		}
	}
	if len(spans) == 0 {
		return line
	}

	// Build a sorted list of (runePos, text) injections.
	type inj struct {
		pos  int
		text string
	}
	var injections []inj
	for _, s := range spans {
		injections = append(injections, inj{s.start, bg}, inj{s.end, ansiReset})
	}

	// Walk the ANSI string, tracking visible rune position, and inject.
	var result strings.Builder
	visRune, injIdx, i := 0, 0, 0
	inHighlight := false
	for i < len(line) {
		for injIdx < len(injections) && injections[injIdx].pos == visRune {
			txt := injections[injIdx].text
			result.WriteString(txt)
			if txt == bg {
				inHighlight = true
			} else if txt == ansiReset {
				inHighlight = false
			}
			injIdx++
		}
		if line[i] == '\x1b' && i+1 < len(line) {
			switch line[i+1] {
			case '[': // CSI — scan to final byte (0x40–0x7E)
				j := i + 2
				for j < len(line) && (line[j] < 0x40 || line[j] > 0x7E) {
					j++
				}
				if j < len(line) {
					j++
				}
				seq := line[i:j]
				if inHighlight {
					if seq == codeBlockBgANSI {
						// suppress code-block background re-injection, keep highlight
						result.WriteString(bg)
					} else {
						result.WriteString(seq)
						if csiHasReset(seq) {
							// re-assert highlight after any reset inside the span
							result.WriteString(bg)
						}
					}
				} else {
					result.WriteString(seq)
				}
				i = j
			case ']': // OSC — scan to ST (ESC \) or BEL
				j := i + 2
				for j < len(line) {
					if line[j] == '\x07' {
						j++
						break
					}
					if line[j] == '\x1b' && j+1 < len(line) && line[j+1] == '\\' {
						j += 2
						break
					}
					j++
				}
				result.WriteString(line[i:j])
				i = j
			default:
				result.WriteByte(line[i])
				result.WriteByte(line[i+1])
				i += 2
			}
		} else {
			r, size := utf8.DecodeRuneInString(line[i:])
			result.WriteRune(r)
			i += size
			visRune++
		}
	}
	for injIdx < len(injections) {
		result.WriteString(injections[injIdx].text)
		injIdx++
	}
	return result.String()
}

// applyHighlights returns content with all match lines highlighted.
// currentMatchLine is the line index of the active match (orange vs yellow).
func applyHighlights(content, query string, matches []int, currentMatchLine int) string {
	if query == "" || len(matches) == 0 {
		return content
	}
	matchSet := make(map[int]bool, len(matches))
	for _, m := range matches {
		matchSet[m] = true
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matchSet[i] {
			lines[i] = highlightLine(line, query, i == currentMatchLine)
		}
	}
	return strings.Join(lines, "\n")
}
