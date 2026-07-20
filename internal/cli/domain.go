// Domain commands list domains and manage the active link-creation domain.
// The active domain is a hostname, matching the public API create-link field.
package cli

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/config"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newDomainsCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "domains",
		Short: "List available domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListDomains(cmd.Context(), ctx.SpaceID)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			section("Domains")
			col := 0
			for _, domain := range response.Domains {
				if width := utf8.RuneCountInString(domain.Hostname); width > col {
					col = width
				}
			}
			for _, domain := range response.Domains {
				meta := domain.Type
				if domain.Tier != nil {
					meta += " · " + *domain.Tier
				}
				pad := strings.Repeat(" ", col-utf8.RuneCountInString(domain.Hostname)+2)
				row := "  " + theme.CommandText.Render(domain.Hostname) + pad + theme.MutedText.Render(meta)
				if domain.Hostname == cfg.ActiveDomain {
					row += "  " + theme.GoodText.Render("active")
				}
				lipgloss.Println(row)
			}
			air()
			return nil
		},
	}
}

func newDomainCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Manage active domain context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"activeDomain": nullString(cfg.ActiveDomain)})
			}
			field("Active domain", emptyLabel(cfg.ActiveDomain), 14)
			air()
			return nil
		},
	}
	cmd.AddCommand(newDomainListCommand(root), newDomainUseCommand(root), newDomainClearCommand(root))
	return cmd
}

func newDomainListCommand(root *rootOptions) *cobra.Command {
	cmd := newDomainsCommand(root)
	cmd.Use = "list"
	cmd.Short = "List available domains"
	return cmd
}

func newDomainUseCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "use <hostname>",
		Short: "Set active domain for new links",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := args[0]
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListDomains(cmd.Context(), ctx.SpaceID)
			if err != nil {
				return err
			}
			found := false
			for _, domain := range response.Domains {
				if domain.Hostname == hostname {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("domain %q is not available in the active space", hostname)
			}
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.ActiveDomain = hostname
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]string{"activeDomain": hostname})
			}
			done("Active domain set to " + hostname)
			air()
			return nil
		},
	}
}

func newDomainClearCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear active domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			cfg.ActiveDomain = ""
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]any{"activeDomain": nil})
			}
			done("Active domain cleared; server default will be used.")
			air()
			return nil
		},
	}
}
