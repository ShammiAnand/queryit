package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/cache"
	"github.com/shammianand/queryit/internal/completion"
)

type AutocompleteModel struct {
	engine      *completion.Engine
	suggestions []string
	selected    int
	visible     bool
	maxShow     int
}

func NewAutocompleteModel(schema *cache.SchemaCache) *AutocompleteModel {
	return &AutocompleteModel{
		engine:  completion.NewEngine(schema),
		maxShow: 8,
	}
}

func (a *AutocompleteModel) Update(input string) {
	if input == "" {
		a.Hide()
		return
	}
	a.suggestions = a.engine.Suggest(input)
	if len(a.suggestions) == 0 {
		a.visible = false
		return
	}
	a.visible = true
	if a.selected >= len(a.suggestions) {
		a.selected = 0
	}
}

func (a *AutocompleteModel) Next() {
	if !a.visible || len(a.suggestions) == 0 {
		return
	}
	a.selected = (a.selected + 1) % len(a.suggestions)
}

func (a *AutocompleteModel) Prev() {
	if !a.visible || len(a.suggestions) == 0 {
		return
	}
	a.selected = (a.selected - 1 + len(a.suggestions)) % len(a.suggestions)
}

func (a *AutocompleteModel) Accept() string {
	if !a.visible || len(a.suggestions) == 0 {
		return ""
	}
	s := a.suggestions[a.selected]
	a.Hide()
	return s
}

func (a *AutocompleteModel) Hide() {
	a.visible = false
	a.selected = 0
	a.suggestions = nil
}

func (a *AutocompleteModel) IsVisible() bool {
	return a.visible
}

func (a *AutocompleteModel) View() string {
	if !a.visible || len(a.suggestions) == 0 {
		return ""
	}

	show := a.suggestions
	offset := 0
	if a.selected >= a.maxShow {
		offset = a.selected - a.maxShow + 1
	}
	end := offset + a.maxShow
	if end > len(show) {
		end = len(show)
	}
	show = show[offset:end]

	var rows []string
	for i, s := range show {
		actualIdx := i + offset
		if actualIdx == a.selected {
			rows = append(rows, styleAutocompleteSelected.Render(s))
		} else {
			rows = append(rows, styleAutocompleteItem.Render(s))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
