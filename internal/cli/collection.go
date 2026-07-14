// Collection commands list, create, inspect, update, and delete collections,
// manage the active collection context, and manage manual-collection
// membership (add/remove links).
package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newCollectionsCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "List collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListCollections(cmd.Context(), ctx.SpaceID)
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
		Short: "Manage collections and the active collection context",
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
	cmd.AddCommand(
		newCollectionListCommand(root),
		newCollectionCreateCommand(root),
		newCollectionShowCommand(root),
		newCollectionLinksCommand(root),
		newCollectionUpdateCommand(root),
		newCollectionDeleteCommand(root),
		newCollectionConvertCommand(root),
		newCollectionAddCommand(root),
		newCollectionRemoveCommand(root),
		newCollectionUseCommand(root),
		newCollectionClearCommand(root),
		newCollectionNoneCommand(root),
	)
	return cmd
}

func newCollectionListCommand(root *rootOptions) *cobra.Command {
	cmd := newCollectionsCommand(root)
	cmd.Use = "list"
	cmd.Short = "List collections"
	return cmd
}

// resolveCollectionArg resolves a collection reference for a subcommand:
// an explicit id/name argument, or the literal/implicit "active" which reads
// the saved context (erroring with a hint when none is set).
func resolveCollectionArg(cmd *cobra.Command, ctx apiContext, input string) (api.Collection, error) {
	if input == "" || input == "active" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return api.Collection{}, err
		}
		input = cfg.ActiveCollection
		if input == "" {
			return api.Collection{}, fmt.Errorf("no active collection is set; provide a collection id or name, or run `zeb collection use <name>`")
		}
	}
	collections, err := ctx.Client.ListCollections(cmd.Context(), ctx.SpaceID)
	if err != nil {
		return api.Collection{}, err
	}
	return resolveCollection(collections.Collections, input)
}

func newCollectionUseCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "use <id-or-name>",
		Short: "Set active collection for new links",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			collection, err := resolveCollectionArg(cmd, ctx, args[0])
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

func newCollectionShowCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show [id-or-name|active]",
		Short: "Show one collection",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			input := ""
			if len(args) > 0 {
				input = args[0]
			}
			resolved, err := resolveCollectionArg(cmd, ctx, input)
			if err != nil {
				return err
			}
			response, err := ctx.Client.GetCollection(cmd.Context(), ctx.SpaceID, resolved.ID)
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
			printCollection(response.Collection, response.Collection.ID == cfg.ActiveCollection)
			return nil
		},
	}
}

