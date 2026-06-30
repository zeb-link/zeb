// Collection commands list collections and manage active collection context.
// The active collection is where create-link commands should add new links.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/spf13/cobra"
)

func newCollectionsCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "List collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListCollections(context.Background(), ctx.SpaceID)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			printCollections(response.Collections, cfg.ActiveCollection)
			return nil
		},
	}
	cmd.AddCommand(newCollectionCreateCommand(root))
	return cmd
}

func newCollectionCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collection",
		Short: "Manage active collection context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"activeCollection": nullString(cfg.ActiveCollection)})
			}
			fmt.Printf("Active collection: %s\n", emptyLabel(cfg.ActiveCollection))
			return nil
		},
	}
	cmd.AddCommand(newCollectionListCommand(root), newCollectionCreateCommand(root), newCollectionUseCommand(root), newCollectionClearCommand(root), newCollectionNoneCommand(root))
	return cmd
}

func newCollectionListCommand(root *rootOptions) *cobra.Command {
	cmd := newCollectionsCommand(root)
	cmd.Use = "list"
	cmd.Short = "List collections"
	return cmd
}

func newCollectionUseCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "use <id-or-name>",
		Short: "Set active collection for new links",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListCollections(context.Background(), ctx.SpaceID)
			if err != nil {
				return err
			}
			collection, err := resolveCollection(response.Collections, args[0])
			if err != nil {
				return err
			}
			if collection.Type == "smart" {
				return fmt.Errorf("collection %q is smart; choose a collection that can accept new links or run zeb collection none", collection.Name)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.ActiveCollection = collection.ID
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"activeCollection": collection.ID, "collection": collection})
			}
			fmt.Printf("Active collection set to %s (%s)\n", collection.Name, collection.ID)
			return nil
		},
	}
}

func newCollectionCreateCommand(root *rootOptions) *cobra.Command {
	var description string
	var use bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("collection name cannot be blank")
			}
			ctx, err := resolveAPIContext(root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.CreateCollection(context.Background(), ctx.SpaceID, api.CreateCollectionInput{
				Name:        name,
				Description: description,
			})
			if err != nil {
				return err
			}
			if use {
				cfg, err := config.LoadConfig()
				if err != nil {
					return err
				}
				cfg.ActiveCollection = response.Collection.ID
				if err := config.SaveConfig(cfg); err != nil {
					return err
				}
			}
			if root.JSON {
				activeCollection := ""
				if use {
					activeCollection = response.Collection.ID
				}
				return writeJSON(map[string]any{
					"collection":       response.Collection,
					"activeCollection": nullString(activeCollection),
				})
			}
			fmt.Printf("Created collection %s (%s)\n", response.Collection.Name, response.Collection.ID)
			if use {
				fmt.Printf("Active collection set to %s\n", response.Collection.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "collection description")
	cmd.Flags().BoolVar(&use, "use", false, "make the new collection active for new links")
	return cmd
}

func newCollectionClearCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear active collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.ActiveCollection = ""
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"activeCollection": nil})
			}
			fmt.Println("Active collection cleared.")
			return nil
		},
	}
}

func newCollectionNoneCommand(root *rootOptions) *cobra.Command {
	cmd := newCollectionClearCommand(root)
	cmd.Use = "none"
	cmd.Short = "Create new links without a default collection"
	return cmd
}

func resolveCollection(collections []api.Collection, input string) (api.Collection, error) {
	for _, collection := range collections {
		if collection.ID == input {
			return collection, nil
		}
	}
	var matches []api.Collection
	for _, collection := range collections {
		if collection.Name == input {
			matches = append(matches, collection)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return api.Collection{}, fmt.Errorf("multiple collections named %q; use the collection id", input)
	}
	return api.Collection{}, fmt.Errorf("collection %q not found", input)
}

func printCollections(collections []api.Collection, activeCollection string) {
	fmt.Println(heading("Collections"))
	for _, collection := range collections {
		active := ""
		if collection.ID == activeCollection {
			active = "  active"
		}
		fmt.Printf("%s  %s  %d links%s\n", collection.ID, collection.Name, collection.LinkCount, active)
	}
}
