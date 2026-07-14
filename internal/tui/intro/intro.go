// Package intro renders Zeb launch animations.
// Renderers are pure string functions so they can run inside Bubble Tea or
// print preview frames for non-interactive inspection.
package intro

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zeb-link/zeb/internal/ui/brand"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

const (
	FrameCount = 16
	FrameDelay = 72 * time.Millisecond
)

type Variant struct {
	Name   string
	Slug   string
	render func(frame int) string
}

var variants = []Variant{
	{Name: "block boot", Slug: "block-boot", render: renderBlockBoot},
	{Name: "block scan", Slug: "block-scan", render: renderBlockScan},
	{Name: "block glitch", Slug: "block-glitch", render: renderBlockGlitch},
	{Name: "block pulse", Slug: "block-pulse", render: renderBlockPulse},
	{Name: "block wipe", Slug: "block-wipe", render: renderBlockWipe},
}

var terminalBlockLogo = []string{
	"          ",
	"▄▄▄▄▄▄▄▄▄ ",
	"▀▀▀▀▀████ ",
	"   ▄███▀  ",
	" ▄███▀    ",
	"█████████ ",
	"          ",
}

func Slugs() []string {
	slugs := make([]string, 0, len(variants))
	for _, variant := range variants {
		slugs = append(slugs, variant.Slug)
	}
	return slugs
}

func Variants() []Variant {
	copied := make([]Variant, len(variants))
	copy(copied, variants)
	return copied
}

func RandomVariant() Variant {
	source := rand.New(rand.NewSource(time.Now().UnixNano()))
	return variants[source.Intn(len(variants))]
}

func VariantByName(name string) (Variant, bool) {
	normalized := normalizeName(name)
	for _, variant := range variants {
		if normalizeName(variant.Name) == normalized || normalizeName(variant.Slug) == normalized {
			return variant, true
		}
	}
	return Variant{}, false
}

func PrintPreview(frames int, variantName string) {
	if frames <= 0 {
		frames = FrameCount
	}
	selected := variants
	if variantName != "" && !strings.EqualFold(variantName, "all") {
		if variant, ok := VariantByName(variantName); ok {
			selected = []Variant{variant}
		} else {
			fmt.Printf("unknown intro %q; available: %s\n\n", variantName, strings.Join(Slugs(), ", "))
			return
		}
	}
	for _, variant := range selected {
		for frame := range frames {
			fmt.Printf("%s frame %02d\n%s\n\n", variant.Name, frame, variant.RenderFrame(frame))
		}
	}
}

func (v Variant) RenderFrame(frame int) string {
	if v.render == nil {
		// Zero-value Variant (should not happen via the registry): fall
		// back to the first registered variant.
		v = variants[0]
	}
	return v.render(clampFrame(frame)) + "\n" + variantLabel(v.Name)
}

func (v Variant) RenderCompactFrame(frame int) string {
	frame = clampFrame(frame)
	switch v.Slug {
	case "block-boot":
		return renderCompactBlockBoot(frame) + "\n" + variantLabel(v.Name)
	case "block-scan":
		return renderCompactBlockScan(frame) + "\n" + variantLabel(v.Name)
	case "block-glitch":
		return renderCompactBlockGlitch(frame) + "\n" + variantLabel(v.Name)
	case "block-pulse":
		return renderCompactBlockPulse(frame) + "\n" + variantLabel(v.Name)
	case "block-wipe":
		return renderCompactBlockWipe(frame) + "\n" + variantLabel(v.Name)
	default:
		return renderCompactBlockBoot(frame) + "\n" + variantLabel(v.Name)
	}
}

func clampFrame(frame int) int {
	if frame < 0 {
		return 0
	}
	if frame >= FrameCount {
		return FrameCount - 1
	}
	return frame
}

func renderBlockBoot(frame int) string {
	return renderBlockIntro("   booting Zeb", renderCompactBlockBoot(frame))
}

func renderCompactBlockBoot(frame int) string {
	return renderTerminalBlockLogo(frame, blockBoot)
}

func renderBlockScan(frame int) string {
	return renderBlockIntro("   scanning routes", renderCompactBlockScan(frame))
}

func renderCompactBlockScan(frame int) string {
	return renderTerminalBlockLogo(frame, blockScan)
}

func renderBlockGlitch(frame int) string {
	return renderBlockIntro("   locking signal", renderCompactBlockGlitch(frame))
}

func renderCompactBlockGlitch(frame int) string {
	return renderTerminalBlockLogo(frame, blockGlitch)
}

func renderBlockPulse(frame int) string {
	return renderBlockIntro("   charging Zeb", renderCompactBlockPulse(frame))
}

func renderCompactBlockPulse(frame int) string {
	return renderTerminalBlockLogo(frame, blockPulse)
}

