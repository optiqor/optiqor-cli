// Package render formats analysis results as styled text or JSON.
//
// **Every renderer must include the ±40% accuracy disclosure.** Removing
// it is a hard rule violation — see ../../CLAUDE.md. The disclosure is
// what makes the CLI a trustworthy funnel: we never overpromise.
//
// Layout philosophy: cost is the headline product, security findings
// are a bonus side-effect of parsing. Renderers therefore split
// findings by [rules.Category] and present them as two distinct
// sections — cost first with full detail, security after as compact
// one-liners — so a user scanning the output sees the value prop in
// the first screenful.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/optiqor/optiqor-cli/internal/render/style"
	"github.com/optiqor/optiqor-cli/pkg/htmlrender"
	"github.com/optiqor/optiqor-cli/pkg/rules"
)

// Brand strings used in the header banner.
const (
	BrandName    = "optiqor"
	BrandTagline = "Helm chart cost optimization · security as a bonus"
	GetURL       = "https://optiqor.dev/get"
)

// Layout constants. Centralised so callers don't sprinkle magic
// numbers and so visual tweaks happen in one place.
const (
	defaultWidth     = 78
	contentIndent    = "  "
	findingIndent    = "    "
	monthsPerYear    = 12
	annualTeaserMin  = 1_00 // show annual projection only above $1/mo savings
	bonusSectionName = "Security findings"
	costSectionName  = "Cost optimizations"

	// cardMinInner is the smallest interior width we'll render a
	// boxed cost finding at. Below this the layout breaks down; we
	// fall back to a flat (un-boxed) rendering on very narrow
	// terminals.
	cardMinInner = 50

	// signalBarWidth is the rune count of the request/limit bar
	// inside a card. Wide enough to be expressive, narrow enough to
	// leave room for the value labels.
	signalBarWidth = 24
)

// Report is the renderer-facing view of an analysis run.
type Report struct {
	Source    string          `json:"source"` // path or label of the input
	Workloads int             `json:"workloads_analyzed"`
	Findings  []rules.Finding `json:"findings"`
}

// Options controls how a Report is rendered. Callers (cmd/optiqor/main.go)
// detect TTY + NO_COLOR + --no-color and set Color accordingly.
type Options struct {
	Color bool // false → plain ASCII, no ANSI; true → branded styled output
	Width int  // terminal width; 0 → defaultWidth

	// Roast swaps the brand tagline and footer quip for the playful
	// `--roast` variants. Findings themselves are roasted upstream
	// (see internal/roast); the renderer only reads the strings here
	// so it stays unaware of where the roast titles came from.
	Roast bool

	// RoastTagline / RoastFooter override the default tagline and
	// footer when Roast is true. Empty values fall back to the
	// non-roast copy. Callers that don't need to override leave them
	// empty; tests use them to assert the wiring without depending
	// on the internal/roast package.
	RoastTagline string
	RoastFooter  string
}

// MonthlySavingsUSDCents totals the predicted savings across findings.
func (r Report) MonthlySavingsUSDCents() int64 {
	var sum int64
	for _, f := range r.Findings {
		sum += f.MonthlyUSDCents
	}
	return sum
}

// Text writes the styled human-readable report. The output is split
// into a branded header, an executive summary, a "Cost optimizations"
// section (full detail), a "Security findings (bonus)" section
// (compact one-liners), and a footer with the accuracy disclosure
// and agent CTA.
func Text(w io.Writer, r Report, opts Options) error {
	t := style.NewTheme(opts.Color)
	width := opts.Width
	if width <= 0 {
		width = defaultWidth
	}

	cost, security := splitByCategory(r.Findings)

	var b strings.Builder
	writeHeader(&b, t, width, opts)
	writeSummary(&b, t, r, len(cost), len(security))

	if len(cost) == 0 && len(security) == 0 {
		fmt.Fprintf(&b, "\n%s%s\n\n", contentIndent, t.OK.Render("✓ Clean. No findings."))
		writeFooter(&b, t, width, 0, opts)
		_, err := io.WriteString(w, b.String())
		return err
	}

	if len(cost) > 0 {
		writeCostSection(&b, t, width, sortCostForDisplay(cost))
	}
	if len(security) > 0 {
		writeSecuritySection(&b, t, width, security)
	}

	writeFooter(&b, t, width, r.MonthlySavingsUSDCents(), opts)
	_, err := io.WriteString(w, b.String())
	return err
}

