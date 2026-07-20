// Link query + resolve commands. `zeb links query` exposes the full filter
// vocabulary (the same LinkFilter the dashboard search, smart collections, and
// the assistant use) over POST /links/query; `zeb links resolve` maps a short
// URL/code back to its link via GET /links/lookup. Both honor --json/--agent
// for machine-readable output.
package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
)

// Value hints mirror the API vocabulary for help text; the server stays the
// validator, so a stale hint here only weakens a suggestion. The spec drift
// test keeps the endpoints — not these enums — honest.
var (
	filterWindowValues     = []string{"1h", "24h", "7d", "30d"}
	filterStatusValues     = []string{"active", "inactive"}
	filterScheduleValues   = []string{"upcoming", "completed"}
	filterCreatedViaValues = []string{"web", "import", "chat", "api"}
	filterAttributionVals  = []string{"utm", "signals", "any"}
	filterNegatableFields  = []string{
		"status", "created", "edited", "clicked", "schedule", "createdVia",
		"attribution", "hasCollection", "shortDomain", "targetHost", "clicks",
		"uniqueClicks",
	}
	// The per-dimension flag names, used to reject combining them with --filter.
	dimensionFlagNames = []string{
		"status", "created", "edited", "clicked", "schedule", "created-via",
		"attribution", "in-collection", "uncollected", "short-domain",
		"target-host", "min-clicks", "max-clicks", "min-unique", "max-unique",
		"not",
	}
)

type queryLinksFlags struct {
	Status       string
	Created      string
	Edited       string
	Clicked      string
	Schedule     string
	CreatedVia   string
	Attribution  string
	InCollection bool
	Uncollected  bool
	ShortDomain  []string
	TargetHost   []string
	MinClicks    int
	MaxClicks    int
	MinUnique    int
	MaxUnique    int
	Not          []string
	Filter       string
	SaveAs       string
	Limit        int
	Offset       int
}

func newLinksQueryCommand(root *rootOptions) *cobra.Command {
	flags := &queryLinksFlags{}
	cmd := &cobra.Command{
		Use:   "query [text]",
		Short: "Find links by filter",
		Long: "Find links by any combination of filter dimensions — the same vocabulary the\n" +
			"dashboard search and smart collections use.\n\n" +
			"Dimensions AND-combine; list flags (--target-host, --short-domain, --not) OR\n" +
			"within a dimension; --not inverts a dimension. The optional [text] is a\n" +
			"free-text match over each link's path, title, and destination.\n\n" +
			"Returns a page of links plus the true uncapped match count. Use --json (or\n" +
			"--agent) for machine-readable {links, total}.\n\n" +
			"Examples:\n" +
			"  zeb links query --status active --not clicked --clicked 30d   # active, stale links\n" +
			"  zeb links query --target-host cnn.com,bbc.com --min-clicks 100\n" +
			"  zeb links query \"newsletter\" --created 7d --json\n" +
			"  zeb links query --filter '{\"status\":\"inactive\"}'",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			input, err := buildQueryInput(cmd, flags, args)
			if err != nil {
				return err
			}
			// --save-as persists the same filter as a smart collection instead
			// of listing: its membership IS this query, kept live.
			if flags.SaveAs != "" {
				if isEmptyFilter(input.LinkFilter) {
					return fmt.Errorf("--save-as needs at least one filter — a smart collection can't be empty")
				}
				filter := input.LinkFilter
				created, err := ctx.Client.CreateCollection(cmd.Context(), ctx.SpaceID, api.CreateCollectionInput{
					Type:   "smart",
					Name:   flags.SaveAs,
					Filter: &filter,
				})
				if err != nil {
					return err
				}
				if root.JSON {
					return writeJSON(created)
				}
				printSmartCollectionCreated(created)
				return nil
			}
			response, err := ctx.Client.QueryLinks(cmd.Context(), ctx.SpaceID, input)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printLinks(response.Links)
			printQuerySummary(response, input)
			return nil
		},
	}
	addQueryLinksFlags(cmd, flags)
	return cmd
}