func newCollectionLinksCommand(root *rootOptions) *cobra.Command {
	flags := &listLinksFlags{}
	cmd := &cobra.Command{
		Use:     "links [id-or-name|active]",
		Aliases: []string{"ls"},
		Short:   "List links in a collection",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			input := ""
			if len(args) > 0 {
				input = args[0]
			}
			collection, err := resolveCollectionArg(cmd, ctx, input)
			if err != nil {
				return err
			}
			response, err := fetchLinks(cmd, ctx, collection.ID, *flags)
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
			printLinkContext(cfg, collection.ID, collection.Name, *flags)
			printLinks(response.Links)
			printNextPageHint(response, fmt.Sprintf("zeb collection links %q", collection.Name))
			return nil
		},
	}
	addListLinksFlags(cmd, flags)
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
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.CreateCollection(cmd.Context(), ctx.SpaceID, api.CreateCollectionInput{
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

func newCollectionUpdateCommand(root *rootOptions) *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update a collection's name or description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := api.UpdateCollectionInput{}
			if cmd.Flags().Changed("name") {
				trimmed := strings.TrimSpace(name)
				if trimmed == "" {
					return fmt.Errorf("--name cannot be blank")
				}
				input.Name = &trimmed
			}
			if cmd.Flags().Changed("description") {
				input.Description = &description
			}
			if input.Name == nil && input.Description == nil {
				return fmt.Errorf("nothing to update; pass --name and/or --description")
			}
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			resolved, err := resolveCollectionArg(cmd, ctx, args[0])
			if err != nil {
				return err
			}
			response, err := ctx.Client.UpdateCollection(cmd.Context(), ctx.SpaceID, resolved.ID, input)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			fmt.Printf("Updated collection %s (%s)\n", response.Collection.Name, response.Collection.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new collection name")
	cmd.Flags().StringVar(&description, "description", "", "new description (empty string clears it)")
	return cmd
}

func newCollectionDeleteCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id-or-name>",
		Aliases: []string{"rm"},
		Short:   "Delete a collection",
		Long:    "Delete a collection. Links in the collection are not deleted — they only leave the collection.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			resolved, err := resolveCollectionArg(cmd, ctx, args[0])
			if err != nil {
				return err
			}
			response, err := ctx.Client.DeleteCollection(cmd.Context(), ctx.SpaceID, resolved.ID)
			if err != nil {
				return err
			}
			// A deleted collection must not survive as the saved create
			// default — that is exactly the stale-context trap.
			clearedActive := false
			cfg, err := config.LoadConfig()
			if err == nil && cfg.ActiveCollection == resolved.ID {
				cfg.ActiveCollection = ""
				if err := config.SaveConfig(cfg); err == nil {
					clearedActive = true
				}
			}
			if root.JSON {
				return writeJSON(map[string]any{
					"deletedCollectionId":     response.DeletedCollectionID,
					"activeCollectionCleared": clearedActive,
				})
			}
			fmt.Printf("Deleted collection %s (%s)\n", resolved.Name, response.DeletedCollectionID)
			if clearedActive {
				fmt.Println("Active collection cleared; new links get no collection.")
			}
			return nil
		},
	}
	return cmd
}

func newCollectionConvertCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "convert <id-or-name>",
		Short: "Convert a smart collection to a manual one",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			resolved, err := resolveCollectionArg(cmd, ctx, args[0])
			if err != nil {
				return err
			}
			response, err := ctx.Client.ConvertCollectionToManual(cmd.Context(), ctx.SpaceID, resolved.ID)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			fmt.Printf("Converted %s to a manual collection; its current links are now direct members.\n", response.Collection.Name)
			return nil
		},
	}
}

func newCollectionAddCommand(root *rootOptions) *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "add <link-id...>",
		Short: "Add links to a collection",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, id := range args {
				if err := validateLinkID(id); err != nil {
					return err
				}
			}
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			resolved, err := resolveCollectionArg(cmd, ctx, to)
			if err != nil {
				return err
			}
			response, err := ctx.Client.AddLinksToCollection(cmd.Context(), ctx.SpaceID, resolved.ID, args)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"collection": resolved, "added": response.Added, "alreadyMember": response.AlreadyMember})
			}
			note := ""
			if response.AlreadyMember > 0 {
				note = fmt.Sprintf(" (%d already in it)", response.AlreadyMember)
			}
			fmt.Printf("Added %d links to %s%s\n", response.Added, resolved.Name, note)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "collection id/name (defaults to the active collection)")
	return cmd
}

func newCollectionRemoveCommand(root *rootOptions) *cobra.Command {
	var from string
	cmd := &cobra.Command{
		Use:   "remove <link-id...>",
		Short: "Remove links from a collection",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, id := range args {
				if err := validateLinkID(id); err != nil {
					return err
				}
			}
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			resolved, err := resolveCollectionArg(cmd, ctx, from)
			if err != nil {
				return err
			}
			response, err := ctx.Client.RemoveLinksFromCollection(cmd.Context(), ctx.SpaceID, resolved.ID, args)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"collection": resolved, "removed": response.Removed})
			}
			fmt.Printf("Removed %d links from %s\n", response.Removed, resolved.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "collection id/name (defaults to the active collection)")
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
	return resolveByIDOrName(collections, input, "collection",
		func(c api.Collection) string { return c.ID },
		func(c api.Collection) string { return c.Name })
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
