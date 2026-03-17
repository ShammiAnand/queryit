# AGENTS.md

## What this is

`queryit` is a keyboard-driven terminal UI (TUI) for PostgreSQL, written in Go.
It uses Bubble Tea (Elm architecture) for the UI, pgxpool for database connections,
and x/crypto/ssh for SSH bastion tunneling.

## Tech stack

- **Go 1.25+**, module `github.com/shammianand/queryit`
- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, `charmbracelet/bubbles`
- **DB**: `jackc/pgx/v5` with `pgxpool` (pool per tab, max 3 conns)
- **CLI**: `spf13/cobra`
- **Config**: `gopkg.in/yaml.v3`, stored at `~/.config/queryit/config.yaml`

## Repository layout

```
main.go                        entrypoint; injects ldflags version into cmd
cmd/
  root.go                      cobra root; --profile flag; launches TUI
  profile.go                   profile list/add/remove subcommands
internal/
  config/config.go             YAML config load/save; XDG paths; $ENV_VAR password expansion
  connection/
    manager.go                 pgxpool.New per tab; SSH tunnel → local port → pool
    tunnel.go                  x/crypto/ssh local port forwarding
  db/
    executor.go                pool.Acquire per query; SELECT vs DML detection; pagination
    introspect.go              information_schema + pg_catalog queries; filters partition children
  cache/schema.go              in-memory SchemaSnapshot; JSON disk cache per profile
  completion/
    engine.go                  context-aware fuzzy suggest (tables/columns only, no keywords)
    keywords.go                SQL keyword list; backslash commands
  tui/
    app.go                     top-level Bubble Tea model; ProfileSelector; ProfileForm modal
    tab.go                     per-tab model; Focus enum; key routing; query execution
    input.go                   multi-line editor; history cycling via up/down; autocomplete hook
    results.go                 table + expanded views; col scroll (colOffset); pagination
    schemabrowser.go           left panel (ctrl+o); list + detail modes; 50 cols wide
    recent.go                  session-only recent queries panel; collapses when unfocused
    history.go                 JSONL disk history; searchable overlay (ctrl+r)
    autocomplete.go            dropdown model; up/down to navigate; enter to accept
    tabbar.go                  tab strip; active tab shows close marker
    statusbar.go               connection state; row count; elapsed time
    styles.go                  Catppuccin Mocha lipgloss theme; all shared styles here
    keys.go                    key binding definitions (mostly informational now)
```

## Architecture

Each open database connection is a **TabModel**. Tabs are independent — separate
pool, schema cache, history, and session state. The app routes key messages to the
active tab; async messages (connect, query done, schema refresh) are broadcast to
all tabs so each handles its own.

### Focus cycle (within a tab)

`esc` rotates: **Input → Recent → Results → Browser → Input**

Panels are skipped if not available (Recent: no entries yet; Browser: collapsed).

### Connection flow

1. `ctrl+t` → ProfileSelector → user picks profile or creates one via ProfileForm
2. `tab.Connect()` returns a `tea.Cmd` that dials SSH tunnel (if bastion) then `pgxpool.New`
3. On `connectDoneMsg` the tab starts a background schema introspection cmd
4. Schema lands as `schemaRefreshDoneMsg`; updates cache + browser

### Query execution

`F5` in the input pane → `tab.executeQuery()`:
- Appends to disk history (`HistoryModel`) and session slice (`sessionQueries`)
- Clears input immediately
- Fires an async `tea.Cmd` that calls `executor.Execute` (acquires pool conn, runs query, releases)
- Result arrives as `queryDoneMsg`; results pane and status bar update

## Key invariants

- **`pgxpool`** per tab — never share a connection between goroutines. Each `Execute` and `IntrospectSchema` call does `pool.Acquire` / `defer conn.Release`.
- **Width arithmetic uses plain string lengths**, not `lipgloss.Width`, for padding calculations. Apply styles after building the string at the correct width. Mixing styled strings into `len()` breaks layout.
- **`sessionQueries`** (in-memory, `[]string`, newest-first) feeds the Recent panel and inline up/down history. `HistoryModel` (disk JSONL) feeds only `ctrl+r` search. New tab = empty `sessionQueries`.
- **Partition children excluded** from schema browser and introspection. Only parent tables shown; partitioned roots tagged with `Partitioned: bool`.
- **`browserWidth = 50`** — all width math in `schemabrowser.go` uses plain `len()` for padding. Do not mix ANSI-styled strings into width calculations.
- **Input box height is fixed** (`inputVisibleLines = 4`, `inputBoxH = 6`). The box pads content to exactly `maxVisibleLines` rows so it never shifts.

## Adding features

**New key binding in a tab**: add a case in `tab.go handleKey()` under the relevant `Focus` block.

**New TUI panel**: implement `Height() int` and `View() string`; wire into `tab.SetSize()` and `tab.View()`. Add a `FocusX` constant and handle it in `setFocus()` and the `esc` cycle.

**New DB query**: add to `db/executor.go` or `db/introspect.go`. Always use `pool.Acquire` / `defer conn.Release` — never pass `*pgx.Conn` directly.

**New config field**: add to `config.Settings` struct and `applyDefaults()` in `config/config.go`.

**New profile form field**: increment `fCount`, add a `formField` entry in `newProfileForm()`, handle in `ProfileForm.submit()` and `ProfileForm.ShowEdit()`.

## Style conventions

- Theme: Catppuccin Mocha. All colours and shared styles in `tui/styles.go`.
- Autocomplete renders above the input box (inside results area) so input never shifts.
- No mouse support. No non-PostgreSQL drivers (interfaces exist at db layer for future addition).
