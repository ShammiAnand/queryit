package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/cache"
)

// SchemaBrowser is a collapsible left panel (ctrl+o to toggle).
//
// List mode  — all tables, scrolling viewport, j/k to navigate.
//              enter → detail mode   space → paste table name into input.
// Detail mode — columns + indexes for the selected table, j/k to scroll.
//              esc → back to list.

const browserWidth = 50

type browserMode int

const (
	bmList browserMode = iota
	bmDetail
)

type SchemaBrowser struct {
	schema    *cache.SchemaCache
	collapsed bool
	focused   bool
	mode      browserMode
	height    int // full tab height

	// list state
	tables  []cache.Table
	listIdx int // absolute cursor over all tables
	scroll  int // index of first visible row in list

	// detail state
	detailTable  cache.Table
	detailScroll int
}

func NewSchemaBrowser(schema *cache.SchemaCache) *SchemaBrowser {
	return &SchemaBrowser{schema: schema, collapsed: true}
}

func (b *SchemaBrowser) SetSize(_, h int)  { b.height = h }
func (b *SchemaBrowser) SetFocused(f bool) { b.focused = f }
func (b *SchemaBrowser) IsCollapsed() bool { return b.collapsed }

func (b *SchemaBrowser) Width() int {
	if b.collapsed {
		return 0
	}
	return browserWidth
}

func (b *SchemaBrowser) Toggle() {
	b.collapsed = !b.collapsed
	if !b.collapsed {
		b.refreshTables()
	}
}

func (b *SchemaBrowser) refreshTables() {
	snap := b.schema.Get()
	if snap == nil {
		b.tables = nil
		return
	}
	b.tables = snap.Tables
	if b.listIdx >= len(b.tables) && len(b.tables) > 0 {
		b.listIdx = len(b.tables) - 1
	}
	b.clampScroll()
}

// maxListRows = usable rows inside the border for the table list.
func (b *SchemaBrowser) maxListRows() int {
	// border(2) + title(1) + sep(1) + footer(1) = 5 fixed rows
	n := b.height - 5
	if n < 1 {
		n = 1
	}
	return n
}

func (b *SchemaBrowser) clampScroll() {
	max := b.maxListRows()
	if b.listIdx < b.scroll {
		b.scroll = b.listIdx
	}
	if b.listIdx >= b.scroll+max {
		b.scroll = b.listIdx - max + 1
	}
	if b.scroll < 0 {
		b.scroll = 0
	}
}

// HandleKey returns a non-empty string when the caller should paste that
// string into the query input.
func (b *SchemaBrowser) HandleKey(key string) string {
	if b.mode == bmDetail {
		return b.handleDetailKey(key)
	}
	return b.handleListKey(key)
}

func (b *SchemaBrowser) handleListKey(key string) string {
	switch key {
	case "j", "down":
		if b.listIdx < len(b.tables)-1 {
			b.listIdx++
			b.clampScroll()
		}
	case "k", "up":
		if b.listIdx > 0 {
			b.listIdx--
			b.clampScroll()
		}
	case "g":
		b.listIdx = 0
		b.scroll = 0
	case "G":
		b.listIdx = len(b.tables) - 1
		b.clampScroll()
	case "enter", "l", "right":
		if b.listIdx < len(b.tables) {
			b.detailTable = b.tables[b.listIdx]
			b.detailScroll = 0
			b.mode = bmDetail
		}
	case " ":
		if b.listIdx < len(b.tables) {
			return b.tables[b.listIdx].Name
		}
	}
	return ""
}

func (b *SchemaBrowser) handleDetailKey(key string) string {
	lines := b.buildDetailLines()
	maxRows := b.maxDetailRows()
	switch key {
	case "j", "down":
		if b.detailScroll < len(lines)-maxRows {
			b.detailScroll++
		}
	case "k", "up":
		if b.detailScroll > 0 {
			b.detailScroll--
		}
	case "g":
		b.detailScroll = 0
	case "G":
		if len(lines) > maxRows {
			b.detailScroll = len(lines) - maxRows
		}
	case "esc", "h", "left":
		b.mode = bmList
	case " ":
		return b.detailTable.Name
	}
	return ""
}

