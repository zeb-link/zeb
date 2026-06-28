// Package playground is a fake-data orientation surface for TUI experiments.
// It models the intended CLI posture: simple link lists plus always-visible
// active domain and collection context.
package playground

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
)

type LinkStatus string

const (
	statusActive   LinkStatus = "active"
	statusInactive LinkStatus = "inactive"
	statusFallback LinkStatus = "fallback"
)

type Link struct {
	ShortURL     string
	Target       string
	Title        string
	Clicks       int
	UniqueClicks int
	Status       LinkStatus
	Collection   int
	CreatedLabel string
}

type Collection struct {
	Name  string
	Count int
	Smart bool
}

type Model struct {
	width                 int
	height                int
	linkIndex             int
	listCollectionIndex   int
	createCollectionIndex int
	domainIndex           int
	contextFocused        bool
	message               string
	commandInput          textinput.Model
}

func New() Model {
	commandInput := textinput.New()
	commandInput.Prompt = "zeb > "
	commandInput.Placeholder = "type a command or URL"
	commandInput.CharLimit = 280
	commandInput.Focus()
	return Model{
		width:                 112,
		height:                30,
		listCollectionIndex:   1,
		createCollectionIndex: -1,
		domainIndex:           0,
		contextFocused:        false,
		message:               "Type a command below. Try: links, collection use none, or https://example.com",
		commandInput:          commandInput,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab", "backtab":
			m.contextFocused = !m.contextFocused
			if m.contextFocused {
				m.commandInput.Blur()
				m.message = "Context controls focused. Press d for domain, c for new-link collection, tab or shift+tab to type."
				return m, nil
			}
			m.commandInput.Focus()
			m.message = "Command line focused. Type a command or URL."
			return m, textinput.Blink
		case "q":
			if m.contextFocused {
				return m, tea.Quit
			}
		case "up":
			m.move(-1)
			return m, nil
		case "down":
			m.move(1)
			return m, nil
		case "c":
			if m.contextFocused {
				m.cycleCreateCollection()
				return m, nil
			}
		case "d":
			if m.contextFocused {
				m.domainIndex = (m.domainIndex + 1) % len(sampleDomains)
				m.message = fmt.Sprintf("New links use domain %s", sampleDomains[m.domainIndex])
				return m, nil
			}
		case "enter":
			value := strings.TrimSpace(m.commandInput.Value())
			if value != "" {
				m.runCommand(value)
				m.commandInput.SetValue("")
				return m, nil
			}
			if links := m.visibleLinks(); len(links) > 0 {
				link := links[m.linkIndex]
				m.message = fmt.Sprintf("Inspecting %s -> %s", link.ShortURL, link.Target)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	width := m.width
	if width < 78 {
		width = 78
	}

	header := m.renderHeader(width)
	list := m.renderLinkList(width)
	footer := m.renderFooter(width)

	content := lipgloss.JoinVertical(lipgloss.Left, header, list)
	spacer := ""
	if m.height > 0 {
		contentHeight := lipgloss.Height(content)
		footerHeight := lipgloss.Height(footer)
		remaining := m.height - contentHeight - footerHeight
		if remaining > 0 {
			spacer = strings.Repeat("\n", remaining)
		}
	}

	return content + spacer + "\n" + footer
}

func (m Model) Preview() string {
	return m.View()
}

func (m *Model) move(delta int) {
	m.linkIndex = clamp(m.linkIndex+delta, 0, len(m.visibleLinks())-1)
}

func (m *Model) cycleCreateCollection() {
	m.createCollectionIndex++
	if m.createCollectionIndex >= len(sampleCollections) {
		m.createCollectionIndex = -1
	}
	value, ok := m.createCollectionLabel()
	if ok {
		m.message = fmt.Sprintf("New links go to %s", value)
		return
	}
	m.message = "New links will not be added to a collection"
}

func (m Model) createCollectionLabel() (string, bool) {
	if m.createCollectionIndex < 0 {
		return "(no collection)", false
	}
	if m.createCollectionIndex >= len(sampleCollections) {
		return "(no collection)", false
	}
	return sampleCollections[m.createCollectionIndex].Name, true
}

func (m *Model) runCommand(command string) {
	lower := strings.ToLower(command)
	switch {
	case looksLikeURL(command):
		domain := sampleDomains[m.domainIndex]
		collection, ok := m.createCollectionLabel()
		if ok {
			m.message = fmt.Sprintf("Would run: zeb %s --domain %s --collection %q", command, domain, collection)
			return
		}
		m.message = fmt.Sprintf("Would run: zeb %s --domain %s --no-collection", command, domain)
	case lower == "links" || lower == "links list":
		collection := sampleCollections[m.listCollectionIndex]
		m.message = fmt.Sprintf("Would run: zeb links --collection %q", collection.Name)
	case lower == "collection use none" || lower == "collection clear":
		m.createCollectionIndex = -1
		m.message = "New links will not be added to a collection"
	case strings.HasPrefix(lower, "collection use "):
		name := strings.TrimSpace(command[len("collection use "):])
		if m.setCreateCollection(name) {
			value, _ := m.createCollectionLabel()
			m.message = fmt.Sprintf("New links go to %s", value)
			return
		}
		m.message = fmt.Sprintf("Collection %q not found in fake data", name)
	case strings.HasPrefix(lower, "domain use "):
		hostname := strings.TrimSpace(command[len("domain use "):])
		if m.setDomain(hostname) {
			m.message = fmt.Sprintf("New links use domain %s", sampleDomains[m.domainIndex])
			return
		}
		m.message = fmt.Sprintf("Domain %q not found in fake data", hostname)
	case lower == "help":
		m.message = "Try: https://example.com, links, domain use zlnk.to, collection use Live analytics, collection clear"
	default:
		m.message = fmt.Sprintf("Unknown playground command: %s", command)
	}
}

func (m *Model) setCreateCollection(name string) bool {
	if strings.EqualFold(name, "none") || strings.EqualFold(name, "no collection") {
		m.createCollectionIndex = -1
		return true
	}
	for i, collection := range sampleCollections {
		if strings.EqualFold(collection.Name, name) {
			m.createCollectionIndex = i
			return true
		}
	}
	return false
}

func (m *Model) setDomain(hostname string) bool {
	for i, domain := range sampleDomains {
		if strings.EqualFold(domain, hostname) {
			m.domainIndex = i
			return true
		}
	}
	return false
}

func (m Model) renderHeader(width int) string {
	collection := sampleCollections[m.listCollectionIndex]
	title := theme.Heading.Render("Links")
	scope := theme.MutedText.Render(fmt.Sprintf("%s collection · %d links", collection.Name, len(m.visibleLinks())))
	if collection.Smart {
		scope = theme.MutedText.Render(fmt.Sprintf("%s smart collection · %d matching links", collection.Name, len(m.visibleLinks())))
	}
	return lipgloss.NewStyle().
		Width(width - 2).
		MarginBottom(1).
		Render(title + "\n" + scope)
}

func (m Model) renderLinkList(width int) string {
	var rows []string
	for i, link := range m.visibleLinks() {
		rows = append(rows, m.renderLinkCard(link, i == m.linkIndex, width))
	}
	if len(rows) == 0 {
		return theme.MutedText.Render("No links in this collection.")
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderLinkCard(link Link, focused bool, width int) string {
	borderColor := lipgloss.Color("238")
	if focused {
		borderColor = theme.Accent
	}
	cardWidth := width - 4
	innerWidth := cardWidth - 4
	if innerWidth < 60 {
		innerWidth = 60
	}

	titleWidth := innerWidth - 14
	if titleWidth < 28 {
		titleWidth = 28
	}

	title := titleStyle(focused).Render(truncate(link.Title, titleWidth))
	created := theme.MutedText.Render(link.CreatedLabel)
	top := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(titleWidth).Render(title),
		theme.MutedText.Render("  "),
		created,
	)

	arrow := theme.MutedText.Render(" -> ")
	route := fmt.Sprintf("%s%s%s", theme.Command.Render(link.ShortURL), arrow, theme.MutedText.Render(truncate(link.Target, innerWidth-24)))

	dot, label := statusParts(link.Status)
	stats := theme.MutedText.Render(fmt.Sprintf("Total clicks: %d   Unique clicks: %d", link.Clicks, link.UniqueClicks))
	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		dot,
		" ",
		label,
		lipgloss.NewStyle().Width(8).Render(""),
		stats,
	)

	body := strings.Join([]string{top, route, statusLine}, "\n")
	return lipgloss.NewStyle().
		Width(cardWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		MarginBottom(1).
		Render(body)
}

func (m Model) renderFooter(width int) string {
	domain := sampleDomains[m.domainIndex]
	domainControl := contextPill("d", "Domain", domain, theme.Accent2, true, m.contextFocused)
	collectionValue, hasCollection := m.createCollectionLabel()
	collectionControl := contextPill("c", "New links go to", collectionValue, theme.Accent, hasCollection, m.contextFocused)
	controls := lipgloss.JoinHorizontal(
		lipgloss.Center,
		domainControl,
		theme.MutedText.Render("  "),
		collectionControl,
	)
	helpText := "tab context  shift+tab context  ↑/↓ move  enter run  esc quit"
	if m.contextFocused {
		helpText = "d domain  c collection  tab type  shift+tab type  q quit"
	}
	right := theme.MutedText.Render(helpText)
	message := theme.MutedText.Render(m.message)
	input := m.renderCommandInput(width)

	contentWidth := width - 2
	spacerWidth := contentWidth - lipgloss.Width(controls) - lipgloss.Width(right)
	var line string
	if spacerWidth >= 2 {
		line = lipgloss.JoinHorizontal(
			lipgloss.Center,
			controls,
			strings.Repeat(" ", spacerWidth),
			right,
		)
	} else {
		line = controls + "\n" + right
	}

	return lipgloss.NewStyle().
		Width(width-2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("238")).
		PaddingTop(1).
		Render(input + "\n" + message + "\n" + line)
}

func (m Model) renderCommandInput(width int) string {
	input := m.commandInput
	input.Width = width - lipgloss.Width(input.Prompt) - 16
	if input.Width < 20 {
		input.Width = 20
	}
	borderColor := theme.Accent2
	if m.contextFocused {
		borderColor = lipgloss.Color("238")
	}
	return lipgloss.NewStyle().
		Width(width-6).
		MarginTop(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(input.View())
}

func contextPill(key string, label string, value string, color lipgloss.Color, active bool, focused bool) string {
	borderColor := lipgloss.Color("238")
	if focused {
		borderColor = color
	}
	keyChip := lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Padding(0, 1).
		Render(key)
	labelText := lipgloss.NewStyle().Foreground(theme.Muted).Render(label)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	if active {
		valueStyle = valueStyle.Foreground(theme.White).Bold(true)
	}
	valueText := valueStyle.Render(value)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(lipgloss.JoinHorizontal(lipgloss.Center, keyChip, " ", labelText, " ", valueText))
}

func statusParts(status LinkStatus) (string, string) {
	switch status {
	case statusActive:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("●"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("Active")
	case statusFallback:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Fallback redirect")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Inactive")
	}
}

func titleStyle(focused bool) lipgloss.Style {
	if focused {
		return theme.Heading
	}
	return lipgloss.NewStyle().Bold(true).Foreground(theme.Ink)
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

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(value) <= max {
		return value
	}
	if max <= 1 {
		return value[:max]
	}
	return value[:max-1] + "…"
}

func (m Model) visibleLinks() []Link {
	var links []Link
	for _, link := range sampleLinks {
		if link.Collection == m.listCollectionIndex {
			links = append(links, link)
		}
	}
	return links
}

func looksLikeURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

var sampleLinks = []Link{
	{ShortURL: "zeb.link/radar", Target: "https://zebralink.app/workbench", Title: "Workbench launch radar", Clicks: 1842, UniqueClicks: 922, Status: statusActive, Collection: 0, CreatedLabel: "27m ago"},
	{ShortURL: "zeb.link/q2", Target: "https://zebralink.app/q2-campaign", Title: "Q2 campaign brief", Clicks: 921, UniqueClicks: 408, Status: statusActive, Collection: 0, CreatedLabel: "1h ago"},
	{ShortURL: "zeb.link/sunset", Target: "https://zebralink.app/old-offer", Title: "Retired launch offer", Clicks: 77, UniqueClicks: 49, Status: statusInactive, Collection: 0, CreatedLabel: "4d ago"},
	{ShortURL: "zeb.link/live", Target: "https://zebralink.app/live-view", Title: "Public live analytics demo", Clicks: 2404, UniqueClicks: 1188, Status: statusActive, Collection: 1, CreatedLabel: "27m ago"},
	{ShortURL: "zlnk.to/chart", Target: "https://zebralink.app/live-chart", Title: "Realtime chart share", Clicks: 1409, UniqueClicks: 743, Status: statusFallback, Collection: 1, CreatedLabel: "2h ago"},
	{ShortURL: "zeb.link/docs", Target: "https://docs.zebralink.app", Title: "Developer documentation", Clicks: 308, UniqueClicks: 201, Status: statusActive, Collection: 3, CreatedLabel: "8h ago"},
}

var sampleCollections = []Collection{
	{Name: "Launch", Count: 18, Smart: false},
	{Name: "Live analytics", Count: 9, Smart: false},
	{Name: "Uncollected high traffic", Count: 12, Smart: true},
	{Name: "Developer docs", Count: 7, Smart: false},
}

var sampleDomains = []string{"zeb.link", "zlnk.to", "go.zebra.test"}
