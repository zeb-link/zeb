// Package shell contains the main Bubble Tea model for `zeb tui`.
// It is a live API-backed link browser with active create context controls.
package shell

import (
	"context"
	"fmt"
	"image/color"
	"net/url"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/tui/intro"
	"github.com/zeb-link/zeb/internal/ui/layout"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

const listLimit = 50

type Data struct {
	Client           *api.Client
	SpaceID          string
	Links            []api.Link
	NextCursor       *string
	Domains          []api.Domain
	Collections      []api.Collection
	ActiveDomain     string
	ActiveCollection string
}

type Model struct {
	spinner         spinner.Model
	commandInput    textinput.Model
	collectionInput textinput.Model
	intro           intro.Variant
	frame           int
	showingIntro    bool

	width  int
	height int

	client  *api.Client
	spaceID string

	links       []api.Link
	nextCursor  *string
	domains     []api.Domain
	collections []api.Collection

	linkIndex         int
	domainIndex       int
	collectionIndex   int
	focus             focusArea
	domainChanged     bool
	collectionChanged bool
	loading           bool
	message           string
}

type focusArea int

const (
	focusInput focusArea = iota
	focusDomain
	focusCollection
	focusNewCollection
)

type introTickMsg time.Time

type createResultMsg struct {
	response api.CreateLinkResponse
	err      error
}

type refreshResultMsg struct {
	response api.ListLinksResponse
	err      error
}

type createCollectionResultMsg struct {
	response api.CreateCollectionResponse
	err      error
}

func New(variant intro.Variant, data Data) Model {
	commandInput := textinput.New()
	commandInput.Prompt = "zeb > "
	commandInput.Placeholder = "paste a URL to create a short link"
	commandInput.CharLimit = 600
	commandInput.Focus()

	collectionInput := textinput.New()
	collectionInput.Prompt = "collection > "
	collectionInput.Placeholder = "new collection name"
	collectionInput.CharLimit = 120

	model := Model{
		spinner:         spinner.New(spinner.WithSpinner(spinner.Dot)),
		commandInput:    commandInput,
		collectionInput: collectionInput,
		intro:           variant,
		showingIntro:    true,
		width:           112,
		height:          32,
		client:          data.Client,
		spaceID:         data.SpaceID,
		links:           data.Links,
		nextCursor:      data.NextCursor,
		domains:         data.Domains,
		collections:     data.Collections,
		message:         "Type a URL to create a link. Tab cycles domain and collection.",
		domainIndex:     -1,
		collectionIndex: -1,
	}
	model.domainIndex = model.indexForDomain(data.ActiveDomain)
	model.collectionIndex = model.indexForCollection(data.ActiveCollection)
	if data.ActiveCollection != "" && model.collectionIndex == 0 {
		model.message = "Active collection cannot accept new links here; choose a collection or no collection."
	}
	return model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, introTick(), textinput.Blink)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		if m.showingIntro {
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			}
			return m, nil
		}
		if m.loading {
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			}
			return m, nil
		}
		if m.focus == focusNewCollection {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.cancelNewCollection()
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.collectionInput.Value())
				if name == "" {
					m.message = "Collection name cannot be blank."
					return m, nil
				}
				m.loading = true
				m.message = "Creating collection..."
				return m, m.createCollectionCmd(name)
			}
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "q":
			if m.focus != focusInput {
				return m, tea.Quit
			}
		case "tab":
			m.focusNext()
			return m, m.focusCmd()
		case "shift+tab", "backtab":
			m.focusPrevious()
			return m, m.focusCmd()
		case "up":
			switch m.focus {
			case focusInput:
				m.moveLink(-1)
			case focusDomain:
				m.cycleDomain(-1)
			case focusCollection:
				m.cycleCollection(-1)
			}
			return m, nil
		case "down":
			switch m.focus {
			case focusInput:
				m.moveLink(1)
			case focusDomain:
				m.cycleDomain(1)
			case focusCollection:
				m.cycleCollection(1)
			}
			return m, nil
		case "left":
			switch m.focus {
			case focusDomain:
				m.cycleDomain(-1)
				return m, nil
			case focusCollection:
				m.cycleCollection(-1)
				return m, nil
			}
		case "right":
			switch m.focus {
			case focusDomain:
				m.cycleDomain(1)
				return m, nil
			case focusCollection:
				m.cycleCollection(1)
				return m, nil
			}
		case "r":
			if m.focus != focusInput {
				m.loading = true
				m.message = "Refreshing links..."
				return m, m.refreshLinksCmd()
			}
		case "n":
			if m.focus == focusCollection {
				m.startNewCollection()
				return m, m.focusCmd()
			}
		case "enter":
			if m.focus == focusInput {
				value := strings.TrimSpace(m.commandInput.Value())
				if value != "" {
					if err := validateHTTPURL(value); err != nil {
						m.message = err.Error()
						return m, nil
					}
					m.loading = true
					m.message = "Creating link..."
					return m, m.createLinkCmd(value)
				}
			} else {
				m.setFocus(focusInput)
				return m, m.focusCmd()
			}
		}
	case introTickMsg:
		if m.showingIntro {
			m.frame++
			if m.frame >= intro.FrameCount {
				m.showingIntro = false
				return m, nil
			}
			return m, introTick()
		}
	case createResultMsg:
		m.loading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.commandInput.SetValue("")
		m.links = append([]api.Link{msg.response.Link}, m.links...)
		m.linkIndex = 0
		m.message = fmt.Sprintf("Created %s", displayShortLink(msg.response.Link))
		return m, nil
	case refreshResultMsg:
		m.loading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.links = msg.response.Links
		m.nextCursor = msg.response.NextCursor
		m.linkIndex = clamp(m.linkIndex, 0, len(m.links)-1)
		m.message = fmt.Sprintf("Loaded %d links", len(m.links))
		return m, nil
	case createCollectionResultMsg:
		m.loading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		collection := msg.response.Collection
		m.collections = append(m.collections, collection)
		m.collectionIndex = m.indexForCollection(collection.ID)
		m.collectionChanged = true
		m.collectionInput.SetValue("")
		m.setFocus(focusCollection)
		m.message = fmt.Sprintf("Created collection %s; new links go there.", collection.Name)
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	switch {
	case m.focus == focusInput && !m.showingIntro:
		var inputCmd tea.Cmd
		m.commandInput, inputCmd = m.commandInput.Update(msg)
		cmd = tea.Batch(cmd, inputCmd)
	case m.focus == focusNewCollection && !m.showingIntro:
		var inputCmd tea.Cmd
		m.collectionInput, inputCmd = m.collectionInput.Update(msg)
		cmd = tea.Batch(cmd, inputCmd)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if m.showingIntro {
		return altView("\n" + m.intro.RenderFrame(m.frame) + "\n")
	}

	width := m.width
	if width < 78 {
		width = 78
	}
	header := m.renderHeader(width)
	list := m.renderLinkList(width)
	footer := m.renderFooter(width)

	content := lipgloss.JoinVertical(lipgloss.Left, header, list)
	if m.height > 0 {
		content = layout.FillHeight(width, m.height-lipgloss.Height(footer)-1, content)
	}
	return altView(content + "\n" + footer)
}

