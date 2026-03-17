# queryit -- Design Specification

## Overview

`queryit` is a keyboard-driven, full-screen TUI for executing SQL queries against PostgreSQL databases. It targets backend developers who know SQL but want a faster, friendlier interface than raw `psql`. Supports direct connections and SSH bastion tunneling. Multiple simultaneous connections via tabs.

**Non-goals (v1):** GUI, mouse interaction, non-PostgreSQL databases, query saving/bookmarks, result export, schema path switching. The DB layer uses interfaces at its boundaries so other drivers (MySQL, SQLite) can be added later.

---

## Technology Stack

| Layer | Choice | Rationale |
|---|---|---|
| Language | Go 1.25+ | Already initialized, strong TUI ecosystem, good concurrency primitives |
| TUI framework | Bubble Tea (charmbracelet/bubbletea) | Elm architecture, composable models per tab, lipgloss styling, large ecosystem |
| Styling | Lipgloss (charmbracelet/lipgloss) | Pairs with Bubble Tea, flexible terminal styling |
| Component base | Bubbles (charmbracelet/bubbles) | Pre-built text input, table, viewport components to extend |
| CLI | Cobra (spf13/cobra) | Standard Go CLI framework for subcommands and flags |
| PostgreSQL driver | pgx (jackc/pgx) | Pure Go, cursor-based fetching, best pg driver |
| SSH tunneling | x/crypto/ssh | Standard Go SSH library, local port forwarding |
| Config format | YAML | Human-readable, widely familiar |
| Config parsing | gopkg.in/yaml.v3 | Standard Go YAML library |

---

## Architecture

```
CLI (cobra)
  |
  +-- profile subcommands (list, add, remove)
  |
  +-- TUI launch (default / --profile flag)
        |
        Bubble Tea App (top-level model)
          |
          +-- Tab Bar Model
          |     +-- Tab 1 (connection A)
          |     +-- Tab 2 (connection B)
          |     +-- [ + ] (new tab trigger)
          |
          +-- Each Tab Model contains:
                +-- Input Pane Model (multi-line text editor + autocomplete)
                +-- Results Pane Model (table or expanded view + pagination)
                +-- Status Bar Model (connection state, row count, query time)
                +-- Connection reference (via Connection Manager)
                +-- Schema Cache reference

Connection Manager
  +-- Direct connections (pgx pool via host:port)
  +-- Tunneled connections (SSH tunnel -> local port -> pgx pool)
  +-- Reconnect logic (error + one-key retry)

Schema Cache
  +-- In-memory per-connection (tables, columns, types, indexes, functions)
  +-- Persisted to $XDG_CACHE_HOME/queryit/<profile>/schema.json
  +-- Loaded from disk on connect (instant autocomplete)
  +-- Async background refresh on connect
  +-- Manual refresh via command

Config
  +-- $XDG_CONFIG_HOME/queryit/config.yaml
  +-- Connection profiles
  +-- Settings (display page size, default view mode, key bindings)
```

---

## Connection Profiles

Stored in `$XDG_CONFIG_HOME/queryit/config.yaml`:

```yaml
profiles:
  local-django:
    host: localhost
    port: 5433
    database: django_db
    user: postgres
    password: mysecretpassword      # plaintext ok for local

  prod-new:
    host: prod-db.example.com
    port: 5432
    database: mydb
    user: appuser
    password: $PROD_DB_PASSWORD       # env var reference
    sslmode: require
    bastion:
      user: ubuntu
      host: 10.0.0.1
      pem: ~/.ssh/bastion-key.pem

  integ:
    host: integ-db.example.com
    port: 5432
    database: mydb
    user: appuser
    password: $INTEG_DB_PASSWORD
    bastion:
      user: ec2-user
      host: 10.0.0.2
      pem: ~/.ssh/dev-key.pem

settings:
  page_size: 10
  default_view: table  # table | expanded
```

### Profile Schema

| Field | Required | Description |
|---|---|---|
| `host` | yes | Database host |
| `port` | yes | Database port |
| `database` | yes | Database name |
| `user` | yes | Database user |
| `password` | yes | Database password (plaintext or `$ENV_VAR` reference) |
| `sslmode` | no | SSL mode: `disable`, `require`, `verify-ca`, `verify-full` (default: `prefer`) |
| `bastion.user` | no | SSH bastion username |
| `bastion.host` | no | SSH bastion host |
| `bastion.pem` | no | Path to PEM file for SSH authentication |

If `bastion` is present, connection goes through SSH tunnel. Otherwise direct.

