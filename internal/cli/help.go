// Custom help surfaces for the zeb root command.
//
// The CLI is used most by agents and scripts, so discovery has to be calm and
// obvious: bare `zeb` shows five doors (printMinimalRoot); `zeb help` / `zeb
// --help` shows the full command set grouped by intent (printRootHelp); the
// copy-paste recipes live one command away in `zeb examples`. Per-command help
// (`zeb <cmd> --help`) keeps Cobra's default rendering.
package cli

import (
	"os"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

// resolveTheme picks the light or dark palette for this run and applies it once,
// before any rendering. Order: an explicit ZEB_THEME override, then NO_COLOR
// (background is irrelevant when color is stripped), then a terminal query.
// lipgloss.HasDarkBackground self-guards — it only queries when both stdin and
// stdout are real terminals, so piped/agent invocations never pay for it.
func resolveTheme() {
	isDark := true
	switch os.Getenv("ZEB_THEME") {
	case "light":
		isDark = false
	case "dark":
		isDark = true
	default:
		if os.Getenv("NO_COLOR") == "" {
			isDark = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
		}
	}
	theme.Apply(isDark)
	applyCLIStyles()
}

// applyCLIStyles refreshes the package-local styles in links.go / collection.go
// from the active theme. Called on every theme change (and at init) so those
// surfaces track the light/dark palette. Run after theme.Apply.
func applyCLIStyles() {
	activeDotStyle = theme.GoodText
	inactiveDotStyle = theme.WarnText
	activeStatusStyle = theme.GoodText
	inactiveStatusStyle = theme.WarnText
	createdHeadingStyle = theme.KeyText
	createdDomainStyle = theme.BodyText
	createdCollectionStyle = theme.CollectionText
	linkShortStyle = theme.LinkText
	linkTargetStyle = theme.BodyText
	linkTitleStyle = theme.MutedText
	reachableStyle = theme.GoodText
	unreachableStyle = theme.BadText
	collectionHeadingStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Collection)
	collectionDotStyle = theme.CollectionText
	collectionSmartStyle = theme.MutedText
	collectionActiveStyle = theme.WarnText
}

func init() { applyCLIStyles() }

const tagline = "The shortlink operating system"

// brandHeader renders the identity block — both lines inside one box so they
// left-align and read as a header:
//
//	┌─────────────────────────────────┐
//	│ Zeb from Links by Zebra         │
//	│ The shortlink operating system  │
//	└─────────────────────────────────┘
//
// Only "Zeb" and "Zebra" wear the brand emerald; "from" recedes to the comment
// tone, "Links by" is the medium tone, and the tagline is a faint whisper.
func brandHeader() string {
	l1 := theme.Wordmark.Render("Zeb") + theme.SubtleText.Render(" from ") +
		theme.MutedText.Render("Links by ") + theme.Wordmark.Render("Zebra")
	l2 := theme.SubtleText.Render(tagline)
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Faint).
		Padding(0, 1).
		Render(l1 + "\n" + l2)
	var boxed strings.Builder
	for _, ln := range strings.Split(box, "\n") {
		boxed.WriteString("  " + ln + "\n")
	}
	return strings.TrimRight(boxed.String(), "\n")
}

// commandLine renders one "zeb <rest>            description" row, aligned to
// col. Scale (dark → light): the "zeb " prefix is muted, the command name is
// bright bone (the focal point), the description is ink. Flag rows go sand.
func commandLine(name, desc string, col int) string {
	var styled string
	switch {
	case strings.HasPrefix(name, "zeb "):
		styled = theme.MutedText.Render("zeb ") + theme.CommandText.Render(strings.TrimPrefix(name, "zeb "))
	case strings.HasPrefix(name, "-"):
		styled = theme.FlagText.Render(name)
	default:
		styled = theme.CommandText.Render(name)
	}
	pad := col - utf8.RuneCountInString(name)
	if pad < 1 {
		pad = 1
	}
	return "  " + styled + strings.Repeat(" ", pad) + theme.SubtleText.Render(desc)
}

// printMinimalRoot is what `zeb` with no arguments prints: five doors, not
// eighteen. The full list is one command away at `zeb help`.
func printMinimalRoot() {
	const col = 20
	var b strings.Builder
	b.WriteString("\n" + brandHeader() + "\n\n")
	for _, r := range [][2]string{
		{"zeb <url>", "create a short link"},
		{"zeb links", "browse your links"},
		{"zeb links query", "find links by any condition"},
		{"zeb analytics", "click analytics"},
	} {
		b.WriteString(commandLine(r[0], r[1], col) + "\n")
	}
	b.WriteString("\n")
	for _, r := range [][2]string{
		{"zeb help", "every command"},
		{"zeb examples", "copy-paste cookbook"},
	} {
		b.WriteString(commandLine(r[0], r[1], col) + "\n")
	}
	b.WriteString("\n  " + theme.FaintText.Render("--agent   machine-readable JSON on every command") + "\n")
	lipgloss.Print(b.String())
}

// rootHelpGroup is one titled cluster of commands in the full help.
type rootHelpGroup struct {
	title string
	rows  [][2]string
}

