package completion

import (
	"strings"

	"github.com/shammianand/queryit/internal/cache"
)

type Engine struct {
	schema *cache.SchemaCache
}

func NewEngine(schema *cache.SchemaCache) *Engine {
	return &Engine{schema: schema}
}

func (e *Engine) Suggest(input string) []string {
	if input == "" {
		return nil
	}

	// Backslash commands — suggest only command names
	if strings.HasPrefix(input, `\`) {
		return filterPrefix(BackslashCommands, input)
	}

	upper := strings.ToUpper(input)
	tokens := tokenize(upper)

	// Determine the word currently being typed
	currentWord := ""
	if len(tokens) > 0 && !strings.HasSuffix(input, " ") {
		currentWord = tokens[len(tokens)-1]
	}

	// Only suggest when the user has typed at least 1 character
	if currentWord == "" {
		return nil
	}

	n := len(tokens)
	prevToken := ""
	prevPrev := ""
	if currentWord == "" {
		if n >= 1 {
			prevToken = tokens[n-1]
		}
		if n >= 2 {
			prevPrev = tokens[n-2]
		}
	} else {
		if n >= 2 {
			prevToken = tokens[n-2]
		}
		if n >= 3 {
			prevPrev = tokens[n-3]
		}
	}
	_ = prevPrev

	var candidates []string

	switch {
	case strings.HasSuffix(strings.TrimRight(input, " "), "."):
		// table.column — suggest columns of that table
		trimmed := strings.TrimRight(input, " ")
		dotIdx := strings.LastIndex(trimmed, ".")
		if dotIdx > 0 {
			tbl := strings.ToLower(trimmed[:dotIdx])
			// strip any prior tokens to get just the table name
			if spaceIdx := strings.LastIndexAny(tbl, " \t"); spaceIdx >= 0 {
				tbl = tbl[spaceIdx+1:]
			}
			candidates = e.schema.ColumnNamesForTable(tbl)
		}

	case prevToken == "FROM" || prevToken == "JOIN" || prevToken == "UPDATE" ||
		prevToken == "INTO" || prevToken == "TABLE":
		candidates = e.schema.TableNames()

	case prevToken == "WHERE" || prevToken == "AND" || prevToken == "OR" ||
		prevToken == "SET" || prevToken == "ON" || prevToken == "BY":
		candidates = e.schema.AllColumnNames()

	case prevToken == "SELECT":
		cols := e.schema.AllColumnNames()
		candidates = append([]string{"*"}, cols...)

	default:
		// Only suggest table names when typing something that could be a table
		tables := e.schema.TableNames()
		cols := e.schema.AllColumnNames()
		candidates = append(tables, cols...)
	}

	return fuzzyFilter(candidates, currentWord)
}

func tokenize(s string) []string {
	return strings.Fields(s)
}

func filterPrefix(list []string, prefix string) []string {
	var out []string
	lp := strings.ToLower(prefix)
	for _, s := range list {
		if strings.HasPrefix(strings.ToLower(s), lp) {
			out = append(out, s)
		}
	}
	return out
}

func fuzzyFilter(list []string, word string) []string {
	if word == "" {
		if len(list) > 20 {
			return list[:20]
		}
		return list
	}
	lw := strings.ToLower(word)
	var out []string
	for _, s := range list {
		if strings.Contains(strings.ToLower(s), lw) {
			out = append(out, s)
		}
	}
	return out
}

func keywordsLower() []string {
	out := make([]string, len(SQLKeywords))
	copy(out, SQLKeywords)
	return out
}
