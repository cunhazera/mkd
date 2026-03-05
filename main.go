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

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
