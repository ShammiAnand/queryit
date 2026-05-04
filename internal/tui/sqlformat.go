package tui

import (
	"strings"
	"unicode"
)

// fmtTok is a single lexical unit produced by tokenizeSQL.
// opaque tokens (string literals, comments) are emitted verbatim and never
// treated as keyword candidates.
type fmtTok struct {
	val    string
	opaque bool
}

// knownSQLKeywords is the set of words that should be uppercased.
var knownSQLKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "JOIN": true,
	"ON": true, "AND": true, "OR": true, "NOT": true,
	"IN": true, "IS": true, "NULL": true, "LIKE": true, "ILIKE": true,
	"BETWEEN": true, "EXISTS": true, "ALL": true, "ANY": true, "SOME": true,
	"DISTINCT": true, "AS": true, "CASE": true, "WHEN": true,
	"THEN": true, "ELSE": true, "END": true,
	"ASC": true, "DESC": true, "NULLS": true, "FIRST": true, "LAST": true,
	"LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
	"FULL": true, "CROSS": true, "NATURAL": true,
	"GROUP": true, "BY": true, "ORDER": true, "HAVING": true,
	"LIMIT": true, "OFFSET": true, "FETCH": true, "NEXT": true,
	"ROWS": true, "ONLY": true, "TIES": true,
	"UNION": true, "INTERSECT": true, "EXCEPT": true,
	"WITH": true, "RECURSIVE": true,
	"VALUES": true, "SET": true, "INTO": true,
	"INSERT": true, "DELETE": true, "UPDATE": true,
	"CREATE": true, "DROP": true, "ALTER": true,
	"TABLE": true, "INDEX": true, "VIEW": true, "SCHEMA": true, "DATABASE": true,
	"IF": true,
	"TRUE": true, "FALSE": true, "DEFAULT": true, "UNKNOWN": true,
	"RETURNING": true, "USING": true,
	"WINDOW": true, "PARTITION": true, "OVER": true, "FILTER": true,
	"RANGE": true, "UNBOUNDED": true,
	"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true, "GROUPS": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"UNIQUE": true, "CHECK": true, "CONSTRAINT": true,
	"COUNT": true, "SUM": true, "AVG": true, "MAX": true, "MIN": true,
	"COALESCE": true, "NULLIF": true, "CAST": true, "GREATEST": true, "LEAST": true,
	"EXTRACT": true, "TRIM": true, "UPPER": true, "LOWER": true, "LENGTH": true,
	"INT": true, "INTEGER": true, "BIGINT": true, "SMALLINT": true, "TINYINT": true,
	"VARCHAR": true, "CHAR": true, "TEXT": true, "NVARCHAR": true,
	"BOOLEAN": true, "BOOL": true, "FLOAT": true, "DOUBLE": true,
	"DECIMAL": true, "NUMERIC": true, "REAL": true, "PRECISION": true,
	"DATE": true, "TIME": true, "TIMESTAMP": true, "DATETIME": true, "INTERVAL": true,
	"SERIAL": true, "BIGSERIAL": true,
	"VARYING": true, "ZONE": true,
	"REPLACE": true, "CONFLICT": true, "DO": true, "NOTHING": true,
	"LATERAL": true, "EXPLAIN": true, "ANALYZE": true, "VERBOSE": true,
}

// clauseNewLine: keywords that start a new non-indented line.
var clauseNewLine = map[string]bool{
	"SELECT":           true,
	"FROM":             true,
	"WHERE":            true,
	"JOIN":             true,
	"INNER JOIN":       true,
	"LEFT JOIN":        true,
	"LEFT OUTER JOIN":  true,
	"RIGHT JOIN":       true,
	"RIGHT OUTER JOIN": true,
	"FULL JOIN":        true,
	"FULL OUTER JOIN":  true,
	"CROSS JOIN":       true,
	"NATURAL JOIN":     true,
	"GROUP BY":         true,
	"ORDER BY":         true,
	"HAVING":           true,
	"LIMIT":            true,
	"OFFSET":           true,
	"UNION":            true,
	"UNION ALL":        true,
	"INTERSECT":        true,
	"INTERSECT ALL":    true,
	"EXCEPT":           true,
	"EXCEPT ALL":       true,
	"WITH":             true,
	"VALUES":           true,
	"SET":              true,
	"UPDATE":           true,
	"INSERT INTO":      true,
	"INSERT":           true,
	"DELETE FROM":      true,
	"DELETE":           true,
	"RETURNING":        true,
}

