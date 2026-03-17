package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shammianand/queryit/internal/cache"
)

// PostgresDriver implements Driver for PostgreSQL (and CockroachDB).
type PostgresDriver struct {
	pool     *pgxpool.Pool
	pageSize int
}

func NewPostgresDriver(pool *pgxpool.Pool, pageSize int) *PostgresDriver {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &PostgresDriver{pool: pool, pageSize: pageSize}
}

func (d *PostgresDriver) DriverName() string { return "postgres" }

func (d *PostgresDriver) SetPageSize(n int) {
	if n > 0 {
		d.pageSize = n
	}
}

func (d *PostgresDriver) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *PostgresDriver) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

func (d *PostgresDriver) Execute(ctx context.Context, query string) (*ResultSet, error) {
	query = strings.TrimSpace(query)
	start := time.Now()

	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return &ResultSet{IsError: true, Message: err.Error(), Elapsed: time.Since(start)}, nil
	}
	defer conn.Release()

	upper := strings.ToUpper(query)
	isSelect := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "TABLE") ||
		strings.HasPrefix(upper, "VALUES") ||
		strings.HasPrefix(upper, "SHOW")

	if !isSelect {
		tag, err := conn.Exec(ctx, query)
		elapsed := time.Since(start)
		if err != nil {
			return &ResultSet{IsError: true, Message: err.Error(), Elapsed: elapsed}, nil
		}
		return &ResultSet{
			Message: fmt.Sprintf("%s — %d rows affected", tag.String(), tag.RowsAffected()),
			Elapsed: elapsed,
		}, nil
	}

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return &ResultSet{IsError: true, Message: err.Error(), Elapsed: time.Since(start)}, nil
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = string(f.Name)
	}

	var allRows []Row
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(Row, len(vals))
		for i, v := range vals {
			row[i] = formatValue(v)
		}
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return &ResultSet{IsError: true, Message: err.Error(), Elapsed: time.Since(start)}, nil
	}
	elapsed := time.Since(start)

	pages := paginate(allRows, d.pageSize)
	if len(pages) == 0 {
		pages = [][]Row{{}}
	}

	return &ResultSet{
		Columns: cols,
		Pages:   pages,
		Total:   len(allRows),
		Elapsed: elapsed,
	}, nil
}

func (d *PostgresDriver) Introspect(ctx context.Context) (*cache.SchemaSnapshot, error) {
	return introspectPostgres(ctx, d.pool)
}