// sortCostForDisplay reorders cost findings so the biggest dollar
// impact leads. The engine's stable sort (workload → severity) is
// excellent for diffs and audit, but it buries high-savings findings
// behind alphabetically-earlier workloads. Display order:
//
//  1. findings with monthly savings, highest USD first
//  2. then findings with no dollar estimate, by severity desc
//  3. ties broken by workload, then detector ID — both stable.
//
// Returns a new slice; the caller's input is not mutated.
func sortCostForDisplay(in []rules.Finding) []rules.Finding {
	out := make([]rules.Finding, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.MonthlyUSDCents > 0) != (b.MonthlyUSDCents > 0) {
			return a.MonthlyUSDCents > b.MonthlyUSDCents
		}
		if a.MonthlyUSDCents != b.MonthlyUSDCents {
			return a.MonthlyUSDCents > b.MonthlyUSDCents
		}
		if a.Severity != b.Severity {
			return severityRank(a.Severity) > severityRank(b.Severity)
		}
		if a.Workload != b.Workload {
			return a.Workload < b.Workload
		}
		return a.DetectorID < b.DetectorID
	})
	return out
}

func severityRank(s rules.Severity) int {
	switch s {
	case rules.SeverityHigh:
		return 3
	case rules.SeverityMed:
		return 2
	case rules.SeverityLow:
		return 1
	}
	return 0
}

// splitByCategory partitions findings while preserving the order
// established by rules.Run (workload → severity → detector ID).
// Findings without a Category fall back to the cost section so they
// remain visible — a defensive default for any custom detector that
// forgets to declare one.
func splitByCategory(findings []rules.Finding) (cost, security []rules.Finding) {
	cost = make([]rules.Finding, 0, len(findings))
	security = make([]rules.Finding, 0, len(findings))
	for _, f := range findings {
		switch f.Category {
		case rules.CategorySecurity:
			security = append(security, f)
		default:
			cost = append(cost, f)
		}
	}
	return cost, security
}

func writeHeader(b *strings.Builder, t style.Theme, width int, opts Options) {
	div := t.DividerLine(width)
	mark := t.BrandMark.Render(style.BrandGlyph)
	brand := t.Brand.Render(BrandName)
	taglineText := BrandTagline
	if opts.Roast && opts.RoastTagline != "" {
		taglineText = opts.RoastTagline
	}
	tag := t.Tagline.Render(taglineText)
	fmt.Fprintf(b, "%s\n", div)
	fmt.Fprintf(b, "%s%s  %s\n", contentIndent, mark, brand)
	fmt.Fprintf(b, "%s%s\n", contentIndent, tag)
	fmt.Fprintf(b, "%s\n\n", div)
}

func writeSummary(b *strings.Builder, t style.Theme, r Report, costCount, secCount int) {
	srcLabel := r.Source
	if srcLabel == "" {
		srcLabel = "(stdin)"
	}
	monthly := r.MonthlySavingsUSDCents()

	rows := [][2]string{
		{"Source", t.Workload.Render(srcLabel)},
		{"Workloads", t.Title.Render(plural(r.Workloads, "workload", "workloads")) + t.Muted.Render(" analyzed")},
		{"Cost", costSummaryValue(t, costCount, monthly)},
	}
	if secCount > 0 {
		rows = append(rows, [2]string{
			"Security",
			t.Muted.Render(plural(secCount, "finding", "findings") + " — bonus, surfaced while parsing"),
		})
	}

	labelWidth := 0
	for _, row := range rows {
		if n := len([]rune(row[0])); n > labelWidth {
			labelWidth = n
		}
	}
	for _, row := range rows {
		label := row[0] + strings.Repeat(" ", labelWidth-len([]rune(row[0])))
		fmt.Fprintf(b, "%s%s   %s\n", contentIndent, t.Muted.Render(label), row[1])
	}
}

