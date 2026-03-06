package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via -ldflags "-X main.version=x.y.z".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: mkd [--print|-v|--version] <file.md>")
		os.Exit(1)
	}

	if os.Args[1] == "-v" || os.Args[1] == "--version" {
		fmt.Println("mkd version", version)
		return
	}

	printMode := false
	filename := os.Args[1]
	if os.Args[1] == "--print" {
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: mkd --print <file.md>")
			os.Exit(1)
		}
		printMode = true
		filename = os.Args[2]
	}

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

	if printMode {
		fmt.Print(rendered)
		return
	}

	m := model{
		content:  rendered,
		filename: filename,
	}

	// WithMouseCellMotion enables Bubbletea's mouse parser. Init() then
	// immediately downgrades the terminal to ?1000h (basic buttons only),
	// which preserves native text selection without requiring Shift.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	// Bubbletea's cleanup disables ?1002h but not ?1000h; do it here.
	fmt.Print("\x1b[?1000l")
}
