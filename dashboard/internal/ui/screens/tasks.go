package screens

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/santifer/career-ops/dashboard/internal/theme"
)

// TaskEntry represents a tracked background action.
type TaskEntry struct {
	ID        int
	Label     string
	Status    string // "running", "completed", "failed", "cancelled"
	StartTime time.Time
	Lines     []string
	ExitCode  int
	Err       error
	Runner    *ActionRunner
	JobURL    string // set for evaluate tasks to track pending status
}

// TasksClosedMsg is emitted when the tasks view is dismissed.
type TasksClosedMsg struct{}

// TasksViewOutputMsg is emitted when the user wants to view a task's output.
type TasksViewOutputMsg struct {
	TaskID int
}

// TasksCancelMsg is emitted when the user cancels a running task.
type TasksCancelMsg struct {
	TaskID int
}

// TasksModel displays the list of background tasks.
type TasksModel struct {
	tasks         []*TaskEntry
	cursor        int
	width, height int
	theme         theme.Theme
}

// NewTasksModel creates a new tasks list view.
func NewTasksModel(t theme.Theme, tasks []*TaskEntry, width, height int) TasksModel {
	return TasksModel{
		tasks:  tasks,
		width:  width,
		height: height,
		theme:  t,
	}
}

func (m *TasksModel) Resize(width, height int) {
	m.width = width
	m.height = height
}

// UpdateTasks refreshes the task list reference.
func (m *TasksModel) UpdateTasks(tasks []*TaskEntry) {
	m.tasks = tasks
	if m.cursor >= len(m.tasks) {
		m.cursor = len(m.tasks) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m TasksModel) Update(msg tea.Msg) (TasksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return TasksClosedMsg{} }

		case "down", "j":
			if len(m.tasks) > 0 {
				m.cursor++
				if m.cursor >= len(m.tasks) {
					m.cursor = len(m.tasks) - 1
				}
			}

		case "up", "k":
			if len(m.tasks) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 0
				}
			}

		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.tasks) {
				taskID := m.tasks[m.cursor].ID
				return m, func() tea.Msg { return TasksViewOutputMsg{TaskID: taskID} }
			}

		case "x":
			if m.cursor >= 0 && m.cursor < len(m.tasks) && m.tasks[m.cursor].Status == "running" {
				taskID := m.tasks[m.cursor].ID
				return m, func() tea.Msg { return TasksCancelMsg{TaskID: taskID} }
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m TasksModel) View() string {
	header := m.renderHeader()
	body := m.renderBody()
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m TasksModel) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 2)

	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Blue).Render("TASKS")

	running := 0
	for _, t := range m.tasks {
		if t.Status == "running" {
			running++
		}
	}

	var info string
	if running > 0 {
		info = lipgloss.NewStyle().Foreground(m.theme.Yellow).Render(fmt.Sprintf("%d running", running))
	} else {
		info = lipgloss.NewStyle().Foreground(m.theme.Subtext).Render(fmt.Sprintf("%d total", len(m.tasks)))
	}

	gap := m.width - lipgloss.Width("TASKS") - lipgloss.Width(info) - 4
	if gap < 1 {
		gap = 1
	}

	return style.Render(title + strings.Repeat(" ", gap) + info)
}

func (m TasksModel) renderBody() string {
	bodyHeight := m.height - 4
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	padStyle := lipgloss.NewStyle().Padding(0, 2)

	if len(m.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)
		content := emptyStyle.Render("No tasks yet. Press  a  from the pipeline to run an action.")
		// Pad to fill
		lines := content
		for i := 1; i < bodyHeight; i++ {
			lines += "\n"
		}
		return padStyle.Render(lines)
	}

	var rows []string
	// Column header
	colHeader := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Subtext)
	headerLine := colHeader.Render(
		fmt.Sprintf("  %-4s  %-20s  %-12s  %-10s  %s", "STS", "ACTION", "ELAPSED", "OUTPUT", ""),
	)
	rows = append(rows, headerLine)
	rows = append(rows, lipgloss.NewStyle().Foreground(m.theme.Overlay).Render(strings.Repeat("─", m.width-4)))

	for i, task := range m.tasks {
		rows = append(rows, m.renderTaskRow(i, task))
	}

	// Pad to fill height
	for len(rows) < bodyHeight {
		rows = append(rows, "")
	}
	if len(rows) > bodyHeight {
		rows = rows[:bodyHeight]
	}

	return padStyle.Render(strings.Join(rows, "\n"))
}

func (m TasksModel) renderTaskRow(idx int, task *TaskEntry) string {
	// Status indicator
	var statusIcon string
	var statusStyle lipgloss.Style
	switch task.Status {
	case "running":
		statusIcon = "⟳"
		statusStyle = lipgloss.NewStyle().Foreground(m.theme.Yellow)
	case "completed":
		statusIcon = "✓"
		statusStyle = lipgloss.NewStyle().Foreground(m.theme.Green)
	case "failed":
		statusIcon = "✗"
		statusStyle = lipgloss.NewStyle().Foreground(m.theme.Red)
	case "cancelled":
		statusIcon = "○"
		statusStyle = lipgloss.NewStyle().Foreground(m.theme.Subtext)
	}

	// Elapsed time
	elapsed := time.Since(task.StartTime).Truncate(time.Second)
	elapsedStr := elapsed.String()

	// Output line count
	outputStr := fmt.Sprintf("%d lines", len(task.Lines))

	// Error info
	extra := ""
	if task.Err != nil {
		extra = lipgloss.NewStyle().Foreground(m.theme.Red).Render(task.Err.Error())
	}

	// Build row
	label := task.Label
	labelRunes := []rune(label)
	if len(labelRunes) > 20 {
		label = string(labelRunes[:17]) + "..."
	}

	row := fmt.Sprintf("  %s  %-20s  %-12s  %-10s  %s",
		statusStyle.Render(statusIcon),
		label,
		elapsedStr,
		outputStr,
		extra,
	)

	if idx == m.cursor {
		return lipgloss.NewStyle().
			Background(m.theme.Overlay).
			Bold(true).
			Foreground(m.theme.Text).
			Width(m.width - 4).
			Render(row)
	}

	return lipgloss.NewStyle().Foreground(m.theme.Text).Render(row)
}

func (m TasksModel) renderFooter() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Subtext).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

	keys := keyStyle.Render("↑↓") + descStyle.Render(" navigate  ") +
		keyStyle.Render("Enter") + descStyle.Render(" view output  ") +
		keyStyle.Render("x") + descStyle.Render(" cancel  ") +
		keyStyle.Render("Esc") + descStyle.Render(" back")

	return style.Render(keys)
}
