package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpViewer struct {
	visible bool
	width   int
	height  int
}

func (h *HelpViewer) IsVisible() bool { return h.visible }
func (h *HelpViewer) Show()           { h.visible = true }
func (h *HelpViewer) Hide()           { h.visible = false }
func (h *HelpViewer) Toggle()         { h.visible = !h.visible }

func (h *HelpViewer) SetSize(w, wh int) {
	h.width = w
	h.height = wh
}

type helpSection struct {
	title string
	rows  [][2]string
}

var helpContent = []helpSection{
	{
		title: "Global",
		rows: [][2]string{
			{"ctrl+t / ctrl+n", "new tab"},
			{"ctrl+w", "close tab"},
			{"tab / ctrl+n", "next tab"},
			{"shift+tab / ctrl+p", "prev tab"},
			{"ctrl+o", "toggle schema browser"},
			{"ctrl+r", "history search"},
			{"ctrl+u", "toggle light/dark theme"},
			{"ctrl+q", "quit"},
			{"?", "toggle this help"},
		},
	},
	{
		title: "Query Input",
		rows: [][2]string{
			{"f5 / ctrl+enter", "execute query"},
			{"f4", "format SQL (pretty-print)"},
			{"ctrl+c", "cancel running / clear input"},
			{"up / down", "cycle query history"},
			{"esc", "switch focus"},
		},
	},
	{
		title: "Results",
		rows: [][2]string{
			{"j / k", "scroll rows"},
			{"h / l", "scroll columns"},
			{"n / p", "next / prev page"},
			{"+ / -", "increase / decrease page size"},
			{"v", "toggle expanded view"},
			{"enter", "view cell as JSON"},
			{"y", "copy / export menu"},
			{"r", "reconnect"},
			{"esc", "switch focus"},
		},
	},
	{
		title: "Schema Browser",
		rows: [][2]string{
			{"j / k", "navigate tables"},
			{"enter", "view table detail"},
			{"space", "paste table name into input"},
			{"esc / h", "back to list (in detail view)"},
		},
	},
	{
		title: "Recent Queries",
		rows: [][2]string{
			{"j / k", "navigate"},
			{"enter", "load query into input"},
			{"space", "collapse / expand panel"},
		},
	},
}

func (h *HelpViewer) View() string {
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent).Width(24)
	descStyle := lipgloss.NewStyle().Foreground(colorFg)
	titleStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).MarginTop(1)
	mutedStyle := lipgloss.NewStyle().Foreground(colorMuted)

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("Keybindings"))
	sb.WriteString("  ")
	sb.WriteString(mutedStyle.Render("press any key to close"))
	sb.WriteByte('\n')

	for _, sec := range helpContent {
		sb.WriteByte('\n')
		sb.WriteString(titleStyle.Render(sec.title))
		sb.WriteByte('\n')
		for _, row := range sec.rows {
			sb.WriteString(keyStyle.Render(row[0]))
			sb.WriteString(descStyle.Render(row[1]))
			sb.WriteByte('\n')
		}
	}

	content := strings.TrimRight(sb.String(), "\n")
	box := styleOverlay.Render(content)

	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, box)
}
