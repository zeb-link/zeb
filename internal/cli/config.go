// Config commands expose the local ~/.zlink files.
// They are intentionally boring so scripts and future TUI screens can rely
// on the same values.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/config"
)

func newConfigCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and edit local CLI configuration",
	}
	cmd.AddCommand(newConfigGetCommand(root), newConfigSetCommand(), newConfigUnsetCommand(), newConfigPathCommand(root))
	return cmd
}

func newConfigGetCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Print local configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(cfg)
			}
			fmt.Println(heading("Configuration"))
			fmt.Printf("apiUrl: %s\n", emptyLabel(cfg.APIURL))
			fmt.Printf("activeSpace: %s\n", emptyLabel(cfg.ActiveSpace))
			fmt.Printf("activeCollection: %s\n", emptyLabel(cfg.ActiveCollection))
			fmt.Printf("activeDomain: %s\n", emptyLabel(cfg.ActiveDomain))
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetValue(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Set %s\n", args[0])
			return nil
		},
	}
}

func newConfigUnsetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.UnsetValue(args[0]); err != nil {
				return err
			}
			fmt.Printf("Unset %s\n", args[0])
			return nil
		},
	}
}

func newConfigPathCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print config file locations",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.Dir()
			if err != nil {
				return err
			}
			credentialsPath, err := config.CredentialsPath()
			if err != nil {
				return err
			}
			configPath, err := config.ConfigPath()
			if err != nil {
				return err
			}
			paths := map[string]string{
				"configDir":       dir,
				"configFile":      configPath,
				"credentialsFile": credentialsPath,
			}
			if root.JSON {
				return writeJSON(paths)
			}
			fmt.Println(heading("Config paths"))
			fmt.Printf("Config dir: %s\n", paths["configDir"])
			fmt.Printf("Config file: %s\n", paths["configFile"])
			fmt.Printf("Credentials file: %s\n", paths["credentialsFile"])
			return nil
		},
	}
}
