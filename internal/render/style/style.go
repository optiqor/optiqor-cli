// Package style centralises the CLI's visual language. Styles
// auto-degrade to plain text when colour is off (non-TTY, NO_COLOR,
// or --no-color).
package style

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// BrandGlyph is a single-rune stand-in for the optiqor logomark.
// Avoids Nerd Font / powerline glyphs that render as boxes on stock CI.
const BrandGlyph = "◐"

// Theme bundles the palette + reusable styles. Construct one per
// render via NewTheme so colour-vs-plain is decided once.
type Theme struct {
	UseColor bool

	Brand        lipgloss.Style
	BrandMark    lipgloss.Style
	Tagline      lipgloss.Style
	HeaderBorder lipgloss.Style

	CardBorder  lipgloss.Style
	BarFilled   lipgloss.Style
	BarEmpty    lipgloss.Style
	BarOverflow lipgloss.Style // ratios > 1 (limit < request, etc.)

	SectionPrimary lipgloss.Style
	SectionBonus   lipgloss.Style
	SectionSubtle  lipgloss.Style

	SevHigh lipgloss.Style
	SevMed  lipgloss.Style
	SevLow  lipgloss.Style
	SevInfo lipgloss.Style

	ConfHigh lipgloss.Style
	ConfMed  lipgloss.Style
	ConfLow  lipgloss.Style

	Workload   lipgloss.Style
	Title      lipgloss.Style
	Detail     lipgloss.Style
	Savings    lipgloss.Style
	BigSavings lipgloss.Style
	NoSavings  lipgloss.Style
	Muted      lipgloss.Style
	Divider    lipgloss.Style
	Disclosure lipgloss.Style
	CallToLink lipgloss.Style
	OK         lipgloss.Style
}

// NewTheme builds a theme. When useColor is true we pin TrueColor on
// our own renderer so pipe-redirected output still emits ANSI — the
// CLI's outer layer already decided "should I color?".
func NewTheme(useColor bool) Theme {
	if !useColor {
		return plainTheme()
	}

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)

	// Adaptive colors: dark variants tuned for the default-dark terminal.
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
		BrandMark:    r.NewStyle().Foreground(cyan).Bold(true),
		Tagline:      r.NewStyle().Foreground(subtle).Italic(true),
		HeaderBorder: r.NewStyle().Foreground(border),

		CardBorder:  r.NewStyle().Foreground(border),
		BarFilled:   r.NewStyle().Foreground(green),
		BarEmpty:    r.NewStyle().Foreground(border),
		BarOverflow: r.NewStyle().Foreground(red),

		SectionPrimary: r.NewStyle().Foreground(brand).Bold(true),
		SectionBonus:   r.NewStyle().Foreground(amber).Bold(true),
		SectionSubtle:  r.NewStyle().Foreground(subtle).Italic(true),

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
		BigSavings: r.NewStyle().Foreground(green).Bold(true),
		NoSavings:  r.NewStyle().Foreground(subtle),
		Muted:      r.NewStyle().Foreground(subtle),
		Divider:    r.NewStyle().Foreground(border),
		Disclosure: r.NewStyle().Foreground(amber),
		// OSC 8 already renders as clickable; adding Underline(true)
		// makes lipgloss emit per-char styling that breaks some terms.
		CallToLink: r.NewStyle().Foreground(brand).Bold(true),
		OK:         r.NewStyle().Foreground(green).Bold(true),
	}
}

func plainTheme() Theme {
	plain := lipgloss.NewStyle()
	return Theme{
		UseColor:       false,
		Brand:          plain,
		BrandMark:      plain,
		Tagline:        plain,
		HeaderBorder:   plain,
		CardBorder:     plain,
		BarFilled:      plain,
		BarEmpty:       plain,
		BarOverflow:    plain,
		SectionPrimary: plain,
		SectionBonus:   plain,
		SectionSubtle:  plain,
		SevHigh:        plain,
		SevMed:         plain,
		SevLow:         plain,
		SevInfo:        plain,
		ConfHigh:       plain,
		ConfMed:        plain,
		ConfLow:        plain,
		Workload:       plain,
		Title:          plain,
		Detail:         plain,
		Savings:        plain,
		BigSavings:     plain,
		NoSavings:      plain,
		Muted:          plain,
		Divider:        plain,
		Disclosure:     plain,
		CallToLink:     plain,
		OK:             plain,
	}
}

// Hyperlink wraps url in an OSC 8 escape (clickable in iTerm2,
// kitty, WezTerm, Ghostty, VSCode). Plain text when colour is off.
func (t Theme) Hyperlink(label, url string) string {
	if !t.UseColor {
		return fmt.Sprintf("%s (%s)", label, url)
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, label)
}

func (t Theme) DividerLine(width int) string {
	if width <= 0 {
		width = 64
	}
	return t.Divider.Render(repeat("─", width))
}

// SignalBar renders a have/want ratio bar. Over-saturated (>1) fills
// in BarOverflow so the eye catches it; the magnitude is left to the
// caller's note. Returns a fixed-rune-width string.
func (t Theme) SignalBar(have, want float64, width int) string {
	if width <= 0 {
		width = 20
	}
	if want <= 0 {
		return t.BarEmpty.Render(repeat("░", width))
	}
	ratio := have / want
	if ratio < 0 {
		ratio = 0
	}

	if ratio <= 1 {
		filled := int(ratio*float64(width) + 0.5)
		if filled > width {
			filled = width
		}
		return t.BarFilled.Render(repeat("█", filled)) +
			t.BarEmpty.Render(repeat("░", width-filled))
	}

	return t.BarOverflow.Render(repeat("█", width))
}

// SectionRule renders a labelled divider in heavy hyphens so it
// outranks header/footer dividers. accent colours both label and rule.
func (t Theme) SectionRule(label string, width int, accent lipgloss.Style) string {
	if width <= 0 {
		width = 64
	}
	prefix := "━━ "
	rendered := accent.Render(prefix + label + " ")
	consumed := len([]rune(prefix + label + " "))
	remaining := width - consumed
	if remaining < 4 {
		remaining = 4
	}
	return rendered + accent.Render(repeat("━", remaining))
}

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

// ConfidenceGlyph is the dot-only variant for dense one-liners.
func (t Theme) ConfidenceGlyph(conf string) string {
	switch conf {
	case "high":
		return t.ConfHigh.Render("●●●")
	case "medium":
		return t.ConfMed.Render("●●") + t.Muted.Render("○")
	case "low":
		return t.ConfLow.Render("●") + t.Muted.Render("○○")
	default:
		return t.Muted.Render("○○○")
	}
}

// IsTTY centralises the isatty import.
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
