// Space commands manage the active space context.
package cli

import (
	"context"
	"fmt"

	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/spf13/cobra"
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
			key, err := config.ResolveAPIKey(root.APIKey)
			if err != nil {
				return err
			}
			if key == "" {
				return fmt.Errorf("not logged in; run zeb auth login")
			}
			apiURL, err := config.ResolveAPIURL(root.APIURL)
			if err != nil {
				return err
			}
			client := api.New(api.Options{APIURL: apiURL, APIKey: key})
			me, err := client.GetMe(context.Background())
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
			key, err := config.ResolveAPIKey(root.APIKey)
			if err != nil {
				return err
			}
			if key == "" {
				return fmt.Errorf("not logged in; run zeb auth login")
			}
			apiURL, err := config.ResolveAPIURL(root.APIURL)
			if err != nil {
				return err
			}
			client := api.New(api.Options{APIURL: apiURL, APIKey: key})
			me, err := client.GetMe(context.Background())
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
	for _, space := range spaces {
		if space.ID == input {
			return space, nil
		}
	}
	var matches []api.SpaceSummary
	for _, space := range spaces {
		if space.Name == input {
			matches = append(matches, space)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return api.SpaceSummary{}, fmt.Errorf("multiple spaces named %q; use the space id", input)
	}
	return api.SpaceSummary{}, fmt.Errorf("space %q not found", input)
}
