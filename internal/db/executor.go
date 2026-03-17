package db

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Row []string

type ResultSet struct {
	Columns []string
	Pages   [][]Row
	Total   int
	Elapsed time.Duration
	Message string
	IsError bool
}

type Executor struct {
	pool     *pgxpool.Pool
	pageSize int
}

func NewExecutor(pool *pgxpool.Pool, pageSize int) *Executor {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &Executor{pool: pool, pageSize: pageSize}
}

func (e *Executor) SetPageSize(n int) {
	if n > 0 {
		e.pageSize = n
	}
}

func (e *Executor) Execute(ctx context.Context, query string) (*ResultSet, error) {
	query = strings.TrimSpace(query)
	start := time.Now()

	upper := strings.ToUpper(query)
	isSelect := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "TABLE") ||
		strings.HasPrefix(upper, "VALUES") ||
		strings.HasPrefix(upper, "SHOW")

	// acquire a dedicated connection for this query
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return &ResultSet{IsError: true, Message: err.Error(), Elapsed: time.Since(start)}, nil
	}
	defer conn.Release()

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

	pages := paginate(allRows, e.pageSize)
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

func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case [16]byte:
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			val[0:4], val[4:6], val[6:8], val[8:10], val[10:16])
	case []byte:
		return fmt.Sprintf("%x", val)
	case net.HardwareAddr:
		return val.String()
	case net.IP:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

func paginate(rows []Row, size int) [][]Row {
	if size <= 0 {
		size = 20
	}
	var pages [][]Row
	for i := 0; i < len(rows); i += size {
		end := i + size
		if end > len(rows) {
			end = len(rows)
		}
		pages = append(pages, rows[i:end])
	}
	return pages
}
