// Status command reports the resolved local CLI context.
// The plain form only reads local files, so it is fast and safe offline.
// --check additionally validates the stored key and context ids against the
// API, so stale state (e.g. after a database wipe) is caught here instead of
// failing some later create.
package cli

import (
	"fmt"
	"os"

	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
	"github.com/spf13/cobra"
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

			fmt.Println(heading("Current Context"))
			fmt.Printf("API URL:  %s (%s)\n", apiURL, apiURLSource(root.APIURL, cfg))
			fmt.Printf("Space:    %s%s\n", emptyLabel(spaceID), sourceSuffix(spaceSource(root.SpaceID, cfg)))
			fmt.Printf("Collection: %s%s\n", emptyLabel(cfg.ActiveCollection), sourceSuffix(collectionSource(cfg)))
			fmt.Printf("Domain:   %s%s\n", emptyLabel(cfg.ActiveDomain), sourceSuffix(domainSource(cfg)))
			fmt.Printf("Auth:     %s\n", authLabel(credentials != nil))
			if check {
				fmt.Println()
				fmt.Println(heading("Checks"))
				for _, result := range checks {
					printCheck(result)
				}
			}
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

func printCheck(result contextCheck) {
	mark := checkOKStyle.Render("✓")
	if !result.OK {
		mark = checkFailStyle.Render("✗")
	}
	fmt.Printf("%s %s  %s\n", mark, result.Name, theme.MutedText.Render(result.Note))
	if result.Hint != "" {
		fmt.Printf("    %s\n", theme.MutedText.Render("fix: "+result.Hint))
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

var (
	checkOKStyle   = activeDotStyle
	checkFailStyle = unreachableStyle
)
