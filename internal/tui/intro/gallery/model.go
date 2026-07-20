// Package gallery displays every intro variant in one fixed screen so visual
// options can be compared and screenshotted without racing the animation.
package gallery

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zeb-link/zeb/internal/tui/intro"
	"github.com/zeb-link/zeb/internal/ui/layout"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

type Model struct {
	width  int
	height int
	frame  int
}

func New(frame int) Model {
	return Model{
		width:  120,
		height: 40,
		frame:  frame,
	}
}

func Preview(frame int, width int) string {
	model := New(frame)
	model.width = width
	return model.render()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	width := m.width
	if width < 72 {
		width = 72
	}

	title := theme.Heading.Render("Zeb intro gallery")
	help := theme.MutedText.Render("Static comparison frame. Press esc or q to close.")
	tiles := m.renderTiles(width)
	content := lipgloss.JoinVertical(lipgloss.Left, title, help, "", tiles)

	if m.height > 0 {
		content = layout.FillHeight(width, m.height, content)
	}
	return "\n" + content
}

func (m Model) renderTiles(width int) string {
	tiles := make([]string, 0)
	for _, variant := range intro.Variants() {
		tiles = append(tiles, renderTile(variant, m.frame))
	}
	return layout.Grid(tiles, tileWidth, tileGap, width)
}

const (
	tileWidth = 36
	tileGap   = 2
)

func renderTile(variant intro.Variant, frame int) string {
	body := variant.RenderCompactFrame(frame)
	return lipgloss.NewStyle().
		Width(tileWidth).
		Height(13).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Dim).
		Padding(1, 1).
		MarginBottom(1).
		Render(body)
}
