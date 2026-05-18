package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/optiqor/optiqor-cli/internal/render"
	"github.com/optiqor/optiqor-cli/internal/render/style"
	"github.com/optiqor/optiqor-cli/pkg/htmlrender"
	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// WriteText renders the diff as styled text. Always includes the
// accuracy disclosure (CLAUDE.md hard rule).
func (r DiffReport) WriteText(w io.Writer, opts render.Options) error {
	t := style.NewTheme(opts.Color)
	width := opts.Width
	if width <= 0 {
		width = 72
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", t.DividerLine(width))
	fmt.Fprintf(&b, "  %s   %s\n", t.Brand.Render("optiqor diff"), t.Tagline.Render("compare two Helm values files"))
	fmt.Fprintf(&b, "%s\n\n", t.DividerLine(width))
	fmt.Fprintf(&b, "  %s %s\n", t.Muted.Render("a:"), t.Workload.Render(r.A))
	fmt.Fprintf(&b, "  %s %s\n\n", t.Muted.Render("b:"), t.Workload.Render(r.B))

	if len(r.Entries) == 0 {
		fmt.Fprintf(&b, "  %s\n\n", t.OK.Render("✓ No workload changes."))
	} else {
		for i, e := range r.Entries {
			writeDiffEntry(&b, t, e)
			if i < len(r.Entries)-1 {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "%s\n", t.DividerLine(width))
	if total := r.MonthlyUSDCentsDelta(); total != 0 {
		sign := "+"
		styled := t.Disclosure
		if total < 0 {
			sign = "-"
			styled = t.Savings
		}
		fmt.Fprintf(&b, "  %s %s   %s\n",
			t.Muted.Render("net monthly delta:"),
			styled.Render(sign+"$"+formatCents(absInt(total))+"/mo"),
			t.Muted.Render("(±40%)"),
		)
	}
	fmt.Fprintf(&b, "  %s\n", t.Disclosure.Render(htmlrender.AccuracyDisclosure))

	_, err := io.WriteString(w, b.String())
	return err
}

// WriteJSON renders the diff as machine-readable JSON.
func (r DiffReport) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		AccuracyDisclosure string      `json:"accuracy_disclosure"`
		A                  string      `json:"a"`
		B                  string      `json:"b"`
		Entries            []DiffEntry `json:"entries"`
		MonthlyUSDDelta    float64     `json:"monthly_usd_delta"`
	}{
		AccuracyDisclosure: htmlrender.AccuracyDisclosure,
		A:                  r.A,
		B:                  r.B,
		Entries:            r.Entries,
		MonthlyUSDDelta:    float64(r.MonthlyUSDCentsDelta()) / 100.0,
	})
}

func writeDiffEntry(b *strings.Builder, t style.Theme, e DiffEntry) {
	tag := ""
	switch {
	case e.NewWorkload:
		tag = t.SevMed.Render(" NEW   ")
	case e.RemovedWorkload:
		tag = t.SevLow.Render(" REM   ")
	default:
		tag = t.SevInfo.Render(" CHG   ")
	}

	fmt.Fprintf(b, "  %s  %s\n", tag, t.Workload.Render(e.Name))
	fmt.Fprintf(b, "    %s %s\n", t.Muted.Render("cpu req:"), styleDelta(t, parser.FormatCPU(parser.Quantity{Value: absInt(e.CPURequestDelta), Set: true}), e.CPURequestDelta))
	fmt.Fprintf(b, "    %s %s\n", t.Muted.Render("cpu lim:"), styleDelta(t, parser.FormatCPU(parser.Quantity{Value: absInt(e.CPULimitDelta), Set: true}), e.CPULimitDelta))
	fmt.Fprintf(b, "    %s %s\n", t.Muted.Render("mem req:"), styleDelta(t, parser.FormatMemory(parser.Quantity{Value: absInt(e.MemoryRequestDelta), Set: true}), e.MemoryRequestDelta))
	fmt.Fprintf(b, "    %s %s\n", t.Muted.Render("mem lim:"), styleDelta(t, parser.FormatMemory(parser.Quantity{Value: absInt(e.MemoryLimitDelta), Set: true}), e.MemoryLimitDelta))

	if e.MonthlyUSDCentsDelta != 0 {
		sign := "+"
		styled := t.Disclosure
		if e.MonthlyUSDCentsDelta < 0 {
			sign = "-"
			styled = t.Savings
		}
		fmt.Fprintf(b, "    %s %s\n", t.Muted.Render("est. delta:"),
			styled.Render(sign+"$"+formatCents(absInt(e.MonthlyUSDCentsDelta))+"/mo"))
	}
}

func styleDelta(t style.Theme, magnitude string, delta int64) string {
	if delta == 0 {
		return t.Muted.Render("0")
	}
	if delta < 0 {
		return t.Savings.Render("-" + magnitude)
	}
	return t.Disclosure.Render("+" + magnitude)
}

func formatCents(c int64) string {
	dollars := c / 100
	cents := c % 100
	if cents == 0 {
		return fmt.Sprintf("%d", dollars)
	}
	return fmt.Sprintf("%d.%02d", dollars, cents)
}

func absInt(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
