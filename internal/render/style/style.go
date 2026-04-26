// Package style centralises the Sevro CLI's visual language: colors,
// badges, dividers, and section formatters. Renderers compose these
// styles; they never reach for raw ANSI codes.
//
// All styles auto-degrade based on terminal capability: when output is
// not a TTY, when NO_COLOR is set, or when the user passes --no-color,
// every Style here renders as its plain-text equivalent.
package style

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme bundles the entire palette + reusable styles. Construct one
// per render invocation via NewTheme so colour-vs-plain is a single
// decision, not threaded through every helper.
type Theme struct {
	UseColor bool

	// Brand
	Brand        lipgloss.Style
	Tagline      lipgloss.Style
	HeaderBorder lipgloss.Style

	// Severity badges
	SevHigh lipgloss.Style
	SevMed  lipgloss.Style
	SevLow  lipgloss.Style
	SevInfo lipgloss.Style

	// Confidence
	ConfHigh lipgloss.Style
	ConfMed  lipgloss.Style
	ConfLow  lipgloss.Style

	// Output elements
	Workload    lipgloss.Style
	Title       lipgloss.Style
	Detail      lipgloss.Style
	Savings     lipgloss.Style
	NoSavings   lipgloss.Style
	Muted       lipgloss.Style
	Divider     lipgloss.Style
	Disclosure  lipgloss.Style
	CallToLink  lipgloss.Style
	OK          lipgloss.Style
}

// NewTheme builds a theme. If useColor is false, every style falls back
// to plain text and bold/foreground attributes are no-ops.
//
// When useColor is true, the theme uses its own renderer with the color
// profile pinned to TrueColor so output is consistent regardless of
// what TTY detection says — the CLI's outer layer already gated on
// TTY/NO_COLOR/--no-color before deciding to call NewTheme(true).
func NewTheme(useColor bool) Theme {
	if !useColor {
		return plainTheme()
	}

	// Use our own renderer so tests and pipe-redirected output still
	// emit ANSI when the caller explicitly asked for color. The CLI's
	// outer layer is the source of truth for "should I color?".
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)

	// Adaptive colors: pick a value that reads well on both dark and
	// light backgrounds. Dark variants are tuned for the most common
	// terminal default (dark).
	brand := lipgloss.AdaptiveColor{Light: "#5C2EE5", Dark: "#A78BFA"}
	red := lipgloss.AdaptiveColor{Light: "#C92A2A", Dark: "#FF6B6B"}
	amber := lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#F59E0B"}
	cyan := lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"}
	green := lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#34D399"}
	gray := lipgloss.AdaptiveColor{Light: "#666666", Dark: "#9CA3AF"}
	subtle := lipgloss.AdaptiveColor{Light: "#999999", Dark: "#6B7280"}
	border := lipgloss.AdaptiveColor{Light: "#D4D4D8", Dark: "#3F3F46"}

	badge := func(c lipgloss.TerminalColor) lipgloss.Style {
		return r.NewStyle().
			Foreground(lipgloss.Color("#0F0F0F")).
			Background(c).
			Bold(true).
			Padding(0, 1)
	}

	return Theme{
		UseColor:     true,
		Brand:        r.NewStyle().Foreground(brand).Bold(true),
		Tagline:      r.NewStyle().Foreground(subtle).Italic(true),
		HeaderBorder: r.NewStyle().Foreground(border),

		SevHigh: badge(red),
		SevMed:  badge(amber),
		SevLow:  badge(cyan),
		SevInfo: badge(gray),

		ConfHigh: r.NewStyle().Foreground(green).Bold(true),
		ConfMed:  r.NewStyle().Foreground(amber).Bold(true),
		ConfLow:  r.NewStyle().Foreground(gray).Bold(true),

		Workload:   r.NewStyle().Foreground(brand).Bold(true),
		Title:      r.NewStyle().Bold(true),
		Detail:     r.NewStyle().Foreground(gray),
		Savings:    r.NewStyle().Foreground(green).Bold(true),
		NoSavings:  r.NewStyle().Foreground(subtle),
		Muted:      r.NewStyle().Foreground(subtle),
		Divider:    r.NewStyle().Foreground(border),
		Disclosure: r.NewStyle().Foreground(amber),
		// The hyperlink already renders as clickable in modern terminals
		// (OSC 8); doubling that with `Underline(true)` makes lipgloss
		// emit per-character styling that confuses some terminals.
		CallToLink: r.NewStyle().Foreground(brand).Bold(true),
		OK:         r.NewStyle().Foreground(green).Bold(true),
	}
}

func plainTheme() Theme {
	plain := lipgloss.NewStyle()
	return Theme{
		UseColor:     false,
		Brand:        plain,
		Tagline:      plain,
		HeaderBorder: plain,
		SevHigh:      plain,
		SevMed:       plain,
		SevLow:       plain,
		SevInfo:      plain,
		ConfHigh:     plain,
		ConfMed:      plain,
		ConfLow:      plain,
		Workload:     plain,
		Title:        plain,
		Detail:       plain,
		Savings:      plain,
		NoSavings:    plain,
		Muted:        plain,
		Divider:      plain,
		Disclosure:   plain,
		CallToLink:   plain,
		OK:           plain,
	}
}

// Hyperlink wraps url with an OSC 8 hyperlink escape so modern
// terminals (iTerm2, kitty, WezTerm, Ghostty, VSCode) render it as a
// clickable link. Falls back to plain text when colours are off.
func (t Theme) Hyperlink(label, url string) string {
	if !t.UseColor {
		return fmt.Sprintf("%s (%s)", label, url)
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, label)
}

// Divider returns a horizontal rule the given width.
func (t Theme) DividerLine(width int) string {
	if width <= 0 {
		width = 64
	}
	return t.Divider.Render(repeat("─", width))
}

// SeverityBadge picks the right badge style for a severity string and
// renders the literal label.
func (t Theme) SeverityBadge(sev string) string {
	switch sev {
	case "HIGH":
		return t.SevHigh.Render(" HIGH ")
	case "MED":
		return t.SevMed.Render(" MED  ")
	case "LOW":
		return t.SevLow.Render(" LOW  ")
	default:
		return t.SevInfo.Render(" INFO ")
	}
}

// ConfidenceDots returns a fixed-width visual confidence indicator.
func (t Theme) ConfidenceDots(conf string) string {
	switch conf {
	case "high":
		return t.ConfHigh.Render("●●●") + " " + t.Muted.Render("high")
	case "medium":
		return t.ConfMed.Render("●●") + t.Muted.Render("○") + " " + t.Muted.Render("medium")
	case "low":
		return t.ConfLow.Render("●") + t.Muted.Render("○○") + " " + t.Muted.Render("low")
	default:
		return t.Muted.Render("○○○ unknown")
	}
}

// IsTTY reports whether the given file is connected to a terminal.
// Centralised here so callers don't import isatty everywhere.
func IsTTY(f *os.File) bool {
	return isTTY(f)
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
