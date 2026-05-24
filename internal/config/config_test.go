package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	for _, tc := range []struct {
		name     string
		setup    func(t *testing.T) (explicit string)
		wantErr  bool
		wantZero bool
		check    func(t *testing.T, c Config)
	}{
		{
			name: "no file returns zero",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				wd, _ := os.Getwd()
				t.Cleanup(func() { _ = os.Chdir(wd) })
				_ = os.Chdir(dir)
				return ""
			},
			wantZero: true,
		},
		{
			name: "picks up dot optiqor in cwd",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				wd, _ := os.Getwd()
				t.Cleanup(func() { _ = os.Chdir(wd) })
				_ = os.Chdir(dir)
				body := `min_severity: med
detectors:
  - cpu-overprovisioned
  - missing-memory-limit
fail_on: high
no_color: true
`
				if err := os.WriteFile(filepath.Join(dir, ConfigName), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
				return ""
			},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.MinSeverity != "med" || c.FailOn != "high" || !c.NoColor {
					t.Errorf("config not loaded: %+v", c)
				}
				if len(c.Detectors) != 2 {
					t.Errorf("detectors lost: %v", c.Detectors)
				}
			},
		},
		{
			name: "explicit path",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "custom.yaml")
				if err := os.WriteFile(path, []byte("min_severity: high"), 0o600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.MinSeverity != "high" {
					t.Errorf("MinSeverity = %q", c.MinSeverity)
				}
			},
		},
		{
			name:    "explicit missing errors",
			setup:   func(_ *testing.T) string { return "/no/such/path" },
			wantErr: true,
		},
		{
			name: "env var fallback",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "env-config.yaml")
				if err := os.WriteFile(path, []byte("min_severity: low"), 0o600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("OPTIQOR_CONFIG", path)
				return ""
			},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.MinSeverity != "low" {
					t.Errorf("MinSeverity = %q", c.MinSeverity)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			explicit := tc.setup(t)
			c, err := Load(explicit)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if tc.wantZero && !reflect.DeepEqual(c, Config{}) {
				t.Errorf("expected zero, got %+v", c)
			}
			if tc.check != nil {
				tc.check(t, c)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cfg       Config
		wantErrIn string // substring; "" means expect nil
	}{
		{
			name:      "bad severity",
			cfg:       Config{MinSeverity: "noisy"},
			wantErrIn: "min_severity",
		},
		{name: "low", cfg: Config{MinSeverity: "low"}},
		{name: "med", cfg: Config{MinSeverity: "med"}},
		{name: "medium alias", cfg: Config{MinSeverity: "medium"}},
		{name: "high", cfg: Config{MinSeverity: "high"}},
		{name: "uppercase LOW", cfg: Config{MinSeverity: "LOW"}},
		{name: "mixed case Med", cfg: Config{MinSeverity: "Med"}},
		{name: "uppercase HIGH", cfg: Config{MinSeverity: "HIGH"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErrIn != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrIn) {
					t.Fatalf("want error containing %q, got %v", tc.wantErrIn, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate(%q): unexpected error %v", tc.cfg.MinSeverity, err)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		wantErr bool
		check   func(t *testing.T, c Config)
	}{
		{
			name: "empty body yields zero",
			body: "",
			check: func(t *testing.T, c Config) {
				t.Helper()
				if !reflect.DeepEqual(c, Config{}) {
					t.Errorf("got %+v", c)
				}
			},
		},
		{
			name:    "bad yaml errors",
			body:    "not: valid: yaml::",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := Decode(strings.NewReader(tc.body))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected yaml error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.check != nil {
				tc.check(t, c)
			}
		})
	}
}
