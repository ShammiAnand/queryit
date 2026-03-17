package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RecentModel is a collapsible panel showing the last N executed queries.
// It sits between the results pane and the input box.
// When collapsed it shows a single summary line.
// When expanded it shows up to maxVisible entries; j/k or up/down navigate,
// Enter pastes the selected query into the input box.
type RecentModel struct {
	entries     []string // newest first
	selected    int
	collapsed   bool
	focused     bool
	maxVisible  int
	width       int
}

func NewRecentModel() *RecentModel {
	return &RecentModel{
		collapsed:  true,
		maxVisible: 6,
	}
}

func (r *RecentModel) SetWidth(w int) { r.width = w }

func (r *RecentModel) SetEntries(entries []string) {
	r.entries = entries
	if r.selected >= len(r.entries) {
		r.selected = 0
	}
}

func (r *RecentModel) SetFocused(f bool) {
	r.focused = f
	if f {
		r.collapsed = false // expand when focused
	} else {
		r.collapsed = true  // collapse when focus leaves
	}
}

func (r *RecentModel) IsCollapsed() bool { return r.collapsed }
func (r *RecentModel) HasEntries() bool  { return len(r.entries) > 0 }

func (r *RecentModel) Toggle() { r.collapsed = !r.collapsed }

func (r *RecentModel) Next() {
	if len(r.entries) == 0 {
		return
	}
	r.selected = (r.selected + 1) % len(r.entries)
}

func (r *RecentModel) Prev() {
	if len(r.entries) == 0 {
		return
	}
	r.selected = (r.selected - 1 + len(r.entries)) % len(r.entries)
}

// Accept returns the currently selected query (empty string if none).
func (r *RecentModel) Accept() string {
	if len(r.entries) == 0 {
		return ""
	}
	return r.entries[r.selected]
}

// Height returns the number of lines this panel will render.
func (r *RecentModel) Height() int {
	if !r.HasEntries() {
		return 0
	}
	if r.collapsed {
		return 1
	}
	n := len(r.entries)
	if n > r.maxVisible {
		n = r.maxVisible
	}
	return n + 1 // +1 for the header line
}

func (r *RecentModel) View() string {
	if !r.HasEntries() {
		return ""
	}

	w := r.width
	if w < 20 {
		w = 80
	}

	arrow := "▶"
	if !r.collapsed {
		arrow = "▼"
	}

	focusMark := ""
	if r.focused {
		focusMark = styleStatusConnected.Render(" ●")
	}

	headerLabel := fmt.Sprintf(" %s Recent queries (%d)%s", arrow, len(r.entries), focusMark)
	if r.collapsed {
		hint := styleMuted.Render("  enter/esc: expand")
		gap := w - lipgloss.Width(headerLabel) - lipgloss.Width(hint)
		if gap < 1 {
			gap = 1
		}
		line := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(headerLabel) +
			strings.Repeat(" ", gap) + hint
		return lipgloss.NewStyle().Width(w).Background(colorTabBg).Render(line)
	}

	hint := styleMuted.Render("  j/k: navigate  enter: use  esc: collapse")
	gap := w - lipgloss.Width(headerLabel) - lipgloss.Width(hint)
	if gap < 1 {
		gap = 1
	}
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(headerLabel) +
		strings.Repeat(" ", gap) + hint
	header = lipgloss.NewStyle().Width(w).Background(colorTabBg).Render(header)

	// which slice to show
	start := 0
	if r.selected >= r.maxVisible {
		start = r.selected - r.maxVisible + 1
	}
	end := start + r.maxVisible
	if end > len(r.entries) {
		end = len(r.entries)
	}

	var rows []string
	rows = append(rows, header)
	for i := start; i < end; i++ {
		q := truncate(r.entries[i], w-6)
		q = strings.ReplaceAll(q, "\n", " ↵ ")
		line := fmt.Sprintf("  %s", q)
		if i == r.selected && r.focused {
			rows = append(rows, styleAutocompleteSelected.Width(w).Render(line))
		} else if i == r.selected {
			// selected but not focused — dimmer highlight
			rows = append(rows, lipgloss.NewStyle().
				Foreground(colorAccent).Width(w).Render(line))
		} else {
			rows = append(rows, styleMuted.Width(w).Render(line))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
