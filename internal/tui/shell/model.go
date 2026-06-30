// Package shell contains the main Bubble Tea model for `zeb tui`.
// It is a live API-backed link browser with active create context controls.
package shell

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/tui/intro"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
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
	spinner      spinner.Model
	commandInput textinput.Model
	intro        intro.Variant
	frame        int
	showingIntro bool

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
	contextFocused    bool
	domainChanged     bool
	collectionChanged bool
	loading           bool
	message           string
}

type introTickMsg time.Time

type createResultMsg struct {
	response api.CreateLinkResponse
	err      error
}

type refreshResultMsg struct {
	response api.ListLinksResponse
	err      error
}

func New(variant intro.Variant, data Data) Model {
	commandInput := textinput.New()
	commandInput.Prompt = "zeb > "
	commandInput.Placeholder = "paste a URL to create a short link"
	commandInput.CharLimit = 600
	commandInput.Focus()

	model := Model{
		spinner:         spinner.New(spinner.WithSpinner(spinner.Dot)),
		commandInput:    commandInput,
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
		message:         "Type a URL to create a link. Tab focuses context controls.",
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
	case tea.KeyMsg:
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
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "q":
			if m.contextFocused {
				return m, tea.Quit
			}
		case "tab", "shift+tab", "backtab":
			m.toggleFocus()
			return m, m.focusCmd()
		case "up":
			m.moveLink(-1)
			return m, nil
		case "down":
			m.moveLink(1)
			return m, nil
		case "d":
			if m.contextFocused {
				m.cycleDomain()
				return m, nil
			}
		case "c":
			if m.contextFocused {
				m.cycleCollection()
				return m, nil
			}
		case "r":
			if m.contextFocused {
				m.loading = true
				m.message = "Refreshing links..."
				return m, m.refreshLinksCmd()
			}
		case "enter":
			if !m.contextFocused {
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
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if !m.contextFocused && !m.showingIntro {
		var inputCmd tea.Cmd
		m.commandInput, inputCmd = m.commandInput.Update(msg)
		cmd = tea.Batch(cmd, inputCmd)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.showingIntro {
		return "\n" + m.intro.RenderFrame(m.frame) + "\n"
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
		remaining := m.height - lipgloss.Height(content) - lipgloss.Height(footer) - 1
		if remaining > 0 {
			content += strings.Repeat("\n", remaining)
		}
	}
	return content + "\n" + footer
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

func (m Model) toggleFocus() {
	m.contextFocused = !m.contextFocused
	if m.contextFocused {
		m.commandInput.Blur()
		m.message = "Context controls focused. Press d for domain, c for collection, r to refresh."
		return
	}
	m.commandInput.Focus()
	m.message = "Command line focused. Type a URL to create a link."
}

func (m Model) focusCmd() tea.Cmd {
	if m.contextFocused {
		return nil
	}
	return textinput.Blink
}

func (m *Model) moveLink(delta int) {
	m.linkIndex = clamp(m.linkIndex+delta, 0, len(m.links)-1)
}

func (m *Model) cycleDomain() {
	if len(m.domainOptions()) == 0 {
		return
	}
	m.domainIndex = (m.domainIndex + 1) % len(m.domainOptions())
	m.domainChanged = true
	m.message = "New links use domain " + domainLabel(m.domainOption().Hostname)
}

func (m *Model) cycleCollection() {
	options := m.collectionOptions()
	if len(options) == 0 {
		return
	}
	m.collectionIndex = (m.collectionIndex + 1) % len(options)
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
	prefix := "  "
	if focused {
		prefix = "> "
	}
	short := displayShortLink(link)
	targetWidth := width - len(short) - 10
	if targetWidth < 28 {
		targetWidth = 28
	}
	dot, status := linkStatus(link.IsActive)
	line := fmt.Sprintf("%s%s %s %s", prefix, dot, theme.Command.Render(short), theme.MutedText.Render("-> "+truncate(link.TargetURL, targetWidth)))
	title := ""
	if link.Title != nil && strings.TrimSpace(*link.Title) != "" {
		title = "\n  " + theme.MutedText.Render(truncate(strings.TrimSpace(*link.Title), width-8))
	}
	meta := "\n  " + theme.MutedText.Render(link.ID+" · ") + status
	return line + title + meta + "\n"
}

func (m Model) renderFooter(width int) string {
	domain := contextPill("d", "Domain", domainLabel(m.domainOption().Hostname), theme.Accent2, true, m.contextFocused)
	collection := contextPill("c", "New links go to", collectionLabel(m.collectionOption()), theme.Accent, m.collectionOption().ID != "", m.contextFocused)
	toolbar := lipgloss.JoinHorizontal(lipgloss.Center, domain, theme.MutedText.Render("  "), collection)

	input := m.commandInput.View()
	if m.contextFocused {
		input = theme.MutedText.Render(m.commandInput.Prompt + m.commandInput.Value())
	}
	help := "tab context  ↑/↓ move  enter create  esc quit"
	if m.contextFocused {
		help = "d domain  c collection  r refresh  tab type  q quit"
	}
	if m.ContextChanged() {
		help += "  · saves on quit"
	}
	left := lipgloss.JoinVertical(lipgloss.Left, toolbar, input, theme.MutedText.Render(m.message))
	right := theme.MutedText.Render(help)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 2 {
		gap = 2
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right)
	return lipgloss.NewStyle().
		Width(width-2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("238")).
		PaddingTop(1).
		Render(body)
}

func contextPill(key string, label string, value string, color lipgloss.Color, active bool, focused bool) string {
	border := lipgloss.Color("238")
	if focused {
		border = color
	}
	valueStyle := lipgloss.NewStyle().Foreground(theme.White)
	if !active {
		valueStyle = theme.MutedText
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(theme.MutedText.Render(key+" "+label) + " " + valueStyle.Render(value))
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
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●"), lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("active")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●"), lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("inactive")
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

func truncate(value string, limit int) string {
	if limit <= 1 || len(value) <= limit {
		return value
	}
	return value[:limit-1] + "..."
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

func introTick() tea.Cmd {
	return tea.Tick(intro.FrameDelay, func(t time.Time) tea.Msg {
		return introTickMsg(t)
	})
}
