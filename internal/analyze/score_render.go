package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/lowplane/sevro/internal/render"
	"github.com/lowplane/sevro/internal/render/style"
)

// WriteText renders the score panel as styled text.
func (s Score) WriteText(w io.Writer, opts render.Options) error {
	t := style.NewTheme(opts.Color)
	width := opts.Width
	if width <= 0 {
		width = 72
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", t.DividerLine(width))
	fmt.Fprintf(&b, "  %s   %s\n", t.Brand.Render("sevro score"), t.Tagline.Render("Helm chart efficiency score"))
	fmt.Fprintf(&b, "%s\n\n", t.DividerLine(width))

	srcLabel := s.Source
	if srcLabel == "" {
		srcLabel = "(stdin)"
	}
	fmt.Fprintf(&b, "  %s %s\n", t.Muted.Render("source:"), t.Workload.Render(srcLabel))
	fmt.Fprintf(&b, "  %s %d\n\n", t.Muted.Render("workloads analyzed:"), s.Workloads)

	scoreStr := fmt.Sprintf("%d / 100", s.Value)
	scoreStyle := t.Savings
	switch {
	case s.Value < 60:
		scoreStyle = t.SevHigh
		scoreStr = " " + scoreStr + " "
	case s.Value < 85:
		scoreStyle = t.SevMed
		scoreStr = " " + scoreStr + " "
	}
	fmt.Fprintf(&b, "  %s %s   %s\n\n",
		t.Muted.Render("score:"),
		scoreStyle.Render(scoreStr),
		t.ConfidenceDots(string(s.Band)),
	)

	if len(s.Penalties) > 0 {
		fmt.Fprintf(&b, "  %s\n", t.Title.Render("Penalty breakdown"))
		// Deterministic order: sort detector IDs alphabetically so
		// golden tests are stable.
		ids := make([]string, 0, len(s.Penalties))
		for det := range s.Penalties {
			ids = append(ids, det)
		}
		sort.Strings(ids)
		for _, det := range ids {
			p := s.Penalties[det]
			fmt.Fprintf(&b, "    %s%s   %s\n",
				t.Muted.Render(det),
				strings.Repeat(" ", maxInt(2, 32-len(det))),
				t.Disclosure.Render(fmt.Sprintf("-%d", p)),
			)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "%s\n", t.DividerLine(width))
	fmt.Fprintf(&b, "  %s\n", t.Disclosure.Render(render.AccuracyDisclosure))
	_, err := io.WriteString(w, b.String())
	return err
}

// WriteJSON renders the score as machine-readable JSON.
func (s Score) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		AccuracyDisclosure string `json:"accuracy_disclosure"`
		Score              Score  `json:"score_report"`
	}{
		AccuracyDisclosure: render.AccuracyDisclosure,
		Score:              s,
	})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
