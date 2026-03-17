package connection

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"

	"github.com/shammianand/queryit/internal/config"
	"github.com/shammianand/queryit/internal/db"
)

// Conn wraps a Driver and the optional SSH tunnel that may underpin it.
type Conn struct {
	Driver  db.Driver
	tunnel  *Tunnel
	Profile string
}

func Connect(ctx context.Context, name string, p *config.Profile, pageSize int) (*Conn, error) {
	switch p.DriverName() {
	case "mysql":
		return connectMySQL(ctx, name, p, pageSize)
	case "sqlite":
		return connectSQLite(ctx, name, p, pageSize)
	default:
		return connectPostgres(ctx, name, p, pageSize)
	}
}

// ── postgres ──────────────────────────────────────────────────────────────────

func connectPostgres(ctx context.Context, name string, p *config.Profile, pageSize int) (*Conn, error) {
	password, err := p.ResolvedPassword()
	if err != nil {
		return nil, err
	}

	var tunnel *Tunnel
	host := p.Host
	port := p.Port

	if p.Bastion != nil {
		if p.Bastion.PEM == "" {
			return nil, fmt.Errorf("bastion.pem is required for SSH tunnel")
		}
		tunnel, err = NewTunnel(p.Bastion.User, p.Bastion.Host, p.Bastion.PEM, p.Host, p.Port)
		if err != nil {
			return nil, fmt.Errorf("SSH tunnel: %w", err)
		}
		host = "127.0.0.1"
		port = tunnel.LocalPort
	}

	sslmode := p.SSLMode
	if sslmode == "" {
		sslmode = "prefer"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s pool_max_conns=3",
		host, port, p.Database, p.User, password, sslmode,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		if tunnel != nil {
			tunnel.Close()
		}
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		if tunnel != nil {
			tunnel.Close()
		}
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Conn{
		Driver:  db.NewPostgresDriver(pool, pageSize),
		tunnel:  tunnel,
		Profile: name,
	}, nil
}

// ── mysql ─────────────────────────────────────────────────────────────────────

func connectMySQL(ctx context.Context, name string, p *config.Profile, pageSize int) (*Conn, error) {
	password, err := p.ResolvedPassword()
	if err != nil {
		return nil, err
	}

	var tunnel *Tunnel
	host := p.Host
	port := p.Port
	if port == 0 {
		port = 3306
	}

	if p.Bastion != nil {
		tunnel, err = NewTunnel(p.Bastion.User, p.Bastion.Host, p.Bastion.PEM, p.Host, port)
		if err != nil {
			return nil, fmt.Errorf("SSH tunnel: %w", err)
		}
		host = "127.0.0.1"
		port = tunnel.LocalPort
	}

	tls := ""
	switch p.SSLMode {
	case "require", "verify-ca", "verify-full":
		tls = "&tls=true"
	case "disable":
		tls = "&tls=false"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true%s",
		p.User, password, host, port, p.Database, tls)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		if tunnel != nil {
			tunnel.Close()
		}
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB.SetMaxOpenConns(3)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		if tunnel != nil {
			tunnel.Close()
		}
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return &Conn{
		Driver:  db.NewMySQLDriver(sqlDB, pageSize),
		tunnel:  tunnel,
		Profile: name,
	}, nil
}

// ── sqlite ────────────────────────────────────────────────────────────────────

func connectSQLite(ctx context.Context, name string, p *config.Profile, pageSize int) (*Conn, error) {
	path := p.Database
	if path == "" {
		return nil, fmt.Errorf("sqlite profile requires database field set to a file path")
	}
	// expand ~ 
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	// SQLite only supports one writer at a time
	sqlDB.SetMaxOpenConns(1)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return &Conn{
		Driver:  db.NewSQLiteDriver(sqlDB, path, pageSize),
		Profile: name,
	}, nil
}

// ── Conn methods ──────────────────────────────────────────────────────────────

func (c *Conn) Close(_ context.Context) {
	if c.Driver != nil {
		c.Driver.Close()
	}
	if c.tunnel != nil {
		c.tunnel.Close()
	}
}

func (c *Conn) Ping(ctx context.Context) error {
	return c.Driver.Ping(ctx)
}
