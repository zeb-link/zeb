// TUI command opens the interactive terminal surface.
// Rendering and state live under internal/tui so Cobra code stays thin.
package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kerns/zlink-zeb/internal/tui/intro"
	"github.com/kerns/zlink-zeb/internal/tui/intro/gallery"
	"github.com/kerns/zlink-zeb/internal/tui/shell"
	"github.com/spf13/cobra"
)

func newTUICommand(root *rootOptions) *cobra.Command {
	var preview bool
	var previewFrames int
	var introName string
	var galleryMode bool
	var galleryFrame int
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive terminal interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			if galleryMode && preview {
				fmt.Println(gallery.Preview(galleryFrame, 120))
				return nil
			}
			if galleryMode {
				_, err := tea.NewProgram(gallery.New(galleryFrame), tea.WithAltScreen()).Run()
				return err
			}
			if preview {
				if previewFrames <= 0 {
					previewFrames = intro.FrameCount
				}
				selected := introName
				if selected == "" || selected == "random" {
					selected = "all"
				}
				intro.PrintPreview(previewFrames, selected)
				return nil
			}
			variant, err := resolveIntroVariant(introName)
			if err != nil {
				return err
			}
			_, err = tea.NewProgram(shell.New(variant), tea.WithAltScreen()).Run()
			return err
		},
	}
	cmd.Flags().BoolVar(&preview, "preview", false, "print intro frames without opening the TUI")
	cmd.Flags().IntVar(&previewFrames, "frames", intro.FrameCount, "number of intro frames to print with --preview")
	cmd.Flags().StringVar(&introName, "intro", "random", "intro animation: random, all, "+strings.Join(intro.Slugs(), ", "))
	cmd.Flags().BoolVar(&galleryMode, "gallery", false, "open a static intro comparison gallery")
	cmd.Flags().IntVar(&galleryFrame, "gallery-frame", intro.FrameCount/2, "intro frame to render in --gallery")
	return cmd
}

func resolveIntroVariant(name string) (intro.Variant, error) {
	if name == "" || name == "random" {
		return intro.RandomVariant(), nil
	}
	if name == "all" {
		return intro.RandomVariant(), nil
	}
	variant, ok := intro.VariantByName(name)
	if !ok {
		return intro.Variant{}, fmt.Errorf("unknown intro %q; available: random, %s", name, strings.Join(intro.Slugs(), ", "))
	}
	return variant, nil
}
