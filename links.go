package main

import (
	"fmt"
	"regexp"
	"strings"
)

// linkRe matches inline markdown links: [text](url)
var linkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

var (
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
	inlineBoldRe = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	inlineEmRe   = regexp.MustCompile(`\*([^*]+)\*`)
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

// autoLinkRe matches bare URLs (http:// or https://) not already inside a
// markdown link. Trailing punctuation that is unlikely to be part of the URL
// is excluded by the character class.
var autoLinkRe = regexp.MustCompile(`https?://\S+`)

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

// extractAutoLinks replaces bare URLs with placeholders. Must run after
// extractLinks so that URLs already inside [text](url) are not double-processed.
func extractAutoLinks(content string) (string, []linkRef) {
	var links []linkRef
	result := autoLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		url := strings.TrimRight(match, ").,")
		idx := len(links)
		links = append(links, linkRef{text: url, url: url})
		return fmt.Sprintf("MDAUTO%04d", idx)
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

// restoreAutoLinks substitutes bare-URL placeholders back with OSC 8 sequences.
func restoreAutoLinks(rendered string, links []linkRef) string {
	for i, l := range links {
		placeholder := fmt.Sprintf("MDAUTO%04d", i)
		osc8 := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
			l.url, linkTextStyle.Render(l.text))
		rendered = strings.ReplaceAll(rendered, placeholder, osc8)
	}
	return rendered
}
