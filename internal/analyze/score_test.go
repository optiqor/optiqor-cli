package analyze

import (
	"testing"

	"github.com/optiqor/optiqor-cli/pkg/rules"
)

func TestCompute(t *testing.T) {
	manyHigh := func(n int) []rules.Finding {
		out := make([]rules.Finding, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, rules.Finding{DetectorID: "a", Severity: rules.SeverityHigh})
		}
		return out
	}
	for _, tc := range []struct {
		name     string
		source   string
		workers  int
		findings []rules.Finding
		check    func(t *testing.T, s Score)
	}{
		{
			name:    "no-findings-perfect-score",
			source:  "clean",
			workers: 3,
			check: func(t *testing.T, s Score) {
				t.Helper()
				if s.Value != 100 {
					t.Errorf("Value = %d, want 100", s.Value)
				}
				if s.Band != rules.ConfidenceHigh {
					t.Errorf("Band = %s, want high", s.Band)
				}
			},
		},
		{
			name:    "high-severity-drops",
			source:  "dirty",
			workers: 1,
			findings: []rules.Finding{
				{DetectorID: "a", Severity: rules.SeverityHigh},
			},
			check: func(t *testing.T, s Score) {
				t.Helper()
				if s.Value != 100-penaltyHigh {
					t.Errorf("Value = %d, want %d", s.Value, 100-penaltyHigh)
				}
			},
		},
		{
			name:     "penalty-caps-at-100",
			source:   "worst",
			workers:  1,
			findings: manyHigh(10),
			check: func(t *testing.T, s Score) {
				t.Helper()
				if s.Value != 0 {
					t.Errorf("Value = %d, want 0 (cap)", s.Value)
				}
				if s.Band != rules.ConfidenceLow {
					t.Errorf("Band = %s, want low", s.Band)
				}
			},
		},
		{
			name:    "band-threshold-perfect-high",
			source:  "x",
			workers: 1,
			check: func(t *testing.T, s Score) {
				t.Helper()
				if s.Value < 100 || s.Value > 100 {
					t.Errorf("Value = %d, want [100,100]", s.Value)
				}
				if s.Band != rules.ConfidenceHigh {
					t.Errorf("Band = %s, want %s", s.Band, rules.ConfidenceHigh)
				}
			},
		},
		{
			name:     "band-threshold-low-severity-still-high",
			source:   "x",
			workers:  1,
			findings: []rules.Finding{{Severity: rules.SeverityLow}},
			check: func(t *testing.T, s Score) {
				t.Helper()
				// 100 - 3 = 97
				if s.Value < 95 || s.Value > 100 {
					t.Errorf("Value = %d, want [95,100]", s.Value)
				}
				if s.Band != rules.ConfidenceHigh {
					t.Errorf("Band = %s, want %s", s.Band, rules.ConfidenceHigh)
				}
			},
		},
		{
			name:    "band-threshold-two-med-mid-band",
			source:  "x",
			workers: 1,
			findings: []rules.Finding{
				{Severity: rules.SeverityMed},
				{Severity: rules.SeverityMed},
			},
			check: func(t *testing.T, s Score) {
				t.Helper()
				// 100 - 20 = 80
				if s.Value < 75 || s.Value > 85 {
					t.Errorf("Value = %d, want [75,85]", s.Value)
				}
				if s.Band != rules.ConfidenceMed {
					t.Errorf("Band = %s, want %s", s.Band, rules.ConfidenceMed)
				}
			},
		},
		{
			name:    "band-threshold-two-high-low-band",
			source:  "x",
			workers: 1,
			findings: []rules.Finding{
				{Severity: rules.SeverityHigh},
				{Severity: rules.SeverityHigh},
			},
			check: func(t *testing.T, s Score) {
				t.Helper()
				// 100 - 50 = 50
				if s.Value < 45 || s.Value > 60 {
					t.Errorf("Value = %d, want [45,60]", s.Value)
				}
				if s.Band != rules.ConfidenceLow {
					t.Errorf("Band = %s, want %s", s.Band, rules.ConfidenceLow)
				}
			},
		},
		{
			name:    "penalties-per-detector",
			source:  "x",
			workers: 1,
			findings: []rules.Finding{
				{DetectorID: "a", Severity: rules.SeverityHigh},
				{DetectorID: "a", Severity: rules.SeverityMed},
				{DetectorID: "b", Severity: rules.SeverityLow},
			},
			check: func(t *testing.T, s Score) {
				t.Helper()
				if got, want := s.Penalties["a"], penaltyHigh+penaltyMed; got != want {
					t.Errorf("a penalty = %d, want %d", got, want)
				}
				if got, want := s.Penalties["b"], penaltyLow; got != want {
					t.Errorf("b penalty = %d, want %d", got, want)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, Compute(tc.source, tc.workers, tc.findings))
		})
	}
}
