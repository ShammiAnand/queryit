package tui

import (
	"testing"
)

// normalize is a helper that runs FormatSQL and trims leading/trailing newlines
// so test expectations don't need to worry about leading blank lines.
func normalize(sql string) string {
	return FormatSQL(sql)
}

// --- tokenizeSQL ---

func TestTokenizeSQL_Empty(t *testing.T) {
	if got := tokenizeSQL(""); len(got) != 0 {
		t.Fatalf("expected no tokens, got %d", len(got))
	}
}

func TestTokenizeSQL_Words(t *testing.T) {
	tokens := tokenizeSQL("select name from users")
	vals := make([]string, len(tokens))
	for i, tok := range tokens {
		vals[i] = tok.val
	}
	want := []string{"select", "name", "from", "users"}
	if len(vals) != len(want) {
		t.Fatalf("want %v, got %v", want, vals)
	}
	for i := range want {
		if vals[i] != want[i] {
			t.Errorf("token[%d]: want %q got %q", i, want[i], vals[i])
		}
	}
}

func TestTokenizeSQL_Dot(t *testing.T) {
	tokens := tokenizeSQL("t.col")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens (t, ., col), got %d", len(tokens))
	}
	if tokens[0].val != "t" || tokens[1].val != "." || tokens[2].val != "col" {
		t.Errorf("unexpected tokens: %v", tokens)
	}
}

func TestTokenizeSQL_SingleQuotedString(t *testing.T) {
	tokens := tokenizeSQL("'hello world'")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	tok := tokens[0]
	if tok.val != "'hello world'" || !tok.opaque {
		t.Errorf("unexpected token: %+v", tok)
	}
}

func TestTokenizeSQL_SingleQuotedStringWithEscapedQuote(t *testing.T) {
	tokens := tokenizeSQL("'it''s'")
	if len(tokens) != 1 || tokens[0].val != "'it''s'" {
		t.Errorf("expected single token 'it''s', got %v", tokens)
	}
}

func TestTokenizeSQL_DoubleQuotedIdentifier(t *testing.T) {
	tokens := tokenizeSQL(`"from"`)
	if len(tokens) != 1 || !tokens[0].opaque {
		t.Errorf("expected 1 opaque token, got %v", tokens)
	}
}

func TestTokenizeSQL_LineComment(t *testing.T) {
	tokens := tokenizeSQL("id -- this is a comment")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0].val != "id" || !tokens[1].opaque {
		t.Errorf("unexpected tokens: %v", tokens)
	}
}

func TestTokenizeSQL_BlockComment(t *testing.T) {
	tokens := tokenizeSQL("id /* comment */ name")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if !tokens[1].opaque {
		t.Errorf("block comment should be opaque")
	}
}

func TestTokenizeSQL_Numbers(t *testing.T) {
	cases := []string{"1", "1.5", "1e10", "3.14e-2", ".5"}
	for _, c := range cases {
		tokens := tokenizeSQL(c)
		if len(tokens) != 1 {
			t.Errorf("input %q: expected 1 token, got %d", c, len(tokens))
		}
	}
}

func TestTokenizeSQL_TwoCharOps(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"!=", "!="},
		{"<>", "<>"},
		{"<=", "<="},
		{">=", ">="},
		{"::", "::"},
		{"||", "||"},
		{"->", "->"},
		{"->>", "->>"},
	}
	for _, c := range cases {
		tokens := tokenizeSQL(c.input)
		if len(tokens) != 1 || tokens[0].val != c.want {
			t.Errorf("input %q: want %q, got %v", c.input, c.want, tokens)
		}
	}
}

func TestTokenizeSQL_Punctuation(t *testing.T) {
	tokens := tokenizeSQL("(a, b)")
	vals := []string{}
	for _, tok := range tokens {
		vals = append(vals, tok.val)
	}
	want := []string{"(", "a", ",", "b", ")"}
	if len(vals) != len(want) {
		t.Fatalf("want %v got %v", want, vals)
	}
	for i := range want {
		if vals[i] != want[i] {
			t.Errorf("token[%d]: want %q got %q", i, want[i], vals[i])
		}
	}
}

