package tui

import "github.com/charmbracelet/lipgloss"

// ThemeColors holds the color palette for a theme.
type ThemeColors struct {
	Bg        lipgloss.Color
	BgAlt     lipgloss.Color // alternate row background
	Fg        lipgloss.Color
	Muted     lipgloss.Color
	Accent    lipgloss.Color
	Green     lipgloss.Color
	Red       lipgloss.Color
	Yellow    lipgloss.Color
	TabActive lipgloss.Color
	TabBg     lipgloss.Color
	Border    lipgloss.Color
}

// DarkTheme is Catppuccin Mocha.
var DarkTheme = ThemeColors{
	Bg:        lipgloss.Color("#1e1e2e"),
	BgAlt:     lipgloss.Color("#232330"),
	Fg:        lipgloss.Color("#cdd6f4"),
	Muted:     lipgloss.Color("#6c7086"),
	Accent:    lipgloss.Color("#89b4fa"),
	Green:     lipgloss.Color("#a6e3a1"),
	Red:       lipgloss.Color("#f38ba8"),
	Yellow:    lipgloss.Color("#f9e2af"),
	TabActive: lipgloss.Color("#89b4fa"),
	TabBg:     lipgloss.Color("#313244"),
	Border:    lipgloss.Color("#45475a"),
}

// LightTheme is Catppuccin Latte.
var LightTheme = ThemeColors{
	Bg:        lipgloss.Color("#eff1f5"),
	BgAlt:     lipgloss.Color("#e6e9ef"),
	Fg:        lipgloss.Color("#4c4f69"),
	Muted:     lipgloss.Color("#9ca0b0"),
	Accent:    lipgloss.Color("#1e66f5"),
	Green:     lipgloss.Color("#40a02b"),
	Red:       lipgloss.Color("#d20f39"),
	Yellow:    lipgloss.Color("#df8e1d"),
	TabActive: lipgloss.Color("#1e66f5"),
	TabBg:     lipgloss.Color("#dce0e8"),
	Border:    lipgloss.Color("#bcc0cc"),
}

// ActiveThemeName is the currently applied theme ("dark" or "light").
var ActiveThemeName = "dark"

// Color variables — initialized to dark theme; reassigned by applyTheme on switch.
var (
	colorBg        = lipgloss.Color("#1e1e2e")
	colorBgAlt     = lipgloss.Color("#232330")
	colorFg        = lipgloss.Color("#cdd6f4")
	colorMuted     = lipgloss.Color("#6c7086")
	colorAccent    = lipgloss.Color("#89b4fa")
	colorGreen     = lipgloss.Color("#a6e3a1")
	colorRed       = lipgloss.Color("#f38ba8")
	colorYellow    = lipgloss.Color("#f9e2af")
	colorTabActive = lipgloss.Color("#89b4fa")
	colorTabBg     = lipgloss.Color("#313244")
	colorBorder    = lipgloss.Color("#45475a")
)

