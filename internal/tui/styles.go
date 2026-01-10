package tui

import "github.com/charmbracelet/lipgloss"

// Color palette based on TUI design
var (
	// Priority colors
	PriorityUrgent = lipgloss.Color("#FF6B6B") // P1 - Red
	PriorityHigh   = lipgloss.Color("#FFB347") // P2 - Orange
	PriorityMedium = lipgloss.Color("#FFE66D") // P3 - Yellow
	PriorityLow    = lipgloss.Color("#4ECDC4") // P4 - Blue

	// Status colors
	Completed   = lipgloss.Color("#95E1A3") // Green
	SyncOK      = lipgloss.Color("#95E1A3") // Green
	SyncPending = lipgloss.Color("#FFE66D") // Yellow
	SyncError   = lipgloss.Color("#FF6B6B") // Red
	Offline     = lipgloss.Color("#6C757D") // Gray

	// UI colors
	Primary    = lipgloss.Color("#4ECDC4")
	Secondary  = lipgloss.Color("#6C757D")
	Background = lipgloss.Color("#1a1a2e")
	Surface    = lipgloss.Color("#16213e")
	Text       = lipgloss.Color("#FFFFFF")
	TextMuted  = lipgloss.Color("#888888")
	Border     = lipgloss.Color("#333333")
	Highlight  = lipgloss.Color("#4ECDC4")
)

// Styles
var (
	// App container
	AppStyle = lipgloss.NewStyle().
			Background(Background)

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(0, 1)

	// Sidebar
	SidebarStyle = lipgloss.NewStyle().
			Width(20).
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true).
			BorderForeground(Border).
			Padding(1, 1)

	// Task list
	TaskListStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Project item
	ProjectItemStyle = lipgloss.NewStyle().
				Padding(0, 1)

	ProjectItemSelectedStyle = lipgloss.NewStyle().
					Padding(0, 1).
					Background(Surface).
					Bold(true)

	// Task item
	TaskItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	TaskItemSelectedStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(Surface).
				Bold(true)

	TaskDoneStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Strikethrough(true).
			Padding(0, 1)

	// Priority badges
	PriorityP1Style = lipgloss.NewStyle().Foreground(PriorityUrgent).Bold(true)
	PriorityP2Style = lipgloss.NewStyle().Foreground(PriorityHigh).Bold(true)
	PriorityP3Style = lipgloss.NewStyle().Foreground(PriorityMedium)
	PriorityP4Style = lipgloss.NewStyle().Foreground(PriorityLow)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(Border)

	// Input modal
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(TextMuted)
)

// GetPriorityStyle returns the style for a given priority
func GetPriorityStyle(priority int) lipgloss.Style {
	switch priority {
	case 1:
		return PriorityP1Style
	case 2:
		return PriorityP2Style
	case 3:
		return PriorityP3Style
	default:
		return PriorityP4Style
	}
}

// FormatPriority returns a formatted priority string
func FormatPriority(priority int) string {
	style := GetPriorityStyle(priority)
	switch priority {
	case 1:
		return style.Render("P1")
	case 2:
		return style.Render("P2")
	case 3:
		return style.Render("P3")
	default:
		return style.Render("P4")
	}
}
