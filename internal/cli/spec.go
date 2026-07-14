// Spec commands keep a local OpenAPI snapshot for drift checks and review.
// The snapshot is a build input, not a replacement for the live Core endpoint.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/zeb-link/zeb/internal/config"
	"github.com/spf13/cobra"
)

// specRelPath is where the snapshot lives inside the zeb repo. Sync resolves
// it against the repo root (found via go.mod), not the current directory, so
// the installed binary can run from anywhere.
const specRelPath = "internal/openapi/openapi.json"

func newSpecCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Manage the local OpenAPI snapshot",
	}
	cmd.AddCommand(newSpecSyncCommand(root), newSpecPathCommand(root))
	return cmd
}

func newSpecSyncCommand(root *rootOptions) *cobra.Command {
	var url string
	var output string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Download the Core OpenAPI spec",
		Long:  "Download the Core OpenAPI spec into the zeb repo's snapshot.\n\nThe URL defaults to the configured API (see `zeb status`) plus /openapi.json.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if output == "" {
				resolved, err := defaultSpecOutputPath()
				if err != nil {
					return err
				}
				output = resolved
			}
			specURL := url
			if specURL == "" {
				apiURL, err := config.ResolveAPIURL(root.APIURL)
				if err != nil {
					return err
				}
				specURL = apiURL + "/openapi.json"
			}
			data, err := fetchSpec(cmd.Context(), specURL)
			if err != nil {
				return err
			}
			var parsed any
			if err := json.Unmarshal(data, &parsed); err != nil {
				return fmt.Errorf("downloaded spec is not valid JSON: %w", err)
			}
			formatted, err := json.MarshalIndent(parsed, "", "  ")
			if err != nil {
				return err
			}
			formatted = append(formatted, '\n')
			if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(output, formatted, 0o644); err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]string{"url": specURL, "output": output})
			}
			fmt.Printf("Synced %s\n", output)
			fmt.Printf("Source: %s\n", specURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "OpenAPI JSON URL; defaults to the configured API + /openapi.json")
	cmd.Flags().StringVarP(&output, "output", "o", "", "file to write (defaults to the snapshot inside the zeb repo)")
	return cmd
}

func newSpecPathCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the local OpenAPI snapshot path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := defaultSpecOutputPath()
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(map[string]string{"path": path})
			}
			fmt.Println(path)
			return nil
		},
	}
}

// defaultSpecOutputPath anchors the snapshot to the zeb repo root by walking
// up from the working directory until a go.mod appears. Outside the repo the
// caller must pass --output explicitly — silently writing an
// internal/openapi/ tree into a random cwd is worse than an error.
func defaultSpecOutputPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, specRelPath), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside the zeb repo; pass --output <file> to write the snapshot elsewhere")
		}
		dir = parent
	}
}

func fetchSpec(ctx context.Context, url string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch OpenAPI spec: %w", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("could not fetch OpenAPI spec: %s returned HTTP %d", url, res.StatusCode)
	}
	return data, nil
}
