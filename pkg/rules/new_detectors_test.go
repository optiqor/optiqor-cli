package rules

import (
	"strings"
	"testing"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

func TestMemoryOverprovisioned_Triggers(t *testing.T) {
	w := parser.Workload{
		Name:     "api",
		Requests: parser.ResourceList{Memory: memQ(2 * 1024 * 1024 * 1024)}, // 2GiB
		Limits:   parser.ResourceList{Memory: memQ(int64(2.5 * 1024 * 1024 * 1024))},
	}
	f := newMemoryOverprovisioned().Run(w)
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

func TestMemoryOverprovisioned_DoesNotTriggerBelowRatio(t *testing.T) {
	w := parser.Workload{
		Requests: parser.ResourceList{Memory: memQ(256 * 1024 * 1024)},
		Limits:   parser.ResourceList{Memory: memQ(2 * 1024 * 1024 * 1024)},
	}
	if f := newMemoryOverprovisioned().Run(w); len(f) != 0 {
		t.Fatalf("ratio 0.125 should not fire; got %v", f)
	}
}

func TestMissingCPULimit_Triggers(t *testing.T) {
	w := parser.Workload{
		Requests: parser.ResourceList{CPU: cpuQ(500)},
		Limits:   parser.ResourceList{},
	}
	f := newMissingCPULimit().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected 1 finding")
	}
	if f[0].Severity != SeverityLow {
		t.Errorf("severity = %s", f[0].Severity)
	}
}

func TestMissingCPULimit_NoTrigger(t *testing.T) {
	w := parser.Workload{
		Requests: parser.ResourceList{CPU: cpuQ(500)},
		Limits:   parser.ResourceList{CPU: cpuQ(1000)},
	}
	if f := newMissingCPULimit().Run(w); len(f) != 0 {
		t.Fatalf("limit set → no finding; got %v", f)
	}
}

func TestImagePinnedLatest_TriggersOnLiteralLatest(t *testing.T) {
	w := parser.Workload{
		Name:  "cache",
		Image: parser.ImageRef{Repository: "redis", Tag: "latest", Set: true},
	}
	f := newImagePinnedLatest().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected 1 finding")
	}
	if !strings.Contains(f[0].Detail, ":latest") {
		t.Errorf("detail should mention :latest: %q", f[0].Detail)
	}
}

func TestImagePinnedLatest_TriggersOnMissingTag(t *testing.T) {
	w := parser.Workload{
		Name:  "cache",
		Image: parser.ImageRef{Repository: "redis", Tag: "", Set: true},
	}
	f := newImagePinnedLatest().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected 1 finding for missing tag")
	}
}

func TestImagePinnedLatest_DoesNotTriggerOnVersionTag(t *testing.T) {
	w := parser.Workload{
		Image: parser.ImageRef{Repository: "redis", Tag: "7.2.4", Set: true},
	}
	if f := newImagePinnedLatest().Run(w); len(f) != 0 {
		t.Fatalf("version tag should not fire; got %v", f)
	}
}

func TestImagePinnedLatest_NoImage(t *testing.T) {
	w := parser.Workload{Name: "x"}
	if f := newImagePinnedLatest().Run(w); len(f) != 0 {
		t.Fatalf("no image → no finding; got %v", f)
	}
}

// Sanity check that All() returns the expected detector count and
// that every ID is unique. Update the count when new detectors are
// added; the IDs themselves are stable wire format.
func TestAll_DetectorCountAndUniqueIDs(t *testing.T) {
	const want = 31 // 16 cost + 15 security (idle-workload added 2026-05-18)
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
