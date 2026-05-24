package analyze

import "github.com/optiqor/optiqor-cli/pkg/rules"

// Score is the result of `optiqor score [chart]`: 0–100 efficiency
// computed as "100 minus per-finding penalty (capped)", plus a
// qualitative confidence band and a Grade (letter + percentile).
// Numerical confidence arrives in Year 2 once we have merged-PR
// outcomes to calibrate against.
type Score struct {
	Workloads int              `json:"workloads_analyzed"`
	Source    string           `json:"source"`
	Value     int              `json:"score"` // 0-100
	Band      rules.Confidence `json:"confidence_band"`
	Grade     Grade            `json:"grade"`
	Penalties map[string]int   `json:"penalties"` // detector_id -> penalty points
	Findings  []rules.Finding  `json:"findings"`
}

// Penalty weights per severity; capped so one HIGH finding alone
// can't push a chart below ~50.
const (
	penaltyHigh = 25
	penaltyMed  = 10
	penaltyLow  = 3
	penaltyInfo = 1
	penaltyCap  = 100
)

// Compute folds findings into a Score.
func Compute(source string, workloads int, findings []rules.Finding) Score {
	penalties := map[string]int{}
	total := 0
	for _, f := range findings {
		p := penaltyFor(f.Severity)
		penalties[f.DetectorID] += p
		total += p
	}
	if total > penaltyCap {
		total = penaltyCap
	}
	value := 100 - total
	if value < 0 {
		value = 0
	}
	return Score{
		Workloads: workloads,
		Source:    source,
		Value:     value,
		Band:      bandFor(value),
		Grade:     GradeFor(value),
		Penalties: penalties,
		Findings:  findings,
	}
}

func penaltyFor(s rules.Severity) int {
	switch s {
	case rules.SeverityHigh:
		return penaltyHigh
	case rules.SeverityMed:
		return penaltyMed
	case rules.SeverityLow:
		return penaltyLow
	default:
		return penaltyInfo
	}
}

// bandFor: stable Year-1 mapping; Year-2 confidence will be
// calibrated against measured outcomes instead.
func bandFor(score int) rules.Confidence {
	switch {
	case score >= 85:
		return rules.ConfidenceHigh
	case score >= 60:
		return rules.ConfidenceMed
	default:
		return rules.ConfidenceLow
	}
}
