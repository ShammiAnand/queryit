package tui

import "github.com/charmbracelet/bubbles/key"

type GlobalKeyMap struct {
	NewTab   key.Binding
	CloseTab key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding
	Quit     key.Binding
}

var GlobalKeys = GlobalKeyMap{
	NewTab:   key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "new tab")),
	CloseTab: key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "close tab")),
	NextTab:  key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next tab")),
	PrevTab:  key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev tab")),
	Quit:     key.NewBinding(key.WithKeys("ctrl+q"), key.WithHelp("ctrl+q", "quit")),
}

type InputKeyMap struct {
	Execute     key.Binding
	Cancel      key.Binding
	History     key.Binding
	SwitchFocus key.Binding
}

var InputKeys = InputKeyMap{
	Execute:     key.NewBinding(key.WithKeys("ctrl+enter", "f5"), key.WithHelp("ctrl+enter/f5", "execute")),
	Cancel:      key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel")),
	History:     key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "history")),
	SwitchFocus: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "switch focus")),
}

type ResultsKeyMap struct {
	NextPage    key.Binding
	PrevPage    key.Binding
	ToggleView  key.Binding
	NextRow     key.Binding
	PrevRow     key.Binding
	NextCol     key.Binding
	PrevCol     key.Binding
	OpenJSON    key.Binding
	CopyMenu    key.Binding
	Reconnect   key.Binding
	SwitchFocus key.Binding
}

var ResultsKeys = ResultsKeyMap{
	NextPage:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next page")),
	PrevPage:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev page")),
	ToggleView:  key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle view")),
	NextRow:     key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "next row")),
	PrevRow:     key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "prev row")),
	NextCol:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "next col")),
	PrevCol:     key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "prev col")),
	OpenJSON:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view JSON")),
	CopyMenu:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy/export")),
	Reconnect:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reconnect")),
	SwitchFocus: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "switch focus")),
}
