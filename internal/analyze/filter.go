package analyze

import (
	"strings"

	"github.com/optiqor/optiqor-cli/internal/render"
	"github.com/optiqor/optiqor-cli/pkg/rules"
)

// FilterOptions narrows a Report's findings before rendering.
type FilterOptions struct {
	MinSeverity  rules.Severity
	DetectorIDs  []string // empty → all detectors
	SecurityOnly bool     // backs the `optiqor audit` command
}

// Filter returns a new Report; the input is not mutated.
func Filter(r render.Report, opts FilterOptions) render.Report {
	if opts.MinSeverity == "" && len(opts.DetectorIDs) == 0 && !opts.SecurityOnly {
		return r
	}
	allowed := make(map[string]struct{}, len(opts.DetectorIDs))
	for _, d := range opts.DetectorIDs {
		allowed[strings.TrimSpace(d)] = struct{}{}
	}
	out := make([]rules.Finding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if opts.SecurityOnly && f.Category != rules.CategorySecurity {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[f.DetectorID]; !ok {
				continue
			}
		}
		if opts.MinSeverity != "" && severityRank(f.Severity) < severityRank(opts.MinSeverity) {
			continue
		}
		out = append(out, f)
	}
	r.Findings = out
	return r
}

func severityRank(s rules.Severity) int {
	switch s {
	case rules.SeverityHigh:
		return 3
	case rules.SeverityMed:
		return 2
	case rules.SeverityLow:
		return 1
	case rules.SeverityInfo:
		return 0
	}
	return 0
}
