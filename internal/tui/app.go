package tui

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shammianand/queryit/internal/config"
)

// ─── Profile form modal ───────────────────────────────────────────────────────

type formField struct {
	label       string
	value       string
	placeholder string
	password    bool // mask input
	optional    bool
}

type ProfileForm struct {
	visible  bool
	fields   []formField
	active   int
	err      string
	width    int
	height   int
	editMode bool
	origName string
}

const (
	fName = iota
	fHost
	fPort
	fDB
	fUser
	fPass
	fSSL
	fBastionUser
	fBastionHost
	fBastionPEM
	fCount
)

func newProfileForm() *ProfileForm {
	return &ProfileForm{
		fields: []formField{
			{label: "Profile name",            placeholder: "my-db"},
			{label: "Host",                    placeholder: "localhost"},
			{label: "Port",                    placeholder: "5432"},
			{label: "Database",                placeholder: "mydb"},
			{label: "User",                    placeholder: "postgres"},
			{label: "Password",                placeholder: "$ENV_VAR or plaintext", password: true},
			{label: "SSL mode",                placeholder: "prefer", optional: true},
			{label: "Bastion user",            placeholder: "ubuntu  (leave empty to skip)", optional: true},
			{label: "Bastion host",            placeholder: "10.0.0.1", optional: true},
			{label: "Bastion PEM",             placeholder: "~/.ssh/key.pem", optional: true},
		},
	}
}

func (f *ProfileForm) Show(width, height int) {
	f.visible = true
	f.active = 0
	f.err = ""
	f.width = width
	f.height = height
	f.editMode = false
	f.origName = ""
	for i := range f.fields {
		f.fields[i].value = ""
	}
}

func (f *ProfileForm) ShowEdit(name string, p *config.Profile, width, height int) {
	f.Show(width, height)
	f.editMode = true
	f.origName = name
	f.fields[fName].value = name
	f.fields[fHost].value = p.Host
	f.fields[fPort].value = fmt.Sprintf("%d", p.Port)
	f.fields[fDB].value = p.Database
	f.fields[fUser].value = p.User
	f.fields[fPass].value = p.Password
	f.fields[fSSL].value = p.SSLMode
	if p.Bastion != nil {
		f.fields[fBastionUser].value = p.Bastion.User
		f.fields[fBastionHost].value = p.Bastion.Host
		f.fields[fBastionPEM].value = p.Bastion.PEM
	}
}

func (f *ProfileForm) Hide() { f.visible = false }
func (f *ProfileForm) IsVisible() bool { return f.visible }

// viewportStart returns the index of the first field to show so the active
// field is always visible within the available height.
func (f *ProfileForm) viewportStart() int {
	// header: title(1) + subtitle(1) + blank(1) = 3 lines
	// each field: label(1) + input(3) = 4 lines
	// footer: blank(1) + err(1) = 2 lines (optional)
	reserved := 3 + 2
	linesPerField := 4
	maxVisible := (f.height - reserved) / linesPerField
	if maxVisible < 2 {
		maxVisible = 2
	}
	if maxVisible > fCount {
		maxVisible = fCount
	}
	start := f.active - maxVisible + 1
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > fCount {
		start = fCount - maxVisible
		if start < 0 {
			start = 0
		}
	}
	return start
}

func (f *ProfileForm) HandleKey(key string, runes []rune) (name string, p *config.Profile, done bool) {
	switch key {
	// ctrl+enter arrives as "ctrl+m" in most terminals/tmux; f5 as fallback
	case "ctrl+enter", "ctrl+m", "f5":
		return f.submit()
	case "esc":
		f.Hide()
		return "", nil, true
	case "tab", "down":
		f.active = (f.active + 1) % fCount
	case "shift+tab", "up":
		f.active = (f.active - 1 + fCount) % fCount
	case "enter":
		// enter on last field submits; otherwise advance to next field
		if f.active == fCount-1 {
			return f.submit()
		}
		f.active = (f.active + 1) % fCount
	case "backspace":
		v := []rune(f.fields[f.active].value)
		if len(v) > 0 {
			f.fields[f.active].value = string(v[:len(v)-1])
		}
	case "ctrl+u":
		f.fields[f.active].value = ""
	default:
		for _, r := range runes {
			if r >= 32 {
				f.fields[f.active].value += string(r)
			}
		}
	}
	return "", nil, false
}

