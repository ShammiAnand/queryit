package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"github.com/shammianand/queryit/internal/cache"
)

// SQLiteDriver implements Driver for SQLite (via modernc pure-Go driver).
type SQLiteDriver struct {
	db       *sql.DB
	path     string
	pageSize int
}

func NewSQLiteDriver(db *sql.DB, path string, pageSize int) *SQLiteDriver {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &SQLiteDriver{db: db, path: path, pageSize: pageSize}
}

func (d *SQLiteDriver) DriverName() string { return "sqlite" }

func (d *SQLiteDriver) SetPageSize(n int) {
	if n > 0 {
		d.pageSize = n
	}
}

func (d *SQLiteDriver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *SQLiteDriver) Close() {
	if d.db != nil {
		d.db.Close()
	}
}

func (d *SQLiteDriver) Execute(ctx context.Context, query string) (*ResultSet, error) {
	query = strings.TrimSpace(query)
	start := time.Now()

	upper := strings.ToUpper(query)
	isSelect := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "PRAGMA") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "VALUES")

	if !isSelect {
		res, err := d.db.ExecContext(ctx, query)
		elapsed := time.Since(start)
		if err != nil {
			return &ResultSet{IsError: true, Message: err.Error(), Elapsed: elapsed}, nil
		}
		affected, _ := res.RowsAffected()
		return &ResultSet{
			Message: fmt.Sprintf("%d rows affected", affected),
			Elapsed: elapsed,
		}, nil
	}

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return &ResultSet{IsError: true, Message: err.Error(), Elapsed: time.Since(start)}, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return &ResultSet{IsError: true, Message: err.Error(), Elapsed: time.Since(start)}, nil
	}

	var allRows []Row
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
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

func (d *SQLiteDriver) Introspect(ctx context.Context) (*cache.SchemaSnapshot, error) {
	return introspectSQLite(ctx, d.db)
}

func introspectSQLite(ctx context.Context, db *sql.DB) (*cache.SchemaSnapshot, error) {
	snap := &cache.SchemaSnapshot{
		RefreshedAt: time.Now(),
		Schemas:     []string{"main"},
		Tables:      []cache.Table{},
		Indexes:     []cache.Index{},
		Functions:   []cache.Function{},
	}

	// tables and views
	tableRows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer tableRows.Close()

	var tableNames []string
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
		snap.Tables = append(snap.Tables, cache.Table{Schema: "main", Name: name})
	}
	tableRows.Close()

	// columns via PRAGMA table_info
	for i, name := range tableNames {
		colRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", name))
		if err != nil {
			continue
		}
		for colRows.Next() {
			var cid int
			var colName, colType string
			var notNull, pk int
			var dflt any
			if err := colRows.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk); err != nil {
				colRows.Close()
				break
			}
			snap.Tables[i].Columns = append(snap.Tables[i].Columns, cache.Column{
				Name:     colName,
				Type:     colType,
				Nullable: notNull == 0,
			})
		}
		colRows.Close()
	}

	// indexes
	idxRows, err := db.QueryContext(ctx,
		`SELECT name, tbl_name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%' ORDER BY tbl_name, name`)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var idxName, tblName string
		if err := idxRows.Scan(&idxName, &tblName); err != nil {
			return nil, err
		}
		snap.Indexes = append(snap.Indexes, cache.Index{
			Schema: "main",
			Table:  tblName,
			Name:   idxName,
		})
	}
	idxRows.Close()

	return snap, nil
}
