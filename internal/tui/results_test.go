package tui

import "testing"

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
