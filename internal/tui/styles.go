package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Brand colors
	primaryColor   = lipgloss.Color("#FF6B6B")
	accentColor    = lipgloss.Color("#4ECDC4")
	successColor   = lipgloss.Color("#95E879")
	errorColor     = lipgloss.Color("#FF6B6B")
	warningColor   = lipgloss.Color("#FFE66D")
	mutedColor     = lipgloss.Color("#6C757D")
	bgColor        = lipgloss.Color("#1A1B26")
	surfaceColor   = lipgloss.Color("#24283B")
	textColor      = lipgloss.Color("#C0CAF5")
	dimTextColor   = lipgloss.Color("#565F89")

	// App frame
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
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
				Foreground(lipgloss.Color("#FFFFFF")).
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
			Foreground(lipgloss.Color("#000000")).
			Background(successColor).
			Bold(true).
			Padding(0, 1)

	disabledBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(mutedColor).
			Padding(0, 1)

	runningBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(warningColor).
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
