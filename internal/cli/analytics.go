// `zeb analytics` — the click-analytics query, the counterpart to
// `zeb links query`. It shares the object-scope flags ("which links") and adds
// click dimensions ("which clicks") plus --group-by/--measure/--range. --json
// (or --agent) returns the raw aggregate for scripts and agents.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

var (
	analyticsGroupByValues = []string{
		"country", "region", "city", "browser", "os", "deviceType",
		"referrerDomain", "fromQr", "shortDomain", "isBot", "botType",
		"botName", "day", "link",
	}
	analyticsMeasureValues = []string{"clicks", "uniqueClicks"}
	analyticsRangeValues   = []string{"15m", "1h", "1d", "7d", "30d", "all", "custom"}
	// Analytics-specific flag names (click dims + aggregation), used with
	// --filter to reject mixing raw JSON with per-dimension flags.
	analyticsOwnFlagNames = []string{
		"country", "continent", "region", "city", "browser", "os",
		"device-type", "referrer", "short-domain", "from-qr", "is-bot",
		"bot-type", "bot-name", "group-by", "measure", "range", "from", "to",
		"limit", "collection", "link",
	}
)

type analyticsFlags struct {
	objectScopeFlags
	// Click dimensions.
	Country     []string
	Continent   []string
	Region      []string
	City        []string
	Browser     []string
	OS          []string
	DeviceType  []string
	Referrer    []string
	ShortDomain []string
	FromQr      bool
	IsBot       bool
	BotType     []string
	BotName     []string
	// Aggregation + scope.
	GroupBy    string
	Measure    string
	Range      string
	From       string
	To         string
	Limit      int
	Collection string
	Link       string
	Filter     string
}

func newAnalyticsCommand(root *rootOptions) *cobra.Command {
	flags := &analyticsFlags{}
	cmd := &cobra.Command{
		Use:   "analytics [text]",
		Short: "Query click analytics",
		Long: "Query click analytics — the click counterpart to `zeb links query`.\n\n" +
			"Two halves: SCOPE which links to count (the same object flags as\n" +
			"`links query`: --status, --created, --target-host, --not, …), and\n" +
			"MEASURE the clicks (--country, --browser, --device-type, bot dims, …).\n" +
			"--group-by returns a breakdown (one row per value); omit it for a\n" +
			"single total. --measure counts clicks (default) or uniqueClicks;\n" +
			"--range sets the window. Use --json (or --agent) for the raw aggregate.\n\n" +
			"Examples:\n" +
			"  zeb analytics --group-by country --range 7d\n" +
			"  zeb analytics --link zbrah.link/2xc --range 7d\n" +
			"  zeb analytics --link zbrah.link/2xc --group-by country\n" +
			"  zeb analytics --status active --not clicked --clicked 30d --group-by browser\n" +
			"  zeb analytics --is-bot --group-by botType --json",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			input, err := buildAnalyticsInput(cmd, flags, args)
			if err != nil {
				return err
			}
			// Resolve the human --link handle (short URL / code / id) to a link
			// id before the query — reusing the same lookup as `zeb links lookup`
			// — so an unknown link errors clearly instead of counting zero clicks.
			if flags.Link != "" {
				linkID, err := resolveAnalyticsLinkID(cmd.Context(), ctx, flags.Link)
				if err != nil {
					return err
				}
				input.LinkID = linkID
			}
			response, err := ctx.Client.QueryAnalytics(cmd.Context(), ctx.SpaceID, input)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printAnalytics(response)
			return nil
		},
	}
	addAnalyticsFlags(cmd, flags)
	return cmd
}

