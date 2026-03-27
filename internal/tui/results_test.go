package tui

import (
	"testing"

	"github.com/shammianand/queryit/internal/db"
)

func TestIsJSON(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{`{"key":"value"}`, true},
		{`[1,2,3]`, true},
		{`{"nested":{"a":1}}`, true},
		{`[]`, true},
		{`{}`, true},
		{`hello`, false},
		{`123`, false},
		{`"string"`, false},
		{``, false},
		{`{broken`, false},
		{`  {"spaced": true}  `, true},
	}
	for _, c := range cases {
		got := isJSON(c.input)
		if got != c.want {
			t.Errorf("isJSON(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func makeTestResult() *db.ResultSet {
	return &db.ResultSet{
		Columns: []string{"id", "name", "data"},
		Pages: [][]db.Row{
			{
				{"1", "alice", `{"role":"admin"}`},
				{"2", "bob", "plain"},
			},
		},
		Total: 2,
	}
}

func TestCellCursorNavigation(t *testing.T) {
	r := NewResultsModel(ViewTable, 20)
	r.SetResult(makeTestResult())

	// initial state
	if r.currentCol != 0 {
		t.Fatalf("initial currentCol: got %d, want 0", r.currentCol)
	}
	if got := r.CurrentCell(); got != "1" {
		t.Errorf("CurrentCell at (0,0): got %q, want %q", got, "1")
	}

	// move right
	r.NextCol()
	if r.currentCol != 1 {
		t.Errorf("after NextCol: got %d, want 1", r.currentCol)
	}
	if got := r.CurrentCell(); got != "alice" {
		t.Errorf("CurrentCell at (0,1): got %q, want %q", got, "alice")
	}

	// move to last col
	r.NextCol()
	if r.currentCol != 2 {
		t.Errorf("after 2nd NextCol: got %d, want 2", r.currentCol)
	}

	// cannot go past last col
	r.NextCol()
	if r.currentCol != 2 {
		t.Errorf("NextCol past end: got %d, want 2", r.currentCol)
	}

	// move left
	r.PrevCol()
	if r.currentCol != 1 {
		t.Errorf("after PrevCol: got %d, want 1", r.currentCol)
	}

	// cannot go before 0
	r.PrevCol()
	r.PrevCol()
	if r.currentCol != 0 {
		t.Errorf("PrevCol past 0: got %d, want 0", r.currentCol)
	}
}

func TestCurrentRow(t *testing.T) {
	r := NewResultsModel(ViewTable, 20)
	r.SetResult(makeTestResult())

	row := r.CurrentRow()
	if len(row) != 3 || row[0] != "1" {
		t.Errorf("CurrentRow: got %v", row)
	}

	r.NextRow()
	row = r.CurrentRow()
	if row[0] != "2" {
		t.Errorf("CurrentRow after NextRow: got %v", row)
	}
}

func TestCurrentCellNilResult(t *testing.T) {
	r := NewResultsModel(ViewTable, 20)
	if got := r.CurrentCell(); got != "" {
		t.Errorf("CurrentCell with nil result: got %q, want empty", got)
	}
	if r.CurrentRow() != nil {
		t.Error("CurrentRow with nil result: want nil")
	}
}