func costSummaryValue(t style.Theme, count int, monthlyCents int64) string {
	if count == 0 {
		return t.OK.Render("✓ no cost waste detected")
	}
	primary := t.Title.Render(plural(count, "optimization", "optimizations"))
	if monthlyCents <= 0 {
		return primary
	}
	monthly := t.BigSavings.Render("save ~$" + formatCents(monthlyCents) + "/mo")
	suffix := ""
	if monthlyCents >= annualTeaserMin {
		suffix = t.Muted.Render(fmt.Sprintf(" (~$%s/yr)", formatCents(monthlyCents*monthsPerYear)))
	}
	return primary + t.Muted.Render(" · ") + monthly + suffix + t.Muted.Render(" ±40%")
}

func writeCostSection(b *strings.Builder, t style.Theme, width int, findings []rules.Finding) {
	b.WriteString("\n")
	fmt.Fprintf(b, "%s\n\n", t.SectionRule(costSectionName, width, t.SectionPrimary))
	for i, f := range findings {
		writeCostFinding(b, t, f, width)
		if i < len(findings)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
}

// writeCostFinding renders a single cost-section finding as a boxed
// "card" with a header (severity · workload · savings), an optional
// signal bar (request/limit ratio + commentary), the body text, and
// a confidence footer. On very narrow terminals (<cardMinInner inner
// width) it gracefully degrades to a flat layout.
func writeCostFinding(b *strings.Builder, t style.Theme, f rules.Finding, width int) {
	// Card body cells = innerWidth + 4 — see writeCardHeaderRule for
	// the breakdown ("│ " on the left, " │" on the right).
	innerWidth := width - len(contentIndent) - 2
	if innerWidth < cardMinInner {
		writeCostFindingFlat(b, t, f, width)
		return
	}

	title := f.Title
	if title == "" && f.DetectorID != "" {
		title = f.DetectorID
	}

	// Header line: ┌─ HIGH · workload ──────── save ~$29/mo ─┐
	left := fmt.Sprintf(" %s · %s ",
		strings.TrimSpace(string(f.Severity)),
		f.Workload,
	)
	right := ""
	if f.MonthlyUSDCents > 0 {
		right = " save ~$" + formatCents(f.MonthlyUSDCents) + "/mo "
	}
	writeCardHeaderRule(b, t, f.Severity, left, right, innerWidth)

	cardLine(b, t, innerWidth, "")
	cardLine(b, t, innerWidth, t.Title.Render(title))
	if f.Signal != nil {
		cardLine(b, t, innerWidth, "")
		cardLine(b, t, innerWidth, formatSignalLine(t, *f.Signal))
	}
	cardLine(b, t, innerWidth, "")
	for _, line := range wrap(f.Detail, innerWidth-2) {
		cardLine(b, t, innerWidth, t.Detail.Render(line))
	}
	cardLine(b, t, innerWidth, "")
	cardLine(b, t, innerWidth, fmt.Sprintf("%s %s",
		t.Muted.Render("confidence:"),
		t.ConfidenceDots(string(f.Confidence)),
	))

	// Bottom rule
	fmt.Fprintf(b, "%s%s\n", contentIndent,
		t.CardBorder.Render("└"+strings.Repeat("─", innerWidth+2)+"┘"),
	)
}

// writeCostFindingFlat is the fall-back layout used when the terminal
// is too narrow to render a card cleanly. Identical content, no box.
func writeCostFindingFlat(b *strings.Builder, t style.Theme, f rules.Finding, width int) {
	badge := t.SeverityBadge(string(f.Severity))
	wl := t.Workload.Render(f.Workload)
	if f.MonthlyUSDCents > 0 {
		savings := t.Savings.Render("save ~$" + formatCents(f.MonthlyUSDCents) + "/mo")
		fmt.Fprintf(b, "%s%s  %s   %s\n", contentIndent, badge, wl, savings)
	} else {
		fmt.Fprintf(b, "%s%s  %s\n", contentIndent, badge, wl)
	}
	title := f.Title
	if title == "" && f.DetectorID != "" {
		title = f.DetectorID
	}
	fmt.Fprintf(b, "%s%s\n", findingIndent, t.Title.Render(title))
	if f.Signal != nil {
		fmt.Fprintf(b, "%s%s\n", findingIndent, formatSignalLine(t, *f.Signal))
	}
	for _, line := range wrap(f.Detail, width-len(findingIndent)) {
		fmt.Fprintf(b, "%s%s\n", findingIndent, t.Detail.Render(line))
	}
	fmt.Fprintf(b, "%s%s %s\n", findingIndent,
		t.Muted.Render("confidence:"),
		t.ConfidenceDots(string(f.Confidence)),
	)
}

// writeCardHeaderRule writes the top of a card with embedded labels:
//
//	┌─ HIGH · api ─────────────────── save ~$29.20/mo ─┐
//
// The severity word inside the rule is colored by the severity. The
// rule is sized to match the body lines emitted by [cardLine] so left
// and right edges align: those lines occupy
// `"│ " + innerWidth runes + " │"`, which is `innerWidth + 4` cells.
// We mirror that here.
func writeCardHeaderRule(b *strings.Builder, t style.Theme, sev rules.Severity, left, right string, innerWidth int) {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	// Card body cells = innerWidth + 4 ("│ " + content + " │").
	// The corners ("┌", "┐") consume 2 of those; the lead-in dash
	// ("┌─") and the matching trailing dash ("─┐") consume 2 more.
	// What's left is what the labels + gap may use.
	usable := innerWidth // = (innerWidth+4) - 2 corners - 2 lead/trail dashes
	if len(leftRunes)+len(rightRunes) > usable {
		maxLeft := usable - len(rightRunes)
		if maxLeft < 4 {
			maxLeft = 4
		}
		leftRunes = leftRunes[:maxLeft]
	}
	gap := usable - len(leftRunes) - len(rightRunes)
	if gap < 1 {
		gap = 1
	}

	leftStyled := stylizeSeverityWord(t, sev, string(leftRunes))
	gapStr := strings.Repeat("─", gap)
	right = string(rightRunes)

	fmt.Fprintf(b, "%s%s%s%s%s%s\n",
		contentIndent,
		t.CardBorder.Render("┌─"),
		leftStyled,
		t.CardBorder.Render(gapStr),
		t.Savings.Render(right),
		t.CardBorder.Render("─┐"),
	)
}

// stylizeSeverityWord re-renders the leading severity token in the
// card header with the matching badge color, leaving the rest of the
// label (workload name) in plain card-border tone.
func stylizeSeverityWord(t style.Theme, sev rules.Severity, label string) string {
	parts := strings.SplitN(label, " · ", 2)
	if len(parts) != 2 {
		return t.CardBorder.Render(label)
	}
	sevTok := parts[0]
	rest := " · " + parts[1]
	var sevStyle = t.Muted
	switch sev {
	case rules.SeverityHigh:
		sevStyle = t.SevHigh
	case rules.SeverityMed:
		sevStyle = t.SevMed
	case rules.SeverityLow:
		sevStyle = t.SevLow
	}
	// Use a foreground-only re-render here — we don't want the
	// background-coloured badge style inside a header rule, just the
	// matching foreground tone. lipgloss styles are value types, so
	// `.Background(...)` returns a new style; the original badge in
	// the Theme is unaffected.
	return sevStyle.Background(noBackground()).Render(sevTok) +
		t.CardBorder.Render(rest)
}

// noBackground returns a sentinel "no background" so the badge style
// can be reused as a foreground-only accent inside the card header.
// Keeping this inside one helper means the rest of the renderer never
// pokes at lipgloss internals.
func noBackground() lipgloss.TerminalColor { return lipgloss.NoColor{} }

// cardLine writes a single body line of a card, padded to innerWidth
// runes between the side rules.
func cardLine(b *strings.Builder, t style.Theme, innerWidth int, content string) {
	visible := visibleRuneCount(content)
	if visible > innerWidth {
		// Should not happen — callers wrap first — but if it does,
		// truncate visibly so the column alignment stays intact.
		content = truncate(content, innerWidth)
		visible = innerWidth
	}
	pad := strings.Repeat(" ", innerWidth-visible)
	fmt.Fprintf(b, "%s%s %s%s %s\n",
		contentIndent,
		t.CardBorder.Render("│"),
		content,
		pad,
		t.CardBorder.Render("│"),
	)
}

// formatSignalLine renders a Signal as a one-liner sized to fit
// inside a card body row:
//
//	CPU  request 200m ████████░░░░░░░░░░░░░ 1   10x burst
//
// Numbers and labels are width-aware so cards with different
// magnitudes line up vertically when stacked.
func formatSignalLine(t style.Theme, s rules.Signal) string {
	bar := t.SignalBar(s.Have, s.Want, signalBarWidth)
	label := s.Label
	if label == "" {
		label = "ratio"
	}
	note := ""
	if s.Note != "" {
		note = "   " + t.Muted.Render(s.Note)
	}
	return fmt.Sprintf("%s %s %s %s%s",
		t.Muted.Render(padRight(label, 8)),
		t.Detail.Render(padRight(s.HaveDisplay, 6)),
		bar,
		t.Detail.Render(s.WantDisplay),
		note,
	)
}

// padRight returns s padded with spaces to width runes (no truncation
// — short callers use width-aware columns elsewhere).
func padRight(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(r))
}