func (f *ProfileForm) submit() (string, *config.Profile, bool) {
	name := strings.TrimSpace(f.fields[fName].value)
	host := strings.TrimSpace(f.fields[fHost].value)
	portStr := strings.TrimSpace(f.fields[fPort].value)
	db := strings.TrimSpace(f.fields[fDB].value)
	user := strings.TrimSpace(f.fields[fUser].value)
	pass := f.fields[fPass].value
	ssl := strings.TrimSpace(f.fields[fSSL].value)
	bastionUser := strings.TrimSpace(f.fields[fBastionUser].value)
	bastionHost := strings.TrimSpace(f.fields[fBastionHost].value)
	bastionPEM := strings.TrimSpace(f.fields[fBastionPEM].value)

	// validate required fields
	missing := []string{}
	if name == "" { missing = append(missing, "profile name") }
	if host == "" { missing = append(missing, "host") }
	if db == "" { missing = append(missing, "database") }
	if user == "" { missing = append(missing, "user") }
	if pass == "" { missing = append(missing, "password") }
	if len(missing) > 0 {
		f.err = "required: " + strings.Join(missing, ", ")
		return "", nil, false
	}

	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 5432
	}
	if ssl == "" {
		ssl = "prefer"
	}

	p := &config.Profile{
		Host:     host,
		Port:     port,
		Database: db,
		User:     user,
		Password: pass,
		SSLMode:  ssl,
	}
	if bastionUser != "" {
		p.Bastion = &config.BastionConfig{
			User: bastionUser,
			Host: bastionHost,
			PEM:  bastionPEM,
		}
	}

	_ = config.AddProfile(name, p)
	f.Hide()
	return name, p, true
}

