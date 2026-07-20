// Context command opens the live picker for active domain and collection.
// It is intentionally narrow: choose defaults for future create commands.
package cli

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/tui/contextpicker"
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
				fmt.Println("Context unchanged.")
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
	fmt.Println("Context saved")
	if selection.DomainChanged {
		if selection.Domain == "" {
			fmt.Println("Domain: server default")
		} else {
			fmt.Printf("Domain: %s\n", selection.Domain)
		}
	}
	if selection.CollectionChanged {
		if selection.CollectionID == "" {
			fmt.Println("Collection: none")
		} else {
			fmt.Printf("Collection: %s (%s)\n", selection.CollectionName, selection.CollectionID)
		}
	}
}