// visibleRuneCount counts terminal cells of s ignoring ANSI/OSC
// escape sequences. Counts in runes (not bytes) so multi-byte
// glyphs — the box-drawing characters, the bar blocks, and the
// confidence dots — are sized correctly. Assumes one cell per rune
// (no wide-CJK in our copy); good enough for card padding.
func visibleRuneCount(s string) int {
	n := 0
	r := []rune(s)
	for i := 0; i < len(r); {
		if r[i] != 0x1b {
			n++
			i++
			continue
		}
		// CSI: ESC [ … <letter in @..~>
		if i+1 < len(r) && r[i+1] == '[' {
			j := i + 2
			for j < len(r) && (r[j] < '@' || r[j] > '~') {
				j++
			}
			if j < len(r) {
				j++
			}
			i = j
			continue
		}
		// OSC: ESC ] … (ESC \ | BEL)
		if i+1 < len(r) && r[i+1] == ']' {
			j := i + 2
			for j < len(r) {
				if r[j] == 0x07 {
					j++
					break
				}
				if r[j] == 0x1b && j+1 < len(r) && r[j+1] == '\\' {
					j += 2
					break
				}
				j++
			}
			i = j
			continue
		}
		i += 2
	}
	return n
}

func writeSecuritySection(b *strings.Builder, t style.Theme, width int, findings []rules.Finding) {
	label := fmt.Sprintf("%s  (bonus, %d)", bonusSectionName, len(findings))
	b.WriteString("\n")
	fmt.Fprintf(b, "%s\n", t.SectionRule(label, width, t.SectionBonus))
	fmt.Fprintf(b, "%s%s\n\n", contentIndent,
		t.SectionSubtle.Render("Spotted while parsing your chart. Cost is the headline; this is a bonus."),
	)

	maxWorkload := 0
	for _, f := range findings {
		if n := len([]rune(f.Workload)); n > maxWorkload {
			maxWorkload = n
		}
	}
	if maxWorkload > 24 { // keep the column reasonable on narrow terminals
		maxWorkload = 24
	}

	for _, f := range findings {
		writeSecurityFinding(b, t, f, maxWorkload)
	}
	b.WriteString("\n")
	fmt.Fprintf(b, "%s%s\n\n", contentIndent,
		t.Muted.Render("Run `optiqor audit` to focus only on these findings."),
	)
}

