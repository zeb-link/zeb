// Package contextpicker renders the live context picker used by `zeb context`.
// It updates the same active domain and collection values that normal commands
// read from ~/.zlink/config.json.
package contextpicker

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
)

type section int

const (
	sectionDomain section = iota
	sectionCollection
)

type DomainOption struct {
	Hostname string
	Type     string
	Tier     *string
}

type CollectionOption struct {
	ID        string
	Name      string
	Type      string
	LinkCount int
}

type Selection struct {
	DomainChanged     bool
	Domain            string
	CollectionChanged bool
	CollectionID      string
	CollectionName    string
}

type Model struct {
	width            int
	activeSection    section
	domainIndex      int
	collectionIndex  int
	domains          []DomainOption
	collections      []CollectionOption
	selection        Selection
	activeDomain     string
	activeCollection string
	message          string
	quitting         bool
}

func New(domains []api.Domain, collections []api.Collection, activeDomain string, activeCollection string) Model {
	model := Model{
		width:            96,
		activeSection:    sectionDomain,
		activeDomain:     activeDomain,
		activeCollection: activeCollection,
		message:          "Press enter to set the highlighted value. Tab switches sections.",
	}
	model.domains = append(model.domains, DomainOption{Hostname: "", Type: "default"})
	for _, domain := range domains {
		model.domains = append(model.domains, DomainOption{
			Hostname: domain.Hostname,
			Type:     domain.Type,
			Tier:     domain.Tier,
		})
	}
	model.collections = append(model.collections, CollectionOption{Name: "(no collection)"})
	for _, collection := range collections {
		model.collections = append(model.collections, CollectionOption{
			ID:        collection.ID,
			Name:      collection.Name,
			Type:      collection.Type,
			LinkCount: collection.LinkCount,
		})
	}
	model.domainIndex = model.indexForDomain(activeDomain)
	model.collectionIndex = model.indexForCollection(activeCollection)
	return model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab", "shift+tab", "backtab", "left", "right":
			m.toggleSection()
			return m, nil
		case "up":
			m.move(-1)
			return m, nil
		case "down":
			m.move(1)
			return m, nil
		case "enter":
			m.applyHighlighted()
			return m, nil
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	width := m.width
	if width < 72 {
		width = 72
	}
	if width > 110 {
		width = 110
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(theme.White).Render("Zeb context")
	subtitle := theme.MutedText.Render("Choose defaults for new links. Flags on create still override these values.")
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderDomains((width-6)/2),
		"  ",
		m.renderCollections((width-6)/2),
	)
	help := theme.MutedText.Render("tab section  ↑/↓ move  enter set  q quit")
	message := theme.MutedText.Render(m.message)
	return "\n" + lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", body, "", message, help) + "\n"
}

func (m Model) Selection() Selection {
	return m.selection
}

func (m *Model) toggleSection() {
	if m.activeSection == sectionDomain {
		m.activeSection = sectionCollection
		return
	}
	m.activeSection = sectionDomain
}

func (m *Model) move(delta int) {
	if m.activeSection == sectionDomain {
		m.domainIndex = clamp(m.domainIndex+delta, 0, len(m.domains)-1)
		return
	}
	m.collectionIndex = clamp(m.collectionIndex+delta, 0, len(m.collections)-1)
}

func (m *Model) applyHighlighted() {
	if m.activeSection == sectionDomain {
		domain := m.domains[m.domainIndex]
		m.selection.DomainChanged = true
		m.selection.Domain = domain.Hostname
		m.activeDomain = domain.Hostname
		m.message = "Domain set to " + domainLabel(domain.Hostname)
		return
	}
	collection := m.collections[m.collectionIndex]
	if collection.Type == "smart" {
		m.message = fmt.Sprintf("%s is smart; choose a collection that can accept new links or no collection.", collection.Name)
		return
	}
	m.selection.CollectionChanged = true
	m.selection.CollectionID = collection.ID
	m.selection.CollectionName = collection.Name
	m.activeCollection = collection.ID
	m.message = "New links go to " + collectionLabel(collection)
}

func (m Model) renderDomains(width int) string {
	var rows []string
	rows = append(rows, sectionTitle("Domain", m.activeSection == sectionDomain))
	for i, domain := range m.domains {
		rows = append(rows, m.renderDomainRow(domain, i == m.domainIndex && m.activeSection == sectionDomain))
	}
	return panel(width, m.activeSection == sectionDomain, strings.Join(rows, "\n"))
}

func (m Model) renderCollections(width int) string {
	var rows []string
	rows = append(rows, sectionTitle("New links go to", m.activeSection == sectionCollection))
	for i, collection := range m.collections {
		rows = append(rows, m.renderCollectionRow(collection, i == m.collectionIndex && m.activeSection == sectionCollection))
	}
	return panel(width, m.activeSection == sectionCollection, strings.Join(rows, "\n"))
}

func (m Model) renderDomainRow(domain DomainOption, focused bool) string {
	value := domainLabel(domain.Hostname)
	meta := domain.Type
	if domain.Tier != nil {
		meta += " · " + *domain.Tier
	}
	return row(value, meta, domain.Hostname == m.activeDomain, focused, false)
}

func (m Model) renderCollectionRow(collection CollectionOption, focused bool) string {
	meta := "none"
	disabled := false
	if collection.ID != "" {
		meta = fmt.Sprintf("%s · %d links", collection.Type, collection.LinkCount)
		disabled = collection.Type == "smart"
	}
	return row(collectionLabel(collection), meta, collection.ID == m.activeCollection, focused, disabled)
}

func row(value string, meta string, active bool, focused bool, disabled bool) string {
	cursor := "  "
	if focused {
		cursor = "> "
	}
	valueStyle := lipgloss.NewStyle().Foreground(theme.White)
	metaStyle := theme.MutedText
	if disabled {
		valueStyle = valueStyle.Foreground(theme.Muted)
		metaStyle = metaStyle.Foreground(lipgloss.Color("240"))
	}
	if active {
		valueStyle = valueStyle.Bold(true)
	}
	activeMark := " "
	if active {
		activeMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
	}
	return cursor + activeMark + " " + valueStyle.Render(value) + "  " + metaStyle.Render(meta)
}

func sectionTitle(label string, active bool) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(theme.Muted)
	if active {
		style = style.Foreground(theme.Accent2)
	}
	return style.Render(label)
}

func panel(width int, focused bool, content string) string {
	border := lipgloss.Color("238")
	if focused {
		border = theme.Accent2
	}
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2).
		Render(content)
}

func domainLabel(hostname string) string {
	if hostname == "" {
		return "server default"
	}
	return hostname
}

func collectionLabel(collection CollectionOption) string {
	if collection.ID == "" {
		return "(no collection)"
	}
	return collection.Name
}

func (m Model) indexForDomain(hostname string) int {
	for i, domain := range m.domains {
		if domain.Hostname == hostname {
			return i
		}
	}
	return 0
}

func (m Model) indexForCollection(collectionID string) int {
	for i, collection := range m.collections {
		if collection.ID == collectionID {
			return i
		}
	}
	return 0
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
