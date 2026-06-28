// Package theme defines shared terminal styles.
// Keep Lip Gloss color choices here so commands and TUI components use one
// visual vocabulary.
package theme

import "github.com/charmbracelet/lipgloss"

const (
	Accent  = lipgloss.Color("212")
	Accent2 = lipgloss.Color("117")
	Muted   = lipgloss.Color("244")
	Ink     = lipgloss.Color("250")
	White   = lipgloss.Color("15")
)

var (
	Heading   = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	MutedText = lipgloss.NewStyle().Foreground(Muted)
	Command   = lipgloss.NewStyle().Foreground(Accent2)
)
