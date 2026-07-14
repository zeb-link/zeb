// Space commands manage the active space context.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/config"
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
			fmt.Printf("Active space: %s\n", emptyLabel(spaceID))
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
			fmt.Println(heading("Spaces"))
			for _, space := range me.AccessibleSpaces {
				fmt.Printf("%s  %s  %s\n", space.ID, space.Role, space.Name)
			}
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
			fmt.Printf("Active space set to %s (%s)\n", space.Name, space.ID)
			return nil
		},
	}
}

func resolveSpace(spaces []api.SpaceSummary, input string) (api.SpaceSummary, error) {
	return resolveByIDOrName(spaces, input, "space",
		func(s api.SpaceSummary) string { return s.ID },
		func(s api.SpaceSummary) string { return s.Name })
}