// clauseIndented: keywords that start a new line indented by 4 spaces.
var clauseIndented = map[string]bool{
	"ON": true,
}

// compoundKeywords lists multi-word keyword sequences to merge, longest first.
// All words are uppercase.
var compoundKeywords = [][]string{
	{"LEFT", "OUTER", "JOIN"},
	{"RIGHT", "OUTER", "JOIN"},
	{"FULL", "OUTER", "JOIN"},
	{"LEFT", "JOIN"},
	{"RIGHT", "JOIN"},
	{"INNER", "JOIN"},
	{"FULL", "JOIN"},
	{"CROSS", "JOIN"},
	{"NATURAL", "JOIN"},
	{"GROUP", "BY"},
	{"ORDER", "BY"},
	{"UNION", "ALL"},
	{"INTERSECT", "ALL"},
	{"EXCEPT", "ALL"},
	{"INSERT", "INTO"},
	{"DELETE", "FROM"},
	{"IS", "NOT", "NULL"},
	{"IS", "NOT"},
	{"IS", "NULL"},
	{"NOT", "IN"},
	{"NOT", "LIKE"},
	{"NOT", "ILIKE"},
	{"NOT", "BETWEEN"},
	{"NOT", "EXISTS"},
}

// tokenizeSQL splits sql into a flat slice of fmtTok.
// Whitespace is discarded. Dots are separate tokens so the formatter can
// suppress spaces around them (e.g. table.column).
func tokenizeSQL(sql string) []fmtTok {
	runes := []rune(sql)
	n := len(runes)
	var out []fmtTok
	i := 0

	for i < n {
		r := runes[i]

		if unicode.IsSpace(r) {
			i++
			continue
		}

		// line comment
		if r == '-' && i+1 < n && runes[i+1] == '-' {
			j := i + 2
			for j < n && runes[j] != '\n' {
				j++
			}
			out = append(out, fmtTok{val: string(runes[i:j]), opaque: true})
			i = j
			continue
		}

		// block comment
		if r == '/' && i+1 < n && runes[i+1] == '*' {
			j := i + 2
			for j+1 < n && !(runes[j] == '*' && runes[j+1] == '/') {
				j++
			}
			if j+1 < n {
				j += 2
			}
			out = append(out, fmtTok{val: string(runes[i:j]), opaque: true})
			i = j
			continue
		}

		// single-quoted string
		if r == '\'' {
			j := i + 1
			for j < n {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				if runes[j] == '\'' {
					j++
					if j < n && runes[j] == '\'' { // escaped ''
						j++
						continue
					}
					break
				}
				j++
			}
			out = append(out, fmtTok{val: string(runes[i:j]), opaque: true})
			i = j
			continue
		}

		// double-quoted identifier or backtick-quoted identifier
		if r == '"' || r == '`' {
			end := r
			j := i + 1
			for j < n {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				if runes[j] == end {
					j++
					break
				}
				j++
			}
			out = append(out, fmtTok{val: string(runes[i:j]), opaque: true})
			i = j
			continue
		}

		// dollar-quoted string (PostgreSQL: $$...$$, $tag$...$tag$)
		if r == '$' && (i+1 >= n || runes[i+1] == '$' || unicode.IsLetter(runes[i+1]) || runes[i+1] == '_') {
			j := i + 1
			for j < n && runes[j] != '$' {
				j++
			}
			if j < n && runes[j] == '$' {
				tag := string(runes[i : j+1])
				j++
				tagRunes := []rune(tag)
				tlen := len(tagRunes)
				for j+tlen <= n {
					if string(runes[j:j+tlen]) == tag {
						j += tlen
						break
					}
					j++
				}
				out = append(out, fmtTok{val: string(runes[i:j]), opaque: true})
				i = j
				continue
			}
			// not dollar-quoted; fall through to treat $ as operator/word char
		}

		// number (digit or .digit)
		if unicode.IsDigit(r) || (r == '.' && i+1 < n && unicode.IsDigit(runes[i+1])) {
			j := i + 1
			hasDot := r == '.'
			for j < n {
				c := runes[j]
				if unicode.IsDigit(c) {
					j++
					continue
				}
				if c == '.' && !hasDot {
					hasDot = true
					j++
					continue
				}
				if (c == 'e' || c == 'E') && j+1 < n {
					k := j + 1
					if runes[k] == '+' || runes[k] == '-' {
						k++
					}
					if k < n && unicode.IsDigit(runes[k]) {
						j = k + 1
						continue
					}
				}
				break
			}
			out = append(out, fmtTok{val: string(runes[i:j])})
			i = j
			continue
		}

		// word (keyword or identifier)
		if unicode.IsLetter(r) || r == '_' || r == '$' {
			j := i + 1
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_' || runes[j] == '$') {
				j++
			}
			out = append(out, fmtTok{val: string(runes[i:j])})
			i = j
			continue
		}

		// punctuation handled individually
		if r == ',' || r == ';' || r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' {
			out = append(out, fmtTok{val: string(r)})
			i++
			continue
		}

		// dot: separate token so formatter can suppress surrounding spaces
		if r == '.' {
			out = append(out, fmtTok{val: "."})
			i++
			continue
		}

		// three-char operators (check before two-char)
		if i+2 < n {
			three := string(runes[i : i+3])
			switch three {
			case "->>", "#>>":
				out = append(out, fmtTok{val: three})
				i += 3
				continue
			}
		}

		// two-char operators
		if i+1 < n {
			two := string(runes[i : i+2])
			switch two {
			case "!=", "<>", "<=", ">=", "::", "||", "->", "#>", "#-",
				"@>", "<@", "&&", "!!", "~*", "=>", ":=", ">>", "<<":
				out = append(out, fmtTok{val: two})
				i += 2
				continue
			}
		}

		// single-char operator or unknown
		out = append(out, fmtTok{val: string(r)})
		i++
	}

	return out
}