func (b *SchemaBrowser) maxDetailRows() int {
	// border(2) + title(1) + sep(1) + footer(1) = 5
	n := b.height - 5
	if n < 1 {
		n = 1
	}
	return n
}

// ── detail content ────────────────────────────────────────────────────────────

type dlKind int

const (
	dlSectionHdr dlKind = iota
	dlRow
	dlBlank
)

type dline struct {
	kind dlKind
	text string
}

func (b *SchemaBrowser) buildDetailLines() []dline {
	t := b.detailTable
	snap := b.schema.Get()
	var out []dline

	out = append(out, dline{dlSectionHdr, "COLUMNS"})
	for _, col := range t.Columns {
		null := ""
		if col.Nullable {
			null = " null"
		}
		// store plain text; rendering styles applied in viewDetail
		out = append(out, dline{dlRow, col.Name + "  " + col.Type + null})
	}

	if snap != nil {
		var idxs []cache.Index
		for _, idx := range snap.Indexes {
			if idx.Table == t.Name && idx.Schema == t.Schema {
				idxs = append(idxs, idx)
			}
		}
		if len(idxs) > 0 {
			out = append(out, dline{dlBlank, ""})
			out = append(out, dline{dlSectionHdr, "INDEXES"})
			for _, idx := range idxs {
				out = append(out, dline{dlRow, idx.Name})
			}
		}
	}

	return out
}

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleBrowserBorderFocused = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(colorAccent)

	styleBrowserBorderBlurred = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(colorBorder)

	styleBrowserHeader = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleBrowserSec    = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	styleBrowserSel    = lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent)
	styleBrowserSelDim = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleBrowserNormal = lipgloss.NewStyle().Foreground(colorFg)
	styleBrowserMuted  = lipgloss.NewStyle().Foreground(colorMuted)
	styleBrowserPart   = lipgloss.NewStyle().Foreground(colorYellow)
)

// ── View ──────────────────────────────────────────────────────────────────────

func (b *SchemaBrowser) View() string {
	if b.collapsed {
		return ""
	}
	if b.mode == bmDetail {
		return b.viewDetail()
	}
	return b.viewList()
}

func (b *SchemaBrowser) viewList() string {
	innerW := browserWidth - 2 // subtract border cols
	b.refreshTables()

	maxRows := b.maxListRows()

	// title
	title := styleBrowserHeader.Render(fmt.Sprintf("Tables (%d)", len(b.tables)))

	// visible slice
	end := b.scroll + maxRows
	if end > len(b.tables) {
		end = len(b.tables)
	}

	var rows []string
	rows = append(rows, title)
	rows = append(rows, styleBrowserMuted.Render(strings.Repeat("─", innerW)))

	for i := b.scroll; i < end; i++ {
		tbl := b.tables[i]
		isSelected := i == b.listIdx

		// All arithmetic in plain-text space — no ANSI codes in width calc.
		partPrefix := ""
		if tbl.Partitioned {
			partPrefix = "⊕ " // 2 plain chars
		}
		colCntStr := fmt.Sprintf("%d", len(tbl.Columns))

		// innerW = total cols available inside the border
		// layout: partPrefix + name + padding + colCntStr
		// leave 1 space between name and count
		nameMaxW := innerW - len(partPrefix) - 1 - len(colCntStr)
		if nameMaxW < 4 {
			nameMaxW = 4
		}
		name := truncate(tbl.Name, nameMaxW)

		usedW := len(partPrefix) + len(name) + len(colCntStr)
		pad := innerW - usedW
		if pad < 1 {
			pad = 1
		}
		// final plain string, exactly innerW wide
		plain := partPrefix + name + strings.Repeat(" ", pad) + colCntStr

		switch {
		case isSelected && b.focused:
			rows = append(rows, styleBrowserSel.Width(innerW).Render(plain))
		case isSelected:
			rows = append(rows, styleBrowserSelDim.Width(innerW).Render(plain))
		default:
			// colour the col-count portion muted, rest normal
			leftPart := partPrefix + name + strings.Repeat(" ", pad)
			if tbl.Partitioned {
				leftPart = styleBrowserPart.Render("⊕ ") + name + strings.Repeat(" ", pad)
			}
			rows = append(rows, leftPart+styleBrowserMuted.Render(colCntStr))
		}
	}

	if len(b.tables) == 0 {
		rows = append(rows, styleBrowserMuted.Render(" loading…"))
		rows = append(rows, styleBrowserMuted.Render(" (run \\refresh)"))
	}

	// pad to full height so the panel fills the window
	totalContentH := b.height - 2 // minus border
	for len(rows) < totalContentH-1 {
		rows = append(rows, "")
	}

	// footer with scroll position
	scrollInfo := ""
	if len(b.tables) > maxRows {
		pct := 0
		if len(b.tables)-maxRows > 0 {
			pct = 100 * b.scroll / (len(b.tables) - maxRows)
		}
		scrollInfo = fmt.Sprintf(" %d%%", pct)
	}
	footer := styleBrowserMuted.Render("j/k·nav  enter·detail  spc·paste" + scrollInfo)
	rows = append(rows, footer)

	content := strings.Join(rows, "\n")
	border := styleBrowserBorderBlurred
	if b.focused {
		border = styleBrowserBorderFocused
	}
	return border.Width(innerW).Height(b.height - 2).Render(content)
}

