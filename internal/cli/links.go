// Link commands list, create, inspect, update, and delete links in the
// active space. Creation builds on the same active domain and collection
// context; multi-URL creates go through the bulk endpoint so partial
// failures stay visible per row.
package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

// bulkChunkSize is the API's per-request cap for /links/bulk create and
// delete. Larger workloads are chunked client-side, never rejected.
const bulkChunkSize = 250

// allPageSize is the page size used by --all when the user did not pass an
// explicit --limit.
const allPageSize = 100

// allProgressInterval is how many accumulated links between --all progress
// notices on stderr.
const allProgressInterval = 1000

// linkSortValues mirrors the API's sort vocabulary for help text and
// completion. The server remains the validator; an out-of-date value here
// only affects the hint. Kept in sync by the spec drift test.
var linkSortValues = []string{
	"creation-date-desc", "creation-date-asc",
	"edit-date-desc", "edit-date-asc",
	"total-clicks-desc", "total-clicks-asc",
	"recent-clicks-desc", "recent-clicks-asc",
}

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

type listLinksFlags struct {
	Limit  int
	Sort   string
	Status string
	Cursor string
	All    bool
}

func addListLinksFlags(cmd *cobra.Command, flags *listLinksFlags) {
	cmd.Flags().IntVarP(&flags.Limit, "limit", "l", 50, "page size (server range 1-1000)")
	cmd.Flags().StringVar(&flags.Sort, "sort", "", "sort order: "+strings.Join(linkSortValues, ", "))
	cmd.Flags().StringVar(&flags.Status, "status", "", "filter by status: active or inactive")
	cmd.Flags().StringVar(&flags.Cursor, "cursor", "", "pagination cursor from a previous page")
	cmd.Flags().BoolVar(&flags.All, "all", false, "follow pagination and fetch every page (can return thousands; narrow with --status/--collection or cap with --limit)")
}

func newLinksCommand(root *rootOptions) *cobra.Command {
	flags := &listLinksFlags{}
	var collection string
	cmd := &cobra.Command{
		Use:   "links",
		Short: "Browse and manage links (use `links query` to filter)",
		Long: "Browse your links: newest first, or paged through in full.\n\n" +
			"`zeb links` is for LISTING — it takes only --status, --collection, and\n" +
			"--sort, and pages the whole table cheaply (--all walks every page).\n" +
			"To FIND links by a condition — destination, clicks, dates, attribution,\n" +
			"negation, free text — use `zeb links query` instead.\n\n" +
			"Rule of thumb: any filter beyond status/collection/sort → `links query`.",
		Example: "  zeb links                              # newest links\n" +
			"  zeb links --all --json                 # every link, machine-readable\n" +
			"  zeb links --status inactive --sort total-clicks-desc\n" +
			"  zeb links --collection \"Campaign\"      # one collection's members\n" +
			"  zeb links query --target-host cnn.com  # FIND by a condition (see: links query)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
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
				collections, err := ctx.Client.ListCollections(cmd.Context(), ctx.SpaceID)
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
			response, err := fetchLinks(cmd, ctx, collectionID, *flags)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printLinkContext(cfg, collectionID, collectionLabel, *flags)
			printLinks(response.Links)
			nextPageCommand := "zeb links"
			if collection != "" {
				nextPageCommand = fmt.Sprintf("zeb links --collection %q", collection)
			}
			printNextPageHint(response, nextPageCommand)
			air()
			return nil
		},
	}
	addListLinksFlags(cmd, flags)
	cmd.Flags().StringVarP(&collection, "collection", "c", "", "collection id/name to list, or 'active'")
	cmd.AddCommand(
		newLinksCreateCommand(root),
		newLinksQueryCommand(root),
		newLinksLookupCommand(root),
		newLinksGetCommand(root),
		newLinksUpdateCommand(root),
		newLinksDeleteCommand(root),
	)
	return cmd
}

