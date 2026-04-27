package rules

import (
	"fmt"

	"github.com/lowplane/sevro/internal/parser"
)

// replicasTooHigh fires when a workload declares many replicas without
// declaring an HPA. The chart author may know what they are doing
// (e.g. they autoscale via Cluster Autoscaler at the node level), but
// for the most common case — over-provisioned dev or stage clusters
// running 10-replica services that average <20% CPU — the cheapest
// fix is to enable the HPA.
type replicasTooHigh struct{}

func newReplicasTooHigh() Detector { return replicasTooHigh{} }

func (replicasTooHigh) ID() string   { return "replicas-too-high" }
func (replicasTooHigh) Name() string { return "Replicas without HPA" }

const replicasThreshold = 5

func (replicasTooHigh) Run(w parser.Workload) []Finding {
	if w.Replicas <= replicasThreshold {
		return nil
	}
	if w.HasHPA {
		return nil
	}
	return []Finding{{
		DetectorID: "replicas-too-high",
		Workload:   w.Name,
		Title:      "High static replica count without an autoscaler",
		Detail:     fmt.Sprintf("Replicas set to %d without an HPA. Static high replica counts pay for unused capacity 24/7. Either enable autoscaling.enabled=true or right-size the static count to your observed P95.", w.Replicas),
		Severity:   SeverityMed,
		Confidence: ConfidenceMed,
		// No dollar estimate — depends entirely on per-replica resource
		// requests, which the cpu/memory-overprovisioned detectors
		// already capture.
	}}
}
