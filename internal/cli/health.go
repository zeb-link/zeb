// Health command pings the API's public health endpoint — a fast way to
// confirm the resolved API URL points at a live Core before debugging auth
// or context problems.
package cli

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newHealthCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check that the API is reachable",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiURL, err := config.ResolveAPIURL(root.APIURL)
			if err != nil {
				return err
			}
			// The endpoint is public; send the key only if one is stored.
			key, err := config.ResolveAPIKey(root.APIKey)
			if err != nil {
				return err
			}
			client := api.New(api.Options{APIURL: apiURL, APIKey: key})
			response, err := client.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("API at %s is not healthy: %w", apiURL, err)
			}
			if root.JSON {
				return writeJSON(map[string]any{"ok": response.OK, "api": response.API, "apiUrl": apiURL})
			}
			lipgloss.Println("  " + theme.GoodText.Render("✓") + " " + theme.BodyText.Render(apiURL) + " " + theme.FaintText.Render("("+response.API+")"))
			return nil
		},
	}
}
