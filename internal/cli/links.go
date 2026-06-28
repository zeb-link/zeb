// Link commands list links in the active space or active collection.
// Creation will build on the same active domain and collection context.
package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
	"github.com/spf13/cobra"
)

type createLinksOptions struct {
	Domain       string
	Collection   string
	NoCollection bool
	Path         string
	ShortCode    string
	Namespace    string
	Title        string
	NoVerify     bool
}

func newLinksCommand(root *rootOptions) *cobra.Command {
	var limit int
	var collection string
	var status string
	cmd := &cobra.Command{
		Use:   "links",
		Short: "List links",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(root)
			if err != nil {
				return err
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			collectionID := ""
			collectionLabel := ""
			if collection != "" {
				collectionID = collection
				if collection == "active" {
					collectionID = cfg.ActiveCollection
				}
				if collectionID == "" {
					return fmt.Errorf("no active collection is set")
				}
				collections, err := ctx.Client.ListCollections(context.Background(), ctx.SpaceID)
				if err != nil {
					return err
				}
				resolved, err := resolveCollection(collections.Collections, collectionID)
				if err != nil {
					return err
				}
				collectionID = resolved.ID
				collectionLabel = resolved.Name
			}
			options := api.ListLinksOptions{Limit: limit, Status: status}
			var response api.ListLinksResponse
			if collectionID != "" {
				response, err = ctx.Client.ListCollectionLinks(context.Background(), ctx.SpaceID, collectionID, options)
			} else {
				response, err = ctx.Client.ListLinks(context.Background(), ctx.SpaceID, options)
			}
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printLinkContext(cfg, collectionID, collectionLabel)
			printLinks(response.Links)
			if response.NextCursor != nil {
				fmt.Printf("\nNext cursor: %s\n", *response.NextCursor)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "number of links to fetch")
	cmd.Flags().StringVarP(&collection, "collection", "c", "", "collection id/name to list, or 'active'")
	cmd.Flags().StringVar(&status, "status", "", "filter by status: active or inactive")
	cmd.AddCommand(newLinksCreateCommand(root))
	return cmd
}

func newLinksCreateCommand(root *rootOptions) *cobra.Command {
	options := &createLinksOptions{}
	cmd := &cobra.Command{
		Use:   "create <url...>",
		Short: "Create short links",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateLinks(root, options, args)
		},
	}
	addCreateLinkFlags(cmd, options)
	return cmd
}

func addCreateLinkFlags(cmd *cobra.Command, options *createLinksOptions) {
	cmd.Flags().StringVarP(&options.Domain, "domain", "d", "", "domain hostname to use instead of the active domain")
	cmd.Flags().StringVarP(&options.Collection, "collection", "c", "", "collection id/name to use instead of the active collection")
	cmd.Flags().BoolVar(&options.NoCollection, "no-collection", false, "create links without adding them to the active collection")
	cmd.Flags().StringVar(&options.Path, "path", "", "custom path for a single URL")
	cmd.Flags().StringVar(&options.ShortCode, "short-code", "", "alias for --path")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "", "namespace for auto-allocated paths")
	cmd.Flags().StringVarP(&options.Title, "title", "t", "", "title for a single URL")
	cmd.Flags().BoolVar(&options.NoVerify, "no-verify", false, "skip checking whether each target URL is reachable")
}

func runCreateLinks(root *rootOptions, options *createLinksOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provide one or more URLs")
	}
	if len(args) > 1 && hasSingleLinkOnlyOptions(options) {
		return fmt.Errorf("--path, --short-code, --namespace, and --title only work with a single URL")
	}
	if options.Path != "" && options.ShortCode != "" && options.Path != options.ShortCode {
		return fmt.Errorf("--path and --short-code specify different values")
	}

	ctx, err := resolveAPIContext(root)
	if err != nil {
		return err
	}

	domain, err := config.ResolveDomain(options.Domain)
	if err != nil {
		return err
	}

	collectionID := ""
	collectionLabel := ""
	if !options.NoCollection {
		collectionInput, err := config.ResolveCollection(options.Collection)
		if err != nil {
			return err
		}
		if collectionInput != "" {
			collections, err := ctx.Client.ListCollections(context.Background(), ctx.SpaceID)
			if err != nil {
				return err
			}
			collection, err := resolveCollection(collections.Collections, collectionInput)
			if err != nil {
				return err
			}
			if collection.Type == "smart" {
				return fmt.Errorf("collection %q is smart; choose a manual collection for new links", collection.Name)
			}
			collectionID = collection.ID
			collectionLabel = collection.Name
		}
	}

	path := firstNonEmpty(options.Path, options.ShortCode)
	created := make([]api.CreateLinkResponse, 0, len(args))
	for _, arg := range args {
		if err := validateHTTPURL(arg); err != nil {
			return err
		}
		input := api.CreateLinkInput{
			TargetURL:  arg,
			Domain:     domain,
			Path:       path,
			Namespace:  options.Namespace,
			Title:      options.Title,
			Collection: collectionID,
		}
		response, err := ctx.Client.CreateLink(context.Background(), ctx.SpaceID, input, !options.NoVerify)
		if err != nil {
			return err
		}
		created = append(created, response)
	}

	if root.JSON {
		return writeJSON(map[string]any{
			"created":    created,
			"domain":     nullString(domain),
			"collection": collectionSummary(collectionID, collectionLabel),
		})
	}

	printCreatedLinks(created, collectionID, collectionLabel)
	return nil
}

