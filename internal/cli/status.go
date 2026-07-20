// Status command reports the resolved local CLI context.
// The plain form only reads local files, so it is fast and safe offline.
// --check additionally validates the stored key and context ids against the
// API, so stale state (e.g. after a database wipe) is caught here instead of
// failing some later create.
package cli

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newStatusCommand(root *rootOptions) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current CLI context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			credentials, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			apiURL, err := config.ResolveAPIURL(root.APIURL)
			if err != nil {
				return err
			}
			spaceID, err := config.ResolveSpace(root.SpaceID)
			if err != nil {
				return err
			}

			status := map[string]any{
				"apiUrl":           apiURL,
				"apiUrlSource":     apiURLSource(root.APIURL, cfg),
				"spaceId":          nullString(spaceID),
				"spaceSource":      nullString(spaceSource(root.SpaceID, cfg)),
				"collectionId":     nullString(cfg.ActiveCollection),
				"collectionSource": nullString(collectionSource(cfg)),
				"domain":           nullString(cfg.ActiveDomain),
				"domainSource":     nullString(domainSource(cfg)),
				"loggedIn":         credentials != nil,
			}

			var checks []contextCheck
			if check {
				checks = runContextChecks(cmd, root, spaceID, cfg)
				status["checks"] = checks
			}

			if root.JSON {
				if err := writeJSON(status); err != nil {
					return err
				}
				return checksOutcome(checks)
			}

			section("Status")
			statusRow("API URL", apiURL, apiURLSource(root.APIURL, cfg))
			statusRow("Space", emptyLabel(spaceID), spaceSource(root.SpaceID, cfg))
			statusRow("Collection", emptyLabel(cfg.ActiveCollection), collectionSource(cfg))
			statusRow("Domain", emptyLabel(cfg.ActiveDomain), domainSource(cfg))
			statusRow("Auth", authLabel(credentials != nil), "")
			if check {
				section("Checks")
				for _, result := range checks {
					printCheck(result)
				}
			}
			air()
			return checksOutcome(checks)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "validate the stored key, space, collection, and domain against the API")
	return cmd
}

type contextCheck struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Note string `json:"note,omitempty"`
	Hint string `json:"hint,omitempty"`
}

// runContextChecks validates each stored value against the API. Failures are
// reported per item — the command keeps going so one dangling id doesn't hide
// the rest of the picture.
func runContextChecks(cmd *cobra.Command, root *rootOptions, spaceID string, cfg config.Config) []contextCheck {
	var checks []contextCheck

	client, _, err := resolveClient(root)
	if err != nil {
		return []contextCheck{{Name: "auth", OK: false, Note: err.Error(), Hint: "zeb auth login"}}
	}
	me, err := client.GetMe(cmd.Context())
	if err != nil {
		return []contextCheck{{Name: "auth", OK: false, Note: "API key was rejected or the API is unreachable: " + err.Error(), Hint: "zeb auth login"}}
	}
	checks = append(checks, contextCheck{Name: "auth", OK: true, Note: "key accepted for " + me.User.Email})

	if spaceID == "" {
		checks = append(checks, contextCheck{Name: "space", OK: false, Note: "no active space", Hint: "zeb space use <id-or-name>"})
		return checks
	}
	space, err := resolveSpace(me.AccessibleSpaces, spaceID)
	if err != nil {
		checks = append(checks, contextCheck{Name: "space", OK: false, Note: fmt.Sprintf("space %q is not accessible with this key", spaceID), Hint: "zeb space list && zeb space use <id-or-name>"})
		return checks
	}
	checks = append(checks, contextCheck{Name: "space", OK: true, Note: fmt.Sprintf("%s (%s)", space.Name, space.ID)})

	if cfg.ActiveCollection != "" {
		collections, err := client.ListCollections(cmd.Context(), space.ID)
		if err != nil {
			checks = append(checks, contextCheck{Name: "collection", OK: false, Note: "could not list collections: " + err.Error()})
		} else if collection, err := resolveCollection(collections.Collections, cfg.ActiveCollection); err != nil {
			checks = append(checks, contextCheck{
				Name: "collection", OK: false,
				Note: fmt.Sprintf("active collection %s no longer exists", cfg.ActiveCollection),
				Hint: "zeb collection clear (or zeb context to pick a new one)",
			})
		} else {
			checks = append(checks, contextCheck{Name: "collection", OK: true, Note: fmt.Sprintf("%s (%s)", collection.Name, collection.ID)})
		}
	}

	if cfg.ActiveDomain != "" {
		domains, err := client.ListDomains(cmd.Context(), space.ID)
		if err != nil {
			checks = append(checks, contextCheck{Name: "domain", OK: false, Note: "could not list domains: " + err.Error()})
		} else {
			found := false
			for _, domain := range domains.Domains {
				if domain.Hostname == cfg.ActiveDomain {
					found = true
					break
				}
			}
			if found {
				checks = append(checks, contextCheck{Name: "domain", OK: true, Note: cfg.ActiveDomain})
			} else {
				checks = append(checks, contextCheck{
					Name: "domain", OK: false,
					Note: fmt.Sprintf("active domain %q is not available in this space", cfg.ActiveDomain),
					Hint: "zeb domain clear (or zeb domains to pick one)",
				})
			}
		}
	}
	return checks
}