// fetchLinks runs one page fetch, or the full pagination loop with --all.
// collectionID scopes the list to a collection when non-empty.
func fetchLinks(cmd *cobra.Command, ctx apiContext, collectionID string, flags listLinksFlags) (api.ListLinksResponse, error) {
	fetch := func(options api.ListLinksOptions) (api.ListLinksResponse, error) {
		if collectionID != "" {
			return ctx.Client.ListCollectionLinks(cmd.Context(), ctx.SpaceID, collectionID, options)
		}
		return ctx.Client.ListLinks(cmd.Context(), ctx.SpaceID, options)
	}
	options := api.ListLinksOptions{
		Limit:  flags.Limit,
		Sort:   flags.Sort,
		Status: flags.Status,
		Cursor: flags.Cursor,
		// The CLI always wants click data in rows — one LEFT JOIN server-side,
		// and "find hot links → act on them" works without a second tool.
		IncludeClicks: true,
	}
	if !flags.All {
		return fetch(options)
	}
	if !cmd.Flags().Changed("limit") {
		options.Limit = allPageSize
	}
	var links []api.Link
	nextNotice := allProgressInterval
	for {
		page, err := fetch(options)
		if err != nil {
			return api.ListLinksResponse{}, err
		}
		links = append(links, page.Links...)
		// Advisory progress on stderr (stdout stays a clean JSON body) so a
		// large --all pull shows life and how to cap it, instead of appearing
		// to hang while it walks thousands of rows.
		if len(links) >= nextNotice {
			lipgloss.Fprintln(os.Stderr, theme.MutedText.Render(
				fmt.Sprintf("fetched %d links so far… (Ctrl-C to stop; use --limit to cap)", len(links)),
			))
			nextNotice += allProgressInterval
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			return api.ListLinksResponse{Links: links}, nil
		}
		options.Cursor = *page.NextCursor
	}
}

func printNextPageHint(response api.ListLinksResponse, command string) {
	if response.NextCursor == nil || *response.NextCursor == "" {
		return
	}
	lipgloss.Printf("\n%s\n", theme.MutedText.Render(
		fmt.Sprintf("More available: %s --cursor %s  (or rerun with --all)", command, *response.NextCursor),
	))
}

func newLinksCreateCommand(root *rootOptions) *cobra.Command {
	options := &createLinksOptions{}
	cmd := &cobra.Command{
		Use:   "create <url...>",
		Short: "Create short links",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateLinks(cmd, root, options, args)
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
	cmd.Flags().BoolVar(&options.NoVerify, "no-verify", false, "skip checking whether the target URL is reachable (single URL only; multi-URL creates never probe)")
}

func runCreateLinks(cmd *cobra.Command, root *rootOptions, options *createLinksOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provide one or more URLs")
	}
	if len(args) > 1 && hasSingleLinkOnlyOptions(options) {
		return fmt.Errorf("--path, --short-code, --namespace, and --title only work with a single URL")
	}
	if options.Path != "" && options.ShortCode != "" && options.Path != options.ShortCode {
		return fmt.Errorf("--path and --short-code specify different values")
	}
	for _, arg := range args {
		if err := validateHTTPURL(arg); err != nil {
			return err
		}
	}

	ctx, err := resolveAPIContext(cmd.Context(), root)
	if err != nil {
		return err
	}

	domain, err := config.ResolveDomain(options.Domain)
	if err != nil {
		return err
	}

	collectionID, collectionLabel, err := resolveCreateCollection(cmd, ctx, options)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return runSingleCreate(cmd, root, ctx, options, args[0], domain, collectionID, collectionLabel)
	}
	return runBulkCreate(cmd, root, ctx, args, domain, collectionID, collectionLabel)
}

