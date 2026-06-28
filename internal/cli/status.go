// Status command reports the resolved local CLI context.
// It mirrors the old CLI's context check without calling the API, so it is fast
// and safe before authentication is fully wired into interactive flows.
package cli

import (
	"fmt"
	"os"

	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
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
			if root.JSON {
				return writeJSON(status)
			}

			fmt.Println(heading("Current Context"))
			fmt.Printf("API URL:  %s (%s)\n", apiURL, apiURLSource(root.APIURL, cfg))
			fmt.Printf("Space:    %s%s\n", emptyLabel(spaceID), sourceSuffix(spaceSource(root.SpaceID, cfg)))
			fmt.Printf("Collection: %s%s\n", emptyLabel(cfg.ActiveCollection), sourceSuffix(collectionSource(cfg)))
			fmt.Printf("Domain:   %s%s\n", emptyLabel(cfg.ActiveDomain), sourceSuffix(domainSource(cfg)))
			fmt.Printf("Auth:     %s\n", authLabel(credentials != nil))
			return nil
		},
	}
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
