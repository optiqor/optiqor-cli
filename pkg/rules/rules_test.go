package rules

import (
	"strings"
	"testing"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

func cpuQ(v int64) parser.Quantity { return parser.Quantity{Value: v, Set: true, Original: "x"} }
func memQ(v int64) parser.Quantity { return parser.Quantity{Value: v, Set: true, Original: "x"} }

func TestCPUOverprovisioned(t *testing.T) {
	for _, tc := range []struct {
		name        string
		in          parser.Workload
		wantCount   int
		wantSev     Severity
		wantSavings bool
	}{
		{
			name: "request-near-limit-triggers",
			in: parser.Workload{
				Name:     "api",
				Requests: parser.ResourceList{CPU: cpuQ(2000), Memory: memQ(1024)},
				Limits:   parser.ResourceList{CPU: cpuQ(2500), Memory: memQ(2048)},
			},
			wantCount:   1,
			wantSev:     SeverityMed,
			wantSavings: true,
		},
		{
			name: "ratio-below-threshold-quiet",
			in: parser.Workload{
				Name:     "api",
				Requests: parser.ResourceList{CPU: cpuQ(500)},
				Limits:   parser.ResourceList{CPU: cpuQ(2000)},
			},
			wantCount: 0,
		},
		{
			name: "no-limit-cannot-compare",
			in: parser.Workload{
				Name:     "api",
				Requests: parser.ResourceList{CPU: cpuQ(500)},
			},
			wantCount: 0,
		},
		{
			name: "no-request-cannot-compare",
			in: parser.Workload{
				Name:   "api",
				Limits: parser.ResourceList{CPU: cpuQ(1000)},
			},
			wantCount: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCPUOverprovisioned().Run(tc.in)
			if len(f) != tc.wantCount {
				t.Fatalf("findings = %d, want %d: %+v", len(f), tc.wantCount, f)
			}
			if tc.wantCount == 0 {
				return
			}
			if f[0].Severity != tc.wantSev {
				t.Errorf("severity = %s, want %s", f[0].Severity, tc.wantSev)
			}
			if tc.wantSavings && f[0].MonthlyUSDCents <= 0 {
				t.Errorf("expected positive savings, got %d", f[0].MonthlyUSDCents)
			}
		})
	}
}

func TestMissingMemoryLimit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		in        parser.Workload
		wantCount int
	}{
		{
			name: "missing-limit-triggers-high",
			in: parser.Workload{
				Name:     "worker",
				Requests: parser.ResourceList{Memory: memQ(1024 * 1024 * 1024)},
				Limits:   parser.ResourceList{},
			},
			wantCount: 1,
		},
		{
			name: "limit-present-quiet",
			in: parser.Workload{
				Name:     "worker",
				Requests: parser.ResourceList{Memory: memQ(1024 * 1024 * 1024)},
				Limits:   parser.ResourceList{Memory: memQ(2 * 1024 * 1024 * 1024)},
			},
			wantCount: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMissingMemoryLimit().Run(tc.in)
			if len(f) != tc.wantCount {
				t.Fatalf("findings = %d, want %d: %+v", len(f), tc.wantCount, f)
			}
			if tc.wantCount == 0 {
				return
			}
			if f[0].Severity != SeverityHigh {
				t.Errorf("severity = %s, want HIGH", f[0].Severity)
			}
			if !strings.Contains(f[0].Detail, "P95") {
				t.Errorf("detail should mention P95: %q", f[0].Detail)
			}
		})
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
		t.Fatalf("expected >=2 findings, got %d: %+v", len(got), got)
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
