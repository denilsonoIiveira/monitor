package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	BaseColor   = lipgloss.Color("#7D56F4")
	AccentColor = lipgloss.Color("#00D7D7")
	GreenColor  = lipgloss.Color("#42E66C")
	YellowColor = lipgloss.Color("#FFD700")
	RedColor    = lipgloss.Color("#FF4D4D")
	MutedColor  = lipgloss.Color("#767676")
	BgDark      = lipgloss.Color("#1A1A24")

	HeaderTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(BaseColor).
				Padding(0, 1)

	StatusLiveStyle = lipgloss.NewStyle().
			Foreground(GreenColor).
			Bold(true)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BaseColor).
			Padding(1, 2).
			MarginRight(1).
			MarginBottom(1)

	MetricTitleStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Bold(true)

	MetricValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Height(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginTop(1)
)
