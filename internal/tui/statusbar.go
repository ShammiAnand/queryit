package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type ConnState int

const (
	StateDisconnected ConnState = iota
	StateConnecting
	StateConnected
	StatePinging
)

type StatusBar struct {
	profile  string
	state    ConnState
	rowCount int
	elapsed  time.Duration
	message  string
	width    int
}

func NewStatusBar(profile string) *StatusBar {
	return &StatusBar{profile: profile, state: StateDisconnected}
}

func (s *StatusBar) SetConnected(profile string) {
	s.profile = profile
	s.state = StateConnected
	s.message = ""
}

func (s *StatusBar) SetDisconnected() {
	s.state = StateDisconnected
	s.message = ""
}

func (s *StatusBar) SetConnecting() {
	s.state = StateConnecting
	s.message = ""
}

func (s *StatusBar) SetPinging() {
	s.state = StatePinging
	s.message = styleMuted.Render("pinging...")
}

func (s *StatusBar) IsConnected() bool {
	return s.state == StateConnected
}

func (s *StatusBar) SetQueryResult(rows int, elapsed time.Duration) {
	s.rowCount = rows
	s.elapsed = elapsed
	s.message = "" // clear "running..." so the row/time info shows
}

func (s *StatusBar) SetMessage(msg string) {
	s.message = msg
}

func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

func (s *StatusBar) View() string {
	var connStr string
	switch s.state {
	case StateConnected:
		connStr = styleStatusConnected.Render("connected")
	case StateConnecting:
		connStr = styleStatusConnecting.Render("connecting...")
	case StatePinging:
		connStr = styleMuted.Render("pinging...")
	default:
		connStr = styleStatusDisconnected.Render("disconnected")
	}

	profile := styleMuted.Render(s.profile)

	var info string
	if s.message != "" {
		info = s.message
	} else if s.state == StateConnected && s.rowCount > 0 {
		info = fmt.Sprintf("%d rows | %s", s.rowCount, s.elapsed.Round(time.Millisecond))
	}

	content := connStr + "  " + profile
	if info != "" {
		content += "  " + styleMuted.Render("|") + "  " + info
	}

	help := styleMuted.Render("tab: next  f5: run  ctrl+c: clear  esc: cycle focus  ?: help  ctrl+q: quit")

	w := s.width
	if w == 0 {
		w = 80
	}

	right := lipgloss.NewStyle().Foreground(colorMuted).Render(help)
	left := content
	gap := w - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := left + lipgloss.NewStyle().Width(gap).Render("") + right
	return styleStatusBar.Width(w).Render(bar)
}
