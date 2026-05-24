// Package config loads `.optiqor.yaml` for the CLI. Flags override
// config when supplied.
//
// Lookup order (first match wins):
//
//  1. --config <path>
//  2. OPTIQOR_CONFIG env var
//  3. ./.optiqor.yaml in cwd
//  4. zero value
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk schema. Add fields additive-only — old
// configs must keep loading after schema growth.
type Config struct {
	MinSeverity string   `yaml:"min_severity,omitempty"`
	Detectors   []string `yaml:"detectors,omitempty"`
	FailOn      string   `yaml:"fail_on,omitempty"`
	NoColor     bool     `yaml:"no_color,omitempty"`
}

// ConfigName is the default filename Load looks for in the cwd.
const ConfigName = ".optiqor.yaml"

// Load returns the zero Config when no file is present (users opt in
// by creating one). Errors only when an explicit --config /
// OPTIQOR_CONFIG path fails to load.
func Load(explicit string) (Config, error) {
	if explicit != "" {
		return readFile(explicit)
	}
	if env := os.Getenv("OPTIQOR_CONFIG"); env != "" {
		return readFile(env)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, nil
	}
	candidate := filepath.Join(cwd, ConfigName)
	if _, err := os.Stat(candidate); err == nil {
		return readFile(candidate)
	}
	return Config{}, nil
}

func readFile(path string) (Config, error) {
	f, err := os.Open(path) //nolint:gosec // user-specified config path
	if err != nil {
		return Config{}, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Decode(f)
}

// Decode parses YAML config bytes and validates the result.
func Decode(r io.Reader) (Config, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Config{}, fmt.Errorf("config: read: %w", err)
	}
	if len(raw) == 0 {
		return Config{}, nil
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("config: yaml: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate rejects unknown severity / fail-on values.
func (c Config) Validate() error {
	for _, key := range []struct {
		name, value string
	}{
		{"min_severity", c.MinSeverity},
		{"fail_on", c.FailOn},
	} {
		if key.value == "" {
			continue
		}
		switch toLower(key.value) {
		case "low", "med", "medium", "high":
		default:
			return fmt.Errorf("config: %s must be low|med|high (got %q)", key.name, key.value)
		}
	}
	return nil
}

// ErrNotFound is returned only when an explicit config path was
// supplied and the file is missing; default Load returns a zero Config.
var ErrNotFound = errors.New("config: file not found")

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
