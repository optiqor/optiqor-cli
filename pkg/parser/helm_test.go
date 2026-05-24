package parser

import (
	"strings"
	"testing"
)

func TestParseValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		wantErr bool
		check   func(t *testing.T, wls []Workload)
	}{
		{
			name: "flat-workloads",
			in: `
api:
  resources:
    requests:
      cpu: "1"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "4Gi"
worker:
  resources:
    requests:
      cpu: 500m
    limits:
      cpu: 1
`,
			check: func(t *testing.T, wls []Workload) {
				t.Helper()
				if len(wls) != 2 {
					t.Fatalf("expected 2 workloads, got %d: %+v", len(wls), wls)
				}
				if wls[0].Name != "api" {
					t.Errorf("wls[0].Name = %q, want api", wls[0].Name)
				}
				if wls[0].Requests.CPU.Value != 1000 {
					t.Errorf("api requests cpu = %d, want 1000", wls[0].Requests.CPU.Value)
				}
				if wls[0].Limits.Memory.Value != 4*1024*1024*1024 {
					t.Errorf("api limits memory = %d, want %d", wls[0].Limits.Memory.Value, 4*1024*1024*1024)
				}
				if wls[1].Name != "worker" {
					t.Errorf("wls[1].Name = %q, want worker", wls[1].Name)
				}
				if wls[1].Requests.Memory.Set {
					t.Errorf("worker requests memory should be unset")
				}
				if wls[1].Limits.Memory.Set {
					t.Errorf("worker limits memory should be unset")
				}
			},
		},
		{
			name: "nested-subchart-flattens-dotted",
			in: `
postgresql:
  primary:
    resources:
      requests:
        cpu: 2
        memory: 8Gi
`,
			check: func(t *testing.T, wls []Workload) {
				t.Helper()
				if len(wls) != 1 {
					t.Fatalf("expected 1 workload, got %+v", wls)
				}
				if wls[0].Name != "postgresql.primary" {
					t.Errorf("nested name = %q, want postgresql.primary", wls[0].Name)
				}
			},
		},
		{
			name: "no-resource-blocks-yields-empty",
			in: `
config:
  level: info
features:
  - one
  - two
`,
			check: func(t *testing.T, wls []Workload) {
				t.Helper()
				if len(wls) != 0 {
					t.Errorf("expected 0 workloads, got %+v", wls)
				}
			},
		},
		{
			name:    "malformed-yaml-errors",
			in:      "this is: not: valid: yaml::",
			wantErr: true,
		},
		{
			name:    "top-level-sequence-errors",
			in:      "- a\n- b\n",
			wantErr: true,
		},
		{
			name: "deterministic-alpha-order",
			in: `
zeta:
  resources:
    requests: {cpu: 1}
alpha:
  resources:
    requests: {cpu: 1}
mike:
  resources:
    requests: {cpu: 1}
`,
			check: func(t *testing.T, wls []Workload) {
				t.Helper()
				want := []string{"alpha", "mike", "zeta"}
				for i := range want {
					if wls[i].Name != want[i] {
						t.Errorf("wls[%d].Name = %q, want %q", i, wls[i].Name, want[i])
					}
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wls, err := ParseValues(strings.NewReader(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseValues: %v", err)
			}
			if tc.check != nil {
				tc.check(t, wls)
			}
		})
	}
}
