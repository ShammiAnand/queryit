package completion

var SQLKeywords = []string{
	"SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN", "IS", "NULL",
	"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL", "CROSS",
	"ON", "AS", "DISTINCT", "ALL", "UNION", "INTERSECT", "EXCEPT",
	"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
	"CREATE", "ALTER", "DROP", "TABLE", "INDEX", "VIEW", "SEQUENCE",
	"TRUNCATE", "BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION",
	"GROUP", "BY", "HAVING", "ORDER", "LIMIT", "OFFSET",
	"CASE", "WHEN", "THEN", "ELSE", "END",
	"COUNT", "SUM", "AVG", "MIN", "MAX",
	"COALESCE", "NULLIF", "GREATEST", "LEAST",
	"CAST", "EXTRACT", "DATE_PART", "NOW", "CURRENT_TIMESTAMP",
	"WITH", "RECURSIVE", "EXISTS", "ANY", "SOME",
	"LIKE", "ILIKE", "BETWEEN", "SIMILAR",
	"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE", "CHECK",
	"DEFAULT", "NOT NULL", "CONSTRAINT",
	"RETURNING", "CONFLICT", "DO", "NOTHING",
	"EXPLAIN", "ANALYZE", "VERBOSE",
	"SHOW", "SET", "RESET",
}

var BackslashCommands = []string{
	`\dt`, `\d`, `\dn`, `\di`, `\df`, `\refresh`,
}