// resolveCreateCollection resolves the collection new links should join.
// A collection named explicitly via --collection must exist and be manual —
// otherwise the create fails. The AMBIENT collection (env or saved context)
// degrades gracefully: if it is gone or not manual, the create proceeds
// without a collection and a warning explains how to fix the stale context,
// so a wiped database never bricks `zeb <url>`.
func resolveCreateCollection(cmd *cobra.Command, ctx apiContext, options *createLinksOptions) (string, string, error) {
	if options.NoCollection {
		return "", "", nil
	}
	explicit := options.Collection != ""
	collectionInput, err := config.ResolveCollection(options.Collection)
	if err != nil {
		return "", "", err
	}
	if collectionInput == "" {
		return "", "", nil
	}
	collections, err := ctx.Client.ListCollections(cmd.Context(), ctx.SpaceID)
	if err != nil {
		return "", "", err
	}
	collection, resolveErr := resolveCollection(collections.Collections, collectionInput)
	reason := ""
	if resolveErr != nil {
		reason = resolveErr.Error()
	} else if collection.Type == "smart" {
		reason = fmt.Sprintf("collection %q is smart and cannot accept new links directly", collection.Name)
	}
	if reason == "" {
		return collection.ID, collection.Name, nil
	}
	if explicit {
		return "", "", fmt.Errorf("%s (list collections with `zeb collections`)", reason)
	}
	lipgloss.Fprintln(os.Stderr, theme.WarnText.Render("warning: ")+theme.SubtleText.Render(
		fmt.Sprintf("%s; creating without a collection. Run `zeb collection clear` or `zeb context` to reset the saved default.", reason),
	))
	return "", "", nil
}

func runSingleCreate(cmd *cobra.Command, root *rootOptions, ctx apiContext, options *createLinksOptions, target string, domain string, collectionID string, collectionLabel string) error {
	input := api.CreateLinkInput{
		TargetURL:  target,
		Domain:     domain,
		Path:       firstNonEmpty(options.Path, options.ShortCode),
		Namespace:  options.Namespace,
		Title:      options.Title,
		Collection: collectionID,
	}
	response, err := ctx.Client.CreateLink(cmd.Context(), ctx.SpaceID, input, !options.NoVerify)
	if err != nil {
		return err
	}
	if root.JSON {
		return writeJSON(map[string]any{
			"created":    []api.CreateLinkResponse{response},
			"failed":     []createFailure{},
			"domain":     nullString(domain),
			"collection": collectionSummary(collectionID, collectionLabel),
		})
	}
	printCreatedLinks([]api.CreateLinkResponse{response}, collectionID, collectionLabel)
	air()
	return nil
}

type createFailure struct {
	Index     int              `json:"index"`
	TargetURL string           `json:"targetUrl"`
	Error     api.BulkRowError `json:"error"`
}

// runBulkCreate creates 2+ URLs through POST /links/bulk: one round-trip per
// 250 URLs and per-row outcomes, so a failed row never hides the rows that
// succeeded (the sequential loop this replaced dropped them on first error).
// Bulk creates skip the reachability probe — that is a single-create feature.
func runBulkCreate(cmd *cobra.Command, root *rootOptions, ctx apiContext, targets []string, domain string, collectionID string, collectionLabel string) error {
	created := make([]api.CreateLinkResponse, 0, len(targets))
	// Non-nil so the JSON report always carries `failed: []`, matching the
	// single-create shape agents parse.
	failed := []createFailure{}
	for start := 0; start < len(targets); start += bulkChunkSize {
		chunk := targets[start:min(start+bulkChunkSize, len(targets))]
		items := make([]api.BulkCreateLinkItem, len(chunk))
		for i, target := range chunk {
			items[i] = api.BulkCreateLinkItem{TargetURL: target, Domain: domain}
		}
		response, err := ctx.Client.BulkCreateLinks(cmd.Context(), ctx.SpaceID, api.BulkCreateLinksInput{
			Collection: collectionID,
			Items:      items,
		})
		if err != nil {
			return bulkCreateTransportError(err, created, failed, start, len(targets))
		}
		for _, row := range response.Results {
			index := start + row.Index
			if row.Success && row.Link != nil {
				created = append(created, api.CreateLinkResponse{Link: *row.Link})
				continue
			}
			failure := createFailure{Index: index, TargetURL: targets[index]}
			if row.Error != nil {
				failure.Error = *row.Error
			}
			failed = append(failed, failure)
		}
	}

	if root.JSON {
		if err := writeJSON(map[string]any{
			"created":    created,
			"failed":     failed,
			"domain":     nullString(domain),
			"collection": collectionSummary(collectionID, collectionLabel),
		}); err != nil {
			return err
		}
		// The JSON body already lists every failed row; a nothing-created batch
		// still exits non-zero without a second error document on stdout.
		if bulkCreateOutcome(created, failed) != nil {
			return errAlreadyReported
		}
		return nil
	}
	if len(created) > 0 {
		printCreatedLinks(created, collectionID, collectionLabel)
	}
	printCreateFailures(failed)
	if len(failed) > 0 {
		lipgloss.Printf("\n%s\n", theme.MutedText.Render(
			fmt.Sprintf("Created %d of %d links", len(created), len(created)+len(failed)),
		))
	}
	air()
	return bulkCreateOutcome(created, failed)
}

