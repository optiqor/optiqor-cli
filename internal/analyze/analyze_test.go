package analyze

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		name          string
		yaml          string
		wantErr       bool
		wantWorkloads int
		mustSeeIDs    []string
	}{
		{
			name: "two-workloads-fire-cpu-and-missing-limit",
			yaml: `
api:
  resources:
    requests:
      cpu: "2"
      memory: "1Gi"
    limits:
      cpu: "2.5"
      memory: "2Gi"
worker:
  resources:
    requests:
      memory: "1Gi"
`,
			wantWorkloads: 2,
			mustSeeIDs:    []string{"cpu-overprovisioned", "missing-memory-limit"},
		},
		{
			name:    "malformed-yaml-errors",
			yaml:    "not: valid: yaml::",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rep, err := Run(strings.NewReader(tc.yaml), Options{Source: "test"})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if rep.Workloads != tc.wantWorkloads {
				t.Errorf("workloads = %d, want %d", rep.Workloads, tc.wantWorkloads)
			}
			if len(rep.Findings) == 0 {
				t.Fatal("expected findings, got 0")
			}
			seen := map[string]bool{}
			for _, f := range rep.Findings {
				seen[f.DetectorID] = true
			}
			for _, id := range tc.mustSeeIDs {
				if !seen[id] {
					t.Errorf("missing detector %q in findings", id)
				}
			}
		})
	}
}

func TestRunPath(t *testing.T) {
	dir := t.TempDir()
	fileValues := filepath.Join(dir, "file-values.yaml")
	if err := os.WriteFile(fileValues, []byte(`
api:
  resources:
    requests:
      cpu: "1"
      memory: "1Gi"
    limits:
      cpu: "1"
      memory: "1Gi"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	dirChart := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirChart, "values.yaml"),
		[]byte(`api: {resources: {requests: {cpu: 1, memory: 1Gi}, limits: {cpu: 1, memory: 1Gi}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name          string
		path          string
		wantErr       bool
		wantWorkloads int
		wantSrcSuffix string
	}{
		{
			name:          "file-path-loads-directly",
			path:          fileValues,
			wantWorkloads: 1,
		},
		{
			name:          "directory-resolves-to-values-yaml",
			path:          dirChart,
			wantWorkloads: 1,
			wantSrcSuffix: "values.yaml",
		},
		{
			name:    "missing-path-errors",
			path:    "/nonexistent/path/values.yaml",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rep, err := RunPath(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RunPath: %v", err)
			}
			if rep.Workloads != tc.wantWorkloads {
				t.Errorf("workloads = %d, want %d", rep.Workloads, tc.wantWorkloads)
			}
			if tc.wantSrcSuffix != "" && !strings.HasSuffix(rep.Source, tc.wantSrcSuffix) {
				t.Errorf("source = %q, want suffix %q", rep.Source, tc.wantSrcSuffix)
			}
		})
	}
}
