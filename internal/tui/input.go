package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/cache"
)

func itoa(n int) string { return strconv.Itoa(n) }

// InputModel is a fixed-height multi-line editor.
// The box always shows visibleLines lines and scrolls internally.
// Up/down on the first/last line cycles through history.
type InputModel struct {
	lines      []string
	cursorLine int
	cursorCol  int
	focused    bool
	// viewOffset is the first line visible inside the fixed window
	viewOffset   int
	visibleLines int // fixed height (default 4); set via SetSize
	width        int

	autocomplete *AutocompleteModel

	// inline history cycling
	history      []string // newest-first slice passed in from HistoryModel
	historyIdx   int      // -1 = not browsing; 0 = newest
	historyDraft string   // saved draft while browsing
}

func NewInputModel(schema *cache.SchemaCache) *InputModel {
	return &InputModel{
		lines:        []string{""},
		autocomplete: NewAutocompleteModel(schema),
		historyIdx:   -1,
		visibleLines: 4,
	}
}

// SetHistory replaces the inline history slice (newest first).
func (m *InputModel) SetHistory(entries []string) {
	m.history = entries
}

// SetSize fixes the width and the number of visible lines.
func (m *InputModel) SetSize(w, visibleLines int) {
	m.width = w
	if visibleLines > 0 {
		m.visibleLines = visibleLines
	}
	m.clampViewOffset()
}

func (m *InputModel) SetFocused(f bool) {
	m.focused = f
	if !f {
		m.autocomplete.Hide()
	}
}

func (m *InputModel) Value() string {
	return strings.Join(m.lines, "\n")
}

