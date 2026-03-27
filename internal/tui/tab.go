package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/cache"
	"github.com/shammianand/queryit/internal/config"
	"github.com/shammianand/queryit/internal/connection"
	"github.com/shammianand/queryit/internal/db"
)

type Focus int

const (
	FocusInput Focus = iota
	FocusRecent
	FocusResults
	FocusBrowser
)

// tabID is a unique identifier assigned once per TabModel at creation.
// All async messages carry the tabID of the tab that originated them so
// every other tab can discard them immediately.
type tabID uint64

var nextTabID tabID = 1

func allocTabID() tabID {
	id := nextTabID
	nextTabID++
	return id
}

// messages — each carries the originating tabID
type queryDoneMsg struct {
	id     tabID
	result *db.ResultSet
}
type connectDoneMsg struct {
	id   tabID
	conn *connection.Conn
	err  error
}
type schemaRefreshDoneMsg struct {
	id   tabID
	snap *cache.SchemaSnapshot
	err  error
}
type reconnectMsg struct{ id tabID }

type TabModel struct {
	id          tabID
	profileName string
	profile     *config.Profile
	settings    config.Settings

	conn        *connection.Conn
	driver      db.Driver
	executor    *db.Executor
	schemaCache *cache.SchemaCache

	input     *InputModel
	results   *ResultsModel
	recent    *RecentModel
	statusBar *StatusBar
	history   *HistoryModel
	browser    *SchemaBrowser
	jsonViewer *JSONViewerModal
	copyMenu   *CopyMenuModal

	focus  Focus
	width  int
	height int

	cancelQuery    context.CancelFunc
	queryRunning   bool
	sessionQueries []string
}

func NewTab(profileName string, profile *config.Profile, settings config.Settings) *TabModel {
	schemaPath := config.CachePath(profileName)
	historyPath := config.DataPath(profileName)

	sc := cache.NewSchemaCache(schemaPath)
	_ = sc.Load()

	t := &TabModel{
		id:          allocTabID(),
		profileName: profileName,
		profile:     profile,
		settings:    settings,
		schemaCache: sc,
		results:     NewResultsModel(ViewTable, settings.PageSize),
		recent:      NewRecentModel(),
		statusBar:   NewStatusBar(profileName),
		history:     NewHistoryModel(historyPath, settings.HistorySize),
		browser:     NewSchemaBrowser(sc),
		jsonViewer:  &JSONViewerModal{},
		copyMenu:    &CopyMenuModal{},
		focus:       FocusInput,
	}
	t.input = NewInputModel(sc)
	t.input.SetFocused(true)
	return t
}

func (t *TabModel) InitCmds() tea.Cmd {
	return nil
}

func (t *TabModel) Connect() tea.Cmd {
	t.statusBar.SetConnecting()
	id := t.id
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		conn, err := connection.Connect(ctx, t.profileName, t.profile, t.settings.PageSize)
		return connectDoneMsg{id: id, conn: conn, err: err}
	}
}

const (
	inputVisibleLines = 4
	inputBoxH         = inputVisibleLines + 2 // +2 for border
	statusH           = 1
)

func (t *TabModel) SetSize(w, h int) {
	t.width = w
	t.height = h

	// browser takes a fixed left column when open
	bw := t.browser.Width() // 0 when collapsed, browserWidth when open
	mainW := w - bw

	t.browser.SetSize(bw, h)
	t.recent.SetWidth(mainW)
	recentH := t.recent.Height()
	resultsH := h - inputBoxH - statusH - recentH
	if resultsH < 3 {
		resultsH = 3
	}
	t.input.SetSize(mainW, inputVisibleLines)
	t.results.SetSize(mainW, resultsH)
	t.statusBar.SetWidth(w)
	t.history.SetSize(mainW-4, h-4)
	t.jsonViewer.SetSize(w, h)
}

