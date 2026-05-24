// Package htmlrender produces a single self-contained HTML document
// from a set of rules.Finding.
//
// Single source of truth: consumed by both the CLI's `optiqor analyze
// --html report.html` and the Optiqor backend's share-page handlers, so
// the local file and the share page are guaranteed to match byte-for-byte.
//
// Constraints:
//   - Apache-2.0 (auditable OSS).
//   - Zero JS dependencies; CSS is inlined.
//   - Mandatory ±40% accuracy disclosure (CLI hard rule) — exposed as
//     AccuracyDisclosure so both callers reuse the exact bytes.
//   - Deterministic output for a given input (golden-testable).
package htmlrender

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/optiqor/optiqor-cli/pkg/rules"
)

// AccuracyDisclosure is the mandatory CLI string. Kept here so this
// package never drifts from the line every other renderer ships.
const AccuracyDisclosure = "Sandbox accuracy: ±40%. Install the Optiqor agent for exact numbers (optiqor.dev/get)."

// Mode discriminates the accuracy banner.
type Mode int

const (
	// ModeSandbox renders the public ±40% sandbox banner (CLI + share preview).
	ModeSandbox Mode = iota
	// ModeAgent renders the exact-accuracy banner for the paid in-cluster path.
	ModeAgent
)

// Data is the renderer input. htmlrender deliberately does NOT depend
// on the CLI's internal/render.Report because internal/ would block
// backend imports.
type Data struct {
	// Source is the human-readable label for what was analysed
	// (path, share-hash, "demo", etc.).
	Source string
	// Workloads is the workload count the parser produced.
	Workloads int
	// Findings is the full set, split by the renderer into cost-first
	// and security-bonus.
	Findings []rules.Finding
	// ShareURL, when non-empty, renders a copy-link button.
	ShareURL string
	// GeneratedAt is stamped into the footer. Truncated to the minute
	// so re-renders within a minute are byte-equal.
	GeneratedAt time.Time
	// Mode controls which accuracy line renders.
	Mode Mode
	// PageTitle overrides the default <title>. Empty uses
	// "Optiqor analysis — <Source>".
	PageTitle string
}

// Render writes a complete HTML5 document to w. Errors are
// infrastructure-class only (template/exec or writer failure).
func Render(w io.Writer, d Data) error {
	v := buildView(d)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, v); err != nil {
		return fmt.Errorf("htmlrender: %w", err)
	}
	out := bytes.TrimRight(buf.Bytes(), "\n")
	out = append(out, '\n')
	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("htmlrender: write: %w", err)
	}
	return nil
}

// RenderString is a convenience wrapper for callers that want a string
// rather than streaming to an io.Writer.
func RenderString(d Data) (string, error) {
	var buf bytes.Buffer
	if err := Render(&buf, d); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type view struct {
	Title              string
	Source             string
	Workloads          int
	GeneratedAtISO     string
	ShareURL           string
	HasShareURL        bool
	AccuracyDisclosure string
	Mode               string
	Totals             totalsView
	Cost               []findingView
	Security           []findingView
	HasCost            bool
	HasSecurity        bool
	Clean              bool
}

type totalsView struct {
	MonthlyUSD     string
	AnnualUSD      string
	HasSavings     bool
	WorkloadsLabel string
}

type findingView struct {
	Severity      string
	SeverityCls   string
	Workload      string
	Title         string
	Detail        string
	HasSavings    bool
	SavingsUSD    string
	Confidence    string
	ConfidenceCls string
}

func buildView(d Data) view {
	cost, sec := split(d.Findings)
	cost = sortCost(cost)

	var totalCents int64
	for _, f := range d.Findings {
		totalCents += f.MonthlyUSDCents
	}

	v := view{
		Title:              titleOrDefault(d),
		Source:             firstNonEmpty(d.Source, "(stdin)"),
		Workloads:          d.Workloads,
		GeneratedAtISO:     stamp(d.GeneratedAt),
		ShareURL:           d.ShareURL,
		HasShareURL:        d.ShareURL != "",
		AccuracyDisclosure: accuracyFor(d.Mode),
		Mode:               modeLabel(d.Mode),
		Totals: totalsView{
			HasSavings:     totalCents > 0,
			MonthlyUSD:     fmtUSD(totalCents),
			AnnualUSD:      fmtUSD(totalCents * 12),
			WorkloadsLabel: pluralised(d.Workloads, "workload"),
		},
		HasCost:     len(cost) > 0,
		HasSecurity: len(sec) > 0,
		Clean:       len(cost) == 0 && len(sec) == 0,
	}
	for _, f := range cost {
		v.Cost = append(v.Cost, toView(f))
	}
	for _, f := range sec {
		v.Security = append(v.Security, toView(f))
	}
	return v
}

func split(in []rules.Finding) (cost, sec []rules.Finding) {
	cost = make([]rules.Finding, 0, len(in))
	sec = make([]rules.Finding, 0, len(in))
	for _, f := range in {
		if f.Category == rules.CategorySecurity {
			sec = append(sec, f)
		} else {
			cost = append(cost, f)
		}
	}
	return
}

func sortCost(in []rules.Finding) []rules.Finding {
	out := make([]rules.Finding, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.MonthlyUSDCents > 0) != (b.MonthlyUSDCents > 0) {
			return a.MonthlyUSDCents > 0
		}
		if a.MonthlyUSDCents != b.MonthlyUSDCents {
			return a.MonthlyUSDCents > b.MonthlyUSDCents
		}
		if a.Workload != b.Workload {
			return a.Workload < b.Workload
		}
		return a.DetectorID < b.DetectorID
	})
	return out
}

func toView(f rules.Finding) findingView {
	v := findingView{
		Severity:      string(f.Severity),
		SeverityCls:   "sev-" + strings.ToLower(string(f.Severity)),
		Workload:      f.Workload,
		Title:         f.Title,
		Detail:        f.Detail,
		Confidence:    string(f.Confidence),
		ConfidenceCls: "conf-" + strings.ToLower(string(f.Confidence)),
	}
	if f.MonthlyUSDCents > 0 {
		v.HasSavings = true
		v.SavingsUSD = fmtUSD(f.MonthlyUSDCents)
	}
	return v
}

func titleOrDefault(d Data) string {
	if d.PageTitle != "" {
		return d.PageTitle
	}
	src := firstNonEmpty(d.Source, "analysis")
	return "Optiqor — " + src
}

func stamp(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Truncate(time.Minute).Format(time.RFC3339)
}

func accuracyFor(m Mode) string {
	if m == ModeAgent {
		return "Agent accuracy: ±15%. Backed by 30 days of Prometheus + your AWS bill."
	}
	return AccuracyDisclosure
}

func modeLabel(m Mode) string {
	if m == ModeAgent {
		return "agent"
	}
	return "sandbox"
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func pluralised(n int, base string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, base)
	}
	return fmt.Sprintf("%d %ss", n, base)
}

func fmtUSD(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	dollars := cents / 100
	rem := cents % 100
	if rem == 0 {
		return fmt.Sprintf("%s$%s", sign, withCommas(dollars))
	}
	return fmt.Sprintf("%s$%s.%02d", sign, withCommas(dollars), rem)
}

func withCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteString(",")
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteString(",")
		}
	}
	return b.String()
}

var tmpl = template.Must(template.New("htmlrender").Parse(documentTemplate))
