package db

import (
	"fmt"
	"net"
	"time"
)

// Row is a slice of string-formatted column values.
type Row []string

// ResultSet is the unified result type returned by all Driver implementations.
type ResultSet struct {
	Columns []string
	Pages   [][]Row
	Total   int
	Elapsed time.Duration
	Message string // DML/DDL success message or error text
	IsError bool
}

// Executor is a thin wrapper kept for backward-compat with tab.go callers
// that call SetPageSize. The real work is in each Driver impl.
type Executor struct {
	driver   Driver
	pageSize int
}

func NewExecutor(driver Driver, pageSize int) *Executor {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &Executor{driver: driver, pageSize: pageSize}
}

func (e *Executor) SetPageSize(n int) {
	if n > 0 {
		e.pageSize = n
		// propagate to underlying driver if it supports it
		if pd, ok := e.driver.(*PostgresDriver); ok {
			pd.SetPageSize(n)
		}
		if md, ok := e.driver.(*MySQLDriver); ok {
			md.SetPageSize(n)
		}
		if sd, ok := e.driver.(*SQLiteDriver); ok {
			sd.SetPageSize(n)
		}
	}
}

func (e *Executor) Driver() Driver { return e.driver }

// ── shared helpers ────────────────────────────────────────────────────────────

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
