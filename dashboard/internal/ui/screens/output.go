package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/santifer/career-ops/dashboard/internal/theme"
)

// OutputClosedMsg is emitted when the output viewer is dismissed.
type OutputClosedMsg struct{}

// OutputModel displays streaming subprocess output.
type OutputModel struct {
	lines        []string
	title        string
	scrollOffset int
	autoScroll   bool
	running      bool
	exitCode     int
	err          error
	width        int
	height       int
	theme        theme.Theme
}

// NewOutputModel creates a new output viewer.
func NewOutputModel(t theme.Theme, title string, width, height int) OutputModel {
	return OutputModel{
		title:      title,
		autoScroll: true,
		running:    true,
		width:      width,
		height:     height,
		theme:      t,
	}
}

func (m *OutputModel) Resize(width, height int) {
	m.width = width
	m.height = height
}

// LoadLines loads existing output lines from a task.
func (m *OutputModel) LoadLines(lines []string, done bool) {
	m.lines = make([]string, len(lines))
	copy(m.lines, lines)
	if done {
		m.running = false
	}
	if m.autoScroll {
		maxScroll := len(m.lines) - m.bodyHeight()
		if maxScroll > 0 {
			m.scrollOffset = maxScroll
		}
	}
}

// MarkDone marks the output as completed with the given exit code and error.
func (m *OutputModel) MarkDone(exitCode int, err error) {
	m.running = false
	m.exitCode = exitCode
	m.err = err
}

// AppendOutput adds a line or marks completion.
func (m *OutputModel) AppendOutput(msg ActionOutputMsg) {
	if msg.Done {
		m.running = false
		m.exitCode = msg.ExitCode
		m.err = msg.Error
		return
	}
	m.lines = append(m.lines, msg.Line)
	if m.autoScroll {
		maxScroll := len(m.lines) - m.bodyHeight()
		if maxScroll > 0 {
			m.scrollOffset = maxScroll
		}
	}
}

func (m OutputModel) bodyHeight() int {
	h := m.height - 4
	if h < 3 {
		h = 3
	}
	return h
}

func (m OutputModel) Update(msg tea.Msg) (OutputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return OutputClosedMsg{} }

		case "down", "j":
			m.autoScroll = false
			m.scrollOffset++
			maxScroll := len(m.lines) - m.bodyHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}

		case "up", "k":
			m.autoScroll = false
			m.scrollOffset--
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}

		case "pgdown", "ctrl+d":
			m.autoScroll = false
			m.scrollOffset += m.bodyHeight() / 2
			maxScroll := len(m.lines) - m.bodyHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}

		case "pgup", "ctrl+u":
			m.autoScroll = false
			m.scrollOffset -= m.bodyHeight() / 2
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}

		case "G", "end":
			m.autoScroll = true
			maxScroll := len(m.lines) - m.bodyHeight()
			if maxScroll > 0 {
				m.scrollOffset = maxScroll
			}

		case "g", "home":
			m.autoScroll = false
			m.scrollOffset = 0
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m OutputModel) View() string {
	header := m.renderHeader()
	body := m.renderBody()
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m OutputModel) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 2)

	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Blue).Render(m.title)

	var status string
	if m.err != nil {
		status = lipgloss.NewStyle().Foreground(m.theme.Red).Render(fmt.Sprintf("Error: %v", m.err))
	} else if m.running {
		status = lipgloss.NewStyle().Foreground(m.theme.Yellow).Render("Running...")
	} else if m.exitCode != 0 {
		status = lipgloss.NewStyle().Foreground(m.theme.Red).Render(fmt.Sprintf("Failed (exit %d)", m.exitCode))
	} else {
		status = lipgloss.NewStyle().Foreground(m.theme.Green).Render("Done")
	}

	gap := m.width - lipgloss.Width(m.title) - lipgloss.Width(status) - 4
	if gap < 1 {
		gap = 1
	}

	return style.Render(title + strings.Repeat(" ", gap) + status)
}

func (m OutputModel) renderBody() string {
	bh := m.bodyHeight()
	padStyle := lipgloss.NewStyle().Padding(0, 2)

	if len(m.lines) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)
		if m.running {
			return padStyle.Render(emptyStyle.Render("Waiting for output..."))
		}
		return padStyle.Render(emptyStyle.Render("(no output)"))
	}

	end := m.scrollOffset + bh
	if end > len(m.lines) {
		end = len(m.lines)
	}
	start := m.scrollOffset
	if start < 0 {
		start = 0
	}
	if start > end {
		start = end
	}
	visible := m.lines[start:end]

	var styled []string
	for _, line := range visible {
		styled = append(styled, m.styleLine(line))
	}

	for len(styled) < bh {
		styled = append(styled, "")
	}

	return padStyle.Render(strings.Join(styled, "\n"))
}

func (m OutputModel) styleLine(line string) string {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "# ") {
		return lipgloss.NewStyle().Bold(true).Foreground(m.theme.Blue).Render(line)
	}
	if strings.HasPrefix(trimmed, "## ") {
		return lipgloss.NewStyle().Bold(true).Foreground(m.theme.Mauve).Render(line)
	}
	if strings.HasPrefix(trimmed, "### ") {
		return lipgloss.NewStyle().Bold(true).Foreground(m.theme.Sky).Render(line)
	}
	if trimmed == "---" || trimmed == "***" {
		return lipgloss.NewStyle().Foreground(m.theme.Overlay).Render(strings.Repeat("─", m.width-4))
	}
	if strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, ":**") {
		return lipgloss.NewStyle().Foreground(m.theme.Yellow).Render(line)
	}
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return lipgloss.NewStyle().Foreground(m.theme.Text).Render(line)
	}
	if strings.HasPrefix(trimmed, "Error") || strings.HasPrefix(trimmed, "error") {
		return lipgloss.NewStyle().Foreground(m.theme.Red).Render(line)
	}

	return lipgloss.NewStyle().Foreground(m.theme.Text).Render(line)
}

func (m OutputModel) renderFooter() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Subtext).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

	keys := keyStyle.Render("↑↓") + descStyle.Render(" scroll  ") +
		keyStyle.Render("PgUp/Dn") + descStyle.Render(" page  ") +
		keyStyle.Render("G") + descStyle.Render(" follow  ") +
		keyStyle.Render("g") + descStyle.Render(" top  ")

	if m.running {
		keys += keyStyle.Render("q") + descStyle.Render(" cancel")
	} else {
		keys += keyStyle.Render("q") + descStyle.Render(" close")
	}

	return style.Render(keys)
}
