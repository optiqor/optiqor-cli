package analyze

import (
	"testing"

	"github.com/optiqor/optiqor-cli/internal/render"
	"github.com/optiqor/optiqor-cli/pkg/rules"
)

func sampleReport(t *testing.T) render.Report {
	t.Helper()
	return render.Report{
		Source:    "x",
		Workloads: 3,
		Findings: []rules.Finding{
			{DetectorID: "cpu-overprovisioned", Severity: rules.SeverityMed, Category: rules.CategoryCost},
			{DetectorID: "memory-overprovisioned", Severity: rules.SeverityMed, Category: rules.CategoryCost},
			{DetectorID: "missing-memory-limit", Severity: rules.SeverityHigh, Category: rules.CategorySecurity},
			{DetectorID: "missing-cpu-limit", Severity: rules.SeverityLow, Category: rules.CategorySecurity},
			{DetectorID: "image-pinned-latest", Severity: rules.SeverityMed, Category: rules.CategorySecurity},
		},
	}
}

func TestFilter(t *testing.T) {
	for _, tc := range []struct {
		name      string
		opts      FilterOptions
		wantLen   int
		wantIDs   []string
		mustOnly  rules.Category
		denyBelow rules.Severity
	}{
		{
			name:    "zero-options-passthrough",
			opts:    FilterOptions{},
			wantLen: 5,
		},
		{
			name:     "security-only-strips-cost",
			opts:     FilterOptions{SecurityOnly: true},
			wantLen:  3,
			mustOnly: rules.CategorySecurity,
		},
		{
			name:      "min-severity-med-drops-low",
			opts:      FilterOptions{MinSeverity: rules.SeverityMed},
			wantLen:   4,
			denyBelow: rules.SeverityMed,
		},
		{
			name:    "detector-allow-list-keeps-named-only",
			opts:    FilterOptions{DetectorIDs: []string{"cpu-overprovisioned", "image-pinned-latest"}},
			wantLen: 2,
			wantIDs: []string{"cpu-overprovisioned", "image-pinned-latest"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := Filter(sampleReport(t), tc.opts)
			if len(out.Findings) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(out.Findings), tc.wantLen)
			}
			if tc.mustOnly != "" {
				for _, f := range out.Findings {
					if f.Category != tc.mustOnly {
						t.Errorf("non-%s finding leaked: %+v", tc.mustOnly, f)
					}
				}
			}
			if tc.denyBelow != "" {
				for _, f := range out.Findings {
					if severityRank(f.Severity) < severityRank(tc.denyBelow) {
						t.Errorf("below-threshold finding leaked: %+v", f)
					}
				}
			}
			if len(tc.wantIDs) > 0 {
				got := map[string]bool{}
				for _, f := range out.Findings {
					got[f.DetectorID] = true
				}
				for _, id := range tc.wantIDs {
					if !got[id] {
						t.Errorf("missing detector %q in filtered set: %v", id, got)
					}
				}
			}
		})
	}
}

func TestFilter_DoesNotMutateInputReport(t *testing.T) {
	r := sampleReport(t)
	before := len(r.Findings)
	_ = Filter(r, FilterOptions{SecurityOnly: true})
	if len(r.Findings) != before {
		t.Errorf("Filter mutated source report: before=%d after=%d", before, len(r.Findings))
	}
}

func TestSeverityRank(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b rules.Severity
		want bool
	}{
		{name: "high-outranks-med", a: rules.SeverityHigh, b: rules.SeverityMed, want: true},
		{name: "med-outranks-low", a: rules.SeverityMed, b: rules.SeverityLow, want: true},
		{name: "unknown-ranks-zero", a: rules.Severity("nonsense"), b: rules.SeverityInfo, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := severityRank(tc.a) > severityRank(tc.b)
			if got != tc.want {
				t.Errorf("rank(%s) > rank(%s) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