// statusRow prints one aligned "Label   value (source)" line: label muted,
// value in body ink, source a faint parenthetical.
func statusRow(label, value, source string) {
	const w = 11
	pad := w - len(label)
	if pad < 1 {
		pad = 1
	}
	line := "  " + theme.MutedText.Render(label) + strings.Repeat(" ", pad) + theme.BodyText.Render(value)
	if source != "" {
		line += " " + theme.FaintText.Render("("+source+")")
	}
	lipgloss.Println(line)
}

func printCheck(result contextCheck) {
	mark := lipgloss.NewStyle().Bold(true).Foreground(theme.Good).Render("✓")
	if !result.OK {
		mark = lipgloss.NewStyle().Bold(true).Foreground(theme.Bad).Render("✗")
	}
	line := "  " + mark + " " + theme.BodyText.Render(result.Name) + "  " + theme.MutedText.Render(result.Note)
	lipgloss.Println(line)
	if result.Hint != "" {
		lipgloss.Println("    " + theme.FaintText.Render("fix: "+result.Hint))
	}
}

// checksOutcome makes `zeb status --check` scriptable: any failed check exits
// non-zero. A plain `zeb status` passes nil checks and always succeeds.
func checksOutcome(checks []contextCheck) error {
	for _, result := range checks {
		if !result.OK {
			return fmt.Errorf("context check failed: %s", result.Note)
		}
	}
	return nil
}

func apiURLSource(flagValue string, cfg config.Config) string {
	if flagValue != "" {
		return "flag"
	}
	if os.Getenv("ZLINK_API_URL") != "" {
		return "env"
	}
	if cfg.APIURL != "" {
		return "config"
	}
	return "default"
}

func spaceSource(flagValue string, cfg config.Config) string {
	if flagValue != "" {
		return "flag"
	}
	if os.Getenv("ZLINK_SPACE") != "" {
		return "env"
	}
	if cfg.ActiveSpace != "" {
		return "config"
	}
	return ""
}

func domainSource(cfg config.Config) string {
	if cfg.ActiveDomain != "" {
		return "config"
	}
	return ""
}

func collectionSource(cfg config.Config) string {
	if cfg.ActiveCollection != "" {
		return "config"
	}
	return ""
}

func authLabel(loggedIn bool) string {
	if loggedIn {
		return "stored API key"
	}
	return "(not logged in)"
}

func sourceSuffix(source string) string {
	if source == "" {
		return ""
	}
	return fmt.Sprintf(" (%s)", source)
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