// altView wraps rendered content as a full-screen (alt-screen) frame. In
// lipgloss/bubbletea v2, alt-screen is requested per-frame via View.AltScreen
// rather than a program option.
func altView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) ActiveDomain() string {
	return m.domainOption().Hostname
}

func (m Model) ActiveCollection() string {
	return m.collectionOption().ID
}

func (m Model) ContextChanged() bool {
	return m.domainChanged || m.collectionChanged
}

func (m Model) DomainChanged() bool {
	return m.domainChanged
}

func (m Model) CollectionChanged() bool {
	return m.collectionChanged
}

func (m *Model) focusNext() {
	m.setFocus((m.focus + 1) % 3)
}

func (m *Model) focusPrevious() {
	if m.focus == focusInput {
		m.setFocus(focusCollection)
		return
	}
	m.setFocus(m.focus - 1)
}

func (m *Model) setFocus(focus focusArea) {
	m.focus = focus
	if focus == focusInput {
		m.commandInput.Focus()
		m.collectionInput.Blur()
		m.message = "Command line focused. Type a URL to create a link."
		return
	}
	m.commandInput.Blur()
	if focus == focusNewCollection {
		m.collectionInput.Focus()
		m.message = "Name the new collection, then press enter."
		return
	}
	m.collectionInput.Blur()
	switch focus {
	case focusDomain:
		m.message = "Domain picker focused. Use arrows to choose the domain."
	case focusCollection:
		m.message = "Collection picker focused. Use arrows to choose where new links go, or n for a new collection."
	}
}

func (m Model) focusCmd() tea.Cmd {
	if m.focus != focusInput && m.focus != focusNewCollection {
		return nil
	}
	return textinput.Blink
}