func (f *ProfileForm) View() string {
	w := f.width - 6
	if w > 72 {
		w = 72
	}
	if w < 48 {
		w = 48
	}
	// styleOverlay has Padding(1,2) = 4 cols, 2 rows consumed by border+padding
	fieldW := w - 8 // inner usable width for the input box content

	accentBold := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	titleStr := "New connection profile"
	if f.editMode {
		titleStr = "Edit profile: " + f.origName
	}
	title := accentBold.Render(titleStr)
	subtitle := styleMuted.Render("tab · next field   shift+tab · prev   enter · save   esc · cancel")

	// how many fields fit
	linesPerField := 4 // label(1) + border-top(1) + content(1) + border-bottom(1)
	reserved := 2 /*title+subtitle*/ + 1 /*blank*/ + 2 /*overlay padding*/ + 2 /*err*/
	maxVisible := (f.height - reserved) / linesPerField
	if maxVisible < 2 {
		maxVisible = 2
	}
	if maxVisible > fCount {
		maxVisible = fCount
	}

	start := f.viewportStart()
	end := start + maxVisible
	if end > fCount {
		end = fCount
	}

	var scrollUp, scrollDown string
	if start > 0 {
		scrollUp = styleMuted.Render("  ↑ " + fmt.Sprintf("%d more above", start))
	}
	if end < fCount {
		scrollDown = styleMuted.Render("  ↓ " + fmt.Sprintf("%d more below", fCount-end))
	}

	var rows []string
	for i := start; i < end; i++ {
		field := f.fields[i]
		isFocused := i == f.active

		// label
		optTag := ""
		if field.optional {
			optTag = styleMuted.Render("  optional")
		}
		var label string
		if isFocused {
			label = accentBold.Render("▶ "+field.label) + optTag
		} else {
			label = styleMuted.Render("  "+field.label) + optTag
		}

		// displayed value
		val := field.value
		if field.password && val != "" {
			val = strings.Repeat("•", len([]rune(val)))
		}

		var content string
		if isFocused {
			// focused: show typed value + cursor; NO placeholder mixed in
			content = val + styleCursor.Render(" ")
		} else {
			if val != "" {
				content = val
			} else {
				content = styleMuted.Render(field.placeholder)
			}
		}

		borderFg := colorBorder
		if isFocused {
			borderFg = colorAccent
		}
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Padding(0, 1).
			Width(fieldW).
			Render(content)

		rows = append(rows, label, inputBox)
	}

	parts := []string{title, subtitle, ""}
	if scrollUp != "" {
		parts = append(parts, scrollUp)
	}
	parts = append(parts, rows...)
	if scrollDown != "" {
		parts = append(parts, scrollDown)
	}
	if f.err != "" {
		parts = append(parts, "", styleError.Render("  ✗ "+f.err))
	} else {
		// always show save hint at bottom so user knows how to submit
		parts = append(parts, "", styleMuted.Render("  enter on last field or tab through all to save"))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return styleOverlay.Width(w).Render(body)
}

// ─── Profile selector list ────────────────────────────────────────────────────

type ProfileSelector struct {
	visible       bool
	profiles      []string
	selected      int
	cfg           *config.Config
	form          *ProfileForm
	editingName   string // non-empty when form is in edit mode
	confirmDelete string // non-empty when waiting for delete confirmation
}

func NewProfileSelector(cfg *config.Config) *ProfileSelector {
	ps := &ProfileSelector{cfg: cfg, form: newProfileForm()}
	ps.refreshProfiles()
	return ps
}

func (ps *ProfileSelector) refreshProfiles() {
	ps.profiles = nil
	for name := range ps.cfg.Profiles {
		ps.profiles = append(ps.profiles, name)
	}
	sort.Strings(ps.profiles)
	if ps.selected >= len(ps.profiles) && len(ps.profiles) > 0 {
		ps.selected = len(ps.profiles) - 1
	}
}

func (ps *ProfileSelector) Show() {
	ps.visible = true
	ps.selected = 0
	ps.confirmDelete = ""
	ps.editingName = ""
	ps.refreshProfiles()
}

func (ps *ProfileSelector) Hide() {
	ps.visible = false
	ps.confirmDelete = ""
	ps.editingName = ""
}

func (ps *ProfileSelector) IsVisible() bool { return ps.visible || ps.form.IsVisible() }

func (ps *ProfileSelector) next() {
	total := len(ps.profiles) + 1
	if ps.selected < total-1 {
		ps.selected++
	}
}
func (ps *ProfileSelector) prev() {
	if ps.selected > 0 {
		ps.selected--
	}
}

func (ps *ProfileSelector) openEditForm(width, height int) {
	if ps.selected >= len(ps.profiles) {
		return
	}
	name := ps.profiles[ps.selected]
	p := ps.cfg.Profiles[name]
	ps.editingName = name
	ps.form.ShowEdit(name, p, width, height)
	ps.visible = false
}

// HandleKey returns (profileName, profile, done).
func (ps *ProfileSelector) HandleKey(key string, runes []rune, width, height int) (string, *config.Profile, bool) {
	// form is open — route to it
	if ps.form.IsVisible() {
		name, p, done := ps.form.HandleKey(key, runes)
		if done {
			if name != "" && p != nil {
				if ps.editingName != "" && ps.editingName != name {
					// name changed — remove old key
					delete(ps.cfg.Profiles, ps.editingName)
					_ = config.RemoveProfile(ps.editingName)
				}
				ps.editingName = ""
				ps.cfg.Profiles[name] = p
				ps.Hide()
				return name, p, true
			}
			// cancelled — back to list
			ps.editingName = ""
			ps.form.Hide()
			ps.visible = true
		}
		return "", nil, false
	}

	// delete confirmation prompt
	if ps.confirmDelete != "" {
		switch key {
		case "y", "Y":
			name := ps.confirmDelete
			ps.confirmDelete = ""
			delete(ps.cfg.Profiles, name)
			_ = config.RemoveProfile(name)
			ps.refreshProfiles()
		case "n", "N", "esc":
			ps.confirmDelete = ""
		}
		return "", nil, false
	}

	// list mode
	switch key {
	case "j", "down":
		ps.next()
	case "k", "up":
		ps.prev()
	case "enter":
		if ps.selected == len(ps.profiles) {
			ps.editingName = ""
			ps.form.Show(width, height)
			ps.visible = false
			return "", nil, false
		}
		name := ps.profiles[ps.selected]
		p := ps.cfg.Profiles[name]
		ps.Hide()
		return name, p, true
	case "e":
		if ps.selected < len(ps.profiles) {
			ps.openEditForm(width, height)
		}
	case "d":
		if ps.selected < len(ps.profiles) {
			ps.confirmDelete = ps.profiles[ps.selected]
		}
	case "esc":
		ps.Hide()
		return "", nil, true
	}
	return "", nil, false
}

func (ps *ProfileSelector) View(width, height int) string {
	if ps.form.IsVisible() {
		return ps.form.View()
	}
	if !ps.visible {
		return ""
	}

	w := width - 6
	if w > 72 {
		w = 72
	}
	if w < 48 {
		w = 48
	}

	accentBold := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	title := accentBold.Render("Connect to a database")
	hint := styleMuted.Render("j/k · navigate   enter · connect   e · edit   d · delete   esc · cancel")

	// delete confirmation takes over the whole modal
	if ps.confirmDelete != "" {
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			styleError.Render("Delete profile \""+ps.confirmDelete+"\"?"),
			"",
			lipgloss.NewStyle().Bold(true).Render("y")+" "+styleMuted.Render("yes — remove permanently")+"   "+
				lipgloss.NewStyle().Bold(true).Render("n / esc")+" "+styleMuted.Render("cancel"),
		)
		return styleOverlay.Width(w).Render(body)
	}

	arrow := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("▶")
	space := "  "

	var rows []string
	for i, name := range ps.profiles {
		p := ps.cfg.Profiles[name]
		isSelected := i == ps.selected

		// line 1: arrow + name
		prefix := space
		nameStyle := lipgloss.NewStyle().Foreground(colorFg).Bold(true)
		if isSelected {
			prefix = arrow + " "
		} else {
			nameStyle = lipgloss.NewStyle().Foreground(colorFg)
		}
		line1 := prefix + nameStyle.Render(name)

		// line 2: indented host detail + ssh tag
		hostStr := fmt.Sprintf("%s:%d / %s", p.Host, p.Port, p.Database)
		sshTag := ""
		if p.Bastion != nil {
			sshTag = "  " + lipgloss.NewStyle().Foreground(colorYellow).Render("⇒ "+p.Bastion.User+"@"+p.Bastion.Host)
		}
		line2 := "    " + styleMuted.Render(hostStr+sshTag)

		rows = append(rows, line1, line2, "") // blank line separator
	}

	// new profile entry (single line)
	newPrefix := space
	newStyle := styleMuted
	if ps.selected == len(ps.profiles) {
		newPrefix = arrow + " "
		newStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	}
	rows = append(rows, newPrefix+newStyle.Render("+ New profile"))

	parts := []string{title, hint, ""}
	parts = append(parts, rows...)
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return styleOverlay.Width(w).Render(body)
}