// bulkCreateTransportError surfaces a mid-batch HTTP failure without hiding
// the rows already created by earlier chunks.
func bulkCreateTransportError(err error, created []api.CreateLinkResponse, failed []createFailure, start int, total int) error {
	if start == 0 {
		return err
	}
	return fmt.Errorf(
		"%w (before the failure: %d links created, %d rows failed; URLs %d-%d were not attempted — rerun with those URLs to finish)",
		err, len(created), len(failed), start+1, total,
	)
}

// bulkCreateOutcome maps per-row results to an exit status: rows failing is
// reported output, not a command failure — unless NOTHING succeeded.
func bulkCreateOutcome(created []api.CreateLinkResponse, failed []createFailure) error {
	if len(created) == 0 && len(failed) > 0 {
		first := failed[0]
		return fmt.Errorf("no links were created (%s: %s)", first.Error.Code, first.Error.Message)
	}
	return nil
}

func printCreateFailures(failed []createFailure) {
	for _, failure := range failed {
		lipgloss.Printf("%s %s\n  %s\n",
			unreachableStyle.Render("✗"),
			linkTargetStyle.Render(truncate(failure.TargetURL, 92)),
			theme.MutedText.Render(fmt.Sprintf("%s · %s", failure.Error.Code, failure.Error.Message)),
		)
	}
}

func newLinksGetCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <link-id>",
		Short: "Show one link",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			linkID := args[0]
			if err := validateLinkID(linkID); err != nil {
				return err
			}
			response, err := ctx.Client.GetLink(cmd.Context(), ctx.SpaceID, linkID)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printLinkDetail(response.Link)
			air()
			return nil
		},
	}
}

func newLinksUpdateCommand(root *rootOptions) *cobra.Command {
	var target, title, description, path string
	var active, inactive bool
	cmd := &cobra.Command{
		Use:   "update <link-id>",
		Short: "Update a link",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			linkID := args[0]
			if err := validateLinkID(linkID); err != nil {
				return err
			}
			if active && inactive {
				return fmt.Errorf("--active and --inactive are mutually exclusive")
			}
			input := api.UpdateLinkInput{}
			if cmd.Flags().Changed("target") {
				if err := validateHTTPURL(target); err != nil {
					return err
				}
				input["targetUrl"] = target
			}
			// An explicitly empty --title/--description clears the field
			// (PATCH null); omitting the flag leaves it untouched.
			if cmd.Flags().Changed("title") {
				input["title"] = nullString(title)
			}
			if cmd.Flags().Changed("description") {
				input["description"] = nullString(description)
			}
			if cmd.Flags().Changed("path") {
				if strings.TrimSpace(path) == "" {
					return fmt.Errorf("--path cannot be blank")
				}
				input["path"] = path
			}
			if active || inactive {
				input["isActive"] = active
			}
			if len(input) == 0 {
				return fmt.Errorf("nothing to update; pass at least one of --target, --title, --description, --path, --active, --inactive")
			}
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.UpdateLink(cmd.Context(), ctx.SpaceID, linkID, input)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			lipgloss.Println(createdHeadingStyle.Render("Updated"))
			lipgloss.Println()
			printLinkDetail(response.Link)
			if response.PathChanged {
				lipgloss.Printf("\n%s\n", theme.WarnText.Render("The short URL changed — the previous path no longer redirects."))
			}
			air()
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "new destination URL")
	cmd.Flags().StringVarP(&title, "title", "t", "", "new title (empty string clears it)")
	cmd.Flags().StringVar(&description, "description", "", "new description (empty string clears it)")
	cmd.Flags().StringVar(&path, "path", "", "new short path")
	cmd.Flags().BoolVar(&active, "active", false, "activate the link")
	cmd.Flags().BoolVar(&inactive, "inactive", false, "deactivate the link")
	return cmd
}

func newLinksDeleteCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <link-id...>",
		Aliases: []string{"rm"},
		Short:   "Delete links",
		Long:    "Delete one or more links by id. Runs through the bulk endpoint with per-row results; batches over 250 ids are chunked automatically.",
		Args:    cobra.MinimumNArgs(1),
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
			results := make([]api.BulkDeleteRowResult, 0, len(args))
			for start := 0; start < len(args); start += bulkChunkSize {
				chunk := args[start:min(start+bulkChunkSize, len(args))]
				response, err := ctx.Client.BulkDeleteLinks(cmd.Context(), ctx.SpaceID, chunk)
				if err != nil {
					if start > 0 {
						return fmt.Errorf("%w (ids %d-%d were already deleted before the failure)", err, 1, start)
					}
					return err
				}
				results = append(results, response.Results...)
			}
			deleted := 0
			for _, row := range results {
				if row.Success {
					deleted++
				}
			}
			if root.JSON {
				if err := writeJSON(map[string]any{
					"results": results,
					"deleted": deleted,
					"failed":  len(results) - deleted,
				}); err != nil {
					return err
				}
				// results[] already carries each row's error; exit non-zero
				// without emitting a second error document on stdout.
				if deleteOutcome(results, deleted) != nil {
					return errAlreadyReported
				}
				return nil
			}
			for _, row := range results {
				if row.Success {
					lipgloss.Printf("%s %s\n", activeDotStyle.Render("✓"), theme.MutedText.Render(row.LinkID))
					continue
				}
				detail := "delete failed"
				if row.Error != nil {
					detail = fmt.Sprintf("%s · %s", row.Error.Code, row.Error.Message)
				}
				lipgloss.Printf("%s %s  %s\n", unreachableStyle.Render("✗"), row.LinkID, theme.MutedText.Render(detail))
			}
			lipgloss.Printf("\n%s\n", createdHeadingStyle.Render(fmt.Sprintf("Deleted %d of %d", deleted, len(results))))
			air()
			return deleteOutcome(results, deleted)
		},
	}
	return cmd
}

// deleteOutcome: per-row failures are reported output; only a batch where
// nothing was deleted fails the command (the core marks whole-request
// invariant failures — not a member, read-only — on every row).
func deleteOutcome(results []api.BulkDeleteRowResult, deleted int) error {
	if deleted == 0 && len(results) > 0 {
		first := results[0]
		if first.Error != nil {
			return fmt.Errorf("no links were deleted (%s: %s)", first.Error.Code, first.Error.Message)
		}
		return fmt.Errorf("no links were deleted")
	}
	return nil
}

