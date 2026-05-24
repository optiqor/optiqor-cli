package htmlrender

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/optiqor/optiqor-cli/pkg/rules"
)

func sample() Data {
	return Data{
		Source:      "demo",
		Workloads:   3,
		ShareURL:    "https://optiqor.dev/r/abc123",
		GeneratedAt: time.Date(2026, 5, 11, 14, 30, 0, 0, time.UTC),
		Findings: []rules.Finding{
			{
				DetectorID:      "cpu-overprovisioned",
				Workload:        "api",
				Title:           "CPU request appears overprovisioned",
				Detail:          "request 2 vs limit 2.5",
				MonthlyUSDCents: 22040,
				Severity:        rules.SeverityHigh,
				Confidence:      rules.ConfidenceMed,
				Category:        rules.CategoryCost,
			},
			{
				DetectorID:      "memory-overprovisioned",
				Workload:        "api",
				Title:           "Memory request appears overprovisioned",
				MonthlyUSDCents: 9640,
				Severity:        rules.SeverityMed,
				Confidence:      rules.ConfidenceMed,
				Category:        rules.CategoryCost,
			},
			{
				DetectorID: "run-as-root",
				Workload:   "worker",
				Title:      "Container runs as root",
				Severity:   rules.SeverityHigh,
				Confidence: rules.ConfidenceHigh,
				Category:   rules.CategorySecurity,
			},
		},
	}
}

func TestRender_HappyPath(t *testing.T) {
	out, err := RenderString(sample())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<!doctype html>",
		"Optiqor",
		"Cost optimizations",
		"Security findings",
		"bonus",
		"CPU request appears overprovisioned",
		"Container runs as root",
		"$316.80", // 22040 + 9640 cents = $316.80
		"±40%",
		"share",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output", want)
		}
	}
}

func TestRender_DisclosureMandatory(t *testing.T) {
	out, err := RenderString(sample())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, AccuracyDisclosure) {
		t.Errorf("sandbox-mode output must contain the verbatim disclosure")
	}
}

func TestRender_AgentModeAccuracyLine(t *testing.T) {
	d := sample()
	d.Mode = ModeAgent
	out, _ := RenderString(d)
	if !strings.Contains(out, "Agent accuracy: ±15%") {
		t.Errorf("agent mode must show ±15%% line")
	}
	if strings.Contains(out, AccuracyDisclosure) {
		t.Errorf("agent mode should not also show sandbox disclosure")
	}
}

func TestRender_NoShareURL_OmitsShareRow(t *testing.T) {
	d := sample()
	d.ShareURL = ""
	out, _ := RenderString(d)
	if strings.Contains(out, "data-copy=") {
		t.Errorf("share UI should not render when ShareURL is empty")
	}
}

func TestRender_NoFindings_CleanState(t *testing.T) {
	d := sample()
	d.Findings = nil
	out, _ := RenderString(d)
	if !strings.Contains(out, "Clean. No findings.") {
		t.Errorf("clean state copy missing:\n%s", out)
	}
}

func TestRender_DeterministicAcrossRuns(t *testing.T) {
	t1 := time.Date(2026, 5, 11, 14, 30, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 11, 14, 30, 45, 0, time.UTC)
	d := sample()
	d.GeneratedAt = t1
	a, _ := RenderString(d)
	d.GeneratedAt = t2
	b, _ := RenderString(d)
	if a != b {
		t.Errorf("renders within the same minute must be byte-equal")
	}
}

func TestRender_CostFindingsLeadBiggestDollar(t *testing.T) {
	d := sample()
	out, _ := RenderString(d)
	cpuIdx := strings.Index(out, "CPU request appears overprovisioned")
	memIdx := strings.Index(out, "Memory request appears overprovisioned")
	if cpuIdx == -1 || memIdx == -1 {
		t.Fatalf("missing finding lines")
	}
	if cpuIdx > memIdx {
		t.Errorf("CPU ($220.40) should render before Memory ($96.40)")
	}
}

func TestRender_WritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	if buf.Len() < 1024 {
		t.Errorf("output suspiciously short: %d bytes", buf.Len())
	}
}

func TestFmtUSD(t *testing.T) {
	for cents, want := range map[int64]string{
		0:          "$0",
		1:          "$0.01",
		100:        "$1",
		12345:      "$123.45",
		12_345_678: "$123,456.78",
	} {
		if got := fmtUSD(cents); got != want {
			t.Errorf("fmtUSD(%d) = %q, want %q", cents, got, want)
		}
	}
}
