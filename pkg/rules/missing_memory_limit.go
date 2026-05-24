package rules

import (
	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// missingMemoryLimit flags workloads with a memory request but no
// memory limit. Pods without memory limits can grow unbounded and
// trigger node-level memory pressure that evicts neighbours.
//
// This is a security/safety finding, not a cost finding — savings are
// 0 but the risk is real. Severity HIGH because it affects co-tenants
// in a shared node group.
type missingMemoryLimit struct{}

func newMissingMemoryLimit() Detector { return missingMemoryLimit{} }

func (missingMemoryLimit) ID() string   { return "missing-memory-limit" }
func (missingMemoryLimit) Name() string { return "Missing memory limit" }

func (missingMemoryLimit) Run(w parser.Workload) []Finding {
	if !w.Requests.Memory.Set {
		return nil
	}
	if w.Limits.Memory.Set {
		return nil
	}
	return []Finding{{
		DetectorID: "missing-memory-limit",
		Workload:   w.Name,
		Title:      "Memory limit not set",
		Detail:     "This workload declares a memory request but no limit. Without a limit it can consume node memory unbounded and evict neighbours under pressure. Set a limit at or above your observed P95.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}