// upcaseKeywords uppercases any non-opaque token whose value is a known SQL keyword.
func upcaseKeywords(tokens []fmtTok) []fmtTok {
	for i, tok := range tokens {
		if !tok.opaque {
			u := strings.ToUpper(tok.val)
			if knownSQLKeywords[u] {
				tokens[i].val = u
			}
		}
	}
	return tokens
}

// mergeCompoundKeywords scans the token slice and merges adjacent non-opaque
// tokens that form a known multi-word keyword (e.g. "GROUP" "BY" → "GROUP BY").
func mergeCompoundKeywords(tokens []fmtTok) []fmtTok {
	out := make([]fmtTok, 0, len(tokens))
	i := 0
	for i < len(tokens) {
		if tokens[i].opaque {
			out = append(out, tokens[i])
			i++
			continue
		}
		merged := false
		for _, compound := range compoundKeywords {
			need := len(compound)
			if i+need > len(tokens) {
				continue
			}
			match := true
			for j, w := range compound {
				t := tokens[i+j]
				if t.opaque || t.val != w {
					match = false
					break
				}
			}
			if match {
				out = append(out, fmtTok{val: strings.Join(compound, " ")})
				i += need
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, tokens[i])
			i++
		}
	}
	return out
}

// compactOperators are written without surrounding spaces (e.g. id::text, data->>'key').
var compactOperators = map[string]bool{
	"::": true,
	"->": true, "->>": true,
	"#>": true, "#>>": true, "#-": true,
}

// alwaysSpaceBeforeParen lists keywords that require a space before "(" so that
// "IN (1,2)" and "NOT (a OR b)" are spaced correctly, while "COUNT(*)" is not.
var alwaysSpaceBeforeParen = map[string]bool{
	"IN": true, "NOT IN": true, "NOT": true, "OR": true, "AND": true,
	"EXISTS": true, "NOT EXISTS": true, "ANY": true, "ALL": true, "SOME": true,
	"BETWEEN": true, "NOT BETWEEN": true, "NOT LIKE": true, "NOT ILIKE": true,
	"FILTER": true, "OVER": true, "AS": true, "RETURNING": true,
}

