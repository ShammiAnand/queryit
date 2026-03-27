package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBg        = lipgloss.Color("#1e1e2e")
	colorFg        = lipgloss.Color("#cdd6f4")
	colorMuted     = lipgloss.Color("#6c7086")
	colorAccent    = lipgloss.Color("#89b4fa")
	colorGreen     = lipgloss.Color("#a6e3a1")
	colorRed       = lipgloss.Color("#f38ba8")
	colorYellow    = lipgloss.Color("#f9e2af")
	colorTabActive = lipgloss.Color("#89b4fa")
	colorTabBg     = lipgloss.Color("#313244")
	colorBorder    = lipgloss.Color("#45475a")

	styleBase = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorFg)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBg).
			Background(colorTabActive).
			Padding(0, 1)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(colorTabBg).
				Padding(0, 1)

	styleTabNew = lipgloss.NewStyle().
			Foreground(colorAccent).
			Background(colorTabBg).
			Padding(0, 1)

	stylePaneTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			PaddingLeft(1)

	stylePaneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleInputBorderFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	styleInputBorderBlurred = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)

	styleStatusBar = lipgloss.NewStyle().
			Background(colorTabBg).
			Foreground(colorFg).
			PaddingLeft(1).
			PaddingRight(1)

	styleStatusConnected = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	styleStatusDisconnected = lipgloss.NewStyle().
				Foreground(colorRed).
				Bold(true)

	styleStatusConnecting = lipgloss.NewStyle().
				Foreground(colorYellow)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Background(colorTabBg).
			Padding(0, 1)

	styleCell = lipgloss.NewStyle().
			Foreground(colorFg).
			Padding(0, 1)

	styleCellAlt = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(lipgloss.Color("#232330")).
			Padding(0, 1)

	styleCellSelected = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorAccent).
			Padding(0, 1)

	styleAutocompleteSelected = lipgloss.NewStyle().
					Foreground(colorBg).
					Background(colorAccent).
					Padding(0, 1)

	styleAutocompleteItem = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(colorTabBg).
				Padding(0, 1)

	styleOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Background(colorTabBg).
			Padding(1, 2)

	styleWelcome = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
)
