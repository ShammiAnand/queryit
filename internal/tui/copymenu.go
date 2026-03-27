package tui

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/db"
)

// rowsToCSV serialises columns + all pages into a CSV string.
func rowsToCSV(columns []string, pages [][]db.Row) string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write(columns)
	for _, page := range pages {
		for _, row := range page {
			_ = w.Write(row)
		}
	}
	w.Flush()
	return sb.String()
}

// rowToCSV serialises a single row as a CSV line (no trailing newline stripped).
func rowToCSV(row db.Row) string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write(row)
	w.Flush()
	return strings.TrimRight(sb.String(), "\n")
}

// copyToClipboard writes text to the system clipboard via the first available
// platform command. Returns an error if no clipboard command is found.
func copyToClipboard(text string) error {
	type candidate struct {
		name string
		args []string
	}
	candidates := []candidate{
		{"pbcopy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"wl-copy", nil},
		{"clip", nil},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c.name); err != nil {
			continue
		}
		var cmd *exec.Cmd
		if len(c.args) > 0 {
			cmd = exec.Command(c.name, c.args...)
		} else {
			cmd = exec.Command(c.name)
		}
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("no clipboard command found (tried pbcopy, xclip, wl-copy, clip)")
}

// exportToFile writes the full result as CSV to ~/queryit_<timestamp>.csv.
// Returns the absolute path written.
func exportToFile(columns []string, pages [][]db.Row) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	fname := "queryit_" + time.Now().Format("20060102T150405") + ".csv"
	path := filepath.Join(home, fname)

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write(columns)
	for _, page := range pages {
		for _, row := range page {
			_ = w.Write(row)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return path, nil
}

// ─── CopyMenuModal ────────────────────────────────────────────────────────────

// CopyMenuModal is a small overlay that lets the user choose what to copy/export.
// It has no internal state beyond visibility; the action dispatch lives in tab.go.
type CopyMenuModal struct {
	visible bool
	width   int
	height  int
}

func (c *CopyMenuModal) Show(w, h int)   { c.visible = true; c.width = w; c.height = h }
func (c *CopyMenuModal) Hide()           { c.visible = false }
func (c *CopyMenuModal) IsVisible() bool { return c.visible }

func (c *CopyMenuModal) View() string {
	w := c.width - 4
	if w > 72 {
		w = 72
	}
	if w < 40 {
		w = 40
	}

	bold := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	title := bold.Render("Copy / Export")
	sep := styleMuted.Render(strings.Repeat("─", w-6))

	key := func(k string) string { return lipgloss.NewStyle().Bold(true).Render("[" + k + "]") }

	row1 := key("c") + " cell    " + key("r") + " row    " + key("t") + " table to clipboard"
	row2 := key("e") + " export table to CSV file    " + styleMuted.Render("[esc] cancel")

	body := strings.Join([]string{title, sep, row1, row2}, "\n")
	return styleOverlay.Width(w).Render(body)
}
