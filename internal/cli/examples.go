// `zeb examples` — a categorized, copy-paste cookbook of common commands.
// It keeps the fuller example set out of every --help block while staying one
// obvious command away. Optional topic arg filters to a section.
package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

type exampleSection struct {
	key   string // topic filter key
	title string
	body  string
}

var exampleSections = []exampleSection{
	{"setup", "Setup — stateless (for agents & scripts)",
		"  export ZLINK_API_KEY=zeb_...        # no `zeb login` needed; --api-key also works\n" +
			"  export ZLINK_SPACE=spc_...          # find yours with: zeb auth whoami --json\n" +
			"  zeb status --check --json           # preflight: verify key + space (non-zero if bad)"},
	{"create", "Create links",
		"  zeb https://example.com                              # fast path (bare URL)\n" +
			"  zeb https://a.com https://b.com                      # batch (per-row results)\n" +
			"  zeb links create https://example.com --short-code launch --title \"Launch\""},
	{"browse", "Browse — `zeb links` (list only: status/collection/sort)",
		"  zeb links                                            # newest first\n" +
			"  zeb links --all --json                               # every link, machine-readable\n" +
			"  zeb links --status inactive --sort clicks-desc\n" +
			"  zeb links --collection \"Campaign\"                    # one collection's members"},
	{"find", "Find — `zeb links query` (any other filter)",
		"  zeb links query --target-host cnn.com,bbc.com --min-clicks 100\n" +
			"  zeb links query --attribution signals                # carries a QR signal\n" +
			"  zeb links query --clicked 30d --not clicked          # NOT clicked in 30 days\n" +
			"  zeb links query \"newsletter\" --created 7d --json     # free text + window\n" +
			"  zeb links query --filter '{\"targetHost\":[\"cnn.com\"],\"attribution\":\"signals\"}'\n" +
			"  zeb links query --status inactive --json | jq .total  # how many match?"},
	{"lookup", "Look up — short URL/code → its link",
		"  zeb links lookup zbrah.link/abc\n" +
			"  zeb links lookup https://zbrah.link/abc --json\n" +
			"  zeb links lookup abc --domain zbrah.link"},
	{"collections", "Smart collections — a saved filter (membership stays live)",
		"  zeb links query --target-host linktr.ee --save-as \"Linktree links\"\n" +
			"  zeb links query --status inactive --not clicked --clicked 30d --save-as \"Dead links\""},
	{"analytics", "Analytics — clicks over the same filter vocabulary",
		"  zeb analytics --group-by country --range 7d          # top countries this week\n" +
			"  zeb analytics --target-host cnn.com --group-by browser\n" +
			"  zeb analytics --is-bot --group-by botType --json     # the AI/bot traffic mix"},
	{"qr", "QR codes",
		"  zeb qr <link-id>                                     # public PNG + SVG URLs\n" +
			"  zeb qr <link-id> --download qr.png --size 1024       # save the rendered image\n" +
			"  zeb qr variants <link-id>                            # named designs"},
	{"manage", "Update / delete (link id OR short URL)",
		"  zeb links update zbrah.link/abc --inactive\n" +
			"  zeb links get lnk_01…  --json\n" +
			"  zeb links delete lnk_01… lnk_02…"},
	{"agent", "The agent contract",
		"  # --json (alias --agent) works on every command.\n" +
			"  # Success AND errors are JSON on stdout; the exit code signals failure.\n" +
			"  # Errors carry a machine-readable code, e.g. {\"error\":{\"code\":\"PATH_TAKEN\",…}}.\n" +
			"  # No interactive prompts — safe to drive unattended."},
}

func newExamplesCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "examples [topic]",
		Short: "Copy-paste cookbook of common commands",
		Long: "Print a categorized, copy-paste cookbook of common `zeb` commands.\n\n" +
			"Pass a topic to filter: setup, create, browse, find, lookup, collections,\n" +
			"analytics, qr, manage, agent. Everything here is ready to run.",
		Example: "  zeb examples            # the whole cookbook\n" +
			"  zeb examples find        # just the `links query` recipes\n" +
			"  zeb examples agent       # the machine-output contract",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := ""
			if len(args) == 1 {
				topic = strings.ToLower(strings.TrimSpace(args[0]))
			}
			shown := 0
			for _, s := range exampleSections {
				if topic != "" && !strings.Contains(s.key, topic) && !strings.Contains(strings.ToLower(s.title), topic) {
					continue
				}
				lipgloss.Println(renderExampleTitle(s.title))
				for _, ln := range strings.Split(s.body, "\n") {
					lipgloss.Println(styleShell(ln))
				}
				lipgloss.Println()
				shown++
			}
			if shown == 0 {
				return fmt.Errorf("no examples for %q; topics: setup, create, browse, find, lookup, collections, analytics, qr, manage, agent", topic)
			}
			return nil
		},
	}
}