func renderBlockWipe(frame int) string {
	return renderBlockIntro("   drawing shortcut", renderCompactBlockWipe(frame))
}

func renderCompactBlockWipe(frame int) string {
	return renderTerminalBlockLogo(frame, blockWipe)
}

func renderBlockIntro(status string, logo string) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render(status))
	out.WriteString("\n\n")
	out.WriteString(logo)
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

type blockMode int

const (
	blockBoot blockMode = iota
	blockScan
	blockGlitch
	blockPulse
	blockWipe
)

func renderTerminalBlockLogo(frame int, mode blockMode) string {
	total := visibleRuneCount(terminalBlockLogo)
	reveal := total
	if frame < FrameCount-1 {
		reveal = 1 + (total*frame)/(FrameCount-1)
	}
	scanColumn := (frame * 2) - 6
	wipeThreshold := frame + 1

	visible := 0
	var out strings.Builder
	for y, line := range terminalBlockLogo {
		out.WriteString("   ")
		for x, ch := range []rune(line) {
			if ch == ' ' {
				out.WriteByte(' ')
				continue
			}
			visible++
			cell := blockCell{
				ch:      ch,
				x:       x,
				y:       y,
				index:   visible,
				frame:   frame,
				total:   total,
				reveal:  reveal,
				scanX:   scanColumn,
				wipeMax: wipeThreshold,
			}
			out.WriteString(renderBlockCell(cell, mode))
		}
		out.WriteByte('\n')
	}
	return out.String()
}

type blockCell struct {
	ch      rune
	x       int
	y       int
	index   int
	frame   int
	total   int
	reveal  int
	scanX   int
	wipeMax int
}

func renderBlockCell(cell blockCell, mode blockMode) string {
	switch mode {
	case blockBoot:
		if cell.index > cell.reveal {
			return " "
		}
		color := blockBaseColor(cell.ch)
		if cell.index > cell.reveal-4 {
			color = theme.Accent2
		}
		return blockGlyph(cell.ch, color, true)
	case blockScan:
		distance := abs(cell.x - cell.scanX)
		switch {
		case distance == 0:
			return blockGlyph(cell.ch, theme.Accent2, true)
		case distance <= 2:
			return blockGlyph(cell.ch, theme.Accent, true)
		default:
			return blockGlyph(cell.ch, blockBaseColor(cell.ch), true)
		}
	case blockGlitch:
		if cell.frame < FrameCount-3 && (cell.x*7+cell.y*11+cell.frame*5)%19 < 3 {
			return blockGlyph(glitchGlyph(cell), theme.Accent2, true)
		}
		if cell.frame < FrameCount-5 && (cell.x+cell.y+cell.frame)%7 == 0 {
			return blockGlyph(cell.ch, theme.Accent, true)
		}
		return blockGlyph(cell.ch, blockBaseColor(cell.ch), true)
	case blockPulse:
		wave := (cell.x + cell.y + cell.frame) % 8
		switch {
		case wave == 0:
			return blockGlyph(cell.ch, theme.Accent2, true)
		case wave <= 2:
			return blockGlyph(cell.ch, theme.Accent, true)
		case wave == 5:
			return blockGlyph(cell.ch, lipgloss.Color("245"), true)
		default:
			return blockGlyph(cell.ch, blockBaseColor(cell.ch), true)
		}
	case blockWipe:
		if cell.x+cell.y > cell.wipeMax {
			return " "
		}
		color := blockBaseColor(cell.ch)
		if cell.x+cell.y >= cell.wipeMax-1 {
			color = theme.Accent
		}
		return blockGlyph(cell.ch, color, true)
	default:
		return blockGlyph(cell.ch, blockBaseColor(cell.ch), true)
	}
}

func blockBaseColor(ch rune) lipgloss.Color {
	switch ch {
	case '█':
		return theme.White
	case '▄':
		return lipgloss.Color("250")
	case '▀':
		return lipgloss.Color("245")
	default:
		return theme.White
	}
}

func blockGlyph(ch rune, color lipgloss.Color, bold bool) string {
	style := lipgloss.NewStyle().Foreground(color)
	if bold {
		style = style.Bold(true)
	}
	return style.Render(string(ch))
}

func glitchGlyph(cell blockCell) rune {
	glyphs := []rune{'█', '▓', '▒', '░', '▄', '▀'}
	return glyphs[(cell.x+cell.y+cell.frame)%len(glyphs)]
}

func visibleRuneCount(lines []string) int {
	count := 0
	for _, line := range lines {
		for _, ch := range line {
			if ch != ' ' {
				count++
			}
		}
	}
	return count
}

func variantLabel(name string) string {
	return theme.MutedText.Render("   (" + name + ")")
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func normalizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
