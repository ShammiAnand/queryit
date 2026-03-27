package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/db"
)

type ViewMode int

const (
	ViewTable ViewMode = iota
	ViewExpanded
)

type ResultsModel struct {
	result      *db.ResultSet
	currentPage int
	currentRow  int
	currentCol  int // selected column index (absolute)
	mode        ViewMode
	pageSize    int
	width       int
	height      int
	focused     bool
	colOffset   int // first visible column index (horizontal scroll)

	// assigned by viewTable on each render (wired in the viewTable update task);
	// used by NextCol to know when to scroll the viewport
	lastRenderedLastCol int
}

func NewResultsModel(mode ViewMode, pageSize int) *ResultsModel {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &ResultsModel{mode: mode, pageSize: pageSize}
}

func (r *ResultsModel) SetResult(res *db.ResultSet) {
	r.result = res
	r.currentPage = 0
	r.currentRow = 0
	r.currentCol = 0
	r.colOffset = 0
}

func (r *ResultsModel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

func (r *ResultsModel) SetFocused(f bool) { r.focused = f }

func (r *ResultsModel) PageSize() int { return r.pageSize }

// SetPageSize re-paginates the existing result with the new page size.
func (r *ResultsModel) SetPageSize(n int) {
	if n < 1 {
		n = 1
	}
	if n > 500 {
		n = 500
	}
	r.pageSize = n
	if r.result == nil {
		return
	}
	// flatten and re-paginate in-place
	all := r.flatRows()
	r.result.Pages = paginate(all, r.pageSize)
	if len(r.result.Pages) == 0 {
		r.result.Pages = [][]db.Row{{}}
	}
	r.currentPage = 0
	r.currentRow = 0
}

func (r *ResultsModel) ChangePageSize(delta int) {
	r.SetPageSize(r.pageSize + delta)
}

func (r *ResultsModel) NextPage() {
	if r.result == nil {
		return
	}
	if r.currentPage < len(r.result.Pages)-1 {
		r.currentPage++
		r.currentRow = 0
	}
}

func (r *ResultsModel) PrevPage() {
	if r.currentPage > 0 {
		r.currentPage--
		r.currentRow = 0
	}
}

func (r *ResultsModel) NextRow() {
	if r.result == nil || len(r.result.Pages) == 0 {
		return
	}
	page := r.result.Pages[r.currentPage]
	if r.currentRow < len(page)-1 {
		r.currentRow++
	} else if r.currentPage < len(r.result.Pages)-1 {
		r.currentPage++
		r.currentRow = 0
	}
}

func (r *ResultsModel) PrevRow() {
	if r.currentRow > 0 {
		r.currentRow--
	} else if r.currentPage > 0 {
		r.currentPage--
		if r.result != nil && len(r.result.Pages) > r.currentPage {
			r.currentRow = len(r.result.Pages[r.currentPage]) - 1
		}
	}
}

func (r *ResultsModel) ScrollColRight() {
	if r.result == nil {
		return
	}
	if r.colOffset < len(r.result.Columns)-1 {
		r.colOffset++
	}
}

func (r *ResultsModel) ScrollColLeft() {
	if r.colOffset > 0 {
		r.colOffset--
	}
}

// NextCol moves the cell cursor one column right and scrolls the viewport if needed.
func (r *ResultsModel) NextCol() {
	if r.result == nil {
		return
	}
	if r.currentCol < len(r.result.Columns)-1 {
		r.currentCol++
	}
	// if the new currentCol is past the last rendered visible col, advance viewport
	if r.lastRenderedLastCol > 0 && r.currentCol >= r.lastRenderedLastCol {
		r.colOffset = r.currentCol
	}
}

// PrevCol moves the cell cursor one column left and scrolls the viewport if needed.
func (r *ResultsModel) PrevCol() {
	if r.currentCol > 0 {
		r.currentCol--
	}
	if r.currentCol < r.colOffset {
		r.colOffset = r.currentCol
	}
}

// CurrentCell returns the raw string value of the selected cell, or "" if none.
func (r *ResultsModel) CurrentCell() string {
	if r.result == nil || len(r.result.Pages) == 0 {
		return ""
	}
	page := r.result.Pages[r.currentPage]
	if r.currentRow >= len(page) {
		return ""
	}
	row := page[r.currentRow]
	if r.currentCol >= len(row) {
		return ""
	}
	return row[r.currentCol]
}

// CurrentRow returns the selected row as a db.Row, or nil if none.
func (r *ResultsModel) CurrentRow() db.Row {
	if r.result == nil || len(r.result.Pages) == 0 {
		return nil
	}
	page := r.result.Pages[r.currentPage]
	if r.currentRow >= len(page) {
		return nil
	}
	return page[r.currentRow]
}

func (r *ResultsModel) ToggleView() {
	if r.mode == ViewTable {
		r.mode = ViewExpanded
	} else {
		r.mode = ViewTable
	}
}

// View renders the results inside a bordered container whose colour reflects focus.
func (r *ResultsModel) View() string {
	inner := r.innerView()

	// inner height = height minus 2 border lines
	innerH := r.height - 2
	if innerH < 1 {
		innerH = 1
	}
	innerW := r.width - 4 // 2 border + 2 padding
	if innerW < 10 {
		innerW = 10
	}

	// pad / crop inner content to exact height so border is stable
	lines := strings.Split(inner, "\n")
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	lines = lines[:innerH]
	inner = strings.Join(lines, "\n")

	var borderStyle lipgloss.Style
	if r.focused {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1).
			Width(innerW)
	} else {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Width(innerW)
	}

	return borderStyle.Render(inner)
}