func hasSingleLinkOnlyOptions(options *createLinksOptions) bool {
	return options.Path != "" || options.ShortCode != "" || options.Namespace != "" || options.Title != ""
}

func validateHTTPURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", value)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%q must start with http:// or https://", value)
	}
	return nil
}

func collectionSummary(id string, name string) any {
	if id == "" {
		return nil
	}
	return map[string]string{"id": id, "name": name}
}

func printCreatedLinks(created []api.CreateLinkResponse, collectionID string, collectionLabel string) {
	countLabel := "links"
	if len(created) == 1 {
		countLabel = "link"
	}
	fmt.Println(createdHeadingStyle.Render(fmt.Sprintf("Created %d %s", len(created), countLabel)))
	fmt.Println()
	for _, response := range created {
		link := response.Link
		short := displayShortLink(link)
		fmt.Printf("%s %s %s\n", linkShortStyle.Render(short), theme.MutedText.Render("->"), linkTargetStyle.Render(link.TargetURL))
		fmt.Printf("  %s\n", createdMetaFooter(link, response.TargetReachable, collectionID, collectionLabel))
	}
}

func createdMetaFooter(link api.Link, reachable *bool, collectionID string, collectionLabel string) string {
	parts := []string{
		theme.MutedText.Render(link.ID),
		theme.MutedText.Render("domain ") + createdDomainStyle.Render(createdDomain(link)),
		theme.MutedText.Render("collection ") + createdCollectionStyle.Render(createdCollection(collectionID, collectionLabel)),
	}
	if note := reachabilityNote(reachable); note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, theme.MutedText.Render(" · "))
}

func createdDomain(link api.Link) string {
	if link.Hostname != "" {
		return link.Hostname
	}
	if link.ShortURL != "" {
		parsed, err := url.Parse(link.ShortURL)
		if err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}
	return "unknown"
}

func createdCollection(collectionID string, collectionLabel string) string {
	if collectionID == "" {
		return "none"
	}
	if collectionLabel == "" {
		return collectionID
	}
	return collectionLabel
}

// reachabilityNote renders the advisory target-reachability marker for a created
// link. Empty when the API didn't check (created with --no-verify, so the field
// is nil) — the link was created regardless; this only annotates.
func reachabilityNote(reachable *bool) string {
	if reachable == nil {
		return ""
	}
	if *reachable {
		return reachableStyle.Render("● verified")
	}
	return unreachableStyle.Render("● unreachable")
}

func printLinkContext(cfg config.Config, collectionID string, collectionLabel string) {
	domain := cfg.ActiveDomain
	if domain == "" {
		domain = "server default"
	}
	fmt.Println(heading("Links"))
	contextLabel := theme.MutedText.Render("New links:")
	domainLabel := theme.Command.Render(domain)
	collectionText := "no collection"
	if cfg.ActiveCollection != "" {
		collectionText = cfg.ActiveCollection
	}
	fmt.Printf("%s  domain %s  %s  collection %s\n", contextLabel, domainLabel, theme.MutedText.Render("·"), theme.Command.Render(collectionText))
	if collectionID != "" {
		fmt.Printf("%s %s %s\n", theme.MutedText.Render("Showing:"), collectionLabel, theme.MutedText.Render("("+collectionID+")"))
	} else {
		fmt.Printf("%s all links\n", theme.MutedText.Render("Showing:"))
	}
	fmt.Println()
}

func printLinks(links []api.Link) {
	if len(links) == 0 {
		fmt.Println("No links found.")
		return
	}
	for idx, link := range links {
		if idx > 0 {
			fmt.Println()
		}
		printLink(link)
	}
}

func printLink(link api.Link) {
	short := displayShortLink(link)
	target := truncate(link.TargetURL, 92)
	dot, status := linkStatus(link.IsActive)

	fmt.Printf("%s %s %s %s\n", dot, linkShortStyle.Render(short), theme.MutedText.Render("->"), linkTargetStyle.Render(target))
	if link.Title != nil && strings.TrimSpace(*link.Title) != "" {
		fmt.Printf("  %s\n", linkTitleStyle.Render(truncate(strings.TrimSpace(*link.Title), 110)))
	}
	fmt.Printf("  %s %s %s\n", theme.MutedText.Render(link.ID), theme.MutedText.Render("·"), status)
}

func displayShortLink(link api.Link) string {
	if link.ShortURL != "" {
		return link.ShortURL
	}
	return shortLink(link.Hostname, link.Path)
}

func shortLink(hostname string, path string) string {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		return hostname
	}
	return hostname + "/" + cleanPath
}

func linkStatus(active bool) (string, string) {
	if active {
		return activeDotStyle.Render("●"), activeStatusStyle.Render("active")
	}
	return inactiveDotStyle.Render("●"), inactiveStatusStyle.Render("inactive")
}

func truncate(value string, limit int) string {
	if limit <= 1 || len(value) <= limit {
		return value
	}
	return value[:limit-1] + "…"
}

var (
	activeDotStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	inactiveDotStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	activeStatusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	inactiveStatusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	createdHeadingStyle    = lipgloss.NewStyle().Bold(true).Foreground(theme.White)
	createdDomainStyle     = lipgloss.NewStyle().Foreground(theme.Command.GetForeground())
	createdCollectionStyle = lipgloss.NewStyle().Foreground(theme.White)
	linkShortStyle         = lipgloss.NewStyle().Foreground(theme.White)
	linkTargetStyle        = lipgloss.NewStyle().Foreground(theme.Ink)
	linkTitleStyle         = lipgloss.NewStyle().Foreground(theme.Muted)
	reachableStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	unreachableStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)
