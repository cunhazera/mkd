package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type model struct {
	viewport      viewport.Model
	content       string
	filename      string
	ready         bool
	searching   bool   // user is typing a query
	searchInput string // query being typed
	searchQuery string // last confirmed query
	matches     []int  // line indices containing matches
	matchIdx    int    // current position in matches
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		verticalSpace := headerHeight + footerHeight
		if !m.ready {
			m.viewport = viewport.New(
				viewport.WithWidth(msg.Width),
				viewport.WithHeight(msg.Height-verticalSpace),
			)
			m.viewport.KeyMap = viewport.KeyMap{}
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalSpace)
		}
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelDown:
			m.viewport.ScrollDown(3)
		case tea.MouseWheelUp:
			m.viewport.ScrollUp(3)
		}
		return m, nil

	case tea.KeyPressMsg:
		// Search input mode — capture all keys for the query
		if m.searching {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.searching = false
				m.searchInput = ""
			case "enter":
				m.searching = false
				m.searchQuery = m.searchInput
				m.matches = findMatches(m.content, m.searchQuery)
				if len(m.matches) > 0 {
					m.matchIdx = 0
					line := m.matches[0]
					m.viewport.SetContent(applyHighlights(m.content, m.searchQuery, m.matches, line))
					m.viewport.SetYOffset(line)
				}
			case "backspace":
				if len(m.searchInput) > 0 {
					r := []rune(m.searchInput)
					m.searchInput = string(r[:len(r)-1])
				}
			case "space":
				m.searchInput += " "
			default:
				if len(msg.Text) > 0 {
					m.searchInput += msg.Text
				}
			}
			return m, nil
		}

		// Normal navigation mode
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.matches = nil
				m.matchIdx = 0
				m.viewport.SetContent(m.content)
			} else {
				return m, tea.Quit
			}
		case "up":
			m.viewport.ScrollUp(1)
		case "down":
			m.viewport.ScrollDown(1)
		case "pgup":
			m.viewport.ScrollUp(m.viewport.Height())
		case "pgdown":
			m.viewport.ScrollDown(m.viewport.Height())
		case "q", "Q":
			return m, tea.Quit
		case "/":
			m.searching = true
			m.searchInput = ""
		case "n":
			if len(m.matches) > 0 {
				m.matchIdx = (m.matchIdx + 1) % len(m.matches)
				line := m.matches[m.matchIdx]
				m.viewport.SetContent(applyHighlights(m.content, m.searchQuery, m.matches, line))
				m.viewport.SetYOffset(line)
			}
		case "N":
			if len(m.matches) > 0 {
				m.matchIdx = (m.matchIdx - 1 + len(m.matches)) % len(m.matches)
				line := m.matches[m.matchIdx]
				m.viewport.SetContent(applyHighlights(m.content, m.searchQuery, m.matches, line))
				m.viewport.SetYOffset(line)
			}
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "k":
			m.viewport.ScrollUp(1)
		case "j":
			m.viewport.ScrollDown(1)
		case "l":
			m.viewport.ScrollDown(15)
		case "p":
			m.viewport.ScrollUp(15)
		case "f":
			m.viewport.ScrollDown(m.viewport.Height())
		case "b":
			m.viewport.ScrollUp(m.viewport.Height())
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() tea.View {
	var content string
	if !m.ready {
		content = "\n  Rendering..."
	} else {
		title := titleStyle.Render(filepath.Base(m.filename))
		divider := dividerStyle.Render(strings.Repeat("─", m.viewport.Width()))

		var footer string
		switch {
		case m.searching:
			footer = footerStyle.Render(fmt.Sprintf("  /%s▌", m.searchInput))
		case m.searchQuery != "":
			var matchInfo string
			if len(m.matches) == 0 {
				matchInfo = "  no matches"
			} else {
				matchInfo = fmt.Sprintf("  [%d/%d]", m.matchIdx+1, len(m.matches))
			}
			footer = footerStyle.Render(fmt.Sprintf(
				"  /%s%s  •  n/N next/prev  •  esc clear  •  q quit",
				m.searchQuery, matchInfo,
			))
		default:
			scrollPct := int(m.viewport.ScrollPercent() * 100)
			footer = footerStyle.Render(fmt.Sprintf(
				"  ↑/↓ j/k  •  l/p page  •  g/G top/bottom  •  / search  •  q quit  •  %d%%",
				scrollPct,
			))
		}

		content = fmt.Sprintf("%s\n%s\n%s\n%s", title, divider, m.viewport.View(), footer)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
