// Package tui implements the arrow-key + mouse picker that replaces
// ghswitch.sh's hand-rolled `arrow_menu` (ghswitch.sh:90-158). A minimal
// bubbletea model is used directly rather than bubbles/list — profile,
// branch, and stash menus are short, unfiltered lists, so list's
// search/pagination machinery isn't needed; mouse support is the only thing
// a raw-ANSI reader genuinely can't do without reimplementing a mouse
// escape-sequence parser, which is what bubbletea buys us.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")).Background(lipgloss.Color("235"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("183"))
)

type model struct {
	title       string
	items       []string
	active      string
	activeLabel string
	cursor      int
	chosen      string
	cancelled   bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.items[m.cursor]
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Rows start at line 2 (title + blank line above the list).
			row := msg.Y - 2
			if row >= 0 && row < len(m.items) {
				m.cursor = row
				m.chosen = m.items[m.cursor]
				return m, tea.Quit
			}
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp && m.cursor > 0 {
			m.cursor--
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown && m.cursor < len(m.items)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m model) View() string {
	s := titleStyle.Render("▸ "+m.title) + "\n"
	s += dimStyle.Render("  ↑↓/j k or mouse · enter/click to select · q to quit") + "\n"
	for i, item := range m.items {
		suffix := ""
		if m.active != "" && item == m.active {
			label := m.activeLabel
			if label == "" {
				label = "active"
			}
			suffix = "  " + activeStyle.Render("● "+label)
		}
		var line string
		if i == m.cursor {
			line = selectedStyle.Render("› " + fmt.Sprintf("%-22s", item))
		} else {
			line = dimStyle.Render(fmt.Sprintf("  %-24s", item))
		}
		s += line + suffix + "\n"
	}
	return s
}

// Pick shows an interactive arrow-key/mouse menu over items and returns the
// chosen one. active/activeLabel mark one entry as the currently-active
// item (mirrors the bash version's "● active" suffix); pass "" to skip.
// Falls back to a plain numbered prompt when stdout isn't a real terminal
// (piped output, CI, non-interactive scripts).
func Pick(title string, items []string, active, activeLabel string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to choose from")
	}
	if !isTTY() {
		return pickPlain(title, items)
	}

	m := model{title: title, items: items, active: active, activeLabel: activeLabel}
	for i, it := range items {
		if it == active {
			m.cursor = i
			break
		}
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithMouseCellMotion())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	fm := final.(model)
	if fm.cancelled || fm.chosen == "" {
		return "", fmt.Errorf("cancelled")
	}
	return fm.chosen, nil
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
