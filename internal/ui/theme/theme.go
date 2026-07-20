// Package theme is the single source of the CLI's visual vocabulary. Every
// token carries a light and a dark value; Apply(isDark) resolves the active
// palette for the detected terminal background and rebuilds the derived styles.
//
// Because Apply runs once at startup (before any rendering), component code must
// read colors/styles from this package at RENDER time — never cache a
// theme-derived style in a package-level var, or it will freeze at the default
// (dark) palette and render wrong on a light terminal.
//
// Direction — warm-ink monochrome with the product's feature colors carrying
// meaning: emerald = links + wordmark, violet = collections, amber = caution,
// red = error, a distinct green = success. Dark values match zlink-web's dark
// mode; light values are their readable-on-white counterparts.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Active role colors, resolved by Apply. In light mode the neutral ladder
// inverts (the most-prominent role becomes the darkest ink); feature colors
// darken/saturate so they read on white.
var (
	Ground, Panel, Panel2, Dim, Ghost, Faint, Subtle, Muted, Ink, Bone, Bright, Sand color.Color
	Link, Collection, Good, Warn, Bad                                                color.Color
)

// Active derived styles, rebuilt by Apply.
var (
	Heading, Title, Wordmark                              lipgloss.Style
	BodyText, MutedText, SubtleText, FaintText, GhostText lipgloss.Style
	KeyText, CommandText, FlagText                        lipgloss.Style
	LinkText, CollectionText, GoodText, WarnText, BadText lipgloss.Style
)

// Apply resolves the palette for a light or dark background and rebuilds every
// derived style. Call once at startup before rendering.
func Apply(isDark bool) {
	ld := lipgloss.LightDark(isDark)
	c := func(light, dark string) color.Color { return ld(lipgloss.Color(light), lipgloss.Color(dark)) }

	// Neutrals — warm monochrome. (light, dark)
	Ground = c("#FBF9F4", "#15120D") // page ground (TUI)
	Panel = c("#F1EDE4", "#1C1811")  // raised surface
	Panel2 = c("#E9E4D9", "#232016")
	Dim = c("#C6BEAC", "#39332A")    // rules, inactive
	Ghost = c("#B6AE9E", "#504A3C")  // near-invisible: # comments
	Faint = c("#918978", "#6B6558")  // borders, operators
	Subtle = c("#746D59", "#8B8375") // readable-faint: descriptions, tagline
	Muted = c("#5E5745", "#C6BEAF")  // medium: the zeb prefix, "Links by"
	Ink = c("#454036", "#D8D2C6")    // body text
	Bone = c("#221E17", "#F5F2EB")   // titles, focus, selection
	Bright = c("#16130E", "#FCFAF6") // the most prominent — command names
	Sand = c("#7C6E45", "#E9E4D6")   // flags, section headers

	// Feature + semantic colors.
	Link = c("#0B7A54", "#5DEAB5")       // brand emerald: links + wordmark
	Collection = c("#6A3FC0", "#B99CF0") // violet: collections
	Good = c("#0E7A4E", "#35C98A")       // success / active
	Warn = c("#B07D12", "#E1AF48")       // caution / inactive
	Bad = c("#D23B28", "#EB6552")        // error / unreachable

	fg := func(x color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(x) }
	bold := func(x color.Color) lipgloss.Style { return lipgloss.NewStyle().Bold(true).Foreground(x) }

	Heading = bold(Sand)
	Title = bold(Bone)
	Wordmark = bold(Link)
	BodyText = fg(Ink)
	MutedText = fg(Muted)
	SubtleText = fg(Subtle)
	FaintText = fg(Faint)
	GhostText = fg(Ghost)
	KeyText = bold(Bone)
	CommandText = bold(Bright)
	FlagText = fg(Sand)
	LinkText = fg(Link)
	CollectionText = fg(Collection)
	GoodText = fg(Good)
	WarnText = fg(Warn)
	BadText = fg(Bad)
}

// init sets a dark default so package-load-time rendering (should any occur
// before Execute detects the terminal) is sane.
func init() { Apply(true) }