func writeSecurityFinding(b *strings.Builder, t style.Theme, f rules.Finding, workloadColWidth int) {
	wl := truncate(f.Workload, workloadColWidth)
	wlPadded := wl + strings.Repeat(" ", workloadColWidth-len([]rune(wl)))
	title := f.Title
	if title == "" && f.DetectorID != "" {
		title = f.DetectorID
	}
	fmt.Fprintf(b, "%s%s  %s   %s   %s\n",
		contentIndent,
		t.SeverityBadge(string(f.Severity)),
		t.Workload.Render(wlPadded),
		t.ConfidenceGlyph(string(f.Confidence)),
		t.Detail.Render(title),
	)
}

func writeFooter(b *strings.Builder, t style.Theme, width int, totalCents int64, opts Options) {
	fmt.Fprintf(b, "%s\n", t.DividerLine(width))
	if totalCents > 0 {
		fmt.Fprintf(b, "%s%s %s   %s\n", contentIndent,
			t.Muted.Render("estimated monthly savings:"),
			t.BigSavings.Render("$"+formatCents(totalCents)+"/mo"),
			t.Muted.Render("(±40%)"),
		)
	}
	// Accuracy disclosure is mandatory and exact (CLAUDE.md hard rule).
	// Roast can add a quip BELOW it; it never replaces it.
	fmt.Fprintf(b, "%s%s\n", contentIndent, t.Disclosure.Render(htmlrender.AccuracyDisclosure))
	linkLabel := t.CallToLink.Render("optiqor.dev/get")
	fmt.Fprintf(b, "%s%s %s\n", contentIndent,
		t.Muted.Render("→ install the agent for exact numbers:"),
		t.Hyperlink(linkLabel, GetURL),
	)
	if opts.Roast && opts.RoastFooter != "" {
		fmt.Fprintf(b, "%s%s\n", contentIndent, t.Tagline.Render(opts.RoastFooter))
	}
}

