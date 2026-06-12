package prompt

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCancelled is returned by Select when the user aborts the menu.
var ErrCancelled = errors.New("selection cancelled")

// Choice is a single selectable row in a Select menu.
type Choice struct {
	Title string
	Desc  string
	Tag   string
}

const selectWindow = 12

var (
	selHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	selTagStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	selDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

// Select renders an arrow-key navigable menu and returns the index of the
// chosen entry, or ErrCancelled if the user aborts. It requires an interactive
// terminal; callers should guard non-TTY contexts.
func Select(header string, choices []Choice) (int, error) {
	if len(choices) == 0 {
		return -1, fmt.Errorf("no choices to select from")
	}
	res, err := tea.NewProgram(selectModel{header: header, choices: choices, chosen: -1}).Run()
	if err != nil {
		return -1, err
	}
	final := res.(selectModel)
	if final.chosen < 0 {
		return -1, ErrCancelled
	}
	return final.chosen, nil
}

type selectModel struct {
	header  string
	choices []Choice
	cursor  int
	chosen  int
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.choices) - 1
		case "enter":
			m.chosen = m.cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.chosen >= 0 {
		return ""
	}

	start := 0
	if len(m.choices) > selectWindow {
		start = m.cursor - selectWindow/2
		if start < 0 {
			start = 0
		}
		if start > len(m.choices)-selectWindow {
			start = len(m.choices) - selectWindow
		}
	}
	end := start + selectWindow
	if end > len(m.choices) {
		end = len(m.choices)
	}

	var b strings.Builder
	b.WriteString(selHeaderStyle.Render(m.header))
	b.WriteString("\n\n")
	if start > 0 {
		b.WriteString(selDescStyle.Render(fmt.Sprintf("  ⋮ %d more above\n", start)))
	}
	for i := start; i < end; i++ {
		c := m.choices[i]
		cursor := "  "
		title := c.Title
		if i == m.cursor {
			cursor = selCursorStyle.Render("> ")
			title = selActiveStyle.Render(title)
		}
		line := cursor + title
		if c.Tag != "" {
			line += " " + selTagStyle.Render("["+c.Tag+"]")
		}
		if c.Desc != "" {
			line += "  " + selDescStyle.Render(c.Desc)
		}
		b.WriteString(line + "\n")
	}
	if end < len(m.choices) {
		b.WriteString(selDescStyle.Render(fmt.Sprintf("  ⋮ %d more below\n", len(m.choices)-end)))
	}
	b.WriteString(selHintStyle.Render("↑/↓ move • enter select • q cancel"))
	return b.String()
}