func (m *Model) startNewCollection() {
	m.collectionInput.SetValue("")
	m.setFocus(focusNewCollection)
}

func (m *Model) cancelNewCollection() {
	m.collectionInput.SetValue("")
	m.setFocus(focusCollection)
	m.message = "New collection cancelled."
}

func (m *Model) moveLink(delta int) {
	m.linkIndex = clamp(m.linkIndex+delta, 0, len(m.links)-1)
}

func (m *Model) cycleDomain(delta int) {
	options := m.domainOptions()
	if len(options) == 0 {
		return
	}
	m.domainIndex = wrapIndex(m.domainIndex+delta, len(options))
	m.domainChanged = true
	m.message = "New links use domain " + domainLabel(m.domainOption().Hostname)
}

func (m *Model) cycleCollection(delta int) {
	options := m.collectionOptions()
	if len(options) == 0 {
		return
	}
	m.collectionIndex = wrapIndex(m.collectionIndex+delta, len(options))
	m.collectionChanged = true
	m.message = "New links go to " + collectionLabel(m.collectionOption())
}

func (m Model) createLinkCmd(targetURL string) tea.Cmd {
	return func() tea.Msg {
		input := api.CreateLinkInput{
			TargetURL:  targetURL,
			Domain:     m.ActiveDomain(),
			Collection: m.ActiveCollection(),
		}
		response, err := m.client.CreateLink(context.Background(), m.spaceID, input, true)
		return createResultMsg{response: response, err: err}
	}
}

func (m Model) refreshLinksCmd() tea.Cmd {
	return func() tea.Msg {
		response, err := m.client.ListLinks(context.Background(), m.spaceID, api.ListLinksOptions{Limit: listLimit})
		return refreshResultMsg{response: response, err: err}
	}
}

func (m Model) createCollectionCmd(name string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.client.CreateCollection(context.Background(), m.spaceID, api.CreateCollectionInput{Name: name})
		return createCollectionResultMsg{response: response, err: err}
	}
}

func (m Model) renderHeader(width int) string {
	title := theme.Heading.Render("Links")
	scope := fmt.Sprintf("%d loaded", len(m.links))
	if m.nextCursor != nil {
		scope += " · more available"
	}
	if m.loading {
		scope += " · " + m.spinner.View()
	}
	return lipgloss.NewStyle().
		Width(width - 2).
		MarginBottom(1).
		Render(title + "\n" + theme.MutedText.Render(scope))
}