// ─── App model ────────────────────────────────────────────────────────────────

type AppModel struct {
	tabs     []*TabModel
	tabBar   *TabBar
	selector *ProfileSelector
	cfg      *config.Config
	width    int
	height   int
}

func NewApp(cfg *config.Config, autoProfile string) (*AppModel, tea.Cmd) {
	app := &AppModel{
		cfg:      cfg,
		tabBar:   NewTabBar(),
		selector: NewProfileSelector(cfg),
	}

	var cmd tea.Cmd
	if autoProfile != "" {
		p, ok := cfg.Profiles[autoProfile]
		if !ok {
			fmt.Fprintf(os.Stderr, "error: profile %q not found\n", autoProfile)
			os.Exit(1)
		}
		cmd = app.openTab(autoProfile, p)
	}

	return app, cmd
}

func (app *AppModel) openTab(name string, p *config.Profile) tea.Cmd {
	tab := NewTab(name, p, app.cfg.Settings)
	app.tabs = append(app.tabs, tab)
	app.tabBar.Add(name)
	app.resizeTabs()
	return tea.Batch(tab.Connect(), tab.InitCmds())
}

func (app *AppModel) resizeTabs() {
	for _, t := range app.tabs {
		t.SetSize(app.width, app.height-1) // -1 for tabbar
	}
}

