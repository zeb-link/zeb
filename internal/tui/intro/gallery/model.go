// Package gallery displays every intro variant in one fixed screen so visual
// options can be compared and screenshotted without racing the animation.
package gallery

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zeb-link/zeb/internal/tui/intro"
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
	return model.View()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	width := m.width
	if width < 72 {
		width = 72
	}

	title := theme.Heading.Render("Zeb intro gallery")
	help := theme.MutedText.Render("Static comparison frame. Press esc or q to close.")
	tiles := m.renderTiles(width)
	content := lipgloss.JoinVertical(lipgloss.Left, title, help, "", tiles)

	if m.height > 0 {
		remaining := m.height - lipgloss.Height(content)
		if remaining > 0 {
			content += strings.Repeat("\n", remaining)
		}
	}
	return "\n" + content
}

func (m Model) renderTiles(width int) string {
	var tiles []string
	for _, variant := range intro.Variants() {
		tiles = append(tiles, renderTile(variant, m.frame))
	}

	columns := 3
	if width < 118 {
		columns = 2
	}
	if width < 80 {
		columns = 1
	}
	if columns == 1 {
		return lipgloss.JoinVertical(lipgloss.Left, tiles...)
	}

	var rows []string
	for i := 0; i < len(tiles); i += columns {
		rowTiles := tiles[i:min(i+columns, len(tiles))]
		parts := make([]string, 0, len(rowTiles)*2)
		for idx, tile := range rowTiles {
			if idx > 0 {
				parts = append(parts, "  ")
			}
			parts = append(parts, tile)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderTile(variant intro.Variant, frame int) string {
	body := variant.RenderCompactFrame(frame)
	return lipgloss.NewStyle().
		Width(36).
		Height(13).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1, 1).
		MarginBottom(1).
		Render(body)
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
