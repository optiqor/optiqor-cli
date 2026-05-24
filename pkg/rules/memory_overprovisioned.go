package rules

import (
	"fmt"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

// memoryOverprovisioned mirrors cpuOverprovisioned for memory: when
// request/limit > 0.5, real-world utilisation rarely justifies it and
// we suggest halving the request. Without 30 days of Prometheus data
// (CLI sandbox mode), this is a heuristic; the agent product replaces
// it with measured P95.
type memoryOverprovisioned struct{}

func newMemoryOverprovisioned() Detector { return memoryOverprovisioned{} }

func (memoryOverprovisioned) ID() string   { return "memory-overprovisioned" }
func (memoryOverprovisioned) Name() string { return "Memory request overprovisioned" }

const (
	memOverprovRatio = 0.5
	// $3.50 per GiB-month (AWS m5.large baseline). Sandbox-only; the
	// agent product uses the customer's actual bill.
	memPricePerGiBMonthCents = 350
)

func (memoryOverprovisioned) Run(w parser.Workload) []Finding {
	req, lim := w.Requests.Memory, w.Limits.Memory
	if !req.Set || !lim.Set || lim.Value == 0 {
		return nil
	}
	if float64(req.Value)/float64(lim.Value) <= memOverprovRatio {
		return nil
	}
	excessBytes := req.Value / 2
	excessGiB := float64(excessBytes) / float64(1024*1024*1024)
	monthlyCents := int64(excessGiB * float64(memPricePerGiBMonthCents))
	return []Finding{{
		DetectorID:      "memory-overprovisioned",
		Workload:        w.Name,
		Title:           "Memory request appears overprovisioned",
		Detail:          fmt.Sprintf("Request %s vs limit %s — typical utilization rarely justifies this ratio. Consider halving the request.", req, lim),
		MonthlyUSDCents: monthlyCents,
		Severity:        SeverityMed,
		Confidence:      ConfidenceMed,
		Signal: &Signal{
			Label:       "memory",
			Have:        float64(req.Value),
			Want:        float64(lim.Value),
			HaveDisplay: req.String(),
			WantDisplay: lim.String(),
			Note:        fmt.Sprintf("%.0f%% of limit", float64(req.Value)*100/float64(lim.Value)),
		},
	}}
}