func (app *AppModel) Init() tea.Cmd { return nil }

func (app *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.width = msg.Width
		app.height = msg.Height
		app.resizeTabs()
		return app, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			app.closeAll()
			return app, tea.Quit
		case "ctrl+t":
			if !app.selector.IsVisible() {
				app.selector.Show()
			}
			return app, nil
		case "ctrl+w":
			app.closeActiveTab()
			return app, nil
		case "tab", "ctrl+tab", "ctrl+n":
			// only switch tabs when selector is not open
			if !app.selector.IsVisible() {
				app.tabBar.Next()
			}
			return app, nil
		case "shift+tab", "ctrl+shift+tab", "ctrl+p":
			if !app.selector.IsVisible() {
				app.tabBar.Prev()
			}
			return app, nil
		}

		if app.selector.IsVisible() {
			name, profile, done := app.selector.HandleKey(msg.String(), msg.Runes, app.width, app.height)
			if done && name != "" && profile != nil {
				cmd := app.openTab(name, profile)
				return app, cmd
			}
			return app, nil
		}

		if len(app.tabs) > 0 {
			idx := app.tabBar.Active()
			if idx >= 0 && idx < len(app.tabs) {
				updated, cmd := app.tabs[idx].Update(msg)
				app.tabs[idx] = updated
				return app, cmd
			}
		}

	default:
		var cmds []tea.Cmd
		for i, t := range app.tabs {
			updated, cmd := t.Update(msg)
			app.tabs[i] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return app, tea.Batch(cmds...)
	}

	return app, nil
}

func (app *AppModel) closeActiveTab() {
	idx := app.tabBar.Active()
	if idx < 0 || idx >= len(app.tabs) {
		return
	}
	app.tabs[idx].Close()
	app.tabs = append(app.tabs[:idx], app.tabs[idx+1:]...)
	app.tabBar.Remove(idx)
}

func (app *AppModel) closeAll() {
	for _, t := range app.tabs {
		t.Close()
	}
}

func (app *AppModel) View() string {
	tabBarView := app.tabBar.View()

	var mainView string
	if len(app.tabs) == 0 {
		mainView = lipgloss.Place(app.width, app.height-1,
			lipgloss.Center, lipgloss.Center,
			styleWelcome.Render("queryit")+"\n\n"+
				styleMuted.Render("ctrl+t  open a connection")+"\n"+
				styleMuted.Render("ctrl+q  quit"),
		)
	} else {
		idx := app.tabBar.Active()
		if idx >= 0 && idx < len(app.tabs) {
			mainView = app.tabs[idx].View()
		}
	}

	if app.selector.IsVisible() {
		overlay := app.selector.View(app.width, app.height)
		overlayH := strings.Count(overlay, "\n") + 1
		// centre overlay vertically inside mainView
		topPad := (app.height - 1 - overlayH) / 2
		if topPad < 0 {
			topPad = 0
		}
		paddedOverlay := strings.Repeat("\n", topPad) + overlay
		// overlay replaces centre of main — simple join is fine for modal feel
		return lipgloss.JoinVertical(lipgloss.Left, tabBarView, paddedOverlay)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBarView, mainView)
}

func Run(cfg *config.Config, autoProfile string) error {
	app, _ := NewApp(cfg, autoProfile)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func Prompt(label string) string {
	fmt.Print(label + ": ")
	sc := bufio.NewScanner(os.Stdin)
	sc.Scan()
	return strings.TrimSpace(sc.Text())
}
