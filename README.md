# queryit

A keyboard-driven terminal UI for PostgreSQL. Faster than psql for interactive work.

## Features

- Multi-tab connections — each tab is an independent session
- Direct connections and SSH bastion tunneling via PEM key
- Schema browser (toggle with `ctrl+o`) — tables, columns, indexes
- Recent queries panel per session; full searchable history with `ctrl+r`
- Autocomplete for table and column names from live schema cache
- Table and expanded row views; horizontal column scrolling
- Connection profiles stored in `~/.config/queryit/config.yaml`

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/shammianand/queryit/main/install.sh | bash
```

Requires Go 1.25+ to build from source.

## Usage

```sh
queryit                        # open with empty tab bar
queryit --profile local-new    # connect to a saved profile on launch
queryit profile list           # list saved profiles
queryit profile add            # add a profile interactively
queryit profile remove <name>  # delete a profile
```

## Key bindings

### Global

| Key | Action |
|-----|--------|
| `ctrl+t` | Open connection selector |
| `tab` / `shift+tab` | Next / previous tab |
| `ctrl+w` | Close current tab |
| `ctrl+o` | Toggle schema browser |
| `ctrl+q` | Quit |

### Input pane

| Key | Action |
|-----|--------|
| `F5` | Execute query |
| `ctrl+c` | Clear input |
| `ctrl+r` | Search query history |
| `up` / `down` | Cycle through session queries |
| `esc` | Cycle focus (input → recent → results → browser) |

### Results pane

| Key | Action |
|-----|--------|
| `n` / `p` | Next / previous page |
| `+` / `-` | Increase / decrease page size |
| `<` / `>` | Scroll columns left / right |
| `v` | Toggle table / expanded row view |
| `j` / `k` | Navigate rows (expanded view) |

### Schema browser

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate tables |
| `enter` | Show table detail (columns, indexes) |
| `space` | Paste table name into input |
| `esc` | Back to table list |

## Configuration

Config file: `~/.config/queryit/config.yaml`

```yaml
profiles:
  local:
    host: localhost
    port: 5432
    database: mydb
    user: postgres
    password: secret

  prod:
    host: db.example.com
    port: 5432
    database: mydb
    user: appuser
    password: $PROD_DB_PASSWORD   # expands from environment
    sslmode: require
    bastion:
      user: ubuntu
      host: 10.0.0.1
      pem: ~/.ssh/bastion.pem

settings:
  page_size: 20
  query_timeout: 30
  history_size: 1000
```

Passwords prefixed with `$` are read from the environment at connect time.

## Backslash commands

| Command | Description |
|---------|-------------|
| `\dt` | List all tables |
| `\d <table>` | Describe a table |
| `\dn` | List schemas |
| `\di` | List indexes |
| `\df` | List functions |
| `\refresh` | Reload schema cache from the database |

## Data locations

| Purpose | Path |
|---------|------|
| Config | `$XDG_CONFIG_HOME/queryit/config.yaml` |
| Schema cache | `$XDG_CACHE_HOME/queryit/<profile>/schema.json` |
| Query history | `$XDG_DATA_HOME/queryit/<profile>/history` |