Password values starting with `$` are resolved as environment variables. If the referenced env var is unset or empty, connection fails with a clear error message.

---

## CLI Interface

### Commands

```
queryit                          # Launch TUI, start with empty tab bar
queryit --profile <name>         # Launch TUI, auto-connect to named profile
queryit profile list             # List all saved profiles (stdout, exit)
queryit profile add              # Interactive profile creation (prompts)
queryit profile remove <name>    # Delete a profile
```

### Flags

| Flag | Description |
|---|---|
| `--profile`, `-p` | Profile name to auto-connect on launch |
| `--config` | Override config file path |
| `--version`, `-v` | Print version and exit |

---

## TUI Layout

```
+--[ local-django ]--[ prod-new ]--[ integ ]--[ + ]--------+
|                                                            |
|  Results Pane                                              |
|  +-------------------------------------------------------+ |
|  | id | date       | org_id   | model_id | weather_id   | |
|  |----|------------|----------|----------|-------------- | |
|  | 1  | 2026-01-20 | 2bbdee74 | abc123   | def456       | |
|  | 2  | 2026-01-20 | 2bbdee74 | abc123   | def456       | |
|  | .. | ..         | ..       | ..       | ..           | |
|  |                                                       | |
|  |                      page 1/24  (10 rows/page)        | |
|  +-------------------------------------------------------+ |
|                                                            |
|  Input Pane                                                |
|  +-------------------------------------------------------+ |
|  | SELECT * FROM storm_impact_daily_forecasts             | |
|  | WHERE date = '2026-01-20';                             | |
|  +-------------------------------------------------------+ |
|                                                            |
|  Status: connected | 240 rows returned | 32ms              |
+------------------------------------------------------------+
```

### Expanded View (toggle with `v`)

```
+--[ local-django ]--[ prod-new ]--[ + ]--------------------+
|                                                            |
|  Row 1 of 240                                              |
|  +-------------------------------------------------------+ |
|  | id              | 2bbdee74-a1b2-c3d4-e5f6-789012345   | |
|  | date            | 2026-01-20                           | |
|  | run             | 2026-01-20T06:00:00Z                 | |
|  | organization_id | 2bbdee74-...                         | |
|  | geo_node_id     | 5427f98e-...                         | |
|  | model_id        | abc123-...                           | |
|  | weather_id      | def456-...                           | |
|  | variables       | {"wind_speed": 45, "outages": 120}   | |
|  |                                                       | |
|  |              fields 1-8 of 8  |  row 1/240            | |
|  +-------------------------------------------------------+ |
|                                                            |
|  Input Pane                                                |
|  +-------------------------------------------------------+ |
|  | SELECT * FROM storm_impact_daily_forecasts             | |
|  | WHERE date = '2026-01-20';                             | |
|  +-------------------------------------------------------+ |
|                                                            |
|  Status: connected | 240 rows returned | 32ms              |
+------------------------------------------------------------+
```

When a row has more fields than fit on screen, fields are paginated (e.g., "fields 1-20 of 35").

---

## Key Bindings

All keyboard-driven. No mouse support.

### Global

| Key | Action |
|---|---|
| `Ctrl+t` | New tab (opens profile selector) |
| `Ctrl+w` | Close current tab |
| `Ctrl+n` | Next tab |
| `Ctrl+p` | Previous tab |
| `Ctrl+q` | Quit application |

### Focus

| Key | Action |
|---|---|
| `Esc` | Switch focus between input and results |

### Input Pane (focused)

