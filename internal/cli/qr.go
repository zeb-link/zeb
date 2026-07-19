// QR commands expose a link's QR code: its stable public image URLs (the
// key-free CDN files an <img> tag or third party embeds) and an on-demand
// render for saving a PNG/SVG to disk. Authoring QR designs (variant styles,
// signals) stays in the studio — the CLI reads and exports, it doesn't style.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

func newQrCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qr <link-id>",
		Short: "Get a link's QR code: public URLs or a downloaded image",
		Long: "Get a link's QR code.\n\n" +
			"By default this prints the design's stable public image URLs — key-free, " +
			"CDN-served, safe to embed in an <img> tag or hand to a third party. Pass " +
			"--download to save the rendered image (PNG or SVG) to a file instead.\n\n" +
			"QR designs (variant styles and signals) are authored in the studio; the " +
			"CLI reads variants (`zeb qr variants <link-id>`) and exports/renders them.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQr(cmd, root, args[0])
		},
	}
	cmd.Flags().String("download", "", "save the rendered image to this file instead of printing URLs")
	cmd.Flags().String("format", "", "image format for --download: png (default) or svg; inferred from the file extension when omitted")
	cmd.Flags().Int("size", 0, "PNG edge length in pixels for --download (ignored for SVG)")
	cmd.Flags().String("variant", "", "render/export a named variant id instead of the link's effective default")
	cmd.AddCommand(newQrVariantsCommand(root))
	return cmd
}

func runQr(cmd *cobra.Command, root *rootOptions, linkID string) error {
	if err := validateLinkID(linkID); err != nil {
		return err
	}
	variant, _ := cmd.Flags().GetString("variant")
	download, _ := cmd.Flags().GetString("download")

	ctx, err := resolveAPIContext(cmd.Context(), root)
	if err != nil {
		return err
	}

	if download != "" {
		return runQrDownload(cmd, root, ctx, linkID, download, variant)
	}
	return runQrExport(root, ctx, cmd, linkID, variant)
}

// runQrExport prints (or JSON-emits) the design's stable public image URLs.
func runQrExport(root *rootOptions, ctx apiContext, cmd *cobra.Command, linkID string, variant string) error {
	response, err := ctx.Client.ExportQr(cmd.Context(), ctx.SpaceID, linkID, variant)
	if err != nil {
		return err
	}
	if root.JSON {
		return writeJSON(response)
	}
	fmt.Println(heading("QR public URLs"))
	fmt.Printf("%s %s\n", theme.MutedText.Render("Link:"), theme.Command.Render(linkID))
	design := "effective default"
	if response.Export.VariantName != nil && *response.Export.VariantName != "" {
		design = *response.Export.VariantName
	}
	fmt.Printf("%s %s\n\n", theme.MutedText.Render("Design:"), theme.Command.Render(design))
	fmt.Printf("PNG: %s\n", response.Export.ImageUrls.PNG)
	fmt.Printf("SVG: %s\n", response.Export.ImageUrls.SVG)
	fmt.Printf("\n%s\n", theme.MutedText.Render("Stable and CDN-served — embed directly; saving the design updates these files in place."))
	return nil
}

// runQrDownload saves the rendered image bytes to a file. Format comes from
// --format, else the file extension, else PNG.
func runQrDownload(cmd *cobra.Command, root *rootOptions, ctx apiContext, linkID string, path string, variant string) error {
	format, _ := cmd.Flags().GetString("format")
	size, _ := cmd.Flags().GetInt("size")
	if format == "" {
		format = formatFromExtension(path)
	}
	if format != "" && format != "png" && format != "svg" {
		return fmt.Errorf("--format must be png or svg")
	}
	data, contentType, err := ctx.Client.GetQrImage(cmd.Context(), ctx.SpaceID, linkID, api.QrImageOptions{
		Format:  format,
		Size:    size,
		Variant: variant,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	if root.JSON {
		return writeJSON(map[string]any{
			"path":        path,
			"bytes":       len(data),
			"contentType": contentType,
		})
	}
	fmt.Printf("%s %s %s\n",
		createdHeadingStyle.Render("Saved"),
		theme.Command.Render(path),
		theme.MutedText.Render(fmt.Sprintf("(%d bytes, %s)", len(data), contentType)),
	)
	return nil
}

func formatFromExtension(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".svg") {
		return "svg"
	}
	return "png"
}

func newQrVariantsCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "variants <link-id>",
		Short: "List a link's QR variants (named designs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			linkID := args[0]
			if err := validateLinkID(linkID); err != nil {
				return err
			}
			ctx, err := resolveAPIContext(cmd.Context(), root)
			if err != nil {
				return err
			}
			response, err := ctx.Client.ListQrVariants(cmd.Context(), ctx.SpaceID, linkID)
			if err != nil {
				return err
			}
			if root.JSON {
				return writeJSON(response)
			}
			printQrVariants(response.QrVariants)
			return nil
		},
	}
}

func printQrVariants(variants []api.QrVariant) {
	fmt.Println(heading("QR variants"))
	if len(variants) == 0 {
		fmt.Println(theme.MutedText.Render("No named variants — the link renders the space default (or stock) look."))
		fmt.Printf("%s\n", theme.MutedText.Render("Get its public URLs with `zeb qr <link-id>` or save it with `zeb qr <link-id> --download qr.png`."))
		return
	}
	for idx, variant := range variants {
		if idx > 0 {
			fmt.Println()
		}
		label := variant.Name
		if idx == 0 {
			label += theme.MutedText.Render("  (effective default)")
		}
		fmt.Printf("%s %s\n", activeDotStyle.Render("●"), linkShortStyle.Render(label))
		fmt.Printf("  %s\n", theme.MutedText.Render(variant.ID))
		if variant.ImageUrls != nil {
			fmt.Printf("  %s %s\n", theme.MutedText.Render("PNG:"), variant.ImageUrls.PNG)
		}
	}
}