// Style variables — initialized to dark theme; rebuilt by applyTheme on switch.
var (
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
			Background(colorBgAlt).
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

// ApplyThemeByName applies "dark" or "light"; unknown names default to dark.
func ApplyThemeByName(name string) {
	if name == "light" {
		applyTheme(LightTheme)
		ActiveThemeName = "light"
	} else {
		applyTheme(DarkTheme)
		ActiveThemeName = "dark"
	}
}

// ToggleTheme flips to the opposite theme and returns the new name.
func ToggleTheme() string {
	if ActiveThemeName == "light" {
		applyTheme(DarkTheme)
		ActiveThemeName = "dark"
	} else {
		applyTheme(LightTheme)
		ActiveThemeName = "light"
	}
	return ActiveThemeName
}

// applyTheme reassigns all color and style vars. Must be kept in sync with
// any new package-level style vars added to this package.
func applyTheme(tc ThemeColors) {
	colorBg = tc.Bg
	colorBgAlt = tc.BgAlt
	colorFg = tc.Fg
	colorMuted = tc.Muted
	colorAccent = tc.Accent
	colorGreen = tc.Green
	colorRed = tc.Red
	colorYellow = tc.Yellow
	colorTabActive = tc.TabActive
	colorTabBg = tc.TabBg
	colorBorder = tc.Border

	styleBase = lipgloss.NewStyle().Background(tc.Bg).Foreground(tc.Fg)

	styleTabActive = lipgloss.NewStyle().
		Bold(true).Foreground(tc.Bg).Background(tc.TabActive).Padding(0, 1)

	styleTabInactive = lipgloss.NewStyle().
		Foreground(tc.Fg).Background(tc.TabBg).Padding(0, 1)

	styleTabNew = lipgloss.NewStyle().
		Foreground(tc.Accent).Background(tc.TabBg).Padding(0, 1)

	stylePaneTitle = lipgloss.NewStyle().
		Bold(true).Foreground(tc.Accent).PaddingLeft(1)

	stylePaneBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tc.Border).Padding(0, 1)

	styleInputBorderFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tc.Accent).Padding(0, 1)

	styleInputBorderBlurred = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tc.Border).Padding(0, 1)

	styleStatusBar = lipgloss.NewStyle().
		Background(tc.TabBg).Foreground(tc.Fg).PaddingLeft(1).PaddingRight(1)

	styleStatusConnected = lipgloss.NewStyle().Foreground(tc.Green).Bold(true)
	styleStatusDisconnected = lipgloss.NewStyle().Foreground(tc.Red).Bold(true)
	styleStatusConnecting = lipgloss.NewStyle().Foreground(tc.Yellow)

	styleError = lipgloss.NewStyle().Foreground(tc.Red)
	styleSuccess = lipgloss.NewStyle().Foreground(tc.Green)
	styleMuted = lipgloss.NewStyle().Foreground(tc.Muted)

	styleHeader = lipgloss.NewStyle().
		Bold(true).Foreground(tc.Accent).Background(tc.TabBg).Padding(0, 1)

	styleCell = lipgloss.NewStyle().Foreground(tc.Fg).Padding(0, 1)

	styleCellAlt = lipgloss.NewStyle().
		Foreground(tc.Fg).Background(tc.BgAlt).Padding(0, 1)

	styleCellSelected = lipgloss.NewStyle().
		Foreground(tc.Bg).Background(tc.Accent).Padding(0, 1)

	styleAutocompleteSelected = lipgloss.NewStyle().
		Foreground(tc.Bg).Background(tc.Accent).Padding(0, 1)

	styleAutocompleteItem = lipgloss.NewStyle().
		Foreground(tc.Fg).Background(tc.TabBg).Padding(0, 1)

	styleOverlay = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tc.Accent).
		Background(tc.TabBg).Padding(1, 2)

	styleWelcome = lipgloss.NewStyle().Foreground(tc.Accent).Bold(true)

	// browser-specific styles (declared in schemabrowser.go, same package)
	styleBrowserBorderFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tc.Accent)
	styleBrowserBorderBlurred = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tc.Border)
	styleBrowserHeader = lipgloss.NewStyle().Bold(true).Foreground(tc.Accent)
	styleBrowserSec = lipgloss.NewStyle().Bold(true).Foreground(tc.Yellow)
	styleBrowserSel = lipgloss.NewStyle().Foreground(tc.Bg).Background(tc.Accent)
	styleBrowserSelDim = lipgloss.NewStyle().Foreground(tc.Accent).Bold(true)
	styleBrowserNormal = lipgloss.NewStyle().Foreground(tc.Fg)
	styleBrowserMuted = lipgloss.NewStyle().Foreground(tc.Muted)
	styleBrowserPart = lipgloss.NewStyle().Foreground(tc.Yellow)

	// cursor style (declared in input.go, same package)
	styleCursor = lipgloss.NewStyle().Foreground(tc.Bg).Background(tc.Accent)
}
