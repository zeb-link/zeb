// Package cli wires Cobra commands for the zeb executable.
// Feature commands should stay thin: parse flags, resolve config, call the
// API/client layer, then format output.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zeb-link/zeb/internal/api"
)

// errAlreadyReported signals that a command has already written its complete
// output (its JSON body or human rows already carry the failure detail) and
// only needs a non-zero exit — Execute must not print an additional line or
// JSON object on top of it. Keeps stdout to exactly one document in --json mode.
var errAlreadyReported = errors.New("already reported")

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
	cmd.SetArgs(expandRootURLShorthand(cmd, os.Args[1:]))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cmd.ExecuteContext(ctx); err != nil {
		if errors.Is(err, errAlreadyReported) {
			os.Exit(1)
		}
		// opts.JSON is set once flags are parsed, but a usage error like an
		// unknown command fails before that — fall back to the raw args so
		// --json/--agent is honored on every failure path.
		if opts.JSON || jsonRequestedIn(os.Args[1:]) {
			writeJSONError(err)
		} else {
			fmt.Fprintln(os.Stderr, "zeb:", err)
		}
		os.Exit(1)
	}
}

// jsonRequestedIn reports whether the machine-output flag (or its --agent
// alias) appears in the raw arguments, stopping at `--` so a literal
// positional after it never counts.
func jsonRequestedIn(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--json" || arg == "--agent" || arg == "-j" {
			return true
		}
	}
	return false
}

// writeJSONError renders a failed command as a single JSON document on stdout,
// so a `zeb … --json` (or --agent) pipeline always parses — success shape or
// {"error":{…}} — and the exit code signals which. API failures keep their
// machine-readable code; validation/transport errors carry the message.
func writeJSONError(err error) {
	errObj := map[string]any{"code": "", "message": err.Error()}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		errObj["code"] = apiErr.Code
		errObj["message"] = apiErr.Message
		if apiErr.Status != 0 {
			errObj["status"] = apiErr.Status
		}
		if len(apiErr.Details) > 0 {
			errObj["details"] = apiErr.Details
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(map[string]any{"error": errObj})
}

func newRootCommand(opts *rootOptions) *cobra.Command {
	createOptions := &createLinksOptions{}
	cmd := &cobra.Command{
		Use:   "zeb [url...]",
		Short: "Manage Zebra spaces from the terminal",
		Long:  "Manage Zebra spaces from the terminal.\n\nRun `zeb <url>` to create a short link, or use subcommands for listing links, choosing domains, and choosing collections.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runCreateLinks(cmd, opts, createOptions, args)
		},
		// Errors are printed once by Execute; a failed command should not
		// bury its message under the full usage block.
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVarP(&opts.JSON, "json", "j", false, "write machine-readable JSON (both success and errors are JSON on stdout; exit code signals failure)")
	// `--agent` is a discoverable alias for `--json`: same machine contract,
	// named for the audience that most wants it. ORed into opts.JSON before any
	// command runs.
	var agent bool
	cmd.PersistentFlags().BoolVar(&agent, "agent", false, "alias for --json: machine-readable output for agents and scripts")
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if agent {
			opts.JSON = true
		}
	}
	cmd.PersistentFlags().StringVar(&opts.APIKey, "api-key", "", "API key override")
	cmd.PersistentFlags().StringVar(&opts.APIURL, "api-url", "", "API base URL or origin")
	// Owner-only escape hatch for pointing at a local Core; every user path
	// goes to the built-in production API. Hidden from all help output on
	// purpose — do not document it.
	_ = cmd.PersistentFlags().MarkHidden("api-url")
	cmd.PersistentFlags().StringVarP(&opts.SpaceID, "space", "s", "", "space id/name override")
	addCreateLinkFlags(cmd, createOptions)

	cmd.AddCommand(
		newAuthCommand(opts),
		// `zeb login` is the front-door spelling; `zeb auth login` stays for
		// symmetry with logout/whoami.
		newLoginCommand(opts),
		newCollectionCommand(opts),
		newCollectionsCommand(opts),
		newConfigCommand(opts),
		newContextCommand(opts),
		newDomainCommand(opts),
		newDomainsCommand(opts),
		newHealthCommand(opts),
		newLinksCommand(opts),
		newQrCommand(opts),
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

func expandRootURLShorthand(cmd *cobra.Command, args []string) []string {
	if firstRootPositionalIsURL(cmd, args) {
		expanded := make([]string, 0, len(args)+2)
		expanded = append(expanded, "links", "create")
		expanded = append(expanded, args...)
		return expanded
	}
	return args
}

func firstRootPositionalIsURL(cmd *cobra.Command, args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return i+1 < len(args) && looksLikeHTTPURL(args[i+1])
		}
		if strings.HasPrefix(arg, "-") {
			if flagConsumesValue(cmd, arg) {
				i++
			}
			continue
		}
		return looksLikeHTTPURL(arg)
	}
	return false
}

// flagConsumesValue reports whether `arg` names a root-level flag that takes
// its value from the NEXT argument. Derived from the command's registered
// flag sets so new create flags can never silently break URL-shorthand
// detection by missing a hardcoded list. `--flag=value` and `-x=value`
// carry their value inline and never consume; boolean flags never consume.
func flagConsumesValue(cmd *cobra.Command, arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	lookup := rootFlagFor(cmd, arg)
	if lookup == nil {
		return false
	}
	return lookup.Value.Type() != "bool"
}

func rootFlagFor(cmd *cobra.Command, arg string) *pflag.Flag {
	// Local create flags and persistent root flags live in separate sets
	// until cobra merges them during execution — check both.
	lookup := func(find func(*pflag.FlagSet) *pflag.Flag) *pflag.Flag {
		if flag := find(cmd.Flags()); flag != nil {
			return flag
		}
		return find(cmd.PersistentFlags())
	}
	if name, ok := strings.CutPrefix(arg, "--"); ok {
		return lookup(func(flags *pflag.FlagSet) *pflag.Flag { return flags.Lookup(name) })
	}
	// Shorthand: only the bare `-x` form consumes the next argument.
	if !strings.HasPrefix(arg, "-") || len(arg) != 2 {
		return nil
	}
	return lookup(func(flags *pflag.FlagSet) *pflag.Flag { return flags.ShorthandLookup(arg[1:2]) })
}

func looksLikeHTTPURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