func addAnalyticsFlags(cmd *cobra.Command, flags *analyticsFlags) {
	// Shared object-scope flags (identical on `zeb links query`).
	addObjectScopeFlags(cmd, &flags.objectScopeFlags)
	// Click dimensions.
	cmd.Flags().StringSliceVar(&flags.Country, "country", nil, "click country code(s), e.g. US, GB")
	cmd.Flags().StringSliceVar(&flags.Continent, "continent", nil, "continent grouping(s): europe, asia, …")
	cmd.Flags().StringSliceVar(&flags.Region, "region", nil, "click region(s) / state(s)")
	cmd.Flags().StringSliceVar(&flags.City, "city", nil, "click city(ies)")
	cmd.Flags().StringSliceVar(&flags.Browser, "browser", nil, "browser(s), e.g. Chrome")
	cmd.Flags().StringSliceVar(&flags.OS, "os", nil, "operating system(s), e.g. iOS")
	cmd.Flags().StringSliceVar(&flags.DeviceType, "device-type", nil, "device type(s): mobile, desktop, tablet")
	cmd.Flags().StringSliceVar(&flags.Referrer, "referrer", nil, "referrer domain(s)")
	cmd.Flags().StringSliceVar(&flags.ShortDomain, "short-domain", nil, "the link's own domain hostname(s) the click came through")
	cmd.Flags().BoolVar(&flags.FromQr, "from-qr", false, "QR scans only (--from-qr=false for non-QR)")
	cmd.Flags().BoolVar(&flags.IsBot, "is-bot", false, "automated traffic only (--is-bot=false for humans)")
	cmd.Flags().StringSliceVar(&flags.BotType, "bot-type", nil, "traffic class(es): ai_assistant, crawler, …")
	cmd.Flags().StringSliceVar(&flags.BotName, "bot-name", nil, "bot product name(s), e.g. GPTBot")
	// Aggregation + scope.
	cmd.Flags().StringVar(&flags.GroupBy, "group-by", "", "break down by: "+strings.Join(analyticsGroupByValues, " | "))
	cmd.Flags().StringVar(&flags.Measure, "measure", "", "count: "+strings.Join(analyticsMeasureValues, " | "))
	cmd.Flags().StringVar(&flags.Range, "range", "", "time window: "+strings.Join(analyticsRangeValues, " | "))
	cmd.Flags().StringVar(&flags.From, "from", "", "custom window start (ISO); with --range custom")
	cmd.Flags().StringVar(&flags.To, "to", "", "custom window end (ISO); with --range custom")
	cmd.Flags().IntVarP(&flags.Limit, "limit", "l", 0, "max breakdown rows (group-by only; default 100)")
	cmd.Flags().StringVar(&flags.Collection, "collection", "", "scope to one collection id (exclusive with the object-attribute filters)")
	cmd.Flags().StringVar(&flags.Link, "link", "", "scope to one link by short URL, code, or id — this link's clicks (exclusive with the object-attribute filters and --collection)")
	cmd.Flags().StringVar(&flags.Filter, "filter", "", "raw analytics query JSON (exclusive with the per-dimension flags and text)")
}

func buildAnalyticsInput(cmd *cobra.Command, flags *analyticsFlags, args []string) (api.AnalyticsQueryInput, error) {
	var input api.AnalyticsQueryInput
	text := strings.TrimSpace(strings.Join(args, " "))

	if flags.Filter != "" {
		if text != "" || anyObjectScopeFlagSet(cmd) || anyAnalyticsOwnFlagSet(cmd) {
			return input, fmt.Errorf("--filter is exclusive with free text and the per-dimension flags; use one or the other")
		}
		if err := json.Unmarshal([]byte(flags.Filter), &input); err != nil {
			return input, fmt.Errorf("--filter is not valid analytics query JSON: %w", err)
		}
		return input, nil
	}

	// --link scopes to exactly one link (resolved to an id in RunE). Like
	// --collection it's an identity scope, so it can't be mixed with free text,
	// the object-attribute filters, or --collection; the click dimensions still
	// apply on top.
	if cmd.Flags().Changed("link") {
		if text != "" || anyObjectScopeFlagSet(cmd) || flags.Collection != "" {
			return input, fmt.Errorf("--link scopes to one link; it can't be combined with free text, the object-scope filters, or --collection")
		}
	}

	// Guardrail: a bare positional is a free-text SEARCH, but a link-shaped one
	// (a URL, a host/code short link, or a link id) is almost always someone who
	// meant to scope to that link and forgot --link. Refuse with a hint rather
	// than run a search that matches nothing and reports a misleading 0 clicks.
	if text != "" && looksLikeLinkReference(text) {
		return input, fmt.Errorf(
			"%q looks like a link, not a search term — a bare argument is a free-text search.\n"+
				"To scope analytics to that one link, use:\n"+
				"  zeb analytics --link %s",
			text, text,
		)
	}

	// Aggregation + scope.
	input.GroupBy = flags.GroupBy
	input.Measure = flags.Measure
	input.Range = flags.Range
	input.From = flags.From
	input.To = flags.To
	input.Limit = flags.Limit
	input.CollectionID = flags.Collection

	// Object scope (shared vocabulary).
	input.Query = text
	input.Status = flags.Status
	input.Created = flags.Created
	input.Edited = flags.Edited
	input.Schedule = flags.Schedule
	input.CreatedVia = flags.CreatedVia
	input.Attribution = flags.Attribution
	input.TargetHost = flags.TargetHost
	input.Negate = flags.Not
	hc, err := flags.hasCollection()
	if err != nil {
		return input, err
	}
	input.HasCollection = hc

	// Click dimensions.
	input.Country = flags.Country
	input.Continents = flags.Continent
	input.Region = flags.Region
	input.City = flags.City
	input.Browser = flags.Browser
	input.OS = flags.OS
	input.DeviceType = flags.DeviceType
	input.ReferrerDomain = flags.Referrer
	input.ShortDomain = flags.ShortDomain
	input.BotType = flags.BotType
	input.BotName = flags.BotName
	if cmd.Flags().Changed("from-qr") {
		v := flags.FromQr
		input.FromQr = &v
	}
	if cmd.Flags().Changed("is-bot") {
		v := flags.IsBot
		input.IsBot = &v
	}

	return input, nil
}