func (t *TabModel) Update(msg tea.Msg) (*TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case connectDoneMsg:
		if msg.id != t.id {
			return t, nil
		}
		if msg.err != nil {
			t.statusBar.SetDisconnected()
			t.statusBar.SetMessage(styleError.Render("connect error: " + msg.err.Error()))
			return t, nil
		}
		t.conn = msg.conn
		t.driver = msg.conn.Driver
		t.executor = db.NewExecutor(t.driver, t.settings.PageSize)
		t.statusBar.SetConnected(t.profileName)
		return t, t.refreshSchema()

	case schemaRefreshDoneMsg:
		if msg.id != t.id {
			return t, nil
		}
		if msg.err == nil && msg.snap != nil {
			_ = t.schemaCache.Set(msg.snap)
			t.browser.refreshTables()
		}
		return t, nil

	case queryDoneMsg:
		if msg.id != t.id {
			return t, nil
		}
		t.queryRunning = false
		t.cancelQuery = nil
		if msg.result != nil {
			t.results.SetResult(msg.result)
			if msg.result.IsError {
				t.statusBar.SetMessage(styleError.Render(truncateMsg(msg.result.Message, 60)))
			} else {
				t.statusBar.SetQueryResult(msg.result.Total, msg.result.Elapsed)
			}
		}
		t.recent.SetEntries(t.sessionQueries)
		t.SetSize(t.width, t.height)
		return t, nil

	case reconnectMsg:
		if msg.id != t.id {
			return t, nil
		}
		return t, t.Connect()

	case tea.KeyMsg:
		return t.handleKey(msg)
	}

	return t, nil
}

