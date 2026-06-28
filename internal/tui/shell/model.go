// Package shell contains the main Bubble Tea model for `zeb tui`.
// Commands own CLI flags; this package owns interactive state and rendering.
package shell

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kerns/zlink-zeb/internal/tui/intro"
	"github.com/kerns/zlink-zeb/internal/ui/brand"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
)

type Model struct {
	spinner      spinner.Model
	intro        intro.Variant
	frame        int
	showingIntro bool
}

type introTickMsg time.Time

func New(variant intro.Variant) Model {
	return Model{
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot)),
		intro:        variant,
		showingIntro: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, introTick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
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
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.showingIntro {
		return "\n" + m.intro.RenderFrame(m.frame) + "\n"
	}

	title := theme.Heading.Render("Zeb")
	secondary := theme.MutedText.Render(brand.Welcome + ". Press q to quit.")
	commands := theme.Command.Render("auth  space  spec  status  links")
	return fmt.Sprintf("\n  %s %s\n  %s\n  %s\n\n", m.spinner.View(), title, secondary, commands)
}

func introTick() tea.Cmd {
	return tea.Tick(intro.FrameDelay, func(t time.Time) tea.Msg {
		return introTickMsg(t)
	})
}
