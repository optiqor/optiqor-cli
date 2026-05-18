package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/optiqor/optiqor-cli/internal/render"
	"github.com/optiqor/optiqor-cli/internal/render/style"
	"github.com/optiqor/optiqor-cli/pkg/htmlrender"
)

const scoreDefaultWidth = 78

// WriteText renders the score panel as styled text. The Grade row is
// the headline — letter + percentile against the calibration set —
// because that is what platform engineers screenshot. The numeric
// score still appears so CI gates and analytics can pin against it.
//
// Layout:
//
//	── header ──
//	  Source / Workloads
//	  Grade        B+   better than 64% of 100 benchmark charts
//	  Score        78 / 100   ●●○ medium confidence
//	  Penalty breakdown …
//	── footer (accuracy disclosure + calibration note) ──
func (s Score) WriteText(w io.Writer, opts render.Options) error {
	t := style.NewTheme(opts.Color)
	width := opts.Width
	if width <= 0 {
		width = scoreDefaultWidth
	}

	var b strings.Builder
	writeScoreHeader(&b, t, width)
	writeScoreSource(&b, t, s)
	writeScoreGradeRow(&b, t, s)
	writeScoreNumericRow(&b, t, s)
	writeScorePenalties(&b, t, s)
	writeScoreFooter(&b, t, width)

	_, err := io.WriteString(w, b.String())
	return err
}

func writeScoreHeader(b *strings.Builder, t style.Theme, width int) {
	div := t.DividerLine(width)
	mark := t.BrandMark.Render(style.BrandGlyph)
	fmt.Fprintf(b, "%s\n", div)
	fmt.Fprintf(b, "  %s  %s   %s\n",
		mark,
		t.Brand.Render("optiqor score"),
		t.Tagline.Render("Helm chart efficiency grade"),
	)
	fmt.Fprintf(b, "%s\n\n", div)
}

func writeScoreSource(b *strings.Builder, t style.Theme, s Score) {
	srcLabel := s.Source
	if srcLabel == "" {
		srcLabel = "(stdin)"
	}
	fmt.Fprintf(b, "  %s %s\n",
		t.Muted.Render("Source     "),
		t.Workload.Render(srcLabel),
	)
	fmt.Fprintf(b, "  %s %s\n\n",
		t.Muted.Render("Workloads  "),
		t.Title.Render(fmt.Sprintf("%d analyzed", s.Workloads)),
	)
}

func writeScoreGradeRow(b *strings.Builder, t style.Theme, s Score) {
	letter := " " + s.Grade.Letter + " "
	beat := fmt.Sprintf("better than %d%% of %d benchmark charts",
		s.Grade.PercentileRank, s.Grade.Sample,
	)
	fmt.Fprintf(b, "  %s %s   %s\n",
		t.Muted.Render("Grade      "),
		gradeBadge(t, s.Value).Render(letter),
		t.Title.Render(beat),
	)
}

func writeScoreNumericRow(b *strings.Builder, t style.Theme, s Score) {
	fmt.Fprintf(b, "  %s %s   %s\n\n",
		t.Muted.Render("Score      "),
		t.Title.Render(fmt.Sprintf("%d / 100", s.Value)),
		t.ConfidenceDots(string(s.Band)),
	)
}

// gradeBadge picks a badge style for the letter grade so it visually
// matches the severity palette: red for F/D, amber for C, green for
// B/A. Reuses the existing severity badges so the brand palette stays
// consistent.
func gradeBadge(t style.Theme, value int) lipgloss.Style {
	switch {
	case value < 60:
		return t.SevHigh
	case value < 85:
		return t.SevMed
	default:
		return t.SevLow
	}
}

func writeScorePenalties(b *strings.Builder, t style.Theme, s Score) {
	if len(s.Penalties) == 0 {
		return
	}
	fmt.Fprintf(b, "  %s\n", t.Title.Render("Penalty breakdown"))
	ids := make([]string, 0, len(s.Penalties))
	for det := range s.Penalties {
		ids = append(ids, det)
	}
	sort.Strings(ids)
	for _, det := range ids {
		p := s.Penalties[det]
		fmt.Fprintf(b, "    %s%s   %s\n",
			t.Muted.Render(det),
			strings.Repeat(" ", maxInt(2, 32-len(det))),
			t.Disclosure.Render(fmt.Sprintf("-%d", p)),
		)
	}
	b.WriteString("\n")
}

func writeScoreFooter(b *strings.Builder, t style.Theme, width int) {
	fmt.Fprintf(b, "%s\n", t.DividerLine(width))
	fmt.Fprintf(b, "  %s\n", t.Disclosure.Render(htmlrender.AccuracyDisclosure))
	fmt.Fprintf(b, "  %s %s\n",
		t.Muted.Render("Calibration:"),
		t.Muted.Render("static benchmark distribution; agent install unlocks live percentile vs your fleet."),
	)
}

// WriteJSON renders the score as machine-readable JSON. Includes the
// full Grade so consumers can drive their own dashboards without
// reimplementing the percentile lookup.
func (s Score) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		AccuracyDisclosure string `json:"accuracy_disclosure"`
		Score              Score  `json:"score_report"`
	}{
		AccuracyDisclosure: htmlrender.AccuracyDisclosure,
		Score:              s,
	})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
