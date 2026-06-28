// Auth commands manage the local API key and default space.
// Login validates the key against `/api/v1/me`, then writes ~/.zlink
// credentials and context files.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	cmd.AddCommand(newLoginCommand(root), newLogoutCommand(), newWhoamiCommand(root))
	return cmd
}

func newLoginCommand(root *rootOptions) *cobra.Command {
	var apiKey string
	var spaceID string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store and validate an API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key := firstNonEmpty(apiKey, root.APIKey)
			if key == "" {
				prompted, err := readSecret("API key")
				if err != nil {
					return err
				}
				key = prompted
			}
			if !strings.HasPrefix(key, "zeb_") {
				return fmt.Errorf("API key should start with zeb_")
			}
			apiURL, err := config.ResolveAPIURL(root.APIURL)
			if err != nil {
				return err
			}
			client := api.New(api.Options{APIURL: apiURL, APIKey: key})
			me, err := client.GetMe(context.Background())
			if err != nil {
				return err
			}

			selectedSpace := firstNonEmpty(spaceID, root.SpaceID)
			if selectedSpace == "" && len(me.AccessibleSpaces) == 1 {
				selectedSpace = me.AccessibleSpaces[0].ID
			}
			if selectedSpace == "" && len(me.AccessibleSpaces) > 1 {
				selectedSpace = chooseSpace(me.AccessibleSpaces)
			}
			if selectedSpace != "" {
				space, err := resolveSpace(me.AccessibleSpaces, selectedSpace)
				if err != nil {
					return err
				}
				selectedSpace = space.ID
			}

			if err := config.SaveCredentials(key); err != nil {
				return err
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.APIURL = apiURL
			if selectedSpace != "" {
				cfg.ActiveSpace = selectedSpace
			}
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}

			if root.JSON {
				return writeJSON(map[string]any{
					"user":        me.User,
					"apiUrl":      apiURL,
					"activeSpace": selectedSpace,
					"spaces":      me.AccessibleSpaces,
				})
			}
			fmt.Println(heading("Logged in"))
			fmt.Printf("User: %s\n", me.User.Email)
			if selectedSpace != "" {
				fmt.Printf("Active space: %s\n", selectedSpace)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to store")
	cmd.Flags().StringVar(&spaceID, "space", "", "space id/name to make active after login")
	return cmd
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.ClearCredentials(); err != nil {
				return err
			}
			fmt.Println("Logged out")
			return nil
		},
	}
}

func newWhoamiCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current API identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := config.ResolveAPIKey(root.APIKey)
			if err != nil {
				return err
			}
			if key == "" {
				return fmt.Errorf("not logged in; run zeb auth login")
			}
			apiURL, err := config.ResolveAPIURL(root.APIURL)
			if err != nil {
				return err
			}
			spaceID, err := config.ResolveSpace(root.SpaceID)
			if err != nil {
				return err
			}
			client := api.New(api.Options{APIURL: apiURL, APIKey: key})
			me, err := client.GetMe(context.Background())
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{
					"user":        me.User,
					"apiKey":      me.APIKey,
					"activeSpace": spaceID,
					"spaces":      me.AccessibleSpaces,
				})
			}
			fmt.Println(heading("Current identity"))
			fmt.Printf("Email: %s\n", me.User.Email)
			if me.User.Name != nil {
				fmt.Printf("Name: %s\n", *me.User.Name)
			}
			fmt.Printf("User ID: %s\n", me.User.ID)
			fmt.Printf("API key: %s\n", me.APIKey.Prefix)
			fmt.Printf("Active space: %s\n", emptyLabel(spaceID))
			fmt.Printf("Accessible spaces: %d\n", len(me.AccessibleSpaces))
			return nil
		},
	}
}

func readSecret(label string) (string, error) {
	fmt.Printf("%s: ", label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return strings.TrimSpace(string(value)), err
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	return strings.TrimSpace(value), err
}

func chooseSpace(spaces []api.SpaceSummary) string {
	fmt.Println("Accessible spaces:")
	for idx, space := range spaces {
		fmt.Printf("  %d. %s (%s, %s)\n", idx+1, space.Name, space.ID, space.Role)
	}
	fmt.Print("Choose active space number: ")
	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	for idx, space := range spaces {
		if value == fmt.Sprintf("%d", idx+1) {
			return space.ID
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func emptyLabel(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}