// resolveAnalyticsLinkID turns the --link handle into a link id. A value that
// already looks like a link id passes through; anything else is resolved the
// same way `zeb links lookup` does (a full/short URL by its path, a bare code by
// key), so `zeb analytics --link zbrah.link/2xc` scopes to that one link.
func resolveAnalyticsLinkID(ctx context.Context, api apiContext, handle string) (string, error) {
	h := strings.TrimSpace(handle)
	if strings.HasPrefix(h, "lnk_") {
		return h, nil
	}
	var shortURL, key string
	if strings.Contains(h, "/") {
		shortURL = h
	} else {
		key = h
	}
	resp, err := api.Client.LookupLink(ctx, api.SpaceID, shortURL, "", key)
	if err != nil {
		return "", err
	}
	return resp.Link.ID, nil
}

// looksLikeLinkReference reports whether a bare positional argument is really a
// link the user meant to scope to (and forgot --link), rather than a genuine
// free-text search term. True for a URL scheme, a link id, or a host/code
// short-link shape (a dotted host, an optional :port, then a /code). A plain
// word, a multi-word phrase, or a bare hostname without a path stays a search.
func looksLikeLinkReference(text string) bool {
	s := strings.TrimSpace(text)
	if s == "" || strings.ContainsAny(s, " \t") {
		return false // multi-word: a genuine search, never a single link
	}
	if strings.HasPrefix(s, "lnk_") {
		return true // a link id
	}
	if strings.Contains(s, "://") {
		return true // a full URL
	}
	// host/code shape: a dotted host (optional :port) then a non-empty /code.
	slash := strings.IndexByte(s, '/')
	if slash <= 0 || slash == len(s)-1 {
		return false
	}
	host := s[:slash]
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return strings.Contains(host, ".")
}

func anyAnalyticsOwnFlagSet(cmd *cobra.Command) bool {
	for _, name := range analyticsOwnFlagNames {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func printAnalytics(r api.AnalyticsQueryResponse) {
	if !r.Configured {
		lipgloss.Println(theme.WarnText.Render("Analytics backend isn't configured for this space."))
		return
	}
	if r.TooLarge {
		lipgloss.Println(theme.MutedText.Render(r.Message))
		return
	}
	if r.GroupBy == nil {
		// Single total.
		clicks, unique := 0, 0
		if len(r.Rows) > 0 {
			clicks, unique = r.Rows[0].Clicks, r.Rows[0].UniqueClicks
		}
		lipgloss.Println("  " + theme.MutedText.Render("clicks ") + theme.CommandText.Render(fmt.Sprintf("%d", clicks)) +
			theme.MutedText.Render("   unique ") + theme.CommandText.Render(fmt.Sprintf("%d", unique)))
		return
	}
	if len(r.Rows) == 0 {
		lipgloss.Println(theme.MutedText.Render("No clicks match."))
		return
	}
	lipgloss.Println(theme.Heading.Render(fmt.Sprintf("%-28s %10s %10s", strings.ToUpper(*r.GroupBy), "clicks", "unique")))
	for _, row := range r.Rows {
		key := "(none)"
		if row.Key != nil && *row.Key != "" {
			key = *row.Key
		}
		lipgloss.Println(theme.CommandText.Render(fmt.Sprintf("%-28s", key)) + " " +
			theme.BodyText.Render(fmt.Sprintf("%10d", row.Clicks)) + " " +
			theme.MutedText.Render(fmt.Sprintf("%10d", row.UniqueClicks)))
	}
}