func (r *ResultsModel) innerView() string {
	if r.result == nil {
		hint := styleMuted.Render("Execute a query with F5 to see results")
		return "\n" + center(hint, r.width-6)
	}

	if r.result.IsError {
		return styleError.Render("Error: " + r.result.Message)
	}

	if r.result.Message != "" && len(r.result.Columns) == 0 {
		return styleSuccess.Render(r.result.Message)
	}

	if len(r.result.Pages) == 0 || (len(r.result.Pages) == 1 && len(r.result.Pages[0]) == 0) {
		return styleMuted.Render("Query returned 0 rows.")
	}

	if r.mode == ViewExpanded {
		return r.viewExpanded()
	}
	return r.viewTable()
}

func (r *ResultsModel) viewTable() string {
	page := r.result.Pages[r.currentPage]
	cols := r.result.Columns
	if len(cols) == 0 {
		return styleMuted.Render("No columns.")
	}

	innerW := r.width - 6
	if innerW < 10 {
		innerW = 40
	}

	const maxColW = 36
	colWidths := make([]int, len(cols))
	for i, c := range cols {
		colWidths[i] = len(c)
	}
	for _, row := range page {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	for i := range colWidths {
		if colWidths[i] > maxColW {
			colWidths[i] = maxColW
		}
	}

	// clamp colOffset
	if r.colOffset >= len(cols) {
		r.colOffset = len(cols) - 1
	}
	if r.colOffset < 0 {
		r.colOffset = 0
	}

	firstCol := r.colOffset
	lastCol := firstCol
	var totalW int
	for i := firstCol; i < len(cols); i++ {
		totalW += colWidths[i] + 3
		if totalW > innerW && i > firstCol {
			break
		}
		lastCol = i + 1
	}
	if lastCol <= firstCol {
		lastCol = firstCol + 1
	}
	if lastCol > len(cols) {
		lastCol = len(cols)
	}
	r.lastRenderedLastCol = lastCol // cache for NextCol

	// header
	var hcells []string
	for i := firstCol; i < lastCol; i++ {
		hcells = append(hcells, styleHeader.Width(colWidths[i]+2).Render(cols[i]))
	}
	header := strings.Join(hcells, styleMuted.Render("│"))
	sep := styleMuted.Render(strings.Repeat("─", innerW))

	var lines []string
	lines = append(lines, header, sep)

	for ri, row := range page {
		var cells []string
		for ci := firstCol; ci < lastCol; ci++ {
			rawVal := ""
			if ci < len(row) {
				rawVal = row[ci]
			}
			val := truncate(rawVal, colWidths[ci])

			// append [J] indicator for JSON cells without mangling content
			if isJSON(rawVal) {
				tag := "[J]"
				maxValW := colWidths[ci] - len(tag)
				if maxValW < 0 {
					maxValW = 0
				}
				val = truncate(rawVal, maxValW) + tag
			}

			// pick cell style: selected > alt row > normal
			var st lipgloss.Style
			if r.focused && ri == r.currentRow && ci == r.currentCol {
				st = styleCellSelected
			} else if ri%2 == 1 {
				st = styleCellAlt
			} else {
				st = styleCell
			}
			cells = append(cells, st.Width(colWidths[ci]+2).Render(val))
		}
		lines = append(lines, strings.Join(cells, styleMuted.Render("│")))
	}

	totalPages := len(r.result.Pages)
	colInfo := ""
	if len(cols) > lastCol-firstCol {
		colInfo = fmt.Sprintf("  │  cols %d-%d/%d  h/l navigate", firstCol+1, lastCol, len(cols))
	}
	pageStr := styleMuted.Render(fmt.Sprintf(
		"page %d/%d · %d rows/page · total %d  │  +/- page size  │  n/p next/prev%s",
		r.currentPage+1, totalPages, r.pageSize, r.result.Total, colInfo,
	))
	lines = append(lines, "", pageStr)

	return strings.Join(lines, "\n")
}

func (r *ResultsModel) viewExpanded() string {
	all := r.flatRows()
	if len(all) == 0 {
		return styleMuted.Render("No rows.")
	}

	globalRow := r.currentPage*r.pageSize + r.currentRow
	if globalRow >= len(all) {
		globalRow = len(all) - 1
	}

	row := all[globalRow]
	cols := r.result.Columns

	innerW := r.width - 6
	if innerW < 20 {
		innerW = 40
	}

	labelW := 0
	for _, c := range cols {
		if len(c) > labelW {
			labelW = len(c)
		}
	}
	if labelW > 30 {
		labelW = 30
	}

	header := stylePaneTitle.Render(fmt.Sprintf("Row %d of %d", globalRow+1, r.result.Total))
	sep := styleMuted.Render(strings.Repeat("─", innerW))

	var lines []string
	lines = append(lines, header, sep)

	for i, col := range cols {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		label := styleHeader.Width(labelW + 2).Render(col)
		valW := innerW - labelW - 5
		if valW < 1 {
			valW = 10
		}
		display := truncate(val, valW)
		if isJSON(val) {
			tag := " [J]"
			maxDisplayW := valW - len(tag)
			if maxDisplayW < 0 {
				maxDisplayW = 0
			}
			display = truncate(val, maxDisplayW) + styleMuted.Render(tag)
		}
		valStr := styleCell.Width(valW).Render(display)
		lines = append(lines, label+styleMuted.Render("│")+valStr)
	}

	lines = append(lines, "", styleMuted.Render(fmt.Sprintf(
		"row %d/%d  │  j/k navigate  │  v table view",
		globalRow+1, r.result.Total,
	)))

	return strings.Join(lines, "\n")
}

func (r *ResultsModel) flatRows() []db.Row {
	var all []db.Row
	for _, page := range r.result.Pages {
		all = append(all, page...)
	}
	return all
}

func paginate(rows []db.Row, size int) [][]db.Row {
	if size <= 0 {
		size = 20
	}
	var pages [][]db.Row
	for i := 0; i < len(rows); i += size {
		end := i + size
		if end > len(rows) {
			end = len(rows)
		}
		pages = append(pages, rows[i:end])
	}
	return pages
}

func center(s string, width int) string {
	sw := lipgloss.Width(s)
	if sw >= width {
		return s
	}
	pad := (width - sw) / 2
	return strings.Repeat(" ", pad) + s
}

// isJSON reports whether s is a valid JSON object or array.
// Checks first byte to avoid Unmarshal cost on obviously non-JSON strings.
func isJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if s[0] != '{' && s[0] != '[' {
		return false
	}
	var discard interface{}
	return json.Unmarshal([]byte(s), &discard) == nil
}
