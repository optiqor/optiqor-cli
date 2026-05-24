package analyze

import (
	"strings"
	"testing"
)

const aOnly = `
api:
  resources:
    requests: {cpu: "1", memory: "1Gi"}
    limits:   {cpu: "1", memory: "1Gi"}
worker:
  resources:
    requests: {cpu: "500m", memory: "1Gi"}
    limits:   {cpu: "500m", memory: "1Gi"}
`

const bOnly = `
api:
  resources:
    requests: {cpu: "500m", memory: "512Mi"}
    limits:   {cpu: "500m", memory: "512Mi"}
worker:
  resources:
    requests: {cpu: "500m", memory: "1Gi"}
    limits:   {cpu: "500m", memory: "1Gi"}
cache:
  resources:
    requests: {cpu: "100m", memory: "256Mi"}
    limits:   {cpu: "200m", memory: "512Mi"}
`

func TestDiff_DetectsAddedRemovedAndChanged(t *testing.T) {
	rep, err := Diff(strings.NewReader(aOnly), strings.NewReader(bOnly), "before", "after")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if rep.A != "before" || rep.B != "after" {
		t.Errorf("labels lost: %+v", rep)
	}

	byName := map[string]DiffEntry{}
	for _, e := range rep.Entries {
		byName[e.Name] = e
	}

	api := byName["api"]
	if api.NewWorkload || api.RemovedWorkload {
		t.Errorf("api should be modified: %+v", api)
	}
	if api.CPURequestDelta != -500 {
		t.Errorf("api cpu request delta = %d, want -500", api.CPURequestDelta)
	}
	if api.MemoryRequestDelta != -(512 * 1024 * 1024) {
		t.Errorf("api memory request delta = %d, want -512Mi", api.MemoryRequestDelta)
	}
	if api.MonthlyUSDCentsDelta >= 0 {
		t.Errorf("api should be a saving (negative cents); got %d", api.MonthlyUSDCentsDelta)
	}

	worker := byName["worker"]
	if worker.CPURequestDelta != 0 || worker.MonthlyUSDCentsDelta != 0 {
		t.Errorf("worker unchanged should have 0 deltas: %+v", worker)
	}

	cache := byName["cache"]
	if !cache.NewWorkload {
		t.Error("cache should be NewWorkload")
	}
	if cache.MonthlyUSDCentsDelta <= 0 {
		t.Errorf("new workload should add cost: %d", cache.MonthlyUSDCentsDelta)
	}
}

func TestDiff_RemovedWorkload_FlagsRemoval(t *testing.T) {
	rep, err := Diff(
		strings.NewReader(aOnly),
		strings.NewReader(`api: {resources: {requests: {cpu: 1, memory: 1Gi}, limits: {cpu: 1, memory: 1Gi}}}`),
		"a", "b",
	)
	if err != nil {
		t.Fatal(err)
	}
	var sawRemoved bool
	for _, e := range rep.Entries {
		if e.Name == "worker" && e.RemovedWorkload {
			sawRemoved = true
		}
	}
	if !sawRemoved {
		t.Errorf("expected worker as RemovedWorkload, got entries=%+v", rep.Entries)
	}
}

func TestDiff_TotalSavings_NetNegativeWhenShrinking(t *testing.T) {
	rep, err := Diff(strings.NewReader(aOnly), strings.NewReader(`api: {resources: {requests: {cpu: 500m, memory: 512Mi}, limits: {cpu: 500m, memory: 512Mi}}}
worker: {resources: {requests: {cpu: 500m, memory: 1Gi}, limits: {cpu: 500m, memory: 1Gi}}}`), "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	if rep.MonthlyUSDCentsDelta() >= 0 {
		t.Errorf("net should be negative (savings); got %d", rep.MonthlyUSDCentsDelta())
	}
}

func TestDiff_Entries_SortedByName(t *testing.T) {
	rep, err := Diff(strings.NewReader(aOnly), strings.NewReader(bOnly), "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(rep.Entries); i++ {
		if rep.Entries[i-1].Name >= rep.Entries[i].Name {
			t.Errorf("entries not sorted: %v", rep.Entries)
		}
	}
}
