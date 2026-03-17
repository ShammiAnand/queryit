# AGENTS.md

## What this is

`queryit` is a keyboard-driven terminal UI (TUI) for relational databases, written in Go.
Supports PostgreSQL, MySQL/MariaDB, and SQLite. Uses Bubble Tea (Elm architecture) for the UI,
a driver interface for database backends, and x/crypto/ssh for SSH bastion tunneling.

## Tech stack

- **Go 1.25+**, module `github.com/shammianand/queryit`
- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, `charmbracelet/bubbles`
- **PostgreSQL**: `jackc/pgx/v5` with `pgxpool` (pool per tab, max 3 conns)
- **MySQL**: `go-sql-driver/mysql` via `database/sql`
- **SQLite**: `modernc.org/sqlite` (pure Go, no CGo) via `database/sql`
- **CLI**: `spf13/cobra`
- **Config**: `gopkg.in/yaml.v3`, stored at `~/.config/queryit/config.yaml`

## Repository layout

```
main.go                          entrypoint; injects ldflags version into cmd
cmd/
  root.go                        cobra root; --profile flag; launches TUI
  profile.go                     profile list/add/remove subcommands
internal/
  config/config.go               YAML config load/save; XDG paths; $ENV_VAR password expansion;
                                 Profile.DriverName() returns "postgres" when driver field is absent
  connection/
    manager.go                   dispatches Connect() by driver name; returns *Conn{Driver, tunnel}
    tunnel.go                    x/crypto/ssh local port forwarding (used by postgres + mysql)
  db/
    driver.go                    Driver interface: Execute, Introspect, Ping, Close, DriverName
    executor.go                  Executor wrapper (SetPageSize propagation); shared Row/ResultSet types;
                                 formatValue, paginate helpers
    postgres.go                  PostgresDriver — pgxpool.Acquire per call
    introspect_postgres.go       Postgres-specific information_schema + pg_catalog queries;
                                 filters partition children; tags partition roots
    mysql.go                     MySQLDriver + introspectMySQL — database/sql, information_schema
    sqlite.go                    SQLiteDriver + introspectSQLite — sqlite_master + PRAGMA table_info
  cache/schema.go                in-memory SchemaSnapshot (tables, columns, indexes, functions);
                                 JSON disk cache per profile; Table.Partitioned bool
  completion/
    engine.go                    context-aware fuzzy suggest (tables/columns only, no keywords)
    keywords.go                  SQL keyword list; backslash commands
  tui/
    app.go                       top-level Bubble Tea model; ProfileSelector list; ProfileForm modal
    tab.go                       per-tab model; Focus enum; key routing; query execution via Driver
    input.go                     multi-line editor; history cycling via up/down; autocomplete hook
    results.go                   table + expanded views; colOffset horizontal scroll; pagination
    schemabrowser.go             left panel (ctrl+o); 50 cols; list + detail modes
    recent.go                    session-only recent queries; collapses when unfocused
    history.go                   JSONL disk history; searchable overlay (ctrl+r)
    autocomplete.go              dropdown; up/down navigate; enter accept
    tabbar.go                    tab strip; active tab shows close marker
    statusbar.go                 connection state; row count; elapsed time
    styles.go                    Catppuccin Mocha lipgloss theme; all shared styles
    keys.go                      key binding definitions
```

## Architecture

Each open database connection is a **TabModel**. Tabs are independent — separate driver,
schema cache, session history, and focus state.

### Driver interface

```go
type Driver interface {
    Execute(ctx, query) (*ResultSet, error)
    Introspect(ctx) (*cache.SchemaSnapshot, error)
    Ping(ctx) error
    Close()
    DriverName() string
}
```

`connection.Connect()` reads `profile.DriverName()` and returns the appropriate implementation.
`tab.go` holds a `db.Driver` field and calls it directly — no pgx types leak into the TUI layer.

### Focus cycle (within a tab)

`esc` rotates: **Input → Recent (if entries) → Results → Browser (if open) → Input**

Panels skipped if not available. `esc` inside browser detail goes back to list before cycling out.

### Connection flow

1. `ctrl+t` → ProfileSelector → user picks profile or creates one via ProfileForm modal
2. `tab.Connect()` fires async cmd: dials SSH tunnel if bastion present, then opens driver
3. `connectDoneMsg` → tab stores `conn.Driver`, creates `Executor`, starts schema introspect cmd
4. `schemaRefreshDoneMsg` → updates cache + browser

### Query execution

`F5` in input → `tab.executeQuery()`:
- Appends to disk history and `sessionQueries` (in-memory, newest-first)
- Clears input immediately
- Fires async `tea.Cmd` → `driver.Execute(ctx, query)`
- `queryDoneMsg` → results pane and status bar update

### Backslash commands

`\dt`, `\d`, `\dn`, `\di`, `\df` dispatch to driver-appropriate SQL in `handleBackslash()`.
`\refresh` calls `driver.Introspect()` in a goroutine and updates the cache.

## Key invariants

- **One Driver per tab** — drivers manage their own concurrency (pgxpool.Acquire, sql.DB pool).
  Never share a connection between goroutines.
- **Width arithmetic uses plain string lengths**, not `lipgloss.Width`, for padding calculations.
  Apply styles after the string is built at the correct length. Mixing ANSI strings into `len()`
  breaks layout (learned the hard way in schemabrowser.go).
- **`sessionQueries`** (`[]string`, newest-first, per TabModel) feeds the Recent panel and
  inline up/down history. `HistoryModel` (JSONL on disk) feeds only `ctrl+r` search.
  New tab = empty `sessionQueries`.
- **Partition children excluded** from schema browser. Only parent tables shown; roots tagged
  `Partitioned: true`. Postgres-only — MySQL and SQLite have no partitioned table concept.
- **`browserWidth = 50`** — all width math in `schemabrowser.go` uses plain `len()`.
- **Input box height is fixed** (`inputVisibleLines = 4`, `inputBoxH = 6`). Padded to exactly
  `maxVisibleLines` rows so it never shifts layout.
- **`driver` field in config is `omitempty`** — defaults to `"postgres"` via `Profile.DriverName()`.
  No existing config breaks.

## Adding a new database driver

1. Create `internal/db/<name>.go` implementing `db.Driver`
2. Use `database/sql` with a pure-Go driver if possible
3. Add `connect<Name>()` in `connection/manager.go` and wire into the `switch` in `Connect()`
4. Add backslash command SQL variants in `tab.go handleBackslash()` for the new driver name
5. No changes needed in config, TUI, cache, or completion packages

## Adding a TUI panel

1. Implement `Height() int` and `View() string`
2. Wire into `tab.SetSize()` (subtract panel height from results)
3. Add a `FocusX` constant to the `Focus` enum
4. Handle in `setFocus()` and the `esc` cycle in `handleKey()`

## Style conventions

- Theme: Catppuccin Mocha. All colours and shared styles in `tui/styles.go`.
- No blink ticker — static solid cursor (`styleCursor`: accent bg, dark fg).
- Autocomplete renders above the input box inside the results area — input never shifts.
- No mouse support. No GUI.
