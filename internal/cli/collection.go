// Collection commands list collections and manage active collection context.
// The active collection is where create-link commands should add new links.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
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
	cmd.AddCommand(newCollectionListCommand(root), newCollectionCreateCommand(root), newCollectionLinksCommand(root), newCollectionUseCommand(root), newCollectionClearCommand(root), newCollectionNoneCommand(root))
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

func newCollectionLinksCommand(root *rootOptions) *cobra.Command {
	var limit int
	var status string
	cmd := &cobra.Command{
		Use:     "links [id-or-name|active]",
		Aliases: []string{"ls"},
		Short:   "List links in a collection",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(root)
			if err != nil {
				return err
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			input := "active"
			if len(args) > 0 {
				input = args[0]
			}
			if input == "active" {
				input = cfg.ActiveCollection
				if input == "" {
					return fmt.Errorf("no active collection is set; provide a collection id or name")
				}
			}
			collections, err := ctx.Client.ListCollections(context.Background(), ctx.SpaceID)
			if err != nil {
				return err
			}
			collection, err := resolveCollection(collections.Collections, input)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListCollectionLinks(context.Background(), ctx.SpaceID, collection.ID, api.ListLinksOptions{
				Limit:  limit,
				Status: status,
			})
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printLinkContext(cfg, collection.ID, collection.Name)
			printLinks(response.Links)
			if response.NextCursor != nil {
				fmt.Printf("\nNext cursor: %s\n", *response.NextCursor)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "number of links to fetch")
	cmd.Flags().StringVar(&status, "status", "", "filter by status: active or inactive")
	return cmd
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
	fmt.Println(collectionHeading())
	if len(collections) == 0 {
		fmt.Println("No collections found.")
		return
	}
	for _, collection := range collections {
		printCollection(collection, collection.ID == activeCollection)
	}
}

func printCollection(collection api.Collection, active bool) {
	dot, status := collectionStatus(collection, active)
	fmt.Printf("%s %s %s\n", dot, collectionNameStyle.Render(collection.Name), theme.MutedText.Render(collectionLinkCountLabel(collection.LinkCount)))
	if description := collectionDescription(collection); description != "" {
		fmt.Printf("  %s\n", collectionDescriptionStyle.Render(truncate(description, 110)))
	}
	meta := []string{theme.MutedText.Render(collection.ID)}
	if status != "" {
		meta = append(meta, status)
	}
	fmt.Printf("  %s\n\n", strings.Join(meta, theme.MutedText.Render(" · ")))
}

func collectionStatus(collection api.Collection, active bool) (string, string) {
	if active {
		return collectionActiveStyle.Render("●"), collectionActiveStyle.Render("active")
	}
	if collection.Type == "smart" {
		return collectionSmartStyle.Render("●"), ""
	}
	return collectionDotStyle.Render("●"), ""
}

func collectionLinkCountLabel(count int) string {
	if count == 1 {
		return "1 link"
	}
	return fmt.Sprintf("%d links", count)
}

func collectionDescription(collection api.Collection) string {
	if collection.Description == nil {
		return ""
	}
	return strings.TrimSpace(*collection.Description)
}

func collectionHeading() string {
	legend := strings.Join([]string{
		collectionDotStyle.Render("●") + " " + theme.MutedText.Render("collection"),
		collectionSmartStyle.Render("●") + " " + theme.MutedText.Render("smart"),
		collectionActiveStyle.Render("●") + " " + theme.MutedText.Render("active"),
	}, theme.MutedText.Render(" · "))
	return collectionHeadingStyle.Render("Collections") + " " + theme.MutedText.Render("(") + legend + theme.MutedText.Render(")")
}

var (
	collectionNameStyle        = linkShortStyle
	collectionDescriptionStyle = linkTitleStyle
	collectionHeadingStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("183"))
	collectionDotStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	collectionSmartStyle       = lipgloss.NewStyle().Foreground(theme.Accent2)
	collectionActiveStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)
