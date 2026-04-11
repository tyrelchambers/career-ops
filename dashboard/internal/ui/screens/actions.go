package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/santifer/career-ops/dashboard/internal/theme"
)

// ActionItem represents a single executable action.
type ActionItem struct {
	Label       string
	Description string
	Category    string // "claude" or "script"
	Command     string // mode name or script filename
	NeedsApp    bool
}

var claudeActions = []ActionItem{
	{Label: "Evaluate", Description: "Run A-F evaluation on JD", Category: "claude", Command: "evaluate", NeedsApp: true},
	{Label: "Generate PDF", Description: "Generate tailored CV", Category: "claude", Command: "pdf", NeedsApp: true},
	{Label: "Contact", Description: "LinkedIn outreach draft", Category: "claude", Command: "contact", NeedsApp: true},
	{Label: "Deep Research", Description: "Company intel report", Category: "claude", Command: "deep", NeedsApp: true},
	{Label: "Interview Prep", Description: "Interview prep materials", Category: "claude", Command: "interview-prep", NeedsApp: true},
	{Label: "Compare", Description: "Rank top offers", Category: "claude", Command: "compare", NeedsApp: false},
	{Label: "Follow-up", Description: "Check follow-up cadence", Category: "claude", Command: "followup", NeedsApp: false},
	{Label: "Scan", Description: "Scan portals for new offers", Category: "claude", Command: "scan", NeedsApp: false},
}

var scriptActions = []ActionItem{
	{Label: "Merge Tracker", Description: "Merge batch additions", Category: "script", Command: "merge-tracker.mjs", NeedsApp: false},
	{Label: "Verify Pipeline", Description: "Check pipeline health", Category: "script", Command: "verify-pipeline.mjs", NeedsApp: false},
	{Label: "Dedup Tracker", Description: "Remove duplicates", Category: "script", Command: "dedup-tracker.mjs", NeedsApp: false},
	{Label: "Normalize", Description: "Fix statuses", Category: "script", Command: "normalize-statuses.mjs", NeedsApp: false},
	{Label: "Patterns", Description: "Rejection analysis", Category: "script", Command: "analyze-patterns.mjs", NeedsApp: false},
	{Label: "Health Check", Description: "System diagnostics", Category: "script", Command: "doctor.mjs", NeedsApp: false},
}

var actionCategories = []struct {
	Name    string
	Actions []ActionItem
}{
	{Name: "Claude CLI", Actions: claudeActions},
	{Name: "Scripts", Actions: scriptActions},
}

// actionsForCategory returns the action list for a given category index.
func actionsForCategory(cat int) []ActionItem {
	if cat < 0 || cat >= len(actionCategories) {
		return nil
	}
	return actionCategories[cat].Actions
}

// overlayActionMenu renders the action menu centered on screen.
func overlayActionMenu(cat, cursor, width, height int, hasApp bool, t theme.Theme) string {
	contentWidth := 48

	dimStyle := lipgloss.NewStyle().Foreground(t.Overlay)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Text).
		Background(t.Overlay).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(t.Subtext).
		Padding(0, 1)

	// Build inner content
	var rows []string

	// Category tabs
	var tabs string
	for i, c := range actionCategories {
		if i == cat {
			tabs += activeTabStyle.Render(c.Name)
		} else {
			tabs += inactiveTabStyle.Render(c.Name)
		}
		if i < len(actionCategories)-1 {
			tabs += "  "
		}
	}
	rows = append(rows, tabs)
	rows = append(rows, lipgloss.NewStyle().Foreground(t.Overlay).Render(strings.Repeat("─", contentWidth)))

	// Action items
	labelWidth := 18
	actions := actionsForCategory(cat)
	for i, a := range actions {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		disabled := a.NeedsApp && !hasApp
		label := prefix + a.Label
		desc := a.Description

		// Pad label to fixed width
		labelRunes := []rune(label)
		for len(labelRunes) < labelWidth {
			labelRunes = append(labelRunes, ' ')
		}
		label = string(labelRunes[:labelWidth])

		if disabled {
			row := dimStyle.Render(label + "  " + desc)
			rows = append(rows, row)
		} else if i == cursor {
			rowStyle := lipgloss.NewStyle().
				Background(t.Overlay).
				Bold(true).
				Foreground(t.Text).
				Width(contentWidth)
			rows = append(rows, rowStyle.Render(label+"  "+desc))
		} else {
			labelStr := lipgloss.NewStyle().Foreground(t.Text).Render(label)
			descStr := lipgloss.NewStyle().Foreground(t.Subtext).Render("  " + desc)
			rows = append(rows, labelStr+descStr)
		}
	}

	// Help line
	rows = append(rows, "")
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Text)
	helpDescStyle := lipgloss.NewStyle().Foreground(t.Subtext)
	help := keyStyle.Render("↑↓") + helpDescStyle.Render(" nav  ") +
		keyStyle.Render("Tab") + helpDescStyle.Render(" category  ") +
		keyStyle.Render("Enter") + helpDescStyle.Render(" run  ") +
		keyStyle.Render("Esc") + helpDescStyle.Render(" close")
	rows = append(rows, help)

	content := strings.Join(rows, "\n")

	// Wrap in a bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Blue).
		Background(t.Base).
		Padding(1, 2).
		Render(content)

	// Add title to border
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Blue).Background(t.Base)
	title := titleStyle.Render(" ACTIONS ")

	boxLines := strings.Split(box, "\n")
	if len(boxLines) > 0 {
		// Replace part of top border with title
		topLine := boxLines[0]
		topRunes := []rune(topLine)
		titleRunes := []rune(title)
		if len(topRunes) > 4+len(titleRunes) {
			var newTop []rune
			newTop = append(newTop, topRunes[:3]...)
			newTop = append(newTop, titleRunes...)
			newTop = append(newTop, topRunes[3+len(titleRunes):]...)
			boxLines[0] = string(newTop)
		}
		box = strings.Join(boxLines, "\n")
	}

	// Center on screen
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(t.Base))
}