// JSON writes the report as machine-readable JSON. Always disclosure-
// gated. Never colored — JSON output is for piping. The schema groups
// findings by category so consumers don't have to replicate the
// renderer's split.
func JSON(w io.Writer, r Report) error {
	cost, security := splitByCategory(r.Findings)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		AccuracyDisclosure string          `json:"accuracy_disclosure"`
		Source             string          `json:"source"`
		Workloads          int             `json:"workloads_analyzed"`
		Findings           []rules.Finding `json:"findings"`
		Cost               []rules.Finding `json:"cost_findings"`
		Security           []rules.Finding `json:"security_findings_bonus"`
		MonthlySavingsUSD  float64         `json:"monthly_savings_usd"`
		AnnualSavingsUSD   float64         `json:"annual_savings_usd"`
	}{
		AccuracyDisclosure: htmlrender.AccuracyDisclosure,
		Source:             r.Source,
		Workloads:          r.Workloads,
		Findings:           r.Findings,
		Cost:               cost,
		Security:           security,
		MonthlySavingsUSD:  float64(r.MonthlySavingsUSDCents()) / 100.0,
		AnnualSavingsUSD:   float64(r.MonthlySavingsUSDCents()*monthsPerYear) / 100.0,
	})
}

func formatCents(c int64) string {
	dollars := c / 100
	cents := c % 100
	if cents == 0 {
		return fmt.Sprintf("%d", dollars)
	}
	return fmt.Sprintf("%d.%02d", dollars, cents)
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, pluralForm)
}

// wrap breaks s into lines no wider than width runes. Naïve word wrap;
// good enough for finding details which are sentences, not paragraphs.
func wrap(s string, width int) []string {
	if width <= 0 || len([]rune(s)) <= width {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	words := strings.Fields(s)
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > width {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteString(" ")
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// truncate clamps s to at most width runes, suffixing "…" when it had
// to cut. Width must be >= 1; callers pass column widths so this is
// trivially satisfied in practice.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}
