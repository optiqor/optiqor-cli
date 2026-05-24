package rules

import (
	"strings"
	"testing"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

func TestMemoryOverprovisioned(t *testing.T) {
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
				Requests: parser.ResourceList{Memory: memQ(2 * 1024 * 1024 * 1024)}, // 2GiB
				Limits:   parser.ResourceList{Memory: memQ(int64(2.5 * 1024 * 1024 * 1024))},
			},
			wantCount:   1,
			wantSev:     SeverityMed,
			wantSavings: true,
		},
		{
			name: "ratio-below-threshold-quiet",
			in: parser.Workload{
				Requests: parser.ResourceList{Memory: memQ(256 * 1024 * 1024)},
				Limits:   parser.ResourceList{Memory: memQ(2 * 1024 * 1024 * 1024)},
			},
			wantCount: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMemoryOverprovisioned().Run(tc.in)
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

func TestMissingCPULimit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		in        parser.Workload
		wantCount int
	}{
		{
			name: "missing-limit-triggers-low",
			in: parser.Workload{
				Requests: parser.ResourceList{CPU: cpuQ(500)},
				Limits:   parser.ResourceList{},
			},
			wantCount: 1,
		},
		{
			name: "limit-present-quiet",
			in: parser.Workload{
				Requests: parser.ResourceList{CPU: cpuQ(500)},
				Limits:   parser.ResourceList{CPU: cpuQ(1000)},
			},
			wantCount: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMissingCPULimit().Run(tc.in)
			if len(f) != tc.wantCount {
				t.Fatalf("findings = %d, want %d: %+v", len(f), tc.wantCount, f)
			}
			if tc.wantCount == 0 {
				return
			}
			if f[0].Severity != SeverityLow {
				t.Errorf("severity = %s, want LOW", f[0].Severity)
			}
		})
	}
}

func TestImagePinnedLatest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		in             parser.Workload
		wantCount      int
		wantDetailHint string
	}{
		{
			name: "explicit-latest-tag-triggers",
			in: parser.Workload{
				Name:  "cache",
				Image: parser.ImageRef{Repository: "redis", Tag: "latest", Set: true},
			},
			wantCount:      1,
			wantDetailHint: ":latest",
		},
		{
			name: "missing-tag-defaults-to-latest",
			in: parser.Workload{
				Name:  "cache",
				Image: parser.ImageRef{Repository: "redis", Tag: "", Set: true},
			},
			wantCount: 1,
		},
		{
			name: "semver-tag-quiet",
			in: parser.Workload{
				Image: parser.ImageRef{Repository: "redis", Tag: "7.2.4", Set: true},
			},
			wantCount: 0,
		},
		{
			name:      "no-image-quiet",
			in:        parser.Workload{Name: "x"},
			wantCount: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newImagePinnedLatest().Run(tc.in)
			if len(f) != tc.wantCount {
				t.Fatalf("findings = %d, want %d: %+v", len(f), tc.wantCount, f)
			}
			if tc.wantDetailHint != "" && !strings.Contains(f[0].Detail, tc.wantDetailHint) {
				t.Errorf("detail should mention %q: %q", tc.wantDetailHint, f[0].Detail)
			}
		})
	}
}

// TestAll_DetectorCountAndUniqueIDs guards the wire-stable detector
// IDs and the running Phase 3+ count.
func TestAll_DetectorCountAndUniqueIDs(t *testing.T) {
	const want = 31 // 15 cost + 15 security + idle-workload (#32)
	dets := All()
	if len(dets) != want {
		t.Fatalf("All() returned %d detectors, want %d", len(dets), want)
	}
	seen := map[string]bool{}
	for _, d := range dets {
		id := d.ID()
		if seen[id] {
			t.Errorf("duplicate detector id %q", id)
		}
		seen[id] = true
	}
}
