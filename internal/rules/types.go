// Package rules holds the deterministic detector library: cost detectors
// (overprovisioned CPU/mem, missing limits, etc.) and security detectors.
//
// All detectors are pure functions over the parser's normalised
// representation. They never call out to networks, never invoke an LLM,
// and never produce nondeterministic output for the same input.
package rules

import (
	"sort"

	"github.com/lowplane/sevro/internal/parser"
)

// Severity captures the suggested triage urgency.
type Severity string

const (
	SeverityHigh Severity = "HIGH"
	SeverityMed  Severity = "MED"
	SeverityLow  Severity = "LOW"
	SeverityInfo Severity = "INFO"
)

// Confidence is the qualitative band the CLI emits. Numerical scores
// land in Year 2 once enough merged PRs have been measured to calibrate
// them — see [docs/idea.md](../../../docs/idea.md).
type Confidence string

const (
	ConfidenceHigh Confidence = "high"
	ConfidenceMed  Confidence = "medium"
	ConfidenceLow  Confidence = "low"
)

// Finding is a single detector output. Renderers consume a slice of
// these; the rule engine never speaks UI.
type Finding struct {
	// Stable detector identifier (e.g. "cpu-overprovisioned"). Used
	// for dismissals, suppressions, and the SaaS detector library.
	DetectorID string

	// Workload the finding refers to (its Name from parser.Workload).
	Workload string

	// Short human-readable title for the PR comment / CLI table.
	Title string

	// Long-form explanation — surfaces in the diff narrative.
	Detail string

	// Suggested cents/month savings if the finding is acted on.
	// CLI is sandbox-grade (±40% disclosed); cluster-installed agent
	// produces exact numbers backed by 30 days of Prometheus data.
	MonthlyUSDCents int64

	Severity   Severity
	Confidence Confidence
}

// Detector inspects a parser.Workload and returns zero or more
// findings. Implementations must be deterministic and side-effect free.
type Detector interface {
	ID() string
	Name() string
	Run(parser.Workload) []Finding
}

// Run applies every detector to every workload and returns the merged,
// sorted findings list. Sort order is stable so renderers and golden
// tests are deterministic.
func Run(workloads []parser.Workload, dets []Detector) []Finding {
	var out []Finding
	for _, w := range workloads {
		for _, d := range dets {
			out = append(out, d.Run(w)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Workload != out[j].Workload {
			return out[i].Workload < out[j].Workload
		}
		if severityRank(out[i].Severity) != severityRank(out[j].Severity) {
			return severityRank(out[i].Severity) > severityRank(out[j].Severity)
		}
		return out[i].DetectorID < out[j].DetectorID
	})
	return out
}

// All returns the registered Year-1 detector set. Callers should always
// invoke this rather than constructing detectors by hand so we add new
// detectors in one place.
func All() []Detector {
	return []Detector{
		newCPUOverprovisioned(),
		newMissingMemoryLimit(),
	}
}

func severityRank(s Severity) int {
	switch s {
	case SeverityHigh:
		return 3
	case SeverityMed:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}
