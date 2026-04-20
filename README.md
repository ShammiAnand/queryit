# queryit

A keyboard-driven terminal UI for PostgreSQL, MySQL, and SQLite.

https://github.com/user-attachments/assets/453bffb4-7984-45f3-ae2b-754d6d61077c

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/ShammiAnand/queryit/main/install.sh | bash
```

Binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` are attached to each [release](https://github.com/ShammiAnand/queryit/releases). To build from source: `make install` (requires Go 1.21+).

## Usage

```sh
queryit                        # launch
queryit --profile staging      # connect to a saved profile on launch
queryit profile add            # add a profile interactively
queryit profile list
queryit profile remove <name>
```

Press `?` at any time for the full keybinding reference.

## Configuration

`~/.config/queryit/config.yaml`

```yaml
profiles:
  local:
    host: localhost
    port: 5432
    database: mydb
    user: postgres
    password: secret          # or $ENV_VAR

  prod:
    host: db.internal
    database: mydb
    user: appuser
    password: $PROD_DB_PASS
    sslmode: require
    bastion:                  # SSH tunnel via PEM key
      user: ubuntu
      host: 10.0.0.1
      pem: ~/.ssh/bastion.pem

  local-mysql:
    driver: mysql             # postgres (default) | mysql | sqlite
    host: localhost
    port: 3306
    database: mydb
    user: root
    password: secret

settings:
  page_size: 20
  query_timeout: 30
  history_size: 1000
  theme: dark                 # dark | light
```
