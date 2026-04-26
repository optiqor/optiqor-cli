package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRoot_Help just exercises the top-level cobra wiring; ensures we
// can build and serialise the help text without panicking.
func TestRoot_Help(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --help: %v", err)
	}
	for _, want := range []string{
		"analyze",
		"demo",
		accuracyDisclosure,
		"Examples:",
		"sevro analyze",
		"--no-color",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("help missing %q:\n%s", want, buf.String())
		}
	}
}

// TestVersion_Output checks the polished version line.
func TestVersion_Output(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}
	for _, want := range []string{"sevro", "Helm chart cost & security analysis"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("version missing %q:\n%s", want, buf.String())
		}
	}
}

// TestDemo_RunsAndIncludesDisclosure exercises the full demo path
// (embedded fixture → parser → rules → render).
func TestDemo_RunsAndIncludesDisclosure(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--no-color", "demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute demo: %v\n%s", err, buf.String())
	}
	out := buf.String()
	for _, want := range []string{
		"sevro",
		"Helm chart cost",
		"api",
		"worker",
		"±40%",
		"sevro.dev/get",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in demo output:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("--no-color output should be ANSI-free; got:\n%s", out)
	}
}

// TestAnalyze_FixtureFile exercises the analyze command against the
// versioned testdata fixture.
func TestAnalyze_FixtureFile(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--no-color", "analyze", "../../testdata/fixtures/basic-chart/values.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute analyze: %v\n%s", err, buf.String())
	}
	out := buf.String()
	for _, want := range []string{"3 workloads", "2 findings", "HIGH", "MED", "$29.20"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in analyze output:\n%s", want, out)
		}
	}
}

// TestAnalyze_JSONShape exercises --json on a fixture and validates
// the schema is intact.
func TestAnalyze_JSONShape(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"analyze", "--json", "../../testdata/fixtures/basic-chart/values.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --json: %v\n%s", err, buf.String())
	}
	out := buf.String()
	for _, want := range []string{
		`"accuracy_disclosure"`,
		`"workloads_analyzed": 3`,
		`"DetectorID": "cpu-overprovisioned"`,
		`"DetectorID": "missing-memory-limit"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in JSON output:\n%s", want, out)
		}
	}
}

func TestResolveColor_NoColorFlag(t *testing.T) {
	cmd := newRootCmd()
	if got := resolveColor(cmd, true); got {
		t.Error("resolveColor with --no-color should be false")
	}
}

func TestResolveColor_NoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	cmd := newRootCmd()
	if got := resolveColor(cmd, false); got {
		t.Error("resolveColor with NO_COLOR=1 should be false")
	}
}

func TestResolveColor_CLICOLORForceWithoutTTY(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("NO_COLOR", "")
	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{}) // not a *os.File so TTY path is false
	if got := resolveColor(cmd, false); !got {
		t.Error("CLICOLOR_FORCE=1 should override TTY check")
	}
}

func TestResolveColor_NoTTYNoColor(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("NO_COLOR", "")
	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{}) // not a TTY
	if got := resolveColor(cmd, false); got {
		t.Error("buffer (non-TTY) without forces should disable color")
	}
}

func TestAtoi(t *testing.T) {
	cases := map[string]struct {
		want int
		err  bool
	}{
		"":     {0, false},
		"0":    {0, false},
		"123":  {123, false},
		"abc":  {0, true},
		"12x":  {0, true},
		"99":   {99, false},
	}
	for in, tc := range cases {
		got, err := atoi(in)
		if tc.err {
			if err == nil {
				t.Errorf("atoi(%q): expected error", in)
			}
			continue
		}
		if err != nil {
			t.Errorf("atoi(%q): unexpected error: %v", in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("atoi(%q) = %d, want %d", in, got, tc.want)
		}
	}
}
