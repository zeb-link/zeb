// Playground command opens a fake-data TUI for interaction design.
// It is intentionally not backed by the API; use it to tune layouts and
// keyboard flows before implementing real resource commands.
package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kerns/zlink-zeb/internal/tui/playground"
	"github.com/spf13/cobra"
)

func newPlaygroundCommand(root *rootOptions) *cobra.Command {
	var preview bool
	cmd := &cobra.Command{
		Use:   "playground",
		Short: "Open the fake-data TUI playground",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := playground.New()
			if preview {
				fmt.Println(model.Preview())
				return nil
			}
			_, err := tea.NewProgram(model).Run()
			return err
		},
	}
	cmd.Flags().BoolVar(&preview, "preview", false, "print the playground layout without opening the TUI")
	return cmd
}
