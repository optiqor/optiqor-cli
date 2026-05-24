package rules

import (
	"strings"
	"testing"

	"github.com/optiqor/optiqor-cli/pkg/parser"
)

func TestIdleWorkload_TriggersOnZeroReplicasNoHPA(t *testing.T) {
	w := parser.Workload{
		Name:     "checkout-batch",
		Replicas: 0,
		HasHPA:   false,
	}
	f := newIdleWorkload().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(f))
	}
	if f[0].DetectorID != "idle-workload" {
		t.Errorf("unexpected detector id: %s", f[0].DetectorID)
	}
	if f[0].Severity != SeverityLow {
		t.Errorf("severity = %s, want LOW", f[0].Severity)
	}
	if f[0].Confidence != ConfidenceLow {
		t.Errorf("confidence = %s, want low", f[0].Confidence)
	}
	if f[0].MonthlyUSDCents <= 0 {
		t.Errorf("expected positive projected savings, got %d", f[0].MonthlyUSDCents)
	}
	if !strings.Contains(f[0].Detail, "checkout-batch") {
		t.Errorf("detail should reference workload name: %q", f[0].Detail)
	}
}

func TestIdleWorkload_DoesNotTriggerWithHPA(t *testing.T) {
	w := parser.Workload{
		Name:     "event-consumer",
		Replicas: 0,
		HasHPA:   true,
	}
	if f := newIdleWorkload().Run(w); len(f) != 0 {
		t.Fatalf("HPA + replicas=0 (scale-from-zero) should not fire; got %v", f)
	}
}

func TestIdleWorkload_DoesNotTriggerWithReplicas(t *testing.T) {
	w := parser.Workload{
		Name:     "api",
		Replicas: 3,
		HasHPA:   false,
	}
	if f := newIdleWorkload().Run(w); len(f) != 0 {
		t.Fatalf("replicas=3 should not fire; got %v", f)
	}
}

func TestIdleWorkload_FallbackWorkloadDisplay(t *testing.T) {
	w := parser.Workload{Name: "", Replicas: 0, HasHPA: false}
	f := newIdleWorkload().Run(w)
	if len(f) != 1 {
		t.Fatalf("expected finding with empty name, got %d", len(f))
	}
	if !strings.Contains(f[0].Detail, "this workload") {
		t.Errorf("empty name should yield 'this workload' fallback: %q", f[0].Detail)
	}
}

func TestIdleWorkload_RegisteredInAll(t *testing.T) {
	found := false
	for _, d := range All() {
		if d.ID() == "idle-workload" {
			found = true
			if d.Category() != CategoryCost {
				t.Errorf("idle-workload should be in cost category, got %s", d.Category())
			}
			break
		}
	}
	if !found {
		t.Fatal("idle-workload not registered in rules.All()")
	}
}
