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
	"github.com/kerns/zlink-zeb/internal/ui/brand"
	"github.com/kerns/zlink-zeb/internal/ui/theme"
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

var puffyLogo = []string{
	" _______ ",
	"(_____  )",
	"     /'/'",
	"   /'/'  ",
	" /'/'___ ",
	"(_______)",
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

var hashBannerLogo = []string{
	"'########:",
	"..... ##::",
	":::: ##:::",
	"::: ##::::",
	":: ##:::::",
	": ##::::::",
	" ########:",
	"........::",
}

// The mark mirrors the Zebra Link logo: three bars on the upper-left, four
// center strokes, and three bars on the lower-right. The circle in the source
// logo is intentionally omitted for terminal use.
var boldZLogo = []string{
	"TTTTTTTTD D D D",
	"TTTTTTTD D D D ",
	"TTTTTTD D D D  ",
	"       D D D D  ",
	"     D D DBBBBBBB",
	"    D D DBBBBBBB ",
	"   D D DBBBBBBB  ",
}

var microZLogo = []string{
	"--- ////",
	"   //// ",
	"  ////  ",
	" ////   ",
	"//// ---",
}

var nanoZLogo = []string{
	"---  ////",
	"    //// ",
	"////  ---",
}

func Names() []string {
	names := make([]string, 0, len(variants))
	for _, variant := range variants {
		names = append(names, variant.Name)
	}
	return names
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
		return renderSlashReveal(frame)
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
	case "puffy":
		return renderCompactPuffy(frame) + "\n" + variantLabel(v.Name)
	case "terminal-block":
		return renderCompactTerminalBlock(frame) + "\n" + variantLabel(v.Name)
	case "hash-banner":
		return renderCompactHashBanner(frame) + "\n" + variantLabel(v.Name)
	case "nano-z":
		return renderCompactNanoZ(frame) + "\n" + variantLabel(v.Name)
	case "micro-z":
		return renderCompactMicroZ(frame) + "\n" + variantLabel(v.Name)
	case "slash-reveal":
		return renderCompactSlashReveal(frame) + "\n" + variantLabel(v.Name)
	case "signal-sweep":
		return renderCompactSignalSweep(frame) + "\n" + variantLabel(v.Name)
	case "zebra-crossing":
		return renderCompactZebraCrossing(frame) + "\n" + variantLabel(v.Name)
	case "bar-build":
		return renderCompactBarBuild(frame) + "\n" + variantLabel(v.Name)
	case "route-weave":
		return renderCompactRouteWeave(frame) + "\n" + variantLabel(v.Name)
	case "bold-pulse":
		return renderCompactBoldPulse(frame) + "\n" + variantLabel(v.Name)
	default:
		return renderCompactSignalSweep(frame) + "\n" + variantLabel(v.Name)
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

func renderPuffy(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   shaping Zeb"))
	out.WriteString("\n\n")
	out.WriteString(renderCompactPuffy(frame))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactPuffy(frame int) string {
	return renderLetterLogo(puffyLogo, frame, true, func(ch rune, x int, y int, index int, frame int) lipgloss.Color {
		switch ch {
		case '/', '\'':
			if (index+frame)%3 == 0 {
				return theme.Accent2
			}
			return theme.Accent
		case '(', ')':
			return lipgloss.Color("245")
		default:
			return theme.White
		}
	})
}

func renderTerminalBlock(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   loading Zeb"))
	out.WriteString("\n\n")
	out.WriteString(renderCompactTerminalBlock(frame))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactTerminalBlock(frame int) string {
	return renderLetterLogo(terminalBlockLogo, frame, true, func(ch rune, x int, y int, index int, frame int) lipgloss.Color {
		switch ch {
		case '█':
			if (x+y+frame)%4 == 0 {
				return theme.Accent
			}
			return theme.White
		case '▄':
			return lipgloss.Color("250")
		case '▀':
			return lipgloss.Color("245")
		default:
			return theme.White
		}
	})
}

func renderHashBanner(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   compiling Zeb"))
	out.WriteString("\n\n")
	out.WriteString(renderCompactHashBanner(frame))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactHashBanner(frame int) string {
	return renderLetterLogo(hashBannerLogo, frame, true, func(ch rune, x int, y int, index int, frame int) lipgloss.Color {
		switch ch {
		case '#':
			if (index+frame)%5 < 2 {
				return theme.Accent
			}
			return theme.White
		case '\'':
			return theme.Accent2
		case '.', ':':
			return lipgloss.Color("240")
		default:
			return lipgloss.Color("245")
		}
	})
}

func renderLetterLogo(lines []string, frame int, animated bool, colorFor func(ch rune, x int, y int, index int, frame int) lipgloss.Color) string {
	total := visibleRuneCount(lines)
	reveal := total
	if animated && frame < FrameCount-1 {
		reveal = 1 + (total*frame)/(FrameCount-1)
	}

	visible := 0
	var out strings.Builder
	for y, line := range lines {
		out.WriteString("   ")
		for x, ch := range []rune(line) {
			if ch == ' ' {
				out.WriteByte(' ')
				continue
			}
			visible++
			if animated && visible > reveal {
				out.WriteByte(' ')
				continue
			}
			color := colorFor(ch, x, y, visible, frame)
			out.WriteString(lipgloss.NewStyle().Bold(true).Foreground(color).Render(string(ch)))
		}
		out.WriteByte('\n')
	}
	return out.String()
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

func renderNanoZ(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   loading Zeb"))
	out.WriteString("\n\n")
	out.WriteString(renderNanoZLogo(frame, true))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactNanoZ(frame int) string {
	return renderNanoZLogo(frame, true)
}

func renderNanoZLogo(frame int, animated bool) string {
	reveal := frame + 3
	var out strings.Builder
	for y, line := range nanoZLogo {
		out.WriteString("   ")
		for x, ch := range []rune(line) {
			if ch == ' ' {
				out.WriteByte(' ')
				continue
			}
			if animated && nanoZRevealOrder(x, y, ch) > reveal {
				out.WriteByte(' ')
				continue
			}
			color := theme.White
			if ch == '/' {
				color = []lipgloss.Color{theme.White, theme.Accent2, theme.Accent}[(x+y+frame)%3]
			} else if (x+y+frame)%4 < 2 {
				color = lipgloss.Color("245")
			}
			out.WriteString(lipgloss.NewStyle().Bold(true).Foreground(color).Render(string(ch)))
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func nanoZRevealOrder(x int, y int, ch rune) int {
	switch {
	case y == 0 && ch == '-':
		return x
	case ch == '/':
		switch y {
		case 0:
			return 3 + (x - 5)
		case 1:
			return 7 + (x - 4)
		default:
			return 11 + x
		}
	case y == 2 && ch == '-':
		return 15 + (x - 6)
	default:
		return x + y
	}
}

func renderMicroZ(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   loading Zeb"))
	out.WriteString("\n\n")
	out.WriteString(renderMicroZLogo(frame, true))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactMicroZ(frame int) string {
	return renderMicroZLogo(frame, true)
}

func renderMicroZLogo(frame int, animated bool) string {
	reveal := frame + 4
	var out strings.Builder
	for y, line := range microZLogo {
		out.WriteString("   ")
		for x, ch := range []rune(line) {
			if ch == ' ' {
				out.WriteByte(' ')
				continue
			}
			if animated && x+y > reveal {
				out.WriteByte(' ')
				continue
			}
			color := theme.White
			if ch == '/' {
				palette := []lipgloss.Color{theme.White, theme.Accent2, theme.Accent}
				color = palette[(x+y+frame)%len(palette)]
			} else if (x+y+frame)%4 < 2 {
				color = lipgloss.Color("245")
			}
			out.WriteString(lipgloss.NewStyle().Bold(true).Foreground(color).Render(string(ch)))
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func renderSlashReveal(frame int) string {
	reveal := frame * 2
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   / / / / / / / / / / / /"))
	out.WriteString("\n")
	out.WriteString(renderLogo(func(x int, y int, group logoGroup) string {
		if x+y > reveal+8 {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(".")
		}
		return logoCell(x, y, frame, group)
	}))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactSlashReveal(frame int) string {
	reveal := frame * 2
	return renderLogo(func(x int, y int, group logoGroup) string {
		if x+y > reveal+8 {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(".")
		}
		return logoCell(x, y, frame, group)
	})
}

func renderSignalSweep(frame int) string {
	sweep := frame * 3
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   resolving route graph"))
	out.WriteString("\n\n")
	out.WriteString(renderLogo(func(x int, y int, group logoGroup) string {
		distance := abs((x + y) - sweep)
		switch {
		case distance == 0:
			return logoGlyph(group, theme.Accent2, true)
		case distance <= 3:
			return logoGlyph(group, theme.Accent, true)
		default:
			return logoGlyph(group, groupMutedColor(group), false)
		}
	}))
	out.WriteString("\n")
	out.WriteString(theme.Command.Render("   zeb.link") + theme.MutedText.Render("  ->  ") + theme.MutedText.Render("ready"))
	return out.String()
}

func renderCompactSignalSweep(frame int) string {
	sweep := frame * 3
	return renderLogo(func(x int, y int, group logoGroup) string {
		distance := abs((x + y) - sweep)
		switch {
		case distance == 0:
			return logoGlyph(group, theme.Accent2, true)
		case distance <= 3:
			return logoGlyph(group, theme.Accent, true)
		default:
			return logoGlyph(group, groupMutedColor(group), false)
		}
	})
}

func renderZebraCrossing(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   crossing into command mode"))
	out.WriteString("\n\n")
	out.WriteString(renderCrossingLogo(frame))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactZebraCrossing(frame int) string {
	return renderCrossingLogo(frame)
}

func renderCrossingLogo(frame int) string {
	var out strings.Builder
	for y, line := range boldZLogo {
		out.WriteString("   ")
		for x, ch := range []rune(line) {
			if group, ok := logoGroupFromRune(ch); ok {
				color := theme.Accent
				if (x+y+frame)%5 < 2 {
					color = theme.White
				}
				out.WriteString(logoGlyph(group, color, true))
				continue
			}
			out.WriteByte(' ')
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func renderBarBuild(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   assembling Zebra mark"))
	out.WriteString("\n\n")
	out.WriteString(renderLogo(func(x int, y int, group logoGroup) string {
		threshold := frame
		if group == groupDiagonal {
			threshold -= 3
		}
		if group == groupBottom {
			threshold -= 5
		}
		if threshold < y/2 {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("·")
		}
		return logoCell(x, y, frame, group)
	}))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   three bars, four strokes, three bars"))
	return out.String()
}

func renderCompactBarBuild(frame int) string {
	return renderLogo(func(x int, y int, group logoGroup) string {
		threshold := frame
		if group == groupDiagonal {
			threshold -= 3
		}
		if group == groupBottom {
			threshold -= 5
		}
		if threshold < y/2 {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("·")
		}
		return logoCell(x, y, frame, group)
	})
}

func renderRouteWeave(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   weaving domains and collections"))
	out.WriteString("\n\n")
	out.WriteString(renderLogo(func(x int, y int, group logoGroup) string {
		if (x+y+frame)%6 == 0 {
			return logoGlyph(group, theme.Accent2, true)
		}
		return logoCell(x, y, frame, group)
	}))
	out.WriteString("\n")
	offset := frame % 12
	out.WriteString(theme.MutedText.Render("   " + strings.Repeat(" ", offset)))
	out.WriteString(theme.Command.Render("domain") + theme.MutedText.Render(" -> ") + theme.Command.Render("collection") + theme.MutedText.Render(" -> ") + theme.Command.Render("link"))
	return out.String()
}

func renderCompactRouteWeave(frame int) string {
	return renderLogo(func(x int, y int, group logoGroup) string {
		if (x+y+frame)%6 == 0 {
			return logoGlyph(group, theme.Accent2, true)
		}
		return logoCell(x, y, frame, group)
	})
}

func renderBoldPulse(frame int) string {
	var out strings.Builder
	out.WriteString(theme.MutedText.Render("   loading Zeb"))
	out.WriteString("\n\n")
	out.WriteString(renderLogo(func(x int, y int, group logoGroup) string {
		if (frame/2)%2 == 0 {
			return logoCell(x, y, frame, group)
		}
		return logoGlyph(group, theme.White, true)
	}))
	out.WriteString("\n")
	out.WriteString(theme.MutedText.Render("   " + brand.Welcome))
	return out.String()
}

func renderCompactBoldPulse(frame int) string {
	return renderLogo(func(x int, y int, group logoGroup) string {
		if (frame/2)%2 == 0 {
			return logoCell(x, y, frame, group)
		}
		return logoGlyph(group, theme.White, true)
	})
}

type logoGroup int

const (
	groupTop logoGroup = iota
	groupDiagonal
	groupBottom
)

func renderLogo(style func(x int, y int, group logoGroup) string) string {
	var out strings.Builder
	for y, line := range boldZLogo {
		out.WriteString("   ")
		for x, ch := range []rune(line) {
			group, ok := logoGroupFromRune(ch)
			if !ok {
				out.WriteByte(' ')
				continue
			}
			out.WriteString(style(x, y, group))
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func logoGroupFromRune(ch rune) (logoGroup, bool) {
	switch ch {
	case 'T':
		return groupTop, true
	case 'D':
		return groupDiagonal, true
	case 'B':
		return groupBottom, true
	default:
		return groupDiagonal, false
	}
}

func logoCell(x int, y int, frame int, group logoGroup) string {
	color := theme.White
	switch group {
	case groupTop, groupBottom:
		if (x+y+frame)%5 < 2 {
			color = lipgloss.Color("245")
		}
	case groupDiagonal:
		palette := []lipgloss.Color{theme.White, theme.Accent, theme.Accent2}
		color = palette[(x+y+frame)%len(palette)]
	}
	return logoGlyph(group, color, true)
}

func logoGlyph(group logoGroup, color lipgloss.Color, bold bool) string {
	glyph := "╱"
	switch group {
	case groupTop, groupBottom:
		glyph = "━"
	}
	style := lipgloss.NewStyle().Foreground(color)
	if bold {
		style = style.Bold(true)
	}
	return style.Render(glyph)
}

func groupMutedColor(group logoGroup) lipgloss.Color {
	if group == groupDiagonal {
		return lipgloss.Color("242")
	}
	return lipgloss.Color("238")
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
