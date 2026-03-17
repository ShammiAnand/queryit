package tui

import "strings"

type TabBar struct {
	tabs      []string
	activeIdx int
	width     int
}

func NewTabBar() *TabBar {
	return &TabBar{}
}

func (tb *TabBar) Add(name string) int {
	tb.tabs = append(tb.tabs, name)
	tb.activeIdx = len(tb.tabs) - 1
	return tb.activeIdx
}

func (tb *TabBar) Remove(idx int) {
	if idx < 0 || idx >= len(tb.tabs) {
		return
	}
	tb.tabs = append(tb.tabs[:idx], tb.tabs[idx+1:]...)
	if tb.activeIdx >= len(tb.tabs) {
		tb.activeIdx = len(tb.tabs) - 1
	}
	if tb.activeIdx < 0 {
		tb.activeIdx = 0
	}
}

func (tb *TabBar) Next() {
	if len(tb.tabs) == 0 {
		return
	}
	tb.activeIdx = (tb.activeIdx + 1) % len(tb.tabs)
}

func (tb *TabBar) Prev() {
	if len(tb.tabs) == 0 {
		return
	}
	tb.activeIdx = (tb.activeIdx - 1 + len(tb.tabs)) % len(tb.tabs)
}

func (tb *TabBar) Active() int {
	return tb.activeIdx
}

func (tb *TabBar) Len() int {
	return len(tb.tabs)
}

func (tb *TabBar) SetWidth(w int) {
	tb.width = w
}

func (tb *TabBar) SetName(idx int, name string) {
	if idx >= 0 && idx < len(tb.tabs) {
		tb.tabs[idx] = name
	}
}

func (tb *TabBar) View() string {
	var parts []string
	for i, name := range tb.tabs {
		if i == tb.activeIdx {
			label := "[ " + name + "  ✕ ]"
			parts = append(parts, styleTabActive.Render(label))
		} else {
			label := "[ " + name + " ]"
			parts = append(parts, styleTabInactive.Render(label))
		}
	}
	parts = append(parts, styleTabNew.Render("[ + ]"))
	hint := styleMuted.Render("  ctrl+w: close tab")
	return strings.Join(parts, "") + hint
}
