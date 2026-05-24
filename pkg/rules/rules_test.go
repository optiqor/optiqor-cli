package rules

import (
	"strings"
	"testing"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

func cpuQ(v int64) parser.Quantity { return parser.Quantity{Value: v, Set: true, Original: "x"} }
func memQ(v int64) parser.Quantity { return parser.Quantity{Value: v, Set: true, Original: "x"} }

func TestCPUOverprovisioned_Triggers(t *testing.T) {
	w := parser.Workload{
		Name:     "api",
		Requests: parser.ResourceList{CPU: cpuQ(2000), Memory: memQ(1024)},
		Limits:   parser.ResourceList{CPU: cpuQ(2500), Memory: memQ(2048)},
	}
	f := newCPUOverprovisioned().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(f))
	}
	if f[0].Severity != SeverityMed {
		t.Errorf("severity = %s", f[0].Severity)
	}
	if f[0].MonthlyUSDCents <= 0 {
		t.Errorf("expected positive savings, got %d", f[0].MonthlyUSDCents)
	}
}

func TestCPUOverprovisioned_DoesNotTriggerBelowRatio(t *testing.T) {
	w := parser.Workload{
		Name:     "api",
		Requests: parser.ResourceList{CPU: cpuQ(500)},
		Limits:   parser.ResourceList{CPU: cpuQ(2000)},
	}
	if f := newCPUOverprovisioned().Run(w); len(f) != 0 {
		t.Fatalf("expected no findings (ratio 0.25); got %v", f)
	}
}

func TestCPUOverprovisioned_NoLimit(t *testing.T) {
	w := parser.Workload{
		Name:     "api",
		Requests: parser.ResourceList{CPU: cpuQ(500)},
	}
	if f := newCPUOverprovisioned().Run(w); len(f) != 0 {
		t.Fatalf("no limit -> no finding; got %v", f)
	}
}

func TestCPUOverprovisioned_NoRequest(t *testing.T) {
	w := parser.Workload{
		Name:   "api",
		Limits: parser.ResourceList{CPU: cpuQ(1000)},
	}
	if f := newCPUOverprovisioned().Run(w); len(f) != 0 {
		t.Fatalf("no request -> no finding; got %v", f)
	}
}

func TestMissingMemoryLimit_Triggers(t *testing.T) {
	w := parser.Workload{
		Name:     "worker",
		Requests: parser.ResourceList{Memory: memQ(1024 * 1024 * 1024)},
		Limits:   parser.ResourceList{},
	}
	f := newMissingMemoryLimit().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(f), f)
	}
	if f[0].Severity != SeverityHigh {
		t.Errorf("severity = %s", f[0].Severity)
	}
	if !strings.Contains(f[0].Detail, "P95") {
		t.Errorf("detail should mention P95: %q", f[0].Detail)
	}
}

func TestMissingMemoryLimit_DoesNotTriggerWhenLimitSet(t *testing.T) {
	w := parser.Workload{
		Name:     "worker",
		Requests: parser.ResourceList{Memory: memQ(1024 * 1024 * 1024)},
		Limits:   parser.ResourceList{Memory: memQ(2 * 1024 * 1024 * 1024)},
	}
	if f := newMissingMemoryLimit().Run(w); len(f) != 0 {
		t.Fatalf("limit set -> no finding; got %v", f)
	}
}

func TestRun_StableOrder(t *testing.T) {
	wls := []parser.Workload{
		{
			Name:     "zeta",
			Requests: parser.ResourceList{Memory: memQ(1024)},
		},
		{
			Name:     "alpha",
			Requests: parser.ResourceList{CPU: cpuQ(2000), Memory: memQ(1024)},
			Limits:   parser.ResourceList{CPU: cpuQ(2500), Memory: memQ(2048)},
		},
	}
	got := Run(wls, All())
	if len(got) < 2 {
		t.Fatalf("expected ≥2 findings, got %d: %+v", len(got), got)
	}
	if got[0].Workload != "alpha" {
		t.Errorf("first workload = %q, want alpha", got[0].Workload)
	}
}

func TestAll_HasExpectedDetectors(t *testing.T) {
	dets := All()
	want := map[string]bool{
		"cpu-overprovisioned":    false,
		"memory-overprovisioned": false,
		"missing-memory-limit":   false,
		"missing-cpu-limit":      false,
		"image-pinned-latest":    false,
	}
	for _, d := range dets {
		want[d.ID()] = true
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("All() missing detector %q", k)
		}
	}
}
