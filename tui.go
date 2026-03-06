package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type scrollTickMsg struct{}

func scrollTick() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg {
		return scrollTickMsg{}
	})
}

// enableBasicMouse downgrades the terminal from cell-motion mouse mode
// (?1002h, captures drag) to basic button mode (?1000h, captures clicks and
// scroll wheel only). Drag events are not captured, so the terminal handles
// them natively — text selection works without holding Shift.
func enableBasicMouse() tea.Cmd {
	return func() tea.Msg {
		os.Stdout.WriteString("\x1b[?1002l\x1b[?1000h")
		return nil
	}
}

type model struct {
	viewport      viewport.Model
	content       string
	filename      string
	ready         bool
	searching     bool   // user is typing a query
	searchInput   string // query being typed
	searchQuery   string // last confirmed query
	matches       []int  // line indices containing matches
	matchIdx      int    // current position in matches
	scrollAcc     int    // pending mouse-wheel lines (positive=down, negative=up)
	scrollPending bool   // a scrollTick is already in flight
}

func (m model) Init() tea.Cmd {
	return enableBasicMouse()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		verticalSpace := headerHeight + footerHeight
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalSpace)
			m.viewport.KeyMap = viewport.KeyMap{}
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalSpace
		}
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelDown:
				if m.scrollAcc < 0 {
					m.scrollAcc = 0 // cancel pending up-scroll
				}
				m.scrollAcc += 3
			case tea.MouseButtonWheelUp:
				if m.scrollAcc > 0 {
					m.scrollAcc = 0 // cancel pending down-scroll
				}
				m.scrollAcc -= 3
			}
			if !m.scrollPending && m.scrollAcc != 0 {
				m.scrollPending = true
				return m, scrollTick()
			}
		}
		return m, nil

	case scrollTickMsg:
		m.scrollPending = false
		acc := m.scrollAcc
		m.scrollAcc = 0
		if acc > 0 {
			m.viewport.LineDown(acc)
		} else if acc < 0 {
			m.viewport.LineUp(-acc)
		}
		return m, nil

	case tea.KeyMsg:
		// Search input mode — capture all keys for the query
		if m.searching {
			switch msg.Type {
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEsc:
				m.searching = false
				m.searchInput = ""
			case tea.KeyEnter:
				m.searching = false
				m.searchQuery = m.searchInput
				m.matches = findMatches(m.content, m.searchQuery)
				if len(m.matches) > 0 {
					m.matchIdx = 0
					line := m.matches[0]
					m.viewport.SetContent(applyHighlights(m.content, m.searchQuery, m.matches, line))
					m.viewport.SetYOffset(line)
				}
			case tea.KeyBackspace:
				if len(m.searchInput) > 0 {
					r := []rune(m.searchInput)
					m.searchInput = string(r[:len(r)-1])
				}
			case tea.KeySpace:
				m.searchInput += " "
			case tea.KeyRunes:
				m.searchInput += string(msg.Runes)
			}
			return m, nil
		}

		// Normal navigation mode
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.matches = nil
				m.matchIdx = 0
				m.viewport.SetContent(m.content)
			} else {
				return m, tea.Quit
			}
		case tea.KeyUp:
			m.viewport.LineUp(1)
		case tea.KeyDown:
			m.viewport.LineDown(1)
		case tea.KeyPgUp:
			m.viewport.LineUp(m.viewport.Height)
		case tea.KeyPgDown:
			m.viewport.LineDown(m.viewport.Height)
		case tea.KeyRunes:
			switch string(msg.Runes) {
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
				m.viewport.LineUp(1)
			case "j":
				m.viewport.LineDown(1)
			case "l":
				m.viewport.LineDown(15)
			case "p":
				m.viewport.LineUp(15)
			case "f":
				m.viewport.LineDown(m.viewport.Height)
			case "b":
				m.viewport.LineUp(m.viewport.Height)
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "\n  Rendering..."
	}

	title := titleStyle.Render(filepath.Base(m.filename))
	divider := dividerStyle.Render(strings.Repeat("─", m.viewport.Width))

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

	return fmt.Sprintf("%s\n%s\n%s\n%s", title, divider, m.viewport.View(), footer)
}