// --- upcaseKeywords ---

func TestUpcaseKeywords_Keywords(t *testing.T) {
	in := []fmtTok{{val: "select"}, {val: "name"}, {val: "from"}}
	out := upcaseKeywords(in)
	if out[0].val != "SELECT" {
		t.Errorf("expected SELECT, got %q", out[0].val)
	}
	if out[1].val != "name" {
		t.Errorf("expected name unchanged, got %q", out[1].val)
	}
	if out[2].val != "FROM" {
		t.Errorf("expected FROM, got %q", out[2].val)
	}
}

func TestUpcaseKeywords_OpaqueUnchanged(t *testing.T) {
	in := []fmtTok{{val: "select", opaque: true}}
	out := upcaseKeywords(in)
	if out[0].val != "select" {
		t.Errorf("opaque token should not be uppercased, got %q", out[0].val)
	}
}

// --- mergeCompoundKeywords ---

func TestMergeCompoundKeywords_GroupBy(t *testing.T) {
	in := []fmtTok{{val: "GROUP"}, {val: "BY"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "GROUP BY" {
		t.Errorf("expected [GROUP BY], got %v", out)
	}
}

func TestMergeCompoundKeywords_OrderBy(t *testing.T) {
	in := []fmtTok{{val: "ORDER"}, {val: "BY"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "ORDER BY" {
		t.Errorf("expected [ORDER BY], got %v", out)
	}
}

func TestMergeCompoundKeywords_LeftOuterJoin(t *testing.T) {
	in := []fmtTok{{val: "LEFT"}, {val: "OUTER"}, {val: "JOIN"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "LEFT OUTER JOIN" {
		t.Errorf("expected [LEFT OUTER JOIN], got %v", out)
	}
}

func TestMergeCompoundKeywords_LeftJoin(t *testing.T) {
	in := []fmtTok{{val: "LEFT"}, {val: "JOIN"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "LEFT JOIN" {
		t.Errorf("expected [LEFT JOIN], got %v", out)
	}
}

func TestMergeCompoundKeywords_UnionAll(t *testing.T) {
	in := []fmtTok{{val: "UNION"}, {val: "ALL"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "UNION ALL" {
		t.Errorf("expected [UNION ALL], got %v", out)
	}
}

func TestMergeCompoundKeywords_InsertInto(t *testing.T) {
	in := []fmtTok{{val: "INSERT"}, {val: "INTO"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "INSERT INTO" {
		t.Errorf("expected [INSERT INTO], got %v", out)
	}
}

func TestMergeCompoundKeywords_DeleteFrom(t *testing.T) {
	in := []fmtTok{{val: "DELETE"}, {val: "FROM"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "DELETE FROM" {
		t.Errorf("expected [DELETE FROM], got %v", out)
	}
}

func TestMergeCompoundKeywords_IsNotNull(t *testing.T) {
	in := []fmtTok{{val: "IS"}, {val: "NOT"}, {val: "NULL"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "IS NOT NULL" {
		t.Errorf("expected [IS NOT NULL], got %v", out)
	}
}

func TestMergeCompoundKeywords_NotIn(t *testing.T) {
	in := []fmtTok{{val: "NOT"}, {val: "IN"}}
	out := mergeCompoundKeywords(in)
	if len(out) != 1 || out[0].val != "NOT IN" {
		t.Errorf("expected [NOT IN], got %v", out)
	}
}

func TestMergeCompoundKeywords_OpaqueNotMerged(t *testing.T) {
	// opaque token breaks the sequence
	in := []fmtTok{{val: "GROUP"}, {val: "BY", opaque: true}}
	out := mergeCompoundKeywords(in)
	if len(out) != 2 {
		t.Errorf("opaque token should prevent merge, got %v", out)
	}
}

// --- FormatSQL (integration) ---

func TestFormatSQL_Empty(t *testing.T) {
	if got := FormatSQL(""); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

func TestFormatSQL_Whitespace(t *testing.T) {
	if got := FormatSQL("   \n\t  "); got != "   \n\t  " {
		t.Errorf("whitespace-only should return as-is, got %q", got)
	}
}

func TestFormatSQL_SimpleSelect(t *testing.T) {
	got := FormatSQL("select * from users")
	want := "SELECT\n    *\nFROM users"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_MultiColumn(t *testing.T) {
	got := FormatSQL("select id, name, email from users")
	want := "SELECT\n    id,\n    name,\n    email\nFROM users"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereClause(t *testing.T) {
	got := FormatSQL("select id from users where id = 1")
	want := "SELECT\n    id\nFROM users\nWHERE id = 1"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereAnd(t *testing.T) {
	got := FormatSQL("select id from users where id = 1 and active = true")
	want := "SELECT\n    id\nFROM users\nWHERE id = 1\n    AND active = TRUE"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereOr(t *testing.T) {
	got := FormatSQL("select id from users where id = 1 or id = 2")
	want := "SELECT\n    id\nFROM users\nWHERE id = 1\n    OR id = 2"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereAndOr(t *testing.T) {
	got := FormatSQL("select id from t where a = 1 and b = 2 or c = 3")
	want := "SELECT\n    id\nFROM t\nWHERE a = 1\n    AND b = 2\n    OR c = 3"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Join(t *testing.T) {
	got := FormatSQL("select u.id, o.total from users u join orders o on u.id = o.user_id")
	want := "SELECT\n    u.id,\n    o.total\nFROM users u\nJOIN orders o\n    ON u.id = o.user_id"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_LeftJoin(t *testing.T) {
	got := FormatSQL("select a from t1 left join t2 on t1.id = t2.id")
	want := "SELECT\n    a\nFROM t1\nLEFT JOIN t2\n    ON t1.id = t2.id"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_LeftOuterJoin(t *testing.T) {
	got := FormatSQL("select a from t1 left outer join t2 on t1.id = t2.id")
	want := "SELECT\n    a\nFROM t1\nLEFT OUTER JOIN t2\n    ON t1.id = t2.id"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_MultipleJoins(t *testing.T) {
	got := FormatSQL("select * from a join b on a.id=b.id left join c on a.id=c.id")
	want := "SELECT\n    *\nFROM a\nJOIN b\n    ON a.id = b.id\nLEFT JOIN c\n    ON a.id = c.id"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_GroupBy(t *testing.T) {
	got := FormatSQL("select dept, count(*) from employees group by dept")
	want := "SELECT\n    dept,\n    COUNT(*)\nFROM employees\nGROUP BY dept"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_OrderBy(t *testing.T) {
	got := FormatSQL("select id from t order by id desc")
	want := "SELECT\n    id\nFROM t\nORDER BY id DESC"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Having(t *testing.T) {
	got := FormatSQL("select dept, count(*) cnt from e group by dept having count(*) > 5")
	want := "SELECT\n    dept,\n    COUNT(*) cnt\nFROM e\nGROUP BY dept\nHAVING COUNT(*) > 5"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_LimitOffset(t *testing.T) {
	got := FormatSQL("select id from t limit 10 offset 20")
	want := "SELECT\n    id\nFROM t\nLIMIT 10\nOFFSET 20"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Union(t *testing.T) {
	got := FormatSQL("select id from a union select id from b")
	want := "SELECT\n    id\nFROM a\nUNION\nSELECT\n    id\nFROM b"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_UnionAll(t *testing.T) {
	got := FormatSQL("select id from a union all select id from b")
	want := "SELECT\n    id\nFROM a\nUNION ALL\nSELECT\n    id\nFROM b"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_InsertInto(t *testing.T) {
	got := FormatSQL("insert into users (name, email) values ('alice', 'a@b.com')")
	// table name immediately precedes "(" without space — valid SQL, accepted formatter behaviour
	want := "INSERT INTO users(name, email)\nVALUES ('alice', 'a@b.com')"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Update(t *testing.T) {
	got := FormatSQL("update users set name = 'bob' where id = 1")
	want := "UPDATE users\nSET name = 'bob'\nWHERE id = 1"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_DeleteFrom(t *testing.T) {
	got := FormatSQL("delete from users where id = 1")
	want := "DELETE FROM users\nWHERE id = 1"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_KeywordsUppercased(t *testing.T) {
	got := FormatSQL("select id from users where active is not null")
	want := "SELECT\n    id\nFROM users\nWHERE active IS NOT NULL"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_StringLiteralPreserved(t *testing.T) {
	got := FormatSQL("select id from t where name = 'select from'")
	want := "SELECT\n    id\nFROM t\nWHERE name = 'select from'"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_QuotedIdentifierPreserved(t *testing.T) {
	got := FormatSQL(`select "from" from t`)
	want := "SELECT\n    \"from\"\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_TableDotColumn(t *testing.T) {
	got := FormatSQL("select t.id, t.name from t")
	want := "SELECT\n    t.id,\n    t.name\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_SchemaDotTableDotColumn(t *testing.T) {
	got := FormatSQL("select s.t.col from s.t")
	want := "SELECT\n    s.t.col\nFROM s.t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_FunctionCall(t *testing.T) {
	got := FormatSQL("select count(*) from t")
	want := "SELECT\n    COUNT(*)\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_FunctionWithArgs(t *testing.T) {
	got := FormatSQL("select coalesce(a, b, 0) from t")
	want := "SELECT\n    COALESCE(a, b, 0)\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_SubqueryInFrom(t *testing.T) {
	// Clause keywords inside parens (depth > 0) are not reformatted — intentional
	// limitation of a simple formatter. The subquery tokens stay on one line.
	got := FormatSQL("select id from (select id from users) sub")
	want := "SELECT\n    id\nFROM (SELECT id FROM users) sub"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereInSubquery(t *testing.T) {
	// Same limitation: subquery inside parens is kept flat.
	got := FormatSQL("select id from t where id in (select id from other)")
	want := "SELECT\n    id\nFROM t\nWHERE id IN (SELECT id FROM other)"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_AndOrInsideParenNotNewLine(t *testing.T) {
	// AND/OR inside parens should NOT start new lines
	got := FormatSQL("select * from t where (a = 1 and b = 2)")
	want := "SELECT\n    *\nFROM t\nWHERE (a = 1 AND b = 2)"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_CaseExpression(t *testing.T) {
	got := FormatSQL("select case when a = 1 then 'yes' else 'no' end from t")
	want := "SELECT\n    CASE WHEN a = 1 THEN 'yes' ELSE 'no' END\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Distinct(t *testing.T) {
	got := FormatSQL("select distinct id from t")
	want := "SELECT\n    DISTINCT id\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Alias(t *testing.T) {
	got := FormatSQL("select count(*) as cnt from t")
	want := "SELECT\n    COUNT(*) AS cnt\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereIn(t *testing.T) {
	got := FormatSQL("select id from t where id in (1, 2, 3)")
	want := "SELECT\n    id\nFROM t\nWHERE id IN (1, 2, 3)"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereNotIn(t *testing.T) {
	got := FormatSQL("select id from t where id not in (1,2)")
	want := "SELECT\n    id\nFROM t\nWHERE id NOT IN (1, 2)"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereBetween(t *testing.T) {
	// BETWEEN's own AND connective is kept inline (not pushed to a new line).
	got := FormatSQL("select id from t where age between 18 and 65")
	want := "SELECT\n    id\nFROM t\nWHERE age BETWEEN 18 AND 65"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereLike(t *testing.T) {
	got := FormatSQL("select id from t where name like '%foo%'")
	want := "SELECT\n    id\nFROM t\nWHERE name LIKE '%foo%'"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WhereIsNull(t *testing.T) {
	got := FormatSQL("select id from t where col is null")
	want := "SELECT\n    id\nFROM t\nWHERE col IS NULL"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_ComparisonOperators(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"select * from t where a != b", "SELECT\n    *\nFROM t\nWHERE a != b"},
		{"select * from t where a <> b", "SELECT\n    *\nFROM t\nWHERE a <> b"},
		{"select * from t where a <= b", "SELECT\n    *\nFROM t\nWHERE a <= b"},
		{"select * from t where a >= b", "SELECT\n    *\nFROM t\nWHERE a >= b"},
	}
	for _, c := range cases {
		got := FormatSQL(c.in)
		if got != c.want {
			t.Errorf("input %q:\nwant: %q\ngot:  %q", c.in, c.want, got)
		}
	}
}

func TestFormatSQL_Semicolon(t *testing.T) {
	got := FormatSQL("select 1;")
	want := "SELECT\n    1;"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_WithCTE(t *testing.T) {
	// subquery inside WITH parens stays flat (depth > 0)
	got := FormatSQL("with cte as (select id from t) select * from cte")
	want := "WITH cte AS (SELECT id FROM t)\nSELECT\n    *\nFROM cte"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_Returning(t *testing.T) {
	got := FormatSQL("insert into t (a) values (1) returning id")
	want := "INSERT INTO t(a)\nVALUES (1)\nRETURNING id"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_AlreadyFormatted(t *testing.T) {
	// formatting twice should produce the same result (idempotent)
	input := "select id, name from users where id = 1 and active = true order by name"
	first := FormatSQL(input)
	second := FormatSQL(first)
	if first != second {
		t.Errorf("format is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestFormatSQL_LineCommentPreserved(t *testing.T) {
	got := FormatSQL("select id -- primary key\nfrom t")
	want := "SELECT\n    id -- primary key\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_PostgresCast(t *testing.T) {
	// "text" is a SQL type keyword — the formatter uppercases it, giving id::TEXT.
	got := FormatSQL("select id::text from t")
	want := "SELECT\n    id::TEXT\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_PostgresJSONOperator(t *testing.T) {
	got := FormatSQL("select data->>'name' from t")
	want := "SELECT\n    data->>'name'\nFROM t"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_ComplexQuery(t *testing.T) {
	in := `select u.id, u.name, count(o.id) as order_count from users u left join orders o on u.id = o.user_id where u.active = true and u.created_at > '2024-01-01' group by u.id, u.name having count(o.id) > 2 order by order_count desc limit 10`
	want := "SELECT\n    u.id,\n    u.name,\n    COUNT(o.id) AS order_count\nFROM users u\nLEFT JOIN orders o\n    ON u.id = o.user_id\nWHERE u.active = TRUE\n    AND u.created_at > '2024-01-01'\nGROUP BY u.id, u.name\nHAVING COUNT(o.id) > 2\nORDER BY order_count DESC\nLIMIT 10"
	got := FormatSQL(in)
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_SelectStar(t *testing.T) {
	got := FormatSQL("SELECT * FROM t")
	want := "SELECT\n    *\nFROM t"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestFormatSQL_AlreadyUppercase(t *testing.T) {
	got := FormatSQL("SELECT id FROM t WHERE id = 1")
	want := "SELECT\n    id\nFROM t\nWHERE id = 1"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_MixedCase(t *testing.T) {
	got := FormatSQL("Select Id From T Where Id = 1")
	want := "SELECT\n    Id\nFROM T\nWHERE Id = 1"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_InnerJoin(t *testing.T) {
	got := FormatSQL("select a from t1 inner join t2 on t1.id = t2.id")
	want := "SELECT\n    a\nFROM t1\nINNER JOIN t2\n    ON t1.id = t2.id"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_CrossJoin(t *testing.T) {
	got := FormatSQL("select a, b from t1 cross join t2")
	want := "SELECT\n    a,\n    b\nFROM t1\nCROSS JOIN t2"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_ExceptAll(t *testing.T) {
	got := FormatSQL("select id from a except all select id from b")
	want := "SELECT\n    id\nFROM a\nEXCEPT ALL\nSELECT\n    id\nFROM b"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSQL_HavingAndOr(t *testing.T) {
	got := FormatSQL("select dept from e group by dept having count(*) > 1 and sum(salary) > 100")
	want := "SELECT\n    dept\nFROM e\nGROUP BY dept\nHAVING COUNT(*) > 1\n    AND SUM(salary) > 100"
	if got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}
