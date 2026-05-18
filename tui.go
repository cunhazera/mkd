package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type scrollTickMsg struct{}

func scrollTick() tea.Cmd {
	return tea.Tick(8*time.Millisecond, func(time.Time) tea.Msg {
		return scrollTickMsg{}
	})
}

// enableAltScroll enables alternate-scroll mode (?1007h). In alt-screen mode
// the terminal converts scroll-wheel events into cursor-up/down key sequences,
// so bubbletea receives them as KeyPressMsg "up"/"down" without any mouse
// reporting being active. No mouse capture means text selection and OSC 8
// hyperlink clicks are handled natively by the terminal emulator.
func enableAltScroll() tea.Cmd {
	return func() tea.Msg {
		os.Stdout.WriteString("\x1b[?1007h")
		return nil
	}
}

type model struct {
	viewport      viewport.Model
	content       string
	filename      string
	ready         bool
	searching     bool
	searchInput   string
	searchQuery   string
	matches       []int
	matchIdx      int
	scrollAcc     int  // pending scroll lines from wheel events (+ = down, - = up)
	scrollPending bool // a scrollTick is already in flight
	vpView        string // cached viewport.View() — recomputed only on position change
}

func (m model) Init() tea.Cmd {
	return enableAltScroll()
}

// refreshVP re-renders the viewport into m.vpView. Call after any operation
// that changes the viewport's scroll position or content.
func (m *model) refreshVP() {
	m.vpView = m.viewport.View()
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
		m.refreshVP()
		return m, cmd

	case scrollTickMsg:
		m.scrollPending = false
		if m.scrollAcc > 0 {
			m.viewport.ScrollDown(m.scrollAcc)
		} else if m.scrollAcc < 0 {
			m.viewport.ScrollUp(-m.scrollAcc)
		}
		m.scrollAcc = 0
		m.refreshVP()
		return m, nil

	case tea.KeyPressMsg:
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
					m.refreshVP()
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

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.matches = nil
				m.matchIdx = 0
				m.viewport.SetContent(m.content)
				m.refreshVP()
			} else {
				return m, tea.Quit
			}
		case "up":
			if m.scrollAcc > 0 {
				m.scrollAcc = 0
			}
			m.scrollAcc--
			if !m.scrollPending {
				m.scrollPending = true
				return m, scrollTick()
			}
			return m, nil
		case "down":
			if m.scrollAcc < 0 {
				m.scrollAcc = 0
			}
			m.scrollAcc++
			if !m.scrollPending {
				m.scrollPending = true
				return m, scrollTick()
			}
			return m, nil
		case "pgup":
			m.viewport.ScrollUp(m.viewport.Height())
			m.refreshVP()
		case "pgdown":
			m.viewport.ScrollDown(m.viewport.Height())
			m.refreshVP()
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
				m.refreshVP()
			}
		case "N":
			if len(m.matches) > 0 {
				m.matchIdx = (m.matchIdx - 1 + len(m.matches)) % len(m.matches)
				line := m.matches[m.matchIdx]
				m.viewport.SetContent(applyHighlights(m.content, m.searchQuery, m.matches, line))
				m.viewport.SetYOffset(line)
				m.refreshVP()
			}
		case "g":
			m.viewport.GotoTop()
			m.refreshVP()
		case "G":
			m.viewport.GotoBottom()
			m.refreshVP()
		case "k":
			m.viewport.ScrollUp(1)
			m.refreshVP()
		case "j":
			m.viewport.ScrollDown(1)
			m.refreshVP()
		case "l":
			m.viewport.ScrollDown(15)
			m.refreshVP()
		case "p":
			m.viewport.ScrollUp(15)
			m.refreshVP()
		case "f":
			m.viewport.ScrollDown(m.viewport.Height())
			m.refreshVP()
		case "b":
			m.viewport.ScrollUp(m.viewport.Height())
			m.refreshVP()
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

		content = fmt.Sprintf("%s\n%s\n%s\n%s", title, divider, m.vpView, footer)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeNone
	return v
}
