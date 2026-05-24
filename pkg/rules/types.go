// Package rules holds the deterministic detector library: cost detectors
// (overprovisioned CPU/mem, missing limits, etc.) and security detectors.
// All detectors are pure functions over parser.Workload — no network, no
// LLM, deterministic output for the same input.
package rules

import (
	"sort"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// Severity is the suggested triage urgency for a Finding.
type Severity string

// Severity values, ordered by triage urgency.
const (
	SeverityHigh Severity = "HIGH"
	SeverityMed  Severity = "MED"
	SeverityLow  Severity = "LOW"
	SeverityInfo Severity = "INFO"
)

// Confidence is the qualitative band the CLI emits. Numerical scores
// land in Year 2 once enough merged PRs have been measured to calibrate
// them.
type Confidence string

// Confidence band values; ranked low → high.
const (
	ConfidenceHigh Confidence = "high"
	ConfidenceMed  Confidence = "medium"
	ConfidenceLow  Confidence = "low"
)

// Category classifies a Finding by product surface. Cost is the
// headline; security surfaces as a bonus side-effect of parsing Helm
// charts and renders separately so the cost signal is never drowned out.
// Set on every Finding by Run from the owning Detector's Category().
type Category string

// Category values. Add new categories here and to the renderer's
// section ordering before introducing them in detectors.
const (
	CategoryCost     Category = "cost"
	CategorySecurity Category = "security"
)

// Finding is a single detector output. Renderers consume a slice of
// these; the rule engine never speaks UI.
type Finding struct {
	// DetectorID is the stable identifier (e.g. "cpu-overprovisioned"),
	// used for dismissals, suppressions, and the SaaS detector library.
	DetectorID string

	// Workload is the parser.Workload.Name this finding refers to.
	Workload string

	// Title is the short human-readable headline.
	Title string

	// Detail is the long-form explanation surfaced in the diff narrative.
	Detail string

	// MonthlyUSDCents is the suggested cents/month savings if acted on.
	// CLI is sandbox-grade (±40% disclosed); the agent product produces
	// exact numbers from 30 days of Prometheus data.
	MonthlyUSDCents int64

	Severity   Severity
	Confidence Confidence

	// Category is filled by Run from the owning Detector.Category().
	// Detectors that pre-set it on their findings keep their value —
	// useful for synthetic findings that span categories.
	Category Category

	// Signal carries quantitative evidence (e.g. "request 200m vs limit
	// 2000m"). Renderers draw a request-vs-limit bar from it. Nil for
	// findings without numeric evidence. Detectors that surface ratios
	// MUST populate it — that bar is the screenshot.
	Signal *Signal
}

// Signal is the structured evidence behind a Finding. Renderers turn
// it into a request-vs-limit bar:
//
//	request 200m ████████████░░░░░░░░ 1 limit  (5x burst)
type Signal struct {
	// Label is the dimension being shown (e.g. "CPU", "memory", "replicas").
	Label string `json:"label"`

	// Have is the smaller / current value (request, observed, floor).
	Have float64 `json:"have"`
	// Want is the larger / target value (limit, recommended, ceiling).
	// Same unit as Have; renderers draw the bar from their ratio.
	Want float64 `json:"want"`

	// HaveDisplay / WantDisplay preserve the chart's original tokens
	// (e.g. "200m") so users see them instead of "0.2".
	HaveDisplay string `json:"have_display"`
	WantDisplay string `json:"want_display"`

	// Note is short optional commentary printed next to the bar
	// (e.g. "10x burst", "80% headroom").
	Note string `json:"note,omitempty"`
}

// Detector inspects a parser.Workload and returns zero or more findings.
// Implementations must be deterministic and side-effect free. Category()
// is stamped onto every emitted Finding by Run.
type Detector interface {
	ID() string
	Name() string
	Category() Category
	Run(parser.Workload) []Finding
}

// Run applies every detector to every workload and returns the merged,
// sorted findings list. Sort order is stable so renderers and golden
// tests are deterministic. Each finding's Category is populated from
// its detector unless the detector already set it.
func Run(workloads []parser.Workload, dets []Detector) []Finding {
	out := make([]Finding, 0, len(workloads)*len(dets))
	for _, w := range workloads {
		for _, d := range dets {
			cat := d.Category()
			for _, f := range d.Run(w) {
				if f.Category == "" {
					f.Category = cat
				}
				out = append(out, f)
			}
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
// invoke this rather than constructing detectors by hand so new
// detectors land in one place.
func All() []Detector {
	return []Detector{
		// ---- Cost / efficiency (16) ----
		newCPUOverprovisioned(),
		newMemoryOverprovisioned(),
		newCPULimitFarAboveRequest(),
		newMemoryLimitFarAboveRequest(),
		newOversizedCPULimit(),
		newOversizedMemoryLimit(),
		newReplicasTooHigh(),
		newExcessiveReplicaCount(),
		newUnboundedImageTag(),
		newCPUWithoutMemoryRequest(),
		newMemoryWithoutCPURequest(),
		newCPURequestEqualsLimit(),
		newMemoryRequestEqualsLimit(),
		newTinyCPURequest(),
		newTinyMemoryRequest(),
		newIdleWorkload(),
		// ---- Security (15) ----
		newMissingCPULimit(),    // also surfaces in audit
		newMissingMemoryLimit(), // also surfaces in audit
		newImagePinnedLatest(),
		newRunAsRoot(),
		newRunsAsUIDZero(),
		newPrivilegedContainer(),
		newHostNetwork(),
		newHostPID(),
		newHostIPC(),
		newReadOnlyRootFSMissing(),
		newAllowPrivilegeEscalation(),
		newHostPathVolume(),
		newDangerousCapabilityAdded(),
		newCapabilitiesNotDroppedAll(),
		newServiceAccountTokenAutomount(),
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
