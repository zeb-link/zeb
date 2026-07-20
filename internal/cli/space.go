// Space commands manage the active space context.
package cli

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newSpaceCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "space",
		Aliases: []string{"spaces"},
		Short:   "Manage active space context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return newSpaceCurrentCommand(root).RunE(cmd, args)
		},
	}
	cmd.AddCommand(newSpaceCurrentCommand(root), newSpaceListCommand(root), newSpaceUseCommand(root))
	return cmd
}

func newSpaceCurrentCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show active space",
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceID, err := config.ResolveSpace(root.SpaceID)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]string{"activeSpace": spaceID})
			}
			field("Active space", emptyLabel(spaceID), 14)
			air()
			return nil
		},
	}
}

func newSpaceListCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List spaces available to the current API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := resolveClient(root)
			if err != nil {
				return err
			}
			me, err := client.GetMe(cmd.Context())
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(me.AccessibleSpaces)
			}
			section("Spaces")
			col := 0
			for _, space := range me.AccessibleSpaces {
				if width := utf8.RuneCountInString(space.Name); width > col {
					col = width
				}
			}
			for _, space := range me.AccessibleSpaces {
				pad := strings.Repeat(" ", col-utf8.RuneCountInString(space.Name)+2)
				lipgloss.Println("  " + theme.CommandText.Render(space.Name) + pad +
					theme.GhostText.Render(space.ID) + "  " + theme.MutedText.Render(space.Role))
			}
			air()
			return nil
		},
	}
}

func newSpaceUseCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "use <space-id-or-name>",
		Short: "Set active space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := resolveClient(root)
			if err != nil {
				return err
			}
			me, err := client.GetMe(cmd.Context())
			if err != nil {
				return err
			}
			space, err := resolveSpace(me.AccessibleSpaces, args[0])
			if err != nil {
				return err
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.ActiveSpace = space.ID
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"activeSpace": space.ID, "space": space})
			}
			doneStyled(theme.BodyText.Render("Active space set to "), theme.Title.Render(space.Name), " ", theme.GhostText.Render("("+space.ID+")"))
			air()
			return nil
		},
	}
}

func resolveSpace(spaces []api.SpaceSummary, input string) (api.SpaceSummary, error) {
	return resolveByIDOrName(spaces, input, "space",
		func(s api.SpaceSummary) string { return s.ID },
		func(s api.SpaceSummary) string { return s.Name })
}
