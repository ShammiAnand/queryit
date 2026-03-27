package tui

import (
	"strings"
	"testing"

	"github.com/shammianand/queryit/internal/db"
)

func TestRowsToCSV(t *testing.T) {
	cols := []string{"id", "name", "data"}
	pages := [][]db.Row{
		{{"1", "alice", "hello"}, {"2", "bob", "world"}},
		{{"3", "carol", `{"json":true}`}},
	}
	got := rowsToCSV(cols, pages)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	if len(lines) != 4 {
		t.Fatalf("want 4 lines (header + 3 rows), got %d: %q", len(lines), got)
	}
	if lines[0] != "id,name,data" {
		t.Errorf("header: got %q, want %q", lines[0], "id,name,data")
	}
	if lines[1] != "1,alice,hello" {
		t.Errorf("row1: got %q", lines[1])
	}
	if lines[3] != `3,carol,"{""json"":true}"` {
		t.Errorf("row3 (JSON quoting): got %q", lines[3])
	}
}

func TestRowToCSV(t *testing.T) {
	row := db.Row{"a", "b,c", `{"x":1}`}
	got := rowToCSV(row)
	want := `a,"b,c","{""x"":1}"`
	if got != want {
		t.Errorf("rowToCSV: got %q, want %q", got, want)
	}
}