func (b *SchemaBrowser) viewDetail() string {
	innerW := browserWidth - 2
	lines := b.buildDetailLines()
	maxRows := b.maxDetailRows()

	// clamp scroll
	if len(lines) <= maxRows {
		b.detailScroll = 0
	} else if b.detailScroll > len(lines)-maxRows {
		b.detailScroll = len(lines) - maxRows
	}
	if b.detailScroll < 0 {
		b.detailScroll = 0
	}

	t := b.detailTable
	partPrefix := ""
	partStyled := ""
	if t.Partitioned {
		partPrefix = "⊕ " // 2 plain chars for width accounting
		partStyled = styleBrowserPart.Render("⊕ ")
	}
	nameMaxW := innerW - len(partPrefix) - 1
	if nameMaxW < 4 {
		nameMaxW = 4
	}
	titleStr := partStyled + styleBrowserHeader.Render(truncate(t.Name, nameMaxW))

	end := b.detailScroll + maxRows
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[b.detailScroll:end]

	var rows []string
	rows = append(rows, titleStr)
	rows = append(rows, styleBrowserMuted.Render(strings.Repeat("─", innerW)))

	for _, dl := range visible {
		switch dl.kind {
		case dlSectionHdr:
			rows = append(rows, styleBrowserSec.Render(dl.text))
		case dlBlank:
			rows = append(rows, "")
		case dlRow:
			// dl.text is plain "name  type [null]"
			// split on double-space so we can colour type separately
			var rendered string
			if parts := strings.SplitN(dl.text, "  ", 2); len(parts) == 2 {
				colName := truncate(parts[0], innerW/2)
				typeStr := truncate(parts[1], innerW-len(colName)-3)
				pad := innerW - 1 - len(colName) - 1 - len(typeStr)
				if pad < 1 {
					pad = 1
				}
				rendered = styleBrowserNormal.Render(colName) +
					strings.Repeat(" ", pad) +
					styleBrowserMuted.Render(typeStr)
			} else {
				rendered = styleBrowserNormal.Render(truncate(dl.text, innerW-1))
			}
			rows = append(rows, " "+rendered)
		}
	}

	// pad to full height
	totalContentH := b.height - 2
	for len(rows) < totalContentH-1 {
		rows = append(rows, "")
	}

	scrollInfo := ""
	if len(lines) > maxRows {
		pct := 0
		if len(lines)-maxRows > 0 {
			pct = 100 * b.detailScroll / (len(lines) - maxRows)
		}
		scrollInfo = fmt.Sprintf(" %d%%", pct)
	}
	footer := styleBrowserMuted.Render("esc·back  j/k·scroll  spc·paste" + scrollInfo)
	rows = append(rows, footer)

	content := strings.Join(rows, "\n")
	border := styleBrowserBorderBlurred
	if b.focused {
		border = styleBrowserBorderFocused
	}
	return border.Width(innerW).Height(b.height - 2).Render(content)
}
