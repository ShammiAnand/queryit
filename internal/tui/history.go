package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type historyEntry struct {
	Query string    `json:"query"`
	TS    time.Time `json:"ts"`
}

type HistoryModel struct {
	entries  []historyEntry
	filtered []historyEntry
	selected int
	search   string
	visible  bool
	path     string
	maxSize  int
	width    int
	height   int
}

func NewHistoryModel(path string, maxSize int) *HistoryModel {
	h := &HistoryModel{path: path, maxSize: maxSize}
	_ = h.load()
	return h
}

func (h *HistoryModel) load() error {
	f, err := os.Open(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e historyEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err == nil {
			h.entries = append(h.entries, e)
		}
	}
	return sc.Err()
}

func (h *HistoryModel) Append(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	e := historyEntry{Query: query, TS: time.Now()}
	h.entries = append(h.entries, e)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[len(h.entries)-h.maxSize:]
	}
	return h.appendLine(e)
}

func (h *HistoryModel) appendLine(e historyEntry) error {
	if err := os.MkdirAll(dirOf(h.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(h.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := json.Marshal(e)
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func (h *HistoryModel) Show() {
	h.visible = true
	h.search = ""
	h.selected = 0
	h.applyFilter()
}

func (h *HistoryModel) Hide() {
	h.visible = false
}

func (h *HistoryModel) IsVisible() bool {
	return h.visible
}

func (h *HistoryModel) SetSize(w, height int) {
	h.width = w
	h.height = height
}

func (h *HistoryModel) TypeChar(ch string) {
	h.search += ch
	h.applyFilter()
}

func (h *HistoryModel) Backspace() {
	if len(h.search) > 0 {
		h.search = h.search[:len(h.search)-1]
		h.applyFilter()
	}
}

func (h *HistoryModel) Next() {
	if h.selected < len(h.filtered)-1 {
		h.selected++
	}
}

func (h *HistoryModel) Prev() {
	if h.selected > 0 {
		h.selected--
	}
}

func (h *HistoryModel) Accept() string {
	if len(h.filtered) == 0 {
		return ""
	}
	q := h.filtered[h.selected].Query
	h.Hide()
	return q
}

// NewestFirst returns all query strings newest-first for inline cycling.
func (h *HistoryModel) NewestFirst() []string {
	out := make([]string, len(h.entries))
	for i, e := range h.entries {
		out[len(h.entries)-1-i] = e.Query
	}
	return out
}

func (h *HistoryModel) applyFilter() {
	h.filtered = nil
	lsearch := strings.ToLower(h.search)
	// iterate in reverse so newest first
	for i := len(h.entries) - 1; i >= 0; i-- {
		e := h.entries[i]
		if lsearch == "" || strings.Contains(strings.ToLower(e.Query), lsearch) {
			h.filtered = append(h.filtered, e)
		}
	}
	if h.selected >= len(h.filtered) {
		h.selected = 0
	}
}

func (h *HistoryModel) View() string {
	if !h.visible {
		return ""
	}
	title := stylePaneTitle.Render("Query History") + "  " + styleMuted.Render("(ctrl+r to close)")
	searchLine := "Search: " + h.search + "█"

	maxItems := h.height - 6
	if maxItems < 1 {
		maxItems = 5
	}

	start := 0
	if h.selected >= maxItems {
		start = h.selected - maxItems + 1
	}
	end := start + maxItems
	if end > len(h.filtered) {
		end = len(h.filtered)
	}

	var rows []string
	for i := start; i < end; i++ {
		entry := h.filtered[i]
		q := truncate(entry.Query, h.width-8)
		line := fmt.Sprintf("%-*s  %s", h.width-28, q, styleMuted.Render(entry.TS.Format("01-02 15:04")))
		if i == h.selected {
			rows = append(rows, styleAutocompleteSelected.Render(line))
		} else {
			rows = append(rows, styleAutocompleteItem.Render(line))
		}
	}

	if len(rows) == 0 {
		rows = append(rows, styleMuted.Render("  no entries"))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, searchLine, ""}, rows...)...,
	)

	return styleOverlay.Width(h.width - 4).Render(body)
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
