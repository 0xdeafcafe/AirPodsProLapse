package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Align(lipgloss.Center)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	activeParamStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	normalParamStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	dimParamStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242"))

	meterBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	meterClipStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	bypassOnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("208"))

	bypassOffStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Align(lipgloss.Center)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75"))
)
