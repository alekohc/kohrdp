package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorText   = lipgloss.Color("15")
	colorAccent = lipgloss.Color("4")
	colorMuted  = lipgloss.Color("8")
	colorWarn   = lipgloss.Color("3")
	colorErr    = lipgloss.Color("1")
	colorOK     = lipgloss.Color("2")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorAccent).
			Padding(0, 1)

	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)

	statusOK   = lipgloss.NewStyle().Foreground(colorOK)
	statusWarn = lipgloss.NewStyle().Foreground(colorWarn)
	statusErr  = lipgloss.NewStyle().Foreground(colorErr)
	statusInfo = lipgloss.NewStyle().Foreground(colorAccent)

	formLabelStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	logBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)
)
