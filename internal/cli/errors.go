// Styled human error output. Every failed command funnels through Execute's
// error path, so this single renderer gives validation errors, guidance hints,
// and API failures one treatment: a red mark and readable message, embedded
// `zeb …` suggestion lines highlighted like commands, and a blank line after
// so the failure doesn't sit bunched against the next shell prompt.
package cli

import (
	"errors"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func printHumanError(err error) {
	message := err.Error()
	code := ""
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		// The wrapped form is "code: message"; humans get the message with
		// the machine code tucked in faintly after it.
		message = apiErr.Message
		code = apiErr.Code
	}

	mark := lipgloss.NewStyle().Bold(true).Foreground(theme.Bad).Render("✗")
	var b strings.Builder
	for i, line := range strings.Split(message, "\n") {
		switch {
		case i == 0:
			b.WriteString(mark + " " + theme.BodyText.Render(line))
			if code != "" {
				b.WriteString(" " + theme.GhostText.Render("("+code+")"))
			}
		case strings.HasPrefix(strings.TrimLeft(line, " "), "zeb "):
			// A suggested command — highlight it the way `zeb examples` does.
			b.WriteString(styleShell(line))
		default:
			b.WriteString(theme.SubtleText.Render(line))
		}
		b.WriteString("\n")
	}
	lipgloss.Fprintln(os.Stderr, strings.TrimRight(b.String(), "\n"))
	lipgloss.Fprintln(os.Stderr)
}