| Key | Action |
|---|---|
| Type | Normal text input with multi-line support |
| `Ctrl+Enter` | Execute query (fallback: `F5` for terminals that don't distinguish Ctrl+Enter) |
| `Tab` | Cycle autocomplete suggestions (when dropdown visible) |
| `Enter` | Accept selected autocomplete suggestion (when dropdown visible) |
| `Esc` | Dismiss autocomplete dropdown / switch focus to results (if no dropdown) |
| `Ctrl+c` | Cancel running query |
| `Ctrl+r` | Open query history (searchable) |

### Results Pane (focused)

| Key | Action |
|---|---|
| `n` | Next page |
| `p` | Previous page |
| `v` | Toggle table/expanded view |
| `j` / `k` | Next/previous row (expanded view) |
| `r` | Reconnect (when connection error shown) |

### Built-in Commands (typed in input pane)

| Command | Action |
|---|---|
| `\dt` | List all tables in current schema |
| `\d <table>` | Describe table (columns, types, constraints) |
| `\dn` | List all schemas |
| `\di` | List all indexes |
| `\df` | List all functions |
| `\refresh` | Force refresh schema cache from live DB |

---

## Query Execution

### Flow

1. User types SQL in input pane
2. `Ctrl+Enter` to execute
3. Query sent to pgx connection as-is (no mutation, no wrapping)
4. Results fetched via cursor (pgx rows iterator)
5. First page of results (page_size rows) rendered in results pane
6. Remaining rows fetched lazily as user paginates
7. Status bar updates: row count, execution time, connection state

### Result Handling

- **SELECT queries** -- render in results pane (table or expanded view)
- **DML queries** (INSERT, UPDATE, DELETE) -- show affected row count in results pane
- **DDL queries** (CREATE, ALTER, DROP) -- show success/failure message
- **Errors** -- pg error message rendered inline in results pane

### Cursor-Based Fetching

Use pgx cursor to avoid loading full result set into memory:

- Execute query, obtain rows iterator
- Read `page_size` rows into display buffer
- On page-forward, read next `page_size` rows
- Cache already-fetched pages for backward pagination (max 100 pages in memory)
- When cache limit exceeded, oldest pages evicted; backward navigation past evicted pages re-executes query with OFFSET
- Configurable page size (default: 10)

---

## Connection Management

### Connection Lifecycle

1. User opens new tab, selects profile (or creates new one)
2. Connection Manager resolves profile config
3. If bastion config present:
   - Establish SSH connection to bastion using PEM
   - Set up local port forwarding (random available port -> db host:port)
   - Connect pgx to `localhost:<forwarded_port>`
4. If direct:
   - Connect pgx to `host:port` directly
5. Connection pool established (pgx pool, small pool size -- 2-3 conns)
6. Schema cache loaded from disk (`$XDG_CACHE_HOME/queryit/<profile>/schema.json`)
7. Background goroutine refreshes schema cache from live DB
8. Tab is ready for queries

### Error and Reconnect

- On connection drop, tab shows error message in results pane
- Status bar shows "disconnected"
- User presses `r` to reconnect
- Reconnect re-establishes SSH tunnel (if applicable) and pgx pool
- No automatic retry -- user-initiated only

### SSH Tunnel

- Uses `x/crypto/ssh` to dial bastion
- Authenticates with PEM file (parsed from disk)
- Opens local port forwarding: `localhost:<random_port>` -> `db_host:db_port`
- pgx connects through the forwarded port
- Tunnel lifecycle tied to tab -- closing tab closes tunnel
- SSH dial timeout matches `query_timeout` setting (default: 30s)

---

## Schema Cache

### Contents

Per-connection cache containing:

```json
{
  "profile": "prod-new",
  "refreshed_at": "2026-03-17T10:30:00Z",
  "schemas": ["public", "etl"],
  "tables": [
    {
      "schema": "public",
      "name": "storm_impact_daily_forecasts",
      "columns": [
        {"name": "id", "type": "uuid", "nullable": false},
        {"name": "date", "type": "date", "nullable": false}
      ]
    }
  ],
  "indexes": [
    {"schema": "public", "table": "storm_impact_daily_forecasts", "name": "idx_forecasts_date", "columns": ["date"]}
  ],
  "functions": [
    {"schema": "public", "name": "now", "return_type": "timestamptz", "arguments": ""}
  ]
}
```

### Lifecycle

1. On connect -- load from `$XDG_CACHE_HOME/queryit/<profile>/schema.json` if exists
2. Immediately available for autocomplete (even if stale)
3. Background goroutine queries `information_schema` and `pg_catalog` to refresh
4. On refresh complete -- update in-memory cache + persist to disk
5. Manual refresh via `\refresh` command

### Introspection Queries

- Tables: `information_schema.tables`
- Columns: `information_schema.columns`
- Indexes: `pg_indexes`
- Functions: `pg_proc` joined with `pg_namespace`
- Schemas: `information_schema.schemata`

---

## Autocomplete

### Sources

1. **SQL keywords** -- hardcoded list (SELECT, FROM, WHERE, JOIN, INSERT, UPDATE, DELETE, CREATE, ALTER, DROP, GROUP BY, ORDER BY, HAVING, LIMIT, OFFSET, etc.)
2. **Table names** -- from schema cache
3. **Column names** -- from schema cache
4. **Schema names** -- from schema cache
5. **Function names** -- from schema cache

### Context Awareness

Basic context-aware suggestions:

| After typing... | Suggest |
|---|---|
| `FROM ` / `JOIN ` / `UPDATE ` / `INTO ` | Table names |
| `SELECT ` (no FROM yet) | Column names (all tables), `*`, SQL keywords |
| `<table>.` | Columns of that table |
| `WHERE ` / `AND ` / `OR ` | Column names of tables in query |
| `\d ` | Table names |
| Beginning of line | SQL keywords, `\` commands |

### Behavior

- Triggers as user types (after 1+ characters)
- Dropdown appears below cursor in input pane
- `Tab` cycles through suggestions
- `Enter` accepts selected suggestion
- `Esc` dismisses dropdown
- Typing narrows suggestions (fuzzy match)
- Case-insensitive matching

---

## Query History

- Stored per-profile in `$XDG_DATA_HOME/queryit/<profile>/history`
- JSON Lines format: `{"query": "...", "ts": "2026-03-17T10:30:00Z"}`
- `Ctrl+r` opens searchable history overlay
- Fuzzy search through past queries
- `Enter` loads selected query into input pane
- Configurable max history size (default: 1000 entries)

---

## Configuration

### File Location

`$XDG_CONFIG_HOME/queryit/config.yaml` (defaults to `~/.config/queryit/config.yaml`)

### Full Schema

```yaml
profiles:
  <name>:
    host: string       # required
    port: int          # required
    database: string   # required
    user: string       # required
    password: string   # required (plaintext or $ENV_VAR)
    sslmode: string    # optional (disable|require|verify-ca|verify-full, default: prefer)
    bastion:           # optional
      user: string
      host: string
      pem: string      # path to PEM file

settings:
  page_size: 10                # rows per page (default: 10)
  default_view: table          # table | expanded (default: table)
  history_size: 1000           # max history entries per profile
  query_timeout: 30            # seconds (default: 30)
```

### XDG Paths

| Purpose | Path |
|---|---|
| Config | `$XDG_CONFIG_HOME/queryit/config.yaml` |
| Schema cache | `$XDG_CACHE_HOME/queryit/<profile>/schema.json` |
| Query history | `$XDG_DATA_HOME/queryit/<profile>/history` |

### Defaults

All settings have sensible defaults. Config file is only needed for connection profiles. Settings section is entirely optional.

---

## Project Structure

```
queryit/
  cmd/
    root.go              # cobra root command, TUI launch
    profile.go           # profile list/add/remove subcommands
  internal/
    config/
      config.go          # YAML config loading, profile management
    connection/
      manager.go         # connection pool management, connect/disconnect
      tunnel.go          # SSH tunnel setup and teardown
    db/
      executor.go        # query execution, cursor-based fetching
      introspect.go      # schema introspection queries
    cache/
      schema.go          # schema cache (in-memory + disk persistence)
    tui/
      app.go             # top-level Bubble Tea model
      tab.go             # per-tab model (composes input + results)
      input.go           # multi-line input pane model
      results.go         # results pane model (table + expanded views)
      tabbar.go          # tab bar model
      statusbar.go       # status bar model
      autocomplete.go    # autocomplete dropdown model
      history.go         # query history overlay model
      styles.go          # lipgloss style definitions
      keys.go            # key binding definitions
    completion/
      engine.go          # autocomplete engine (context-aware matching)
      keywords.go        # SQL keyword list
  main.go                # entrypoint
  go.mod
  go.sum
```

---

## Status Bar

Always visible at bottom of each tab. Shows:

- Connection state: `connected` / `disconnected` / `connecting...`
- Profile name
- After query execution: row count + execution time (e.g., "240 rows | 32ms")
- Error messages (brief, full error in results pane)

---

## New Tab Flow

1. User presses `Ctrl+t`
2. Modal overlay appears listing saved profiles + "Create new profile" option
3. User navigates with `j`/`k`, selects with `Enter`
4. If existing profile: connect and open tab
5. If "Create new": inline prompts for host, port, db, user, password, optional bastion fields. Save to config, connect, open tab.

---

## Edge Cases

- **Empty state** -- app launches with no tabs. Shows welcome message: "Press Ctrl+t to open a connection"
- **All tabs closed** -- returns to empty state (same welcome message)
- **Invalid profile on --profile flag** -- print error to stderr, exit 1
- **Config file missing** -- create default empty config on first run
- **PEM file not found** -- show error in tab, don't crash
- **Query timeout** -- pgx context timeout (default: 30s, configurable), show timeout error in results pane
- **Query cancellation** -- `Ctrl+c` while query is running cancels via context cancellation
- **Very wide results** -- auto-fit columns, truncate cell content with `...` if exceeds terminal width. Full value visible in expanded view.
