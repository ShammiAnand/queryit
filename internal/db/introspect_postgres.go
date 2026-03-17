package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shammianand/queryit/internal/cache"
)

const pgQuerySchemas = `
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('pg_catalog','information_schema','pg_toast')
ORDER BY schema_name`

const pgQueryTables = `
SELECT t.table_schema, t.table_name,
       (c.relkind = 'p') AS is_partitioned
FROM information_schema.tables t
JOIN pg_class     c  ON c.relname   = t.table_name
JOIN pg_namespace ns ON ns.oid      = c.relnamespace
                     AND ns.nspname = t.table_schema
WHERE t.table_schema NOT IN ('pg_catalog','information_schema','pg_toast')
  AND t.table_type = 'BASE TABLE'
  AND c.relkind IN ('r','p')
  AND NOT EXISTS (
    SELECT 1
    FROM pg_inherits i
    JOIN pg_class parent ON parent.oid = i.inhparent
    WHERE i.inhrelid = c.oid
      AND parent.relkind = 'p'
  )
ORDER BY t.table_schema, t.table_name`

const pgQueryColumns = `
SELECT table_schema, table_name, column_name, data_type,
       CASE WHEN is_nullable = 'YES' THEN true ELSE false END AS nullable
FROM information_schema.columns
WHERE table_schema NOT IN ('pg_catalog','information_schema','pg_toast')
ORDER BY table_schema, table_name, ordinal_position`

const pgQueryIndexes = `
SELECT schemaname, tablename, indexname, indexdef
FROM pg_indexes
WHERE schemaname NOT IN ('pg_catalog','information_schema','pg_toast')
ORDER BY schemaname, tablename, indexname`

const pgQueryFunctions = `
SELECT n.nspname AS schema, p.proname AS name,
       pg_catalog.pg_get_function_result(p.oid) AS return_type,
       pg_catalog.pg_get_function_arguments(p.oid) AS arguments
FROM pg_catalog.pg_proc p
JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
ORDER BY n.nspname, p.proname`

func introspectPostgres(ctx context.Context, pool *pgxpool.Pool) (*cache.SchemaSnapshot, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	snap := &cache.SchemaSnapshot{
		RefreshedAt: time.Now(),
		Tables:      []cache.Table{},
		Indexes:     []cache.Index{},
		Functions:   []cache.Function{},
	}

	rows, err := conn.Query(ctx, pgQuerySchemas)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return nil, err
		}
		snap.Schemas = append(snap.Schemas, s)
	}
	rows.Close()

	tableIndex := map[string]int{}
	rows, err = conn.Query(ctx, pgQueryTables)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var schema, name string
		var isPartitioned bool
		if err := rows.Scan(&schema, &name, &isPartitioned); err != nil {
			rows.Close()
			return nil, err
		}
		key := schema + "." + name
		tableIndex[key] = len(snap.Tables)
		snap.Tables = append(snap.Tables, cache.Table{
			Schema:      schema,
			Name:        name,
			Partitioned: isPartitioned,
		})
	}
	rows.Close()

	rows, err = conn.Query(ctx, pgQueryColumns)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tSchema, tName, colName, dataType string
		var nullable bool
		if err := rows.Scan(&tSchema, &tName, &colName, &dataType, &nullable); err != nil {
			rows.Close()
			return nil, err
		}
		key := tSchema + "." + tName
		if idx, ok := tableIndex[key]; ok {
			snap.Tables[idx].Columns = append(snap.Tables[idx].Columns, cache.Column{
				Name:     colName,
				Type:     dataType,
				Nullable: nullable,
			})
		}
	}
	rows.Close()

	rows, err = conn.Query(ctx, pgQueryIndexes)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var schema, table, name, def string
		if err := rows.Scan(&schema, &table, &name, &def); err != nil {
			rows.Close()
			return nil, err
		}
		snap.Indexes = append(snap.Indexes, cache.Index{Schema: schema, Table: table, Name: name})
	}
	rows.Close()

	rows, err = conn.Query(ctx, pgQueryFunctions)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var schema, name, returnType, arguments string
		if err := rows.Scan(&schema, &name, &returnType, &arguments); err != nil {
			rows.Close()
			return nil, err
		}
		snap.Functions = append(snap.Functions, cache.Function{
			Schema:     schema,
			Name:       name,
			ReturnType: returnType,
			Arguments:  arguments,
		})
	}
	rows.Close()

	return snap, nil
}
