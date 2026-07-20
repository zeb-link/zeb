// TUI command opens the interactive terminal surface.
// Rendering and state live under internal/tui so Cobra code stays thin.
package cli

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/tui/intro"
	"github.com/zeb-link/zeb/internal/tui/intro/gallery"
	"github.com/zeb-link/zeb/internal/tui/shell"
)

func newTUICommand(root *rootOptions) *cobra.Command {
	var preview bool
	var previewFrames int
	var introName string
	var galleryMode bool
	var galleryFrame int
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive terminal interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			if galleryMode && preview {
				fmt.Println(gallery.Preview(galleryFrame, 120))
				return nil
			}
			if galleryMode {
				_, err := tea.NewProgram(gallery.New(galleryFrame)).Run()
				return err
			}
			if preview {
				if previewFrames <= 0 {
					previewFrames = intro.FrameCount
				}
				selected := introName
				if selected == "" || selected == "random" {
					selected = "all"
				}
				intro.PrintPreview(previewFrames, selected)
				return nil
			}
			variant, err := resolveIntroVariant(introName)
			if err != nil {
				return err
			}
			data, cfg, err := loadTUIData(cmd, root)
			if err != nil {
				return err
			}
			final, err := tea.NewProgram(shell.New(variant, data)).Run()
			if err != nil {
				return err
			}
			model, ok := final.(shell.Model)
			if !ok {
				return fmt.Errorf("tui returned unexpected model")
			}
			if !model.ContextChanged() {
				return nil
			}
			if model.DomainChanged() {
				cfg.ActiveDomain = model.ActiveDomain()
			}
			if model.CollectionChanged() {
				cfg.ActiveCollection = model.ActiveCollection()
			}
			return config.SaveConfig(cfg)
		},
	}
	cmd.Flags().BoolVar(&preview, "preview", false, "print intro frames without opening the TUI")
	cmd.Flags().IntVar(&previewFrames, "frames", intro.FrameCount, "number of intro frames to print with --preview")
	cmd.Flags().StringVar(&introName, "intro", "random", "intro animation: random, all, "+strings.Join(intro.Slugs(), ", "))
	cmd.Flags().BoolVar(&galleryMode, "gallery", false, "open a static intro comparison gallery")
	cmd.Flags().IntVar(&galleryFrame, "gallery-frame", intro.FrameCount/2, "intro frame to render in --gallery")
	return cmd
}

func loadTUIData(cmd *cobra.Command, root *rootOptions) (shell.Data, config.Config, error) {
	ctx, err := resolveAPIContext(cmd.Context(), root)
	if err != nil {
		return shell.Data{}, config.Config{}, err
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return shell.Data{}, config.Config{}, err
	}
	domains, err := ctx.Client.ListDomains(cmd.Context(), ctx.SpaceID)
	if err != nil {
		return shell.Data{}, config.Config{}, err
	}
	collections, err := ctx.Client.ListCollections(cmd.Context(), ctx.SpaceID)
	if err != nil {
		return shell.Data{}, config.Config{}, err
	}
	// The TUI browses the active collection when one is set (the same scope
	// new links go to), so the first paint matches what the footer chip says.
	links, err := listInitialTUILinks(cmd, ctx, collections.Collections, cfg.ActiveCollection)
	if err != nil {
		return shell.Data{}, config.Config{}, err
	}
	return shell.Data{
		Client:           ctx.Client,
		SpaceID:          ctx.SpaceID,
		Links:            links.Links,
		NextCursor:       links.NextCursor,
		Domains:          domains.Domains,
		Collections:      collections.Collections,
		ActiveDomain:     cfg.ActiveDomain,
		ActiveCollection: cfg.ActiveCollection,
	}, cfg, nil
}

// listInitialTUILinks scopes the first page to the active collection when it
// is a real, selectable (non-smart) collection; otherwise it lists the space.
func listInitialTUILinks(cmd *cobra.Command, ctx apiContext, collections []api.Collection, activeCollection string) (api.ListLinksResponse, error) {
	options := api.ListLinksOptions{Limit: 50, Sort: "creation-date-desc", IncludeClicks: true}
	if activeCollection != "" {
		for _, collection := range collections {
			if collection.ID == activeCollection && collection.Type != "smart" {
				return ctx.Client.ListCollectionLinks(cmd.Context(), ctx.SpaceID, collection.ID, options)
			}
		}
	}
	return ctx.Client.ListLinks(cmd.Context(), ctx.SpaceID, options)
}

func resolveIntroVariant(name string) (intro.Variant, error) {
	if name == "" || name == "random" {
		return intro.RandomVariant(), nil
	}
	if name == "all" {
		return intro.RandomVariant(), nil
	}
	variant, ok := intro.VariantByName(name)
	if !ok {
		return intro.Variant{}, fmt.Errorf("unknown intro %q; available: random, %s", name, strings.Join(intro.Slugs(), ", "))
	}
	return variant, nil
}