func truncateMsg(s string, max int) string {
	// single-line version of the message
	s = strings.SplitN(s, "\n", 2)[0]
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func (t *TabModel) setFocus(f Focus) {
	t.focus = f
	t.input.SetFocused(f == FocusInput)
	t.recent.SetFocused(f == FocusRecent)
	t.results.SetFocused(f == FocusResults)
	t.browser.SetFocused(f == FocusBrowser)
	t.SetSize(t.width, t.height)
}

func (t *TabModel) handleKey(msg tea.KeyMsg) (*TabModel, tea.Cmd) {
	k := msg.String()

	// history search overlay captures everything
	if t.history.IsVisible() {
		switch k {
		case "ctrl+r", "esc":
			t.history.Hide()
		case "enter":
			if q := t.history.Accept(); q != "" {
				t.input.SetValue(q)
			}
			t.setFocus(FocusInput)
		case "j", "down":
			t.history.Next()
		case "k", "up":
			t.history.Prev()
		case "backspace":
			t.history.Backspace()
		default:
			if len(msg.Runes) > 0 {
				t.history.TypeChar(string(msg.Runes))
			}
		}
		return t, nil
	}

	// JSON viewer captures all keys when open
	if t.jsonViewer.IsVisible() {
		switch k {
		case "j", "down":
			t.jsonViewer.ScrollDown()
		case "k", "up":
			t.jsonViewer.ScrollUp()
		case "y":
			if err := copyToClipboard(t.jsonViewer.RawJSON()); err != nil {
				t.statusBar.SetMessage(styleError.Render("clipboard: " + err.Error()))
			} else {
				t.statusBar.SetMessage(styleSuccess.Render("JSON copied to clipboard"))
			}
			t.jsonViewer.Hide()
		case "esc":
			t.jsonViewer.Hide()
		}
		return t, nil
	}

	// copy menu captures all keys when open
	if t.copyMenu.IsVisible() {
		t.copyMenu.Hide() // close on any key; action keys handled below
		switch k {
		case "c":
			cell := t.results.CurrentCell()
			if err := copyToClipboard(cell); err != nil {
				t.statusBar.SetMessage(styleError.Render("clipboard: " + err.Error()))
			} else {
				t.statusBar.SetMessage(styleSuccess.Render("cell copied to clipboard"))
			}
		case "r":
			row := t.results.CurrentRow()
			if row == nil {
				return t, nil
			}
			if err := copyToClipboard(rowToCSV(row)); err != nil {
				t.statusBar.SetMessage(styleError.Render("clipboard: " + err.Error()))
			} else {
				t.statusBar.SetMessage(styleSuccess.Render("row copied to clipboard"))
			}
		case "t":
			rs := t.results.ResultSet()
			if rs == nil {
				return t, nil
			}
			if err := copyToClipboard(rowsToCSV(rs.Columns, rs.Pages)); err != nil {
				t.statusBar.SetMessage(styleError.Render("clipboard: " + err.Error()))
			} else {
				t.statusBar.SetMessage(styleSuccess.Render("table copied to clipboard"))
			}
		case "e":
			rs := t.results.ResultSet()
			if rs == nil {
				return t, nil
			}
			path, err := exportToFile(rs.Columns, rs.Pages)
			if err != nil {
				t.statusBar.SetMessage(styleError.Render("export: " + err.Error()))
			} else {
				t.statusBar.SetMessage(styleSuccess.Render("exported to " + path))
			}
		}
		return t, nil
	}

	// ctrl+o toggles the schema browser
	if k == "ctrl+o" {
		t.browser.Toggle()
		t.SetSize(t.width, t.height)
		if !t.browser.IsCollapsed() {
			t.setFocus(FocusBrowser)
		} else if t.focus == FocusBrowser {
			t.setFocus(FocusInput)
		}
		return t, nil
	}

	// ctrl+r → history search overlay (any focus)
	if k == "ctrl+r" {
		t.history.Show()
		return t, nil
	}

	// ctrl+c: cancel or clear
	if k == "ctrl+c" {
		if t.queryRunning {
			if t.cancelQuery != nil {
				t.cancelQuery()
			}
			t.queryRunning = false
			t.statusBar.SetMessage(styleMuted.Render("cancelled"))
		} else {
			t.input.Clear()
		}
		return t, nil
	}

	// esc cycles focus: Browser → Results → Recent → Input → (Browser if open else Results)
	if k == "esc" {
		switch t.focus {
		case FocusBrowser:
			// esc inside detail → back to list; esc in list → move focus on
			if t.browser.mode == bmDetail {
				t.browser.mode = bmList
				return t, nil
			}
			// fall through to next focus: Input
			t.setFocus(FocusInput)
		case FocusInput:
			if t.recent.HasEntries() {
				t.setFocus(FocusRecent)
			} else {
				t.setFocus(FocusResults)
			}
		case FocusRecent:
			t.setFocus(FocusResults)
		case FocusResults:
			if !t.browser.IsCollapsed() {
				t.setFocus(FocusBrowser)
			} else {
				t.setFocus(FocusInput)
			}
		}
		return t, nil
	}

	// --- focus-specific handling ---

	switch t.focus {
	case FocusInput:
		consumed, execRequested, clearRequested := t.input.Update(msg)
		if clearRequested {
			t.input.Clear()
			return t, nil
		}
		if execRequested {
			return t, t.executeQuery()
		}
		_ = consumed

	case FocusRecent:
		switch k {
		case "j", "down":
			t.recent.Next()
		case "k", "up":
			t.recent.Prev()
		case "enter":
			if q := t.recent.Accept(); q != "" {
				t.input.SetValue(q)
				t.setFocus(FocusInput)
			}
		case " ": // space toggles collapse
			t.recent.Toggle()
			t.SetSize(t.width, t.height)
		}

	case FocusBrowser:
		// browser handles esc internally (detail→list) above;
		// space pastes table name into input
		paste := t.browser.HandleKey(k)
		if paste != "" {
			t.input.SetValue(paste)
			t.setFocus(FocusInput)
		}

	case FocusResults:
		switch k {
		case "enter":
			cell := t.results.CurrentCell()
			if isJSON(cell) {
				t.jsonViewer.Show(cell, t.width, t.height)
			} else if cell != "" {
				t.statusBar.SetMessage(styleMuted.Render("not a JSON cell"))
			}
		case "y":
			if t.results.ResultSet() != nil {
				t.copyMenu.Show(t.width, t.height)
			}
		case "n":
			t.results.NextPage()
		case "p":
			t.results.PrevPage()
		case "v":
			t.results.ToggleView()
		case "j", "down":
			t.results.NextRow()
		case "k", "up":
			t.results.PrevRow()
		case "l":
			t.results.NextCol()
		case "h":
			t.results.PrevCol()
		case "+", "=":
			t.results.ChangePageSize(+10)
			if t.executor != nil {
				t.executor.SetPageSize(t.results.PageSize())
			}
		case "-":
			t.results.ChangePageSize(-10)
			if t.executor != nil {
				t.executor.SetPageSize(t.results.PageSize())
			}
		case "r":
			if t.conn == nil {
				return t, t.Connect()
			}
		}
	}

	return t, nil
}

func (t *TabModel) executeQuery() tea.Cmd {
	raw := strings.TrimSpace(t.input.Value())
	if raw == "" {
		return nil
	}

	// persist to disk history for ctrl+r search
	_ = t.history.Append(raw)
	// prepend to session-only list (newest first)
	t.sessionQueries = append([]string{raw}, t.sessionQueries...)
	t.input.SetHistory(t.sessionQueries)
	t.recent.SetEntries(t.sessionQueries)
	t.SetSize(t.width, t.height)
	t.input.Clear()

	// handle backslash commands synchronously
	if strings.HasPrefix(raw, `\`) {
		result := t.handleBackslash(raw)
		t.results.SetResult(result)
		return nil
	}

	if t.driver == nil {
		t.statusBar.SetMessage(styleError.Render("not connected"))
		return nil
	}

	t.queryRunning = true
	t.statusBar.SetMessage(styleStatusConnecting.Render("running…"))

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(t.settings.QueryTimeout)*time.Second)
	t.cancelQuery = cancel

	query := raw
	driver := t.driver
	id := t.id

	return func() tea.Msg {
		defer cancel()
		res, err := driver.Execute(ctx, query)
		if err != nil {
			return queryDoneMsg{id: id, result: &db.ResultSet{IsError: true, Message: err.Error()}}
		}
		return queryDoneMsg{id: id, result: res}
	}
}

func (t *TabModel) handleBackslash(cmd string) *db.ResultSet {
	if t.driver == nil {
		return &db.ResultSet{IsError: true, Message: "not connected"}
	}
	ctx := context.Background()

	driverName := t.driver.DriverName()

	switch {
	case cmd == `\dt`:
		switch driverName {
		case "mysql":
			return t.runQuery(ctx, "SELECT TABLE_NAME, TABLE_TYPE, ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() ORDER BY TABLE_NAME")
		case "sqlite":
			return t.runQuery(ctx, "SELECT name, type FROM sqlite_master WHERE type IN ('table','view') ORDER BY name")
		default:
			return t.runQuery(ctx, "SELECT schemaname, tablename, tableowner FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema') ORDER BY schemaname, tablename")
		}
	case strings.HasPrefix(cmd, `\d `):
		table := strings.TrimSpace(cmd[3:])
		switch driverName {
		case "mysql":
			return t.runQuery(ctx, fmt.Sprintf("DESCRIBE `%s`", table))
		case "sqlite":
			return t.runQuery(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
		default:
			return t.runQuery(ctx, fmt.Sprintf(
				"SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name = '%s' ORDER BY ordinal_position", table))
		}
	case cmd == `\dn`:
		switch driverName {
		case "mysql":
			return t.runQuery(ctx, "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA ORDER BY SCHEMA_NAME")
		case "sqlite":
			return t.runQuery(ctx, "PRAGMA database_list")
		default:
			return t.runQuery(ctx, "SELECT schema_name, schema_owner FROM information_schema.schemata ORDER BY schema_name")
		}
	case cmd == `\di`:
		switch driverName {
		case "mysql":
			return t.runQuery(ctx, "SELECT TABLE_NAME, INDEX_NAME, NON_UNIQUE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() GROUP BY TABLE_NAME, INDEX_NAME, NON_UNIQUE ORDER BY TABLE_NAME, INDEX_NAME")
		case "sqlite":
			return t.runQuery(ctx, "SELECT name, tbl_name FROM sqlite_master WHERE type='index' ORDER BY tbl_name, name")
		default:
			return t.runQuery(ctx, "SELECT schemaname, tablename, indexname FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog','information_schema') ORDER BY schemaname, tablename, indexname")
		}
	case cmd == `\df`:
		switch driverName {
		case "mysql":
			return t.runQuery(ctx, "SELECT ROUTINE_NAME, ROUTINE_TYPE, DTD_IDENTIFIER FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = DATABASE() ORDER BY ROUTINE_NAME")
		case "sqlite":
			return &db.ResultSet{Message: "SQLite does not support stored functions."}
		default:
			return t.runQuery(ctx, "SELECT n.nspname AS schema, p.proname AS name, pg_get_function_result(p.oid) AS returns FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') ORDER BY schema, name")
		}
	case cmd == `\refresh`:
		go func() {
			snap, err := t.refreshSchemaSync(ctx)
			if err == nil && snap != nil {
				_ = t.schemaCache.Set(snap)
			}
		}()
		return &db.ResultSet{Message: "Schema cache refresh started in background."}
	}

	return &db.ResultSet{IsError: true, Message: fmt.Sprintf("unknown command: %s", cmd)}
}

func (t *TabModel) runQuery(ctx context.Context, q string) *db.ResultSet {
	res, err := t.driver.Execute(ctx, q)
	if err != nil {
		return &db.ResultSet{IsError: true, Message: err.Error()}
	}
	return res
}

func (t *TabModel) refreshSchema() tea.Cmd {
	driver := t.driver
	sc := t.schemaCache
	profileName := t.profileName
	id := t.id
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		snap, err := driver.Introspect(ctx)
		if err != nil {
			return schemaRefreshDoneMsg{id: id, err: err}
		}
		snap.Profile = profileName
		_ = sc.Set(snap)
		return schemaRefreshDoneMsg{id: id, snap: snap}
	}
}

func (t *TabModel) refreshSchemaSync(ctx context.Context) (*cache.SchemaSnapshot, error) {
	snap, err := t.driver.Introspect(ctx)
	if err != nil {
		return nil, err
	}
	snap.Profile = t.profileName
	snap.RefreshedAt = time.Now()
	return snap, nil
}

func (t *TabModel) Close() {
	if t.cancelQuery != nil {
		t.cancelQuery()
	}
	if t.conn != nil {
		t.conn.Close(context.Background())
	}
}

func (t *TabModel) View() string {
	// full-screen overlays take priority
	if t.jsonViewer.IsVisible() {
		return lipgloss.Place(t.width, t.height,
			lipgloss.Center, lipgloss.Center,
			t.jsonViewer.View(),
		)
	}
	if t.copyMenu.IsVisible() {
		return lipgloss.Place(t.width, t.height,
			lipgloss.Center, lipgloss.Center,
			t.copyMenu.View(),
		)
	}

	bw := t.browser.Width()
	mainW := t.width - bw

	recentH := t.recent.Height()
	resultsH := t.height - inputBoxH - statusH - recentH
	if resultsH < 3 {
		resultsH = 3
	}

	// autocomplete floats above input inside the results area
	acView := t.input.AutocompleteView()
	var resultsContent string
	if acView != "" {
		acLines := strings.Count(acView, "\n") + 1
		innerH := resultsH - acLines
		if innerH < 1 {
			innerH = 1
		}
		inner := lipgloss.NewStyle().Height(innerH).MaxHeight(innerH).Width(mainW).Render(t.results.View())
		resultsContent = lipgloss.JoinVertical(lipgloss.Left, inner, acView)
	} else {
		resultsContent = lipgloss.NewStyle().Height(resultsH).MaxHeight(resultsH).Width(mainW).Render(t.results.View())
	}

	// history overlay
	if t.history.IsVisible() {
		mainCol := lipgloss.JoinVertical(lipgloss.Left,
			resultsContent,
			t.history.View(),
			t.statusBar.View(),
		)
		if bw > 0 {
			return lipgloss.JoinHorizontal(lipgloss.Top, t.browser.View(), mainCol)
		}
		return mainCol
	}

	// normal layout
	mainParts := []string{resultsContent}
	if recentH > 0 {
		mainParts = append(mainParts, t.recent.View())
	}
	mainParts = append(mainParts, t.input.View(), t.statusBar.View())
	mainCol := lipgloss.JoinVertical(lipgloss.Left, mainParts...)

	if bw > 0 {
		return lipgloss.JoinHorizontal(lipgloss.Top, t.browser.View(), mainCol)
	}
	return mainCol
}