func (m *InputModel) SetValue(s string) {
	m.lines = strings.Split(s, "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.cursorLine = len(m.lines) - 1
	m.cursorCol = len(m.lines[m.cursorLine])
	m.historyIdx = -1
	m.clampViewOffset()
	m.updateAutocomplete()
}

func (m *InputModel) Clear() {
	m.lines = []string{""}
	m.cursorLine = 0
	m.cursorCol = 0
	m.viewOffset = 0
	m.historyIdx = -1
	m.autocomplete.Hide()
}

// Update processes a key and returns (consumed, execRequested, clearRequested, formatRequested).
func (m *InputModel) Update(msg tea.Msg) (consumed, execRequested, clearRequested, formatRequested bool) {
	if !m.focused {
		return false, false, false, false
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return false, false, false, false
	}

	// autocomplete captures up/down/enter/esc when visible
	if m.autocomplete.IsVisible() {
		switch keyMsg.String() {
		case "up":
			m.autocomplete.Prev()
			return true, false, false, false
		case "down":
			m.autocomplete.Next()
			return true, false, false, false
		case "enter":
			if accepted := m.autocomplete.Accept(); accepted != "" {
				m.insertCompletion(accepted)
			}
			return true, false, false, false
		case "esc":
			m.autocomplete.Hide()
			return true, false, false, false
		}
	}

	switch keyMsg.String() {
	case "ctrl+enter", "f5":
		return true, true, false, false

	case "f4":
		return true, false, false, true

	case "esc":
		m.autocomplete.Hide()
		return false, false, false, false // propagate → switch focus

	case "ctrl+c":
		return true, false, true, false // clear

	case "backspace":
		m.backspace()
		m.historyIdx = -1
		m.updateAutocomplete()
		return true, false, false, false

	case "delete":
		m.deleteForward()
		m.historyIdx = -1
		m.updateAutocomplete()
		return true, false, false, false

	case "left", "ctrl+b":
		m.moveCursorLeft()
		return true, false, false, false

	case "right", "ctrl+f":
		m.moveCursorRight()
		return true, false, false, false

	case "up":
		// single-line with no content below/above → cycle history
		if m.cursorLine == 0 {
			m.historyUp()
			return true, false, false, false
		}
		m.cursorLine--
		if m.cursorCol > len(m.lines[m.cursorLine]) {
			m.cursorCol = len(m.lines[m.cursorLine])
		}
		m.clampViewOffset()
		return true, false, false, false

	case "down":
		if m.cursorLine == len(m.lines)-1 {
			m.historyDown()
			return true, false, false, false
		}
		m.cursorLine++
		if m.cursorCol > len(m.lines[m.cursorLine]) {
			m.cursorCol = len(m.lines[m.cursorLine])
		}
		m.clampViewOffset()
		return true, false, false, false

	case "home", "ctrl+a":
		m.cursorCol = 0
		return true, false, false, false

	case "end", "ctrl+e":
		m.cursorCol = len(m.lines[m.cursorLine])
		return true, false, false, false

	case "enter":
		m.insertNewline()
		m.historyIdx = -1
		m.updateAutocomplete()
		return true, false, false, false

	case "ctrl+r":
		return false, false, false, false // parent handles history overlay

	case "ctrl+tab", "ctrl+t", "ctrl+w", "ctrl+n", "ctrl+p", "ctrl+q":
		return false, false, false, false // global — don't consume
	}

	if len(keyMsg.Runes) > 0 {
		for _, r := range keyMsg.Runes {
			m.insertRune(r)
		}
		m.historyIdx = -1
		m.updateAutocomplete()
		return true, false, false, false
	}

	return false, false, false, false
}

// historyUp loads an older entry.
func (m *InputModel) historyUp() {
	if len(m.history) == 0 {
		return
	}
	if m.historyIdx == -1 {
		m.historyDraft = m.Value()
	}
	next := m.historyIdx + 1
	if next >= len(m.history) {
		return
	}
	m.historyIdx = next
	m.setRaw(m.history[m.historyIdx])
}

// historyDown goes back toward the draft.
func (m *InputModel) historyDown() {
	if m.historyIdx == -1 {
		return
	}
	m.historyIdx--
	if m.historyIdx < 0 {
		m.historyIdx = -1
		m.setRaw(m.historyDraft)
		return
	}
	m.setRaw(m.history[m.historyIdx])
}

func (m *InputModel) setRaw(s string) {
	m.lines = strings.Split(s, "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.cursorLine = len(m.lines) - 1
	m.cursorCol = len(m.lines[m.cursorLine])
	m.clampViewOffset()
	m.autocomplete.Hide()
}

func (m *InputModel) insertRune(r rune) {
	line := m.lines[m.cursorLine]
	m.lines[m.cursorLine] = line[:m.cursorCol] + string(r) + line[m.cursorCol:]
	m.cursorCol++
}

func (m *InputModel) insertNewline() {
	line := m.lines[m.cursorLine]
	left, right := line[:m.cursorCol], line[m.cursorCol:]
	m.lines[m.cursorLine] = left
	tail := make([]string, len(m.lines)-m.cursorLine-1)
	copy(tail, m.lines[m.cursorLine+1:])
	m.lines = append(m.lines[:m.cursorLine+1], append([]string{right}, tail...)...)
	m.cursorLine++
	m.cursorCol = 0
	m.clampViewOffset()
}

func (m *InputModel) backspace() {
	if m.cursorCol > 0 {
		line := m.lines[m.cursorLine]
		m.lines[m.cursorLine] = line[:m.cursorCol-1] + line[m.cursorCol:]
		m.cursorCol--
	} else if m.cursorLine > 0 {
		prev := m.lines[m.cursorLine-1]
		cur := m.lines[m.cursorLine]
		m.cursorCol = len(prev)
		m.lines[m.cursorLine-1] = prev + cur
		m.lines = append(m.lines[:m.cursorLine], m.lines[m.cursorLine+1:]...)
		m.cursorLine--
		m.clampViewOffset()
	}
}

func (m *InputModel) deleteForward() {
	line := m.lines[m.cursorLine]
	if m.cursorCol < len(line) {
		m.lines[m.cursorLine] = line[:m.cursorCol] + line[m.cursorCol+1:]
	} else if m.cursorLine < len(m.lines)-1 {
		m.lines[m.cursorLine] = line + m.lines[m.cursorLine+1]
		m.lines = append(m.lines[:m.cursorLine+1], m.lines[m.cursorLine+2:]...)
	}
}

func (m *InputModel) moveCursorLeft() {
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		m.cursorLine--
		m.cursorCol = len(m.lines[m.cursorLine])
		m.clampViewOffset()
	}
}

func (m *InputModel) moveCursorRight() {
	if m.cursorCol < len(m.lines[m.cursorLine]) {
		m.cursorCol++
	} else if m.cursorLine < len(m.lines)-1 {
		m.cursorLine++
		m.cursorCol = 0
		m.clampViewOffset()
	}
}

func (m *InputModel) insertCompletion(word string) {
	line := m.lines[m.cursorLine]
	left := line[:m.cursorCol]
	right := line[m.cursorCol:]
	lastSpace := strings.LastIndexAny(left, " \t")
	prefix := ""
	if lastSpace >= 0 {
		prefix = left[:lastSpace+1]
	}
	m.lines[m.cursorLine] = prefix + word + " " + right
	m.cursorCol = len(prefix) + len(word) + 1
}

func (m *InputModel) updateAutocomplete() {
	if !m.focused {
		return
	}
	var b strings.Builder
	for i := 0; i < m.cursorLine; i++ {
		b.WriteString(m.lines[i])
		b.WriteByte(' ')
	}
	if m.cursorLine < len(m.lines) {
		b.WriteString(m.lines[m.cursorLine][:m.cursorCol])
	}
	m.autocomplete.Update(b.String())
}

// clampViewOffset keeps cursorLine visible inside the fixed window.
func (m *InputModel) clampViewOffset() {
	if m.cursorLine < m.viewOffset {
		m.viewOffset = m.cursorLine
	}
	if m.cursorLine >= m.viewOffset+m.visibleLines {
		m.viewOffset = m.cursorLine - m.visibleLines + 1
	}
}

var (
	styleCursor = lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent)
)