// buildFormatted applies pretty-print rules to an already-processed token slice.
func buildFormatted(tokens []fmtTok) string {
	var sb strings.Builder
	depth := 0    // paren/bracket depth
	clause := ""  // current top-level clause keyword
	prevTok := "" // previous token value
	// emitBeforeNext is injected before the next regular token (set by clause handlers).
	emitBeforeNext := ""
	// afterBetween prevents BETWEEN's own AND from being pushed to a new line.
	afterBetween := false

	// addSpace emits one space unless the builder already ends with whitespace,
	// "(", or "[".
	addSpace := func() {
		s := sb.String()
		if len(s) == 0 {
			return
		}
		last := s[len(s)-1]
		if last == ' ' || last == '\n' || last == '\t' || last == '(' || last == '[' {
			return
		}
		sb.WriteByte(' ')
	}

	// stripTrailingSpaces removes trailing spaces (not newlines) from sb.
	stripTrailingSpaces := func() {
		s := sb.String()
		trimmed := strings.TrimRight(s, " ")
		if len(trimmed) < len(s) {
			sb.Reset()
			sb.WriteString(trimmed)
		}
	}

	for _, tok := range tokens {
		v := tok.val

		switch v {
		case "(", "[":
			depth++
			// consume any pending prefix from a preceding clause keyword
			if emitBeforeNext != "" {
				sb.WriteString(emitBeforeNext)
				emitBeforeNext = ""
			}
			prevUpper := strings.ToUpper(prevTok)
			needSp := clauseNewLine[prevUpper] || clauseIndented[prevUpper] ||
				alwaysSpaceBeforeParen[prevUpper] ||
				prevTok == "" || prevTok == "(" || prevTok == "[" || prevTok == "."
			if needSp {
				addSpace()
			}
			sb.WriteString(v)
			prevTok = v
			continue

		case ")", "]":
			depth--
			emitBeforeNext = "" // closing paren cancels any pending indent
			sb.WriteString(v)
			prevTok = v
			continue

		case ".":
			emitBeforeNext = ""
			sb.WriteString(v)
			prevTok = v
			continue

		case ",":
			emitBeforeNext = ""
			if depth == 0 && clause == "SELECT" {
				sb.WriteString(",\n    ")
			} else {
				sb.WriteString(", ")
			}
			prevTok = v
			continue

		case ";":
			emitBeforeNext = ""
			sb.WriteString(";")
			prevTok = v
			continue
		}

		// clause-level keywords only apply at the top level (depth == 0).
		if !tok.opaque && depth == 0 {
			upper := v // already uppercased or compound keyword string

			if clauseNewLine[upper] {
				emitBeforeNext = "" // discard; newline separates
				stripTrailingSpaces()
				s := sb.String()
				if len(s) > 0 && s[len(s)-1] != '\n' {
					sb.WriteByte('\n')
				}
				sb.WriteString(upper)
				clause = upper
				if upper == "SELECT" {
					emitBeforeNext = "\n    "
				} else {
					emitBeforeNext = " "
				}
				prevTok = upper
				continue
			}

			if clauseIndented[upper] {
				emitBeforeNext = ""
				stripTrailingSpaces()
				s := sb.String()
				if len(s) > 0 && s[len(s)-1] != '\n' {
					sb.WriteString("\n    ")
				}
				sb.WriteString(upper)
				clause = upper
				emitBeforeNext = " "
				prevTok = upper
				continue
			}

			// AND / OR inside WHERE, HAVING, or a JOIN's ON gets an indented line —
			// unless the AND is the connective of a BETWEEN expression.
			if (upper == "AND" || upper == "OR") &&
				(clause == "WHERE" || clause == "HAVING" || clause == "ON" ||
					strings.HasSuffix(clause, "JOIN")) {
				if upper == "AND" && afterBetween {
					// this AND belongs to BETWEEN...AND — keep inline
					afterBetween = false
					// fall through to regular token handling below
				} else {
					afterBetween = false
					sb.WriteString("\n    ")
					sb.WriteString(upper)
					sb.WriteByte(' ')
					prevTok = upper
					continue
				}
			}
		}

		// regular token: resolve display value, then emit with spacing.
		display := v
		if !tok.opaque {
			u := strings.ToUpper(v)
			if knownSQLKeywords[u] {
				display = u
			}
		}

		if display == "BETWEEN" {
			afterBetween = true
		}

		// compact operators (e.g. :: -> ->>) and dot-qualified names suppress spaces.
		noSpace := compactOperators[display] || compactOperators[prevTok] || prevTok == "."
		if noSpace {
			emitBeforeNext = ""
		} else if emitBeforeNext != "" {
			sb.WriteString(emitBeforeNext)
			emitBeforeNext = ""
		} else {
			addSpace()
		}
		sb.WriteString(display)
		prevTok = v
	}

	return strings.TrimRight(sb.String(), " ")
}

// FormatSQL pretty-prints a SQL query.
// Keywords are uppercased. Major clauses start on new lines.
// SELECT column lists and WHERE/HAVING conditions are indented.
// Returns the original string unchanged when it is blank.
func FormatSQL(sql string) string {
	if strings.TrimSpace(sql) == "" {
		return sql
	}

	tokens := tokenizeSQL(sql)
	if len(tokens) == 0 {
		return sql
	}

	tokens = upcaseKeywords(tokens)
	tokens = mergeCompoundKeywords(tokens)
	return buildFormatted(tokens)
}
