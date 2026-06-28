// Package cli wires Cobra commands for the zeb executable.
// Feature commands should stay thin: parse flags, resolve config, call the
// API/client layer, then format output.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	JSON    bool
	APIKey  string
	APIURL  string
	SpaceID string
	Version string
}

func Execute(version string) {
	opts := &rootOptions{Version: version}
	cmd := newRootCommand(opts)
	cmd.SetArgs(expandRootURLShorthand(os.Args[1:]))
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand(opts *rootOptions) *cobra.Command {
	createOptions := &createLinksOptions{}
	cmd := &cobra.Command{
		Use:   "zeb [url...]",
		Short: "Manage Zebra Link spaces from the terminal",
		Long:  "Manage Zebra Link spaces from the terminal.\n\nRun `zeb <url>` to create a short link, or use subcommands for listing links, choosing domains, and choosing collections.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runCreateLinks(opts, createOptions, args)
		},
	}

	cmd.PersistentFlags().BoolVarP(&opts.JSON, "json", "j", false, "write machine-readable JSON")
	cmd.PersistentFlags().StringVar(&opts.APIKey, "api-key", "", "API key override")
	cmd.PersistentFlags().StringVar(&opts.APIURL, "api-url", "", "API base URL or origin")
	cmd.PersistentFlags().StringVarP(&opts.SpaceID, "space", "s", "", "space id/name override")
	addCreateLinkFlags(cmd, createOptions)

	cmd.AddCommand(
		newAuthCommand(opts),
		newCollectionCommand(opts),
		newCollectionsCommand(opts),
		newConfigCommand(opts),
		newContextCommand(opts),
		newDomainCommand(opts),
		newDomainsCommand(opts),
		newLinksCommand(opts),
		newPlaygroundCommand(opts),
		newSpaceCommand(opts),
		newSpecCommand(opts),
		newStatusCommand(opts),
		newTUICommand(opts),
		newVersionCommand(opts),
	)
	cmd.Version = opts.Version
	return cmd
}

func newVersionCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Zeb version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.JSON {
				return writeJSON(map[string]string{"version": opts.Version})
			}
			fmt.Printf("zeb version %s\n", opts.Version)
			return nil
		},
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func heading(text string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render(text)
}

func expandRootURLShorthand(args []string) []string {
	if firstRootPositionalIsURL(args) {
		expanded := make([]string, 0, len(args)+2)
		expanded = append(expanded, "links", "create")
		expanded = append(expanded, args...)
		return expanded
	}
	return args
}

func firstRootPositionalIsURL(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return i+1 < len(args) && looksLikeHTTPURL(args[i+1])
		}
		if strings.HasPrefix(arg, "-") {
			if flagConsumesValue(arg) && !strings.Contains(arg, "=") {
				i++
			}
			continue
		}
		return looksLikeHTTPURL(arg)
	}
	return false
}

func flagConsumesValue(flag string) bool {
	switch flag {
	case "--api-key", "--api-url", "--space",
		"--collection", "-c", "--domain", "-d", "--path",
		"--short-code", "--namespace", "--title", "-t", "-s":
		return true
	default:
		return false
	}
}

func looksLikeHTTPURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