func validateLinkID(id string) error {
	if !strings.HasPrefix(id, "lnk_") {
		return fmt.Errorf("%q does not look like a link id (expected lnk_…; find ids with `zeb links --json`)", id)
	}
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
	lipgloss.Println(createdHeadingStyle.Render(fmt.Sprintf("Created %d %s", len(created), countLabel)))
	lipgloss.Println()
	for _, response := range created {
		link := response.Link
		short := displayShortLink(link)
		lipgloss.Printf("%s %s %s\n", linkShortStyle.Render(short), theme.MutedText.Render("->"), linkTargetStyle.Render(link.TargetURL))
		lipgloss.Printf("  %s\n", createdMetaFooter(link, response.TargetReachable, collectionID, collectionLabel))
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
// link. Empty when the API didn't check (created with --no-verify or through
// the bulk endpoint, so the field is nil) — the link was created regardless;
// this only annotates.
func reachabilityNote(reachable *bool) string {
	if reachable == nil {
		return ""
	}
	if *reachable {
		return reachableStyle.Render("● verified")
	}
	return unreachableStyle.Render("● unreachable")
}

func printLinkContext(cfg config.Config, collectionID string, collectionLabel string, flags listLinksFlags) {
	domain := cfg.ActiveDomain
	if domain == "" {
		domain = "server default"
	}
	lipgloss.Println(heading("Links"))
	contextLabel := theme.MutedText.Render("New links:")
	domainLabel := theme.BodyText.Render(domain)
	collectionText := "no collection"
	if cfg.ActiveCollection != "" {
		collectionText = cfg.ActiveCollection
	}
	lipgloss.Printf("%s  domain %s  %s  collection %s\n", contextLabel, domainLabel, theme.MutedText.Render("·"), theme.CollectionText.Render(collectionText))
	lipgloss.Printf("%s %s\n", theme.MutedText.Render("Showing:"), showingLabel(collectionID, collectionLabel, flags))
	lipgloss.Println()
}

// showingLabel describes exactly what the list below contains — collection
// scope, status filter, and non-default sort — so a filtered view never
// claims to be "all links".
func showingLabel(collectionID string, collectionLabel string, flags listLinksFlags) string {
	scope := "all links"
	if collectionID != "" {
		scope = collectionLabel + " " + theme.MutedText.Render("("+collectionID+")")
	}
	parts := []string{scope}
	if flags.Status != "" {
		parts = append(parts, flags.Status+" only")
	}
	if flags.Sort != "" {
		parts = append(parts, "sorted by "+flags.Sort)
	}
	return strings.Join(parts, theme.MutedText.Render(" · "))
}

func printLinks(links []api.Link) {
	if len(links) == 0 {
		lipgloss.Println("No links found.")
		return
	}
	for idx, link := range links {
		if idx > 0 {
			lipgloss.Println()
		}
		printLink(link)
	}
}

func printLink(link api.Link) {
	short := displayShortLink(link)
	target := truncate(link.TargetURL, 92)
	dot, status := linkStatus(link.IsActive)

	shortStyle := linkShortStyle
	if !link.IsActive {
		// Switched off — no live emerald on the URL.
		shortStyle = theme.BodyText
	}
	lipgloss.Printf("%s %s %s %s\n", dot, shortStyle.Render(short), theme.MutedText.Render("->"), linkTargetStyle.Render(target))
	if link.Title != nil && strings.TrimSpace(*link.Title) != "" {
		lipgloss.Printf("  %s\n", linkTitleStyle.Render(truncate(strings.TrimSpace(*link.Title), 110)))
	}
	meta := []string{theme.MutedText.Render(link.ID), status}
	if link.TotalClicks != nil {
		meta = append(meta, theme.MutedText.Render(clicksLabel(*link.TotalClicks)))
	}
	lipgloss.Printf("  %s\n", strings.Join(meta, theme.MutedText.Render(" · ")))
}

func clicksLabel(count int) string {
	if count == 1 {
		return "1 click"
	}
	return fmt.Sprintf("%d clicks", count)
}

func printLinkDetail(link api.Link) {
	printLink(link)
	lipgloss.Printf("  %s\n", theme.MutedText.Render("created "+link.CreatedAt))
	if link.Description != nil && strings.TrimSpace(*link.Description) != "" {
		lipgloss.Printf("  %s\n", linkTitleStyle.Render(strings.TrimSpace(*link.Description)))
	}
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

// truncate shortens to `limit` characters, counting runes — byte slicing
// would split multibyte titles/URLs mid-rune.
func truncate(value string, limit int) string {
	if limit <= 1 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

// Link rendering styles, resolved from the active theme by applyCLIStyles (in
// help.go) so they track the detected light/dark palette rather than freezing
// at the package-load default.
var (
	activeDotStyle, inactiveDotStyle           lipgloss.Style
	activeStatusStyle, inactiveStatusStyle     lipgloss.Style
	createdHeadingStyle                        lipgloss.Style
	createdDomainStyle, createdCollectionStyle lipgloss.Style
	linkShortStyle, linkTargetStyle            lipgloss.Style
	linkTitleStyle                             lipgloss.Style
	reachableStyle, unreachableStyle           lipgloss.Style
)
