package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Claude brand colors
	// Primary palette
	claudeDark      = lipgloss.Color("#141413") // Dark backgrounds
	claudeLight     = lipgloss.Color("#faf9f5") // Light text
	claudeMidGray   = lipgloss.Color("#b0aea5") // Secondary elements
	claudeLightGray = lipgloss.Color("#e8e6dc") // Subtle backgrounds

	// Accent colors
	claudeOrange = lipgloss.Color("#d97757") // Primary accent
	claudeBlue   = lipgloss.Color("#6a9bcc") // Secondary accent
	claudeGreen  = lipgloss.Color("#788c5d") // Tertiary accent

	// Mapped colors for TUI
	primaryColor = claudeOrange
	accentColor  = claudeBlue
	successColor = claudeGreen
	errorColor   = lipgloss.Color("#c45c4a") // Darker orange-red for errors
	warningColor = claudeOrange
	mutedColor   = claudeMidGray
	bgColor      = claudeDark
	surfaceColor = lipgloss.Color("#1e1e1d") // Slightly lighter than dark
	textColor    = claudeLight
	dimTextColor = claudeMidGray

	// App frame
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(claudeLight).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(dimTextColor).
				Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(claudeLight).
				Background(primaryColor).
				Bold(true).
				Padding(0, 1)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Padding(0, 1)

	dimRowStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			Padding(0, 1)

	// Status badges
	enabledBadge = lipgloss.NewStyle().
			Foreground(claudeDark).
			Background(successColor).
			Bold(true).
			Padding(0, 1)

	disabledBadge = lipgloss.NewStyle().
			Foreground(claudeLight).
			Background(mutedColor).
			Padding(0, 1)

	runningBadge = lipgloss.NewStyle().
			Foreground(claudeDark).
			Background(claudeBlue).
			Bold(true).
			Padding(0, 1)

	// Form styles
	formStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			MarginTop(1)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			MarginBottom(0)

	focusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	blurredInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(dimTextColor).
				Padding(0, 1)

	// Output view
	outputHeaderStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				MarginBottom(1)

	outputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimTextColor).
			Padding(1, 2)

	// Status indicators
	statusOK = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	statusFail = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	statusRunning = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	statusPending = lipgloss.NewStyle().
			Foreground(dimTextColor)

	// Help bar
	helpBarStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(dimTextColor)

	// Misc
	subtitleStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			Italic(true)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successMsgStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	// Box for empty state
	emptyBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimTextColor).
			Foreground(dimTextColor).
			Padding(2, 4).
			Align(lipgloss.Center)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(dimTextColor)
)

func statusBadge(enabled bool, running bool) string {
	if running {
		return runningBadge.Render("RUNNING")
	}
	if enabled {
		return enabledBadge.Render("ENABLED")
	}
	return disabledBadge.Render("DISABLED")
}

func runStatusStyle(status string) lipgloss.Style {
	switch status {
	case "completed":
		return statusOK
	case "failed":
		return statusFail
	case "running":
		return statusRunning
	default:
		return statusPending
	}
}
