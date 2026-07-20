// Context command opens the live picker for active domain and collection.
// It is intentionally narrow: choose defaults for future create commands.
package cli

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/tui/contextpicker"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newContextCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Pick active domain and collection",
		Long: "Pick active domain and collection for new links.\n\n" +
			"This writes the same ~/.zlink config used by normal commands. " +
			"Create flags like --domain, --collection, and --no-collection still override it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			domains, err := ctx.Client.ListDomains(cmd.Context(), ctx.SpaceID)
			if err != nil {
				return err
			}
			collections, err := ctx.Client.ListCollections(cmd.Context(), ctx.SpaceID)
			if err != nil {
				return err
			}

			model := contextpicker.New(domains.Domains, collections.Collections, cfg.ActiveDomain, cfg.ActiveCollection)
			final, err := tea.NewProgram(model).Run()
			if err != nil {
				return err
			}
			picker, ok := final.(contextpicker.Model)
			if !ok {
				return fmt.Errorf("context picker returned unexpected model")
			}
			selection := picker.Selection()
			changed := false
			if selection.DomainChanged {
				cfg.ActiveDomain = selection.Domain
				changed = true
			}
			if selection.CollectionChanged {
				cfg.ActiveCollection = selection.CollectionID
				changed = true
			}
			if !changed {
				lipgloss.Println(theme.MutedText.Render("Context unchanged."))
				air()
				return nil
			}
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			printContextSaved(selection)
			return nil
		},
	}
}

func printContextSaved(selection contextpicker.Selection) {
	done("Context saved")
	if selection.DomainChanged {
		value := theme.BodyText.Render(selection.Domain)
		if selection.Domain == "" {
			value = theme.FaintText.Render("server default")
		}
		lipgloss.Println("  " + theme.MutedText.Render("Domain      ") + value)
	}
	if selection.CollectionChanged {
		value := theme.CollectionText.Render(selection.CollectionName) + " " + theme.GhostText.Render("("+selection.CollectionID+")")
		if selection.CollectionID == "" {
			value = theme.FaintText.Render("none")
		}
		lipgloss.Println("  " + theme.MutedText.Render("Collection  ") + value)
	}
	air()
}
