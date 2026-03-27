package tui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONViewerModal is a full-screen overlay that pretty-prints a JSON string.
// Opened with enter when the current cell is JSON; closed with esc.
type JSONViewerModal struct {
	visible      bool
	rawJSON      string
	lines        []string // pretty-printed lines
	scrollOffset int
	width        int
	height       int
}

func (j *JSONViewerModal) IsVisible() bool { return j.visible }
func (j *JSONViewerModal) Hide()           { j.visible = false }
func (j *JSONViewerModal) RawJSON() string { return j.rawJSON }

// Show opens the viewer for the given raw JSON string.
func (j *JSONViewerModal) Show(raw string, w, h int) {
	j.visible = true
	j.rawJSON = raw
	j.scrollOffset = 0
	j.width = w
	j.height = h

	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err == nil {
		b, err := json.MarshalIndent(v, "", "  ")
		if err == nil {
			j.lines = strings.Split(string(b), "\n")
			return
		}
	}
	// fallback: show raw, split on newlines
	j.lines = strings.Split(raw, "\n")
}

// SetSize updates dimensions (called on window resize).
func (j *JSONViewerModal) SetSize(w, h int) {
	j.width = w
	j.height = h
	// clamp scroll after resize so View() cannot produce start > end
	if max := len(j.lines) - j.visibleLines(); j.scrollOffset > max && max >= 0 {
		j.scrollOffset = max
	}
}

func (j *JSONViewerModal) ScrollDown() {
	max := len(j.lines) - j.visibleLines()
	if max < 0 {
		max = 0
	}
	if j.scrollOffset < max {
		j.scrollOffset++
	}
}

func (j *JSONViewerModal) ScrollUp() {
	if j.scrollOffset > 0 {
		j.scrollOffset--
	}
}

func (j *JSONViewerModal) visibleLines() int {
	// height minus: border(2) + padding(2) + title(1) + sep(1) + scroll-info(2)
	v := j.height - 8
	if v < 1 {
		v = 1
	}
	return v
}

func (j *JSONViewerModal) View() string {
	w := j.width - 4
	if w > 160 {
		w = 160
	}
	if w < 40 {
		w = 40
	}

	if len(j.lines) == 0 {
		title := stylePaneTitle.Render("JSON Viewer") +
			"  " + styleMuted.Render("esc close")
		return styleOverlay.Width(w).Render(title + "\n" + styleMuted.Render("(empty)"))
	}

	title := stylePaneTitle.Render("JSON Viewer") +
		"  " + styleMuted.Render("j/k scroll · y copy · esc close")
	sep := styleMuted.Render(strings.Repeat("─", w-6))

	vis := j.visibleLines()
	start := j.scrollOffset
	end := start + vis
	if end > len(j.lines) {
		end = len(j.lines)
	}

	body := strings.Join(j.lines[start:end], "\n")

	scrollInfo := ""
	if len(j.lines) > vis {
		scrollInfo = "\n\n" + styleMuted.Render(
			fmt.Sprintf("lines %d–%d / %d", start+1, end, len(j.lines)),
		)
	}

	content := strings.Join([]string{title, sep, body}, "\n") + scrollInfo
	return styleOverlay.Width(w).Render(content)
}
