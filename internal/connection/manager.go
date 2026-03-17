package connection

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shammianand/queryit/internal/config"
)

type Conn struct {
	Pool    *pgxpool.Pool
	tunnel  *Tunnel
	Profile string
}

func Connect(ctx context.Context, name string, p *config.Profile) (*Conn, error) {
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

	// verify the pool can actually reach the db
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		if tunnel != nil {
			tunnel.Close()
		}
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Conn{
		Pool:    pool,
		tunnel:  tunnel,
		Profile: name,
	}, nil
}

func (c *Conn) Close(_ context.Context) {
	if c.Pool != nil {
		c.Pool.Close()
	}
	if c.tunnel != nil {
		c.tunnel.Close()
	}
}

func (c *Conn) Ping(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}
