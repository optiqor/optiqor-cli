// Package rules holds the deterministic detector library: cost detectors
// (overprovisioned CPU/mem, missing limits, etc.) and security detectors.
//
// All detectors are pure functions over the parser's normalised
// representation. They never call out to networks, never invoke an LLM,
// and never produce nondeterministic output for the same input.
package rules

import (
	"sort"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// Severity captures the suggested triage urgency.
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
// them — see [docs/idea.md](../../../docs/idea.md).
type Confidence string

// Confidence band values; ranked low → high.
const (
	ConfidenceHigh Confidence = "high"
	ConfidenceMed  Confidence = "medium"
	ConfidenceLow  Confidence = "low"
)

// Category classifies a finding by what it actually helps the user do.
// Optiqor's headline product is cost optimization; security findings
// surface as a bonus side-effect of parsing Helm charts and render
// separately so the cost signal is never drowned out.
//
// Category is set on every [Finding] by the engine in [Run] from the
// owning [Detector]'s Category() method. Detectors should declare it
// once at the type level rather than per-finding.
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

	// Category is filled by [Run] from the owning detector's
	// [Detector.Category]. Detectors that pre-set it on their findings
	// keep their value — useful for synthetic findings that span
	// categories.
	Category Category

	// Signal carries the quantitative evidence for findings that have
	// it (e.g. "request 200m vs limit 2000m"). Renderers draw a bar
	// from this when present; nil for findings without numeric
	// evidence (host-network, runs-as-root, etc.). Detectors that
	// surface ratios MUST populate it — that bar is the screenshot.
	Signal *Signal
}

// Signal is the structured evidence behind a Finding. Renderers turn
// it into a request-vs-limit bar:
//
//	request 200m ████████████░░░░░░░░ 1 limit  (5x burst)
//
// Have / Want share a unit; HaveDisplay / WantDisplay are the
// human-readable original tokens (e.g. "200m", "2Gi"). Note is
// optional commentary the renderer prints to the right of the bar
// (utilization headroom, burst ratio, percentile).
type Signal struct {
	// Label identifies the dimension being shown (e.g. "CPU",
	// "memory", "replicas"). Renderers may align by label.
	Label string `json:"label"`

	// Have is the smaller / current value (request, observed,
	// floor). Want is the larger / target value (limit, recommended,
	// ceiling). Both are in the same unit; renderers use their ratio
	// to draw the bar.
	Have float64 `json:"have"`
	Want float64 `json:"want"`

	// Display strings preserve the chart's original tokens so users
	// see "200m" rather than "0.2".
	HaveDisplay string `json:"have_display"`
	WantDisplay string `json:"want_display"`

	// Note is short commentary printed next to the bar (e.g.
	// "10x burst", "80% headroom"). Optional.
	Note string `json:"note,omitempty"`
}

// Detector inspects a parser.Workload and returns zero or more
// findings. Implementations must be deterministic and side-effect free.
//
// Category() declares which product surface the detector serves; the
// engine stamps it onto every Finding the detector emits so renderers
// and filters can group without inspecting detector IDs.
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
// invoke this rather than constructing detectors by hand so we add new
// detectors in one place.
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