var rootHelpGroups = []rootHelpGroup{
	{"CREATE", [][2]string{
		{"zeb <url>", "make a short link — bare URL is the fast path"},
		{"zeb links create", "create with options (--title, --short-code, …)"},
	}},
	{"FIND", [][2]string{
		{"zeb links", "browse — newest first, or --all"},
		{"zeb links query", "filter by destination, clicks, dates, attribution"},
		{"zeb links lookup", "a short URL or code → its link"},
		{"zeb analytics", "click analytics over the same filters"},
	}},
	{"ORGANIZE", [][2]string{
		{"zeb collections", "list collections"},
		{"zeb collection", "manage a collection + the active collection"},
		{"zeb domains", "list domains"},
		{"zeb qr", "a link's QR image and public URLs"},
	}},
	{"CONTEXT", [][2]string{
		{"zeb context", "pick the active domain + collection"},
		{"zeb space", "the active space"},
		{"zeb config", "inspect / edit local config"},
		{"zeb status", "the current CLI context"},
	}},
	{"ACCOUNT", [][2]string{
		{"zeb login", "log in with your API key"},
		{"zeb auth", "manage authentication"},
	}},
	{"HELP", [][2]string{
		{"zeb examples", "copy-paste cookbook of common commands"},
		{"zeb version", "print the zeb version"},
	}},
}

var rootHelpGlobals = [][2]string{
	{"--json / --agent", "machine-readable JSON on stdout; exit code signals failure"},
	{"-s, --space", "space id/name override"},
	{"--api-key", "API key override"},
}

// printRootHelp renders the full, grouped command set for `zeb help` /
// `zeb --help`.
func printRootHelp() {
	const col = 22
	var b strings.Builder
	b.WriteString("\n" + brandHeader() + "\n\n")
	b.WriteString("  " + theme.MutedText.Render("Create and manage Zebra short links, collections, QR codes, and") + "\n")
	b.WriteString("  " + theme.MutedText.Render("analytics — from your shell or a script.") + "\n\n")

	for _, g := range rootHelpGroups {
		b.WriteString("  " + sectionMark() + theme.Heading.Render(g.title) + "\n")
		for _, r := range g.rows {
			b.WriteString(commandLine(r[0], r[1], col) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("  " + sectionMark() + theme.Heading.Render("GLOBAL") + "\n")
	for _, r := range rootHelpGlobals {
		b.WriteString(commandLine(r[0], r[1], col) + "\n")
	}
	b.WriteString("\n  " + theme.FaintText.Render("zeb <command> --help") + theme.MutedText.Render("   details on any command") + "\n")
	lipgloss.Print(b.String())
}

// sectionMark is the emerald pointer that gives sections shape.
func sectionMark() string { return theme.LinkText.Render("▸ ") }

// section prints a styled command-output heading (blank line, emerald mark,
// sand-bold title), through the profile-aware writer.
func section(title string) {
	lipgloss.Println("\n" + sectionMark() + theme.Heading.Render(title))
}

// field prints an aligned "label   value" row: label muted, value body ink,
// padded so a block of labels lines up.
func field(label, value string, col int) {
	pad := col - utf8.RuneCountInString(label)
	if pad < 1 {
		pad = 1
	}
	lipgloss.Println("  " + theme.MutedText.Render(label) + strings.Repeat(" ", pad) + theme.BodyText.Render(value))
}

// done prints a success confirmation line led by a bold green check.
func done(msg string) {
	check := lipgloss.NewStyle().Bold(true).Foreground(theme.Good).Render("✓")
	lipgloss.Println(check + " " + theme.BodyText.Render(msg))
}

// styleShell syntax-highlights one shell example line, preserving every
// original space so aligned trailing comments stay aligned:
//
//	zeb (faint) · verb (bone) · --flags (sand) · values (ink) · # comment (faint)
func styleShell(line string) string {
	if strings.TrimSpace(line) == "" {
		return ""
	}
	if strings.HasPrefix(strings.TrimLeft(line, " "), "#") {
		return theme.GhostText.Render(line) // whole-line comment — nearly invisible
	}
	code, comment := line, ""
	if idx := strings.Index(line, " #"); idx >= 0 { // " #" avoids catching # in URLs
		code, comment = line[:idx], line[idx:]
	}
	var b strings.Builder
	i, word := 0, 0
	sawFlag := false
	for i < len(code) {
		if code[i] == ' ' {
			j := i
			for j < len(code) && code[j] == ' ' {
				j++
			}
			b.WriteString(code[i:j])
			i = j
			continue
		}
		j := i
		for j < len(code) && code[j] != ' ' {
			j++
		}
		b.WriteString(styleShellWord(code[i:j], word, &sawFlag))
		word++
		i = j
	}
	if comment != "" {
		b.WriteString(theme.GhostText.Render(comment)) // trailing comment — nearly invisible
	}
	return b.String()
}

func styleShellWord(w string, idx int, sawFlag *bool) string {
	switch {
	case w == "zeb":
		return theme.MutedText.Render(w)
	case strings.HasPrefix(w, "-"):
		*sawFlag = true
		return theme.FlagText.Render(w)
	case w == "|" || w == ">" || w == `\` || w == "jq":
		return theme.FaintText.Render(w)
	case strings.Contains(w, "zbrah.link") || strings.Contains(w, "zbra.link"):
		return theme.LinkText.Render(w) // a Zebra short link → emerald
	case !*sawFlag && idx <= 2:
		return theme.CommandText.Render(w) // subcommand verb — brightest
	default:
		return theme.BodyText.Render(w) // values / args
	}
}

// renderExampleTitle styles a cookbook section title: an emerald mark, sand-bold
// prose, and any `code` spans lifted to bone.
func renderExampleTitle(title string) string {
	var b strings.Builder
	for i, part := range strings.Split(title, "`") {
		if i%2 == 1 {
			b.WriteString(theme.KeyText.Render(part))
		} else {
			b.WriteString(theme.Heading.Render(part))
		}
	}
	return sectionMark() + b.String()
}

// installRootHelp wires the custom root help while leaving Cobra's default
// rendering in place for every subcommand's `--help`.
func installRootHelp(root *cobra.Command) {
	base := root.HelpFunc()
	root.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c.HasParent() {
			base(c, args)
			return
		}
		printRootHelp()
	})
}