func (m Model) renderLinkList(width int) string {
	if len(m.links) == 0 {
		return theme.MutedText.Render("No links found.")
	}
	rows := make([]string, 0, len(m.links))
	for i, link := range m.links {
		rows = append(rows, m.renderLink(link, i == m.linkIndex, width))
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderLink(link api.Link, focused bool, width int) string {
	short := displayShortLink(link)
	targetWidth := width - len(short) - 10
	if targetWidth < 28 {
		targetWidth = 28
	}
	dot, status := linkStatus(link.IsActive)
	line := fmt.Sprintf("%s%s %s %s", layout.Gutter(focused), dot, theme.LinkText.Render(short), theme.MutedText.Render("→ "+layout.Truncate(link.TargetURL, targetWidth)))
	title := ""
	if link.Title != nil && strings.TrimSpace(*link.Title) != "" {
		title = "\n  " + theme.MutedText.Render(layout.Truncate(strings.TrimSpace(*link.Title), width-8))
	}
	meta := "\n  " + theme.MutedText.Render(link.ID+" · ") + status
	return line + title + meta + "\n"
}

func (m Model) renderFooter(width int) string {
	domain := contextPill("Domain", domainLabel(m.domainOption().Hostname), theme.Sand, true, m.focus == focusDomain)
	collection := contextPill("Collection", collectionLabel(m.collectionOption()), theme.Collection, m.collectionOption().ID != "", m.focus == focusCollection)
	toolbar := lipgloss.JoinHorizontal(lipgloss.Center, domain, theme.MutedText.Render("  "), collection)

	input := m.renderFooterInput()
	help := footerHelp(m.focus, m.ContextChanged())
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		input,
		toolbar,
		theme.MutedText.Render(m.message),
		help,
	)
	return lipgloss.NewStyle().
		Width(width-2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(theme.Dim).
		PaddingTop(1).
		Render(body)
}

func (m Model) renderFooterInput() string {
	if m.focus == focusNewCollection {
		return m.collectionInput.View()
	}
	if m.focus == focusInput {
		return m.commandInput.View()
	}
	return theme.MutedText.Render(m.commandInput.Prompt + m.commandInput.Value())
}

type helpItem struct {
	key    string
	action string
}

func footerHelp(focus focusArea, changed bool) string {
	items := []helpItem{
		{key: "tab", action: "next"},
		{key: "↑/↓", action: "move"},
		{key: "enter", action: "create"},
		{key: "esc", action: "quit"},
	}
	switch focus {
	case focusDomain:
		items = []helpItem{
			{key: "←/→", action: "domain"},
			{key: "tab", action: "collection"},
			{key: "enter", action: "type"},
			{key: "r", action: "refresh"},
			{key: "q", action: "quit"},
		}
	case focusCollection:
		items = []helpItem{
			{key: "←/→", action: "collection"},
			{key: "n", action: "new"},
			{key: "tab", action: "type"},
			{key: "enter", action: "type"},
			{key: "r", action: "refresh"},
			{key: "q", action: "quit"},
		}
	case focusNewCollection:
		items = []helpItem{
			{key: "enter", action: "create collection"},
			{key: "esc", action: "cancel"},
			{key: "ctrl+c", action: "quit"},
		}
	}
	parts := make([]string, 0, len(items)+1)
	for _, item := range items {
		parts = append(parts, theme.KeyText.Render(item.key)+" "+theme.MutedText.Render(item.action))
	}
	if changed {
		parts = append(parts, theme.GoodText.Render("saves on quit"))
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(theme.Dim).Render("  ·  "))
}

func contextPill(label string, value string, tone color.Color, active bool, focused bool) string {
	return layout.Chip(label, value, layout.ChipOpts{Tone: tone, Active: active, Focused: focused})
}

func (m Model) domainOptions() []domainOption {
	options := []domainOption{{Hostname: "", Type: "default"}}
	for _, domain := range m.domains {
		options = append(options, domainOption{Hostname: domain.Hostname, Type: domain.Type})
	}
	return options
}

func (m Model) collectionOptions() []collectionOption {
	options := []collectionOption{{Name: "(no collection)"}}
	for _, collection := range m.collections {
		if collection.Type == "smart" {
			continue
		}
		options = append(options, collectionOption{ID: collection.ID, Name: collection.Name})
	}
	return options
}

func (m Model) domainOption() domainOption {
	options := m.domainOptions()
	if len(options) == 0 {
		return domainOption{}
	}
	return options[clamp(m.domainIndex, 0, len(options)-1)]
}

func (m Model) collectionOption() collectionOption {
	options := m.collectionOptions()
	if len(options) == 0 {
		return collectionOption{Name: "(no collection)"}
	}
	return options[clamp(m.collectionIndex, 0, len(options)-1)]
}

func (m Model) indexForDomain(hostname string) int {
	for i, option := range m.domainOptions() {
		if option.Hostname == hostname {
			return i
		}
	}
	return 0
}

func (m Model) indexForCollection(collectionID string) int {
	for i, option := range m.collectionOptions() {
		if option.ID == collectionID {
			return i
		}
	}
	return 0
}

type domainOption struct {
	Hostname string
	Type     string
}

type collectionOption struct {
	ID   string
	Name string
}

func validateHTTPURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", value)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%q must start with http:// or https://", value)
	}
	return nil
}

func displayShortLink(link api.Link) string {
	if link.ShortURL != "" {
		return link.ShortURL
	}
	return shortLink(link.Hostname, link.Path)
}

func shortLink(hostname string, path string) string {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		return hostname
	}
	return hostname + "/" + cleanPath
}

func linkStatus(active bool) (string, string) {
	if active {
		return layout.Dot(theme.Good), theme.GoodText.Render("active")
	}
	return layout.Dot(theme.Warn), theme.WarnText.Render("inactive")
}

func domainLabel(hostname string) string {
	if hostname == "" {
		return "server default"
	}
	return hostname
}

func collectionLabel(collection collectionOption) string {
	if collection.ID == "" {
		return "(no collection)"
	}
	return collection.Name
}

func clamp(value int, min int, max int) int {
	if max < min {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func wrapIndex(value int, length int) int {
	if length <= 0 {
		return 0
	}
	value %= length
	if value < 0 {
		value += length
	}
	return value
}

func introTick() tea.Cmd {
	return tea.Tick(intro.FrameDelay, func(t time.Time) tea.Msg {
		return introTickMsg(t)
	})
}