func addQueryLinksFlags(cmd *cobra.Command, flags *queryLinksFlags) {
	cmd.Flags().StringVar(&flags.Status, "status", "", "status: "+strings.Join(filterStatusValues, " | "))
	cmd.Flags().StringVar(&flags.Created, "created", "", "created within a window: "+strings.Join(filterWindowValues, " | "))
	cmd.Flags().StringVar(&flags.Edited, "edited", "", "last edited within a window: "+strings.Join(filterWindowValues, " | "))
	cmd.Flags().StringVar(&flags.Clicked, "clicked", "", "last clicked within a window: "+strings.Join(filterWindowValues, " | "))
	cmd.Flags().StringVar(&flags.Schedule, "schedule", "", "schedule state: "+strings.Join(filterScheduleValues, " | "))
	cmd.Flags().StringVar(&flags.CreatedVia, "created-via", "", "creation source: "+strings.Join(filterCreatedViaValues, " | "))
	cmd.Flags().StringVar(&flags.Attribution, "attribution", "", "attribution carried: "+strings.Join(filterAttributionVals, " | "))
	cmd.Flags().BoolVar(&flags.InCollection, "in-collection", false, "only links that are in at least one collection")
	cmd.Flags().BoolVar(&flags.Uncollected, "uncollected", false, "only links that are in no collection")
	cmd.Flags().StringSliceVar(&flags.ShortDomain, "short-domain", nil, "the link's own domain hostname(s); repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&flags.TargetHost, "target-host", nil, "destination hostname(s); repeatable or comma-separated")
	cmd.Flags().IntVar(&flags.MinClicks, "min-clicks", 0, "more than N total clicks")
	cmd.Flags().IntVar(&flags.MaxClicks, "max-clicks", 0, "fewer than N total clicks")
	cmd.Flags().IntVar(&flags.MinUnique, "min-unique", 0, "more than N unique clicks")
	cmd.Flags().IntVar(&flags.MaxUnique, "max-unique", 0, "fewer than N unique clicks")
	cmd.Flags().StringSliceVar(&flags.Not, "not", nil, "invert a dimension; repeatable. Fields: "+strings.Join(filterNegatableFields, ", "))
	cmd.Flags().StringVar(&flags.Filter, "filter", "", "raw LinkFilter JSON (exclusive with the per-dimension flags and text)")
	cmd.Flags().StringVar(&flags.SaveAs, "save-as", "", "persist this filter as a smart collection with this name (its membership stays live) instead of listing")
	cmd.Flags().IntVarP(&flags.Limit, "limit", "l", 20, "page size (server range 1-100)")
	cmd.Flags().IntVar(&flags.Offset, "offset", 0, "result offset for paging")
}

// buildQueryInput turns the flags + optional free text into a QueryLinksInput,
// enforcing the small set of client-side contradictions (both a min and a max
// on one count dimension, --filter mixed with per-dimension flags, etc.). The
// server remains the authority on values.
func buildQueryInput(cmd *cobra.Command, flags *queryLinksFlags, args []string) (api.QueryLinksInput, error) {
	var input api.QueryLinksInput
	input.Limit = flags.Limit
	input.Offset = flags.Offset

	text := strings.TrimSpace(strings.Join(args, " "))

	if flags.Filter != "" {
		if text != "" || anyDimensionFlagSet(cmd) {
			return input, fmt.Errorf("--filter is exclusive with free text and the per-dimension flags; use one or the other")
		}
		if err := json.Unmarshal([]byte(flags.Filter), &input.LinkFilter); err != nil {
			return input, fmt.Errorf("--filter is not valid LinkFilter JSON: %w", err)
		}
		return input, nil
	}

	f := &input.LinkFilter
	f.Query = text
	f.Status = flags.Status
	f.Created = flags.Created
	f.Edited = flags.Edited
	f.Clicked = flags.Clicked
	f.Schedule = flags.Schedule
	f.CreatedVia = flags.CreatedVia
	f.Attribution = flags.Attribution
	f.ShortDomain = flags.ShortDomain
	f.TargetHost = flags.TargetHost
	f.Negate = flags.Not

	if flags.InCollection && flags.Uncollected {
		return input, fmt.Errorf("--in-collection and --uncollected are mutually exclusive")
	}
	if flags.InCollection {
		yes := true
		f.HasCollection = &yes
	}
	if flags.Uncollected {
		no := false
		f.HasCollection = &no
	}

	clicks, err := clickThreshold(cmd, "min-clicks", "max-clicks", flags.MinClicks, flags.MaxClicks, "clicks")
	if err != nil {
		return input, err
	}
	f.Clicks = clicks
	unique, err := clickThreshold(cmd, "min-unique", "max-unique", flags.MinUnique, flags.MaxUnique, "unique clicks")
	if err != nil {
		return input, err
	}
	f.UniqueClicks = unique

	return input, nil
}

// clickThreshold resolves the min/max flag pair for one count dimension into a
// single {op, value} threshold. The API takes one comparison per dimension, so
// setting both a floor and a ceiling is a client-side error.
func clickThreshold(cmd *cobra.Command, minFlag, maxFlag string, minVal, maxVal int, label string) (*api.ClickThreshold, error) {
	hasMin := cmd.Flags().Changed(minFlag)
	hasMax := cmd.Flags().Changed(maxFlag)
	if hasMin && hasMax {
		return nil, fmt.Errorf("--%s and --%s both set: %s takes a single threshold (more-than OR fewer-than)", minFlag, maxFlag, label)
	}
	if hasMin {
		return &api.ClickThreshold{Op: "greaterThan", Value: minVal}, nil
	}
	if hasMax {
		return &api.ClickThreshold{Op: "lessThan", Value: maxVal}, nil
	}
	return nil, nil
}

// isEmptyFilter reports whether a filter carries no dimension at all — the one
// case Core rejects for a smart collection (membership must be defined).
func isEmptyFilter(f api.LinkFilter) bool {
	return f.Query == "" && f.Status == "" && f.Created == "" && f.Edited == "" &&
		f.Clicked == "" && f.Schedule == "" && f.CreatedVia == "" &&
		f.Attribution == "" && f.HasCollection == nil && len(f.ShortDomain) == 0 &&
		len(f.TargetHost) == 0 && f.Clicks == nil && f.UniqueClicks == nil &&
		len(f.Negate) == 0
}

func printSmartCollectionCreated(created api.CreateCollectionResponse) {
	fmt.Printf("Created smart collection %q (%s) — %d link(s).\n",
		created.Collection.Name, created.Collection.ID, created.Collection.LinkCount)
	if created.RulesSummary != nil && *created.RulesSummary != "" {
		fmt.Printf("Rule: %s\n", *created.RulesSummary)
	}
}

func anyDimensionFlagSet(cmd *cobra.Command) bool {
	for _, name := range dimensionFlagNames {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func printQuerySummary(response api.QueryLinksResponse, input api.QueryLinksInput) {
	if response.Total == 0 {
		fmt.Println("\nNo links match.")
		return
	}
	end := input.Offset + len(response.Links)
	fmt.Printf("\n%d match(es); showing %d–%d.\n", response.Total, input.Offset+1, end)
	if end < response.Total {
		fmt.Printf("More: rerun with --offset %d\n", end)
	}
}

func newLinksResolveCommand(root *rootOptions) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "resolve <short-url|code>",
		Short: "Resolve a short link to its record",
		Long: "Resolve a short link to the link it addresses — the inverse of creating one.\n\n" +
			"Pass a full short URL (https://zbrah.link/abc or zbrah.link/abc), or a bare\n" +
			"code with --domain. Use --json (or --agent) for the {link} object. An unknown\n" +
			"link exits non-zero with a machine-readable error.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			arg := strings.TrimSpace(args[0])
			var shortURL, key string
			if strings.Contains(arg, "/") {
				shortURL = arg
			} else {
				key = arg
			}
			response, err := ctx.Client.LookupLink(cmd.Context(), ctx.SpaceID, shortURL, domain, key)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printLinkDetail(response.Link)
			return nil
		},
	}
	cmd.Flags().StringVarP(&domain, "domain", "d", "", "domain hostname when resolving a bare code")
	return cmd
}
