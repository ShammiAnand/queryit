package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/shammianand/queryit/internal/cache"
)

// MySQLDriver implements Driver for MySQL and MariaDB.
type MySQLDriver struct {
	db       *sql.DB
	pageSize int
}

func NewMySQLDriver(db *sql.DB, pageSize int) *MySQLDriver {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &MySQLDriver{db: db, pageSize: pageSize}
}

func (d *MySQLDriver) DriverName() string { return "mysql" }

func (d *MySQLDriver) SetPageSize(n int) {
	if n > 0 {
		d.pageSize = n
	}
}

func (d *MySQLDriver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *MySQLDriver) Close() {
	if d.db != nil {
		d.db.Close()
	}
}

func (d *MySQLDriver) Execute(ctx context.Context, query string) (*ResultSet, error) {
	query = strings.TrimSpace(query)
	start := time.Now()

	upper := strings.ToUpper(query)
	isSelect := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "TABLE")

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

func (d *MySQLDriver) Introspect(ctx context.Context) (*cache.SchemaSnapshot, error) {
	return introspectMySQL(ctx, d.db)
}

func introspectMySQL(ctx context.Context, db *sql.DB) (*cache.SchemaSnapshot, error) {
	snap := &cache.SchemaSnapshot{
		RefreshedAt: time.Now(),
		Tables:      []cache.Table{},
		Indexes:     []cache.Index{},
		Functions:   []cache.Function{},
	}

	// current database name
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName); err != nil {
		return nil, err
	}
	snap.Schemas = []string{dbName}

	// tables
	tableRows, err := db.QueryContext(ctx, `
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`)
	if err != nil {
		return nil, err
	}
	defer tableRows.Close()

	tableIndex := map[string]int{}
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err != nil {
			return nil, err
		}
		tableIndex[name] = len(snap.Tables)
		snap.Tables = append(snap.Tables, cache.Table{Schema: dbName, Name: name})
	}
	tableRows.Close()

	// columns
	colRows, err := db.QueryContext(ctx, `
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE,
		       CASE WHEN IS_NULLABLE='YES' THEN 1 ELSE 0 END
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME, ORDINAL_POSITION`)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	for colRows.Next() {
		var tName, colName, dataType string
		var nullable int
		if err := colRows.Scan(&tName, &colName, &dataType, &nullable); err != nil {
			return nil, err
		}
		if idx, ok := tableIndex[tName]; ok {
			snap.Tables[idx].Columns = append(snap.Tables[idx].Columns, cache.Column{
				Name:     colName,
				Type:     dataType,
				Nullable: nullable == 1,
			})
		}
	}
	colRows.Close()

	// indexes
	idxRows, err := db.QueryContext(ctx, `
		SELECT TABLE_NAME, INDEX_NAME
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		GROUP BY TABLE_NAME, INDEX_NAME
		ORDER BY TABLE_NAME, INDEX_NAME`)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var tName, idxName string
		if err := idxRows.Scan(&tName, &idxName); err != nil {
			return nil, err
		}
		snap.Indexes = append(snap.Indexes, cache.Index{
			Schema: dbName,
			Table:  tName,
			Name:   idxName,
		})
	}
	idxRows.Close()

	// routines (functions + procedures)
	fnRows, err := db.QueryContext(ctx, `
		SELECT ROUTINE_NAME, ROUTINE_TYPE, DTD_IDENTIFIER
		FROM information_schema.ROUTINES
		WHERE ROUTINE_SCHEMA = DATABASE()
		ORDER BY ROUTINE_NAME`)
	if err != nil {
		return nil, err
	}
	defer fnRows.Close()

	for fnRows.Next() {
		var name, rType, returnType string
		if err := fnRows.Scan(&name, &rType, &returnType); err != nil {
			return nil, err
		}
		snap.Functions = append(snap.Functions, cache.Function{
			Schema:     dbName,
			Name:       name,
			ReturnType: returnType,
			Arguments:  rType,
		})
	}
	fnRows.Close()

	return snap, nil
}
