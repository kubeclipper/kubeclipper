package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Status colors
	StatusSuccessful = lipgloss.NewStyle().Foreground(lipgloss.Color("#6a9955"))
	StatusFailed     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f44747"))
	StatusRunning    = lipgloss.NewStyle().Foreground(lipgloss.Color("#dcdcaa"))
	StatusPending    = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))

	// Layout
	StepPanelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, false, true, true).Padding(0, 1)
	LogPanelStyle  = lipgloss.NewStyle().Padding(0, 1)

	// Header
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ec9b0"))

	// Help bar
	HelpStyle = lipgloss.NewStyle().Faint(true)

	// Selected item
	SelectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("#2d2d3f"))

	// Log text
	LogStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))

	// Step indicator marks
	StepSuccessMark = "✓"
	StepFailedMark  = "✗"
	StepRunningMark = "●"
	StepPendingMark = "○"
)