func (m *InputModel) View() string {
	// render only the visible window of lines
	end := m.viewOffset + m.visibleLines
	if end > len(m.lines) {
		end = len(m.lines)
	}
	window := m.lines[m.viewOffset:end]

	var rendered []string
	for i, line := range window {
		li := i + m.viewOffset
		if li == m.cursorLine && m.focused {
			col := m.cursorCol
			if col > len(line) {
				col = len(line)
			}
			left := line[:col]
			right := line[col:]
			ch := " "
			rest := right
			if len(right) > 0 {
				runes := []rune(right)
				ch = string(runes[0])
				rest = string(runes[1:])
			}
			rendered = append(rendered, left+styleCursor.Render(ch)+rest)
		} else {
			rendered = append(rendered, line)
		}
	}

	// pad to exactly visibleLines so the box height never changes
	for len(rendered) < m.visibleLines {
		rendered = append(rendered, "")
	}

	content := strings.Join(rendered, "\n")

	borderW := m.width - 4
	if borderW < 10 {
		borderW = 10
	}
	// scroll indicator: right-align ↑N / ↓N inside the last content line.
	// This is the empty padding row when content < visibleLines, so it
	// never overwrites SQL text in the normal case.
	above := m.viewOffset
	below := len(m.lines) - (m.viewOffset + m.visibleLines)
	if below < 0 {
		below = 0
	}
	if above > 0 || below > 0 {
		var parts []string
		if above > 0 {
			parts = append(parts, "↑"+itoa(above))
		}
		if below > 0 {
			parts = append(parts, "↓"+itoa(below))
		}
		indStr := styleMuted.Render(strings.Join(parts, " "))
		indW := lipgloss.Width(indStr)

		// inner content width = borderW minus lipgloss Padding(0,1) on each side
		innerW := borderW - 2
		lastIdx := len(rendered) - 1
		existingW := lipgloss.Width(rendered[lastIdx])
		spaces := innerW - existingW - indW
		if spaces < 1 {
			spaces = 1
		}
		rendered[lastIdx] = rendered[lastIdx] + strings.Repeat(" ", spaces) + indStr
		// rebuild content after modifying last line
		content = strings.Join(rendered, "\n")
	}

	var border lipgloss.Style
	if m.focused {
		border = styleInputBorderFocused.Width(borderW)
	} else {
		border = styleInputBorderBlurred.Width(borderW)
	}
	return border.Render(content)
}

// AutocompleteView returns the autocomplete popup separately so it can be
// overlaid by the parent above the input box.
func (m *InputModel) AutocompleteView() string {
	if !m.focused || !m.autocomplete.IsVisible() {
		return ""
	}
	return m.autocomplete.View()
}
