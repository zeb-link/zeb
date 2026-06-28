// Spec commands keep a local OpenAPI snapshot for code generation and review.
// The snapshot is a build input for future client generation, not a replacement
// for the live Core endpoint.
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

	"github.com/spf13/cobra"
)

const defaultSpecPath = "internal/openapi/openapi.json"

var defaultSpecURLs = []string{
	"http://localhost:3000/api/v1/openapi.json",
}

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
		RunE: func(cmd *cobra.Command, args []string) error {
			if output == "" {
				output = defaultSpecPath
			}
			usedURL, data, err := fetchSpec(url)
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
				return writeJSON(map[string]string{"url": usedURL, "output": output})
			}
			fmt.Printf("Synced %s\n", output)
			fmt.Printf("Source: %s\n", usedURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "OpenAPI JSON URL; tries local Core defaults when omitted")
	cmd.Flags().StringVarP(&output, "output", "o", defaultSpecPath, "file to write")
	return cmd
}

func newSpecPathCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the local OpenAPI snapshot path",
		RunE: func(cmd *cobra.Command, args []string) error {
			if root.JSON {
				return writeJSON(map[string]string{"path": defaultSpecPath})
			}
			fmt.Println(defaultSpecPath)
			return nil
		},
	}
}

func fetchSpec(explicitURL string) (string, []byte, error) {
	urls := defaultSpecURLs
	if explicitURL != "" {
		urls = []string{explicitURL}
	}
	var lastErr error
	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			return "", nil, err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		data, readErr := io.ReadAll(res.Body)
		closeErr := res.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if closeErr != nil {
			lastErr = closeErr
			continue
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s returned HTTP %d", url, res.StatusCode)
			continue
		}
		return url, data, nil
	}
	return "", nil, fmt.Errorf("could not fetch OpenAPI spec: %w", lastErr)
}
