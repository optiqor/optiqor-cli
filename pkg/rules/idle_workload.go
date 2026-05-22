package rules

import (
	"fmt"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// idleWorkload fires when a Helm chart declares a workload that is
// statically dormant: replicas is set to 0 and there is no HPA that
// would scale it up on demand. In a live cluster this is the chart
// pattern that correlates with "idle workload" findings (a Deployment
// that exists but has no running pods and no traffic-driven autoscale
// path). Agent-mode replaces this heuristic with Prometheus-grounded
// detection ("no traffic + no CPU over 7 days"); the CLI catches the
// most obvious version statically.
//
// Why this isn't covered by the existing replica detectors:
//   - replicas-too-high fires on > 5 with no HPA (high count)
//   - excessive-replica-count fires on > 20 even with HPA
//   - Neither catches the replicas=0 case
//
// Charts with feature-flag-style deployments (replicas=0 by default,
// flipped to N>0 in production overlays) will surface this finding.
// That's intentional: a chart shipped with replicas=0 and no HPA is
// indistinguishable from a forgotten workload at static-analysis time.
// The customer dismisses the finding once if it's intentional.
type idleWorkload struct{}

func newIdleWorkload() Detector { return idleWorkload{} }

func (idleWorkload) ID() string   { return "idle-workload" }
func (idleWorkload) Name() string { return "Workload appears idle" }

// Heuristic constants tracked via golden tests.
const (
	// idleWorkloadAssumedReplicas is the replica count we *would* have
	// recommended for cost-savings math if the workload had been
	// configured normally. Sandbox-grade — agent-mode replaces this with
	// the observed traffic profile.
	idleWorkloadAssumedReplicas = 2
	// idleWorkloadAssumedCPURequestMilli is a conservative CPU request
	// shape for the "saved" capacity. Used only because the workload
	// has no requests declared (its idleness is exactly the signal).
	idleWorkloadAssumedCPURequestMilli = 250
)

func (idleWorkload) Run(w parser.Workload) []Finding {
	if w.Replicas != 0 {
		return nil
	}
	if w.HasHPA {
		// HPA with minReplicas=0 (KEDA-style scale-to-zero) is a
		// legitimate scale-from-zero pattern; the autoscaler is the
		// signal. Don't flag.
		return nil
	}
	// Rough projection: had this workload run with sensible defaults,
	// it would have cost ~N×cpu×$/hr. We attribute the saved amount to
	// the chart already shipping replicas=0.
	monthlyCents := int64(idleWorkloadAssumedReplicas) *
		int64(idleWorkloadAssumedCPURequestMilli) *
		int64(cpuOverprovMonthlyHours) *
		int64(cpuPricePerCoreHourCents) / 1000
	return []Finding{{
		DetectorID: "idle-workload",
		Workload:   w.Name,
		Title:      "Workload declared with replicas=0 and no autoscaler",
		Detail: fmt.Sprintf(
			"%s ships with replicas=0 and no HPA. In a live cluster this is the chart pattern that correlates with abandoned workloads — declared but never scheduled. If this is intentional (e.g. feature-flagged deployment), dismiss. Otherwise the chart can be slimmer: remove the workload entirely, or document the activation path.",
			workloadDisplay(w),
		),
		MonthlyUSDCents: monthlyCents,
		Severity:        SeverityLow,
		Confidence:      ConfidenceLow,
	}}
}

// workloadDisplay renders a workload reference for finding text.
// Falls back to "this workload" when the name is empty.
func workloadDisplay(w parser.Workload) string {
	if w.Name == "" {
		return "this workload"
	}
	return w.Name
}
