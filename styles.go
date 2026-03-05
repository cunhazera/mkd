package main

import "github.com/charmbracelet/lipgloss"

const (
	headerHeight = 2
	footerHeight = 2

	matchHL   = "\x1b[48;5;220m\x1b[38;5;232m" // yellow bg, dark fg
	currentHL = "\x1b[48;5;208m\x1b[38;5;232m" // orange bg, dark fg
	ansiReset = "\x1b[0m"

	codeBlockBg     = "235"
	codeBlockBgANSI = "\x1b[48;5;235m"
)

// glamourTheme is a custom glamour theme that simulates heading size hierarchy
// through the 5 terminal levers: background, spacing, bold, color brightness,
// and full-width decorators — since terminals cannot change font size.
var glamourTheme = []byte(`{
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

var (
	inlineCodeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("147")).Background(lipgloss.Color("236"))
	inlineBoldStyle = lipgloss.NewStyle().Bold(true)
	inlineEmStyle   = lipgloss.NewStyle().Italic(true)

	linkTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("87")).
			Bold(true).
			Underline(true)

	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")).Padding(0, 1)
	tableCellStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)
)
